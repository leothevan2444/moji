package indexerdefs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/storage"
)

// Service owns update jobs and leases for one Manager generation. Start and
// Close belong to the application lifecycle, not an HTTP request lifecycle.
type Service struct {
	streamID     string
	repo         *repository
	root         string
	source       Source
	validator    Validator
	mu           sync.Mutex
	checkMu      sync.Mutex
	leaseMu      sync.Mutex
	cancel       context.CancelFunc
	jobCancel    context.CancelFunc
	jobID        string
	done         chan struct{}
	wake         chan struct{}
	running      bool
	active       *Bundle
	loadError    string
	leases       map[string]int
	events       map[chan struct{}]struct{}
	eventsClosed bool
}

// A validator is mandatory, including when the current configuration has no
// built-in instances: its guard still fences concurrent configuration changes.
func NewService(manager *storage.Manager, source Source, validator Validator) (*Service, error) {
	if manager == nil || manager.Core == nil || source == nil || validator == nil {
		return nil, ErrUnavailable
	}
	return &Service{streamID: uuid.NewString(), repo: &repository{db: manager.Core}, root: filepath.Join(filepath.Dir(manager.Core.Path), "indexers", "bundles"), source: source, validator: validator, wake: make(chan struct{}, 1), leases: map[string]int{}, events: map[chan struct{}]struct{}{}}, nil
}

func (s *Service) Start(parent context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running || s.done != nil {
		return ErrBusy
	}
	if err := s.repo.recoverInterrupted(parent, time.Now()); err != nil {
		return err
	}
	state, err := s.repo.state(parent)
	if err != nil {
		return err
	}
	s.active = nil
	s.loadError = ""
	assetErr := s.cleanupTemporary(parent)
	if assetErr != nil {
		s.loadError = ErrUnavailable.Error()
	}
	if state.Active != "" && assetErr == nil {
		_, checksum, err := s.repo.manifest(parent, state.Active)
		if err == nil {
			s.active, err = LoadBundle(parent, filepath.Join(s.root, state.Active), checksum)
		}
		if err != nil {
			s.loadError = ErrUnavailable.Error()
		}
	}
	if err := parent.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.running = true
	go s.run(ctx, s.done)
	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.running = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	s.CloseSubscriptions()
}

// CloseSubscriptions releases WebSockets before the HTTP owner waits for
// requests. Worker shutdown and resource lease release remain separate.
func (s *Service) CloseSubscriptions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventsClosed = true
	for ch := range s.events {
		close(ch)
		delete(s.events, ch)
	}
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, err := s.repo.snapshot(ctx)
	if err != nil {
		return Status{}, err
	}
	status.StreamID = s.streamID
	status.Available = s.active != nil && s.active.Manifest.Revision == status.State.Active
	status.ErrorCode = s.loadError
	if status.State.Active != "" && !status.Available && status.ErrorCode == "" {
		status.ErrorCode = ErrUnavailable.Error()
	}
	if status.State.Active == "" {
		status.ErrorCode = ErrNotInstalled.Error()
	}
	return status, nil
}

// CurrentJob reads the latest persisted lifecycle without checking for updates.
func (s *Service) CurrentJob(ctx context.Context) (*Job, error) { return s.repo.currentJob(ctx) }

func (s *Service) Job(ctx context.Context, id string) (*Job, error) { return s.repo.job(ctx, id) }

// Subscribe coalesces changes; consumers fetch the latest persisted snapshot.
// The initial signal permits reconnect without relying on an event history.
func (s *Service) Subscribe() (<-chan struct{}, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.eventsClosed {
		return nil, nil, ErrNotRunning
	}
	if len(s.events) >= 128 {
		return nil, nil, ErrBusy
	}
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	s.events[ch] = struct{}{}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			if _, ok := s.events[ch]; ok {
				delete(s.events, ch)
				close(ch)
			}
			s.mu.Unlock()
		})
	}, nil
}

func (s *Service) emit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.events {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Check only discovers updates. Installing a discovered revision is a separate
// persisted job, so default periodic checks cannot silently change rules.
func (s *Service) Check(ctx context.Context) (State, error) {
	s.checkMu.Lock()
	defer s.checkMu.Unlock()
	return s.checkLocked(ctx)
}

// CheckThrottled is the interactive boundary. Concurrent browser refreshes
// share the last persisted check for at least one minute, including failures.
func (s *Service) CheckThrottled(ctx context.Context) (State, error) {
	s.checkMu.Lock()
	defer s.checkMu.Unlock()
	state, err := s.repo.state(ctx)
	if err != nil {
		return State{}, err
	}
	if state.LastCheckedAt != nil && time.Since(*state.LastCheckedAt) < time.Minute {
		return state, nil
	}
	return s.checkLocked(ctx)
}

func (s *Service) checkLocked(ctx context.Context) (State, error) {
	state, err := s.repo.state(ctx)
	if err != nil {
		return State{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	revision, unchanged, err := s.source.Latest(requestCtx, state.ETag)
	if err != nil {
		if ctx.Err() != nil {
			return state, ctx.Err()
		}
		if saveErr := s.repo.recordCheck(ctx, "", "", errorCode(err), time.Now()); saveErr != nil {
			return state, saveErr
		}
		s.emit()
		return state, err
	}
	if unchanged {
		revision.Commit = state.Available
		revision.ETag = state.ETag
	}
	if !ValidRevision(revision.Commit) {
		return state, ErrSource
	}
	if err := s.repo.recordCheck(ctx, revision.Commit, revision.ETag, "", time.Now()); err != nil {
		return state, err
	}
	s.emit()
	return s.repo.state(ctx)
}

// Begin returns an existing identical active job for duplicate submissions.
// A different concurrent operation returns that job with ErrBusy.
func (s *Service) Begin(ctx context.Context, kind, target, expectedActive string) (*Job, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil, ErrNotRunning
	}
	current, err := s.repo.currentJob(ctx)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if current != nil && !current.State.Terminal() {
		s.mu.Unlock()
		if current.Kind == kind && (target == "" || target == current.Target) && current.ExpectedActive == expectedActive {
			return current, nil
		}
		return current, ErrBusy
	}
	state, err := s.repo.state(ctx)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if state.Active != expectedActive {
		s.mu.Unlock()
		return nil, ErrConflict
	}
	switch kind {
	case "INSTALL", "UPDATE":
		if kind == "INSTALL" && state.Active != "" || kind == "UPDATE" && state.Active == "" {
			s.mu.Unlock()
			return nil, ErrConflict
		}
		if target == "" {
			target = state.Available
		}
		if target != state.Available || !ValidRevision(target) || target == state.Active {
			s.mu.Unlock()
			return nil, ErrConflict
		}
	case "ROLLBACK":
		if state.Previous == "" {
			s.mu.Unlock()
			return nil, ErrNoPrevious
		}
		if target == "" {
			target = state.Previous
		}
		if target != state.Previous {
			s.mu.Unlock()
			return nil, ErrConflict
		}
	case "REPAIR":
		if state.Active == "" {
			s.mu.Unlock()
			return nil, ErrNotInstalled
		}
		if target == "" {
			target = state.Active
		}
		if target != state.Active {
			s.mu.Unlock()
			return nil, ErrConflict
		}
	default:
		s.mu.Unlock()
		return nil, ErrConflict
	}
	job := &Job{ID: uuid.NewString(), Kind: kind, State: Queued, Target: target, ExpectedActive: expectedActive, CorrelationID: uuid.NewString(), StartedAt: time.Now().UTC()}
	err = s.repo.begin(ctx, job)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	s.emit()
	return job, nil
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	s.mu.Lock()
	err := s.repo.cancel(ctx, id, time.Now())
	if err == nil && s.jobID == id && s.jobCancel != nil {
		s.jobCancel()
	}
	s.mu.Unlock()
	if err == nil {
		s.emit()
	}
	return err
}

func (s *Service) run(ctx context.Context, done chan struct{}) {
	defer func() { s.mu.Lock(); s.running = false; s.mu.Unlock(); close(done) }()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := s.repo.state(ctx)
			if err == nil && state.Active != "" && (state.LastCheckedAt == nil || time.Since(*state.LastCheckedAt) >= 24*time.Hour) {
				_, _ = s.Check(ctx)
			}
		case <-s.wake:
			job, err := s.repo.currentJob(ctx)
			if err != nil || job == nil || job.State != Queued {
				continue
			}
			s.process(ctx, job)
		}
	}
}

func (s *Service) transition(ctx context.Context, job *Job, state JobState) error {
	job.State = state
	if err := s.repo.updateJob(ctx, job); err != nil {
		return err
	}
	s.emit()
	return nil
}

func (s *Service) process(parent context.Context, job *Job) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	s.mu.Lock()
	s.jobID = job.ID
	s.jobCancel = cancel
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); s.jobID = ""; s.jobCancel = nil; s.mu.Unlock() }()
	err := s.execute(ctx, job)
	if err == nil {
		return
	}
	// Shutdown cleanup has a bounded lifetime and only records interruption. It
	// cannot continue downloads or start a new activation after app cancellation.
	cleanup, cancelCleanup := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
	defer cancelCleanup()
	current, readErr := s.repo.job(cleanup, job.ID)
	if readErr != nil || current == nil || current.State.Terminal() {
		return
	}
	current.State = Failed
	current.ErrorCode = errorCode(err)
	if parent.Err() != nil {
		current.State = Interrupted
		current.ErrorCode = "DEFINITION_UPDATE_INTERRUPTED"
	}
	now := time.Now().UTC()
	current.FinishedAt = &now
	if s.repo.updateJob(cleanup, current) == nil {
		s.emit()
	}
}

func (s *Service) execute(ctx context.Context, job *Job) error {
	var bundle *Bundle
	var staged *Bundle
	if job.Kind == "ROLLBACK" {
		if err := s.transition(ctx, job, Validating); err != nil {
			return err
		}
		_, checksum, err := s.repo.manifest(ctx, job.Target)
		if err != nil {
			return err
		}
		bundle, err = LoadBundle(ctx, filepath.Join(s.root, job.Target), checksum)
		if err != nil {
			return err
		}
	} else {
		if err := s.Prune(ctx); err != nil {
			return err
		}
		usedBeforeDownload, err := s.storageUsage(ctx)
		if err != nil {
			return err
		}
		downloadLimit := min(MaxArchiveBytes, MaxStoredBytes-usedBeforeDownload)
		if downloadLimit <= 0 {
			return ErrBudget
		}
		if err := s.transition(ctx, job, Downloading); err != nil {
			return err
		}
		if err := s.ensureRoot(); err != nil {
			return err
		}
		archive, err := os.CreateTemp(s.root, ".download-*.tar.gz")
		if err != nil {
			return err
		}
		defer func() { archive.Close(); os.Remove(archive.Name()) }()
		writer := &progressWriter{writer: archive, limit: downloadLimit, progress: func(count int64) error {
			job.DownloadedBytes = count
			if err := s.repo.updateJob(ctx, job); err != nil {
				return err
			}
			s.emit()
			return nil
		}}
		if err := s.source.Download(ctx, job.Target, writer); err != nil {
			return err
		}
		job.DownloadedBytes = writer.count
		if err := archive.Sync(); err != nil {
			return err
		}
		if _, err := archive.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if err := s.transition(ctx, job, Validating); err != nil {
			return err
		}
		used, err := s.storageUsage(ctx)
		if err != nil {
			return err
		}
		stage, err := stageArchive(ctx, s.root, job.Target, archive, time.Now(), MaxStoredBytes-used)
		if err != nil {
			return err
		}
		defer func() {
			if filepath.Base(stage.Directory) != stage.Manifest.Revision {
				os.RemoveAll(stage.Directory)
			}
		}()
		if err := s.checkStorageBudget(ctx); err != nil {
			return err
		}
		bundle, staged = stage, stage
	}
	if err := s.transition(ctx, job, CheckingInstances); err != nil {
		return err
	}
	guard, err := s.validator.Validate(ctx, bundle)
	if err != nil {
		return err
	}
	if guard == nil {
		return ErrIncompatible
	}
	if staged != nil {
		var err error
		if job.Kind == "REPAIR" {
			bundle, err = s.repair(ctx, staged)
		} else {
			s.leaseMu.Lock()
			bundle, err = Publish(ctx, s.root, staged)
			s.leaseMu.Unlock()
		}
		if err != nil {
			return err
		}
	}
	activate := func() error {
		// Cancellation and the start of activation share this lock. A user
		// cancellation either wins durably or sees ACTIVATING/COMPLETED.
		s.mu.Lock()
		defer s.mu.Unlock()
		job.State = Activating
		if err := s.repo.updateJob(ctx, job); err != nil {
			return err
		}
		if err := s.repo.activate(ctx, job, bundle, time.Now()); err != nil {
			// A cancelled Commit can have an ambiguous return. Reconcile the
			// durable job and pointer before reporting failure or serving old plans.
			check, cancelCheck := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			committed, readErr := s.repo.activationCommitted(check, job.ID, bundle.Manifest.Revision)
			cancelCheck()
			if readErr != nil || !committed {
				return err
			}
		}
		s.active = bundle
		s.loadError = ""
		return nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		err = guard.WithCurrent(ctx, activate)
		if !errors.Is(err, ErrConflict) || job.State != CheckingInstances || attempt == 2 {
			break
		}
		// Configuration changed while compiling/publishing. Revalidate against
		// the new revision, with a finite bound if settings are changing rapidly.
		guard, err = s.validator.Validate(ctx, bundle)
		if err != nil {
			break
		}
		if guard == nil {
			err = ErrIncompatible
			break
		}
	}
	if err == nil {
		s.emit()
	}
	return err
}

type progressWriter struct {
	writer   io.Writer
	limit    int64
	count    int64
	last     int64
	at       time.Time
	progress func(int64) error
}

func (w *progressWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > w.limit-w.count {
		return 0, ErrBudget
	}
	n, err := w.writer.Write(data)
	w.count += int64(n)
	if err == nil && w.count-w.last >= 128<<10 && time.Since(w.at) >= time.Second {
		err = w.progress(w.count)
		w.last = w.count
		w.at = time.Now()
	}
	return n, err
}

func errorCode(err error) string {
	for _, known := range []error{ErrNotInstalled, ErrBusy, ErrConflict, ErrInvalidBundle, ErrIncompatible, ErrUnavailable, ErrNotRunning, ErrNoPrevious, ErrBudget, ErrSource, ErrJobTerminal, ErrCancelling} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "DEFINITION_UPDATE_TIMEOUT"
	}
	return "DEFINITION_UPDATE_FAILED"
}
