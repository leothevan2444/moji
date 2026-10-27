package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

const TaskStageErrorFileReview = "FILE_SELECTION_REQUIRED"
const TaskStageErrorFileSelection = "FILE_SELECTION_FAILED"
const TaskStageErrorMediaPreparation = "MEDIA_PREPARATION_FAILED"

type TorrentFileClient interface {
	GetTorrentContents(context.Context, string, []string) ([]qbittorrent.TorrentContentFile, error)
	SetTorrentFilePriority(context.Context, string, []int, qbittorrent.TorrentFilePriority) error
	PauseTorrents(context.Context, []string) error
	ResumeTorrents(context.Context, []string) error
}

func WithMediaPreparation(provider func() stashsync.IntegrationConfig, preparer mediafiles.Preparer) Option {
	return func(s *Service) {
		s.mediaConfig = provider
		s.mediaPreparer = preparer
		if s.mediaPreparer == nil {
			s.mediaPreparer = mediafiles.FFmpeg{}
		}
		s.mediaContext = context.Background()
		s.mediaJobs = map[string]context.CancelFunc{}
		s.mediaSlots = make(chan struct{}, 1)
	}
}
func (s *Service) SetMediaPreparationContext(ctx context.Context) {
	s.mediaMu.Lock()
	defer s.mediaMu.Unlock()
	s.mediaContext = ctx
}
func (t *Task) FilePlan() mediafiles.Plan {
	var p mediafiles.Plan
	_ = json.Unmarshal([]byte(t.FilePlanJSON), &p)
	if p.Files == nil {
		p.Files = []mediafiles.File{}
	}
	return p
}
func (t *Task) setFilePlan(p mediafiles.Plan) {
	data, _ := json.Marshal(p)
	t.FilePlanJSON = string(data)
}
func (s *Service) torrentFiles() (TorrentFileClient, error) {
	client, ok := s.qbt.(TorrentFileClient)
	if !ok {
		return nil, errors.New("qBittorrent file-selection API is unavailable")
	}
	return client, nil
}
func (s *Service) updateFilePolicy(p *mediafiles.Plan) {
	keep := s.mediaConfig().KeepSubtitles
	if p.KeepSubtitles != keep {
		p.KeepSubtitles = keep
		p.Applied = false
		for i := range p.Files {
			if p.Files[i].Kind == mediafiles.Subtitle && p.Files[i].Reason != "SUBTITLE_OTHER_RELEASE" {
				p.Files[i].Selected = keep
			}
		}
	}
}
func (s *Service) applyFileSelection(ctx context.Context, t *Task, p *mediafiles.Plan) error {
	client, err := s.torrentFiles()
	if err != nil {
		return err
	}
	// Persist the chosen order/policy and unapplied checkpoint before qB can
	// pause, change priorities or resume. A failed acknowledgment is retried
	// from this durable intent; these qB settings are idempotent.
	p.Applied = false
	if p.Validate() == nil && t.Stage == TaskStageDownloading {
		setTaskStage(t, TaskStageDownloading, TaskStageStatusRunning)
		clearTaskStageError(t)
	}
	t.setFilePlan(*p)
	t.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, t); err != nil {
		return fmt.Errorf("persist file selection before qB changes: %w", err)
	}
	if err := client.PauseTorrents(ctx, []string{t.TorrentHash}); err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}
	selected, discarded := []int{}, []int{}
	for _, entry := range p.Files {
		if entry.Selected {
			selected = append(selected, entry.Index)
		} else {
			discarded = append(discarded, entry.Index)
		}
	}
	if len(discarded) > 0 {
		if err := client.SetTorrentFilePriority(ctx, t.TorrentHash, discarded, qbittorrent.TorrentFilePriorityDontDownload); err != nil {
			return err
		}
	}
	if len(selected) > 0 {
		if err := client.SetTorrentFilePriority(ctx, t.TorrentHash, selected, qbittorrent.TorrentFilePriorityNormal); err != nil {
			return err
		}
	}
	if !p.PauseRequested {
		if err := client.ResumeTorrents(ctx, []string{t.TorrentHash}); err != nil {
			return err
		}
	}
	p.Applied = true
	return nil
}

// qB metadata may not exist on the first tick for a magnet. Do not infer file
// contents or mark the task downloaded until a file plan has been applied.
func (s *Service) syncFileSelection(ctx context.Context, t *Task) (bool, error) {
	client, err := s.torrentFiles()
	if err != nil {
		return false, err
	}
	files, err := client.GetTorrentContents(ctx, t.TorrentHash, nil)
	if err != nil {
		return false, err
	}
	if len(files) == 0 {
		return false, nil
	}
	p := t.FilePlan()
	s.updateFilePolicy(&p)
	if len(p.Files) == 0 {
		entries := make([]mediafiles.File, 0, len(files))
		for _, file := range files {
			entries = append(entries, mediafiles.File{Index: file.Index, Path: file.Name, Size: file.Size})
		}
		classified := mediafiles.Classify(t.Code, entries, p.KeepSubtitles)
		classified.PauseRequested = p.PauseRequested
		p = classified
	}
	if !p.Applied {
		if err := s.applyFileSelection(ctx, t, &p); err != nil {
			t.setFilePlan(p)
			return false, err
		}
	}
	t.setFilePlan(p)
	complete := true
	wanted := map[int]bool{}
	for _, entry := range p.Files {
		if entry.Selected {
			wanted[entry.Index] = true
		}
	}
	found := 0
	var total, done float64
	for _, file := range files {
		if wanted[file.Index] {
			found++
			total += float64(file.Size)
			done += float64(file.Size) * file.Progress
			if file.Progress < 1 {
				complete = false
			}
		}
	}
	if found != len(wanted) {
		return false, errors.New("qBittorrent file list changed after selection")
	}
	if total > 0 {
		t.Progress = done / total
	}
	return complete, nil
}

// ResolveTaskFiles accepts file indices in the requested playback order.
func (s *Service) ResolveTaskFiles(ctx context.Context, id string, indices []int) (*Task, error) {
	unlock := s.lockTask(id)
	defer unlock()
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.RestartStateJSON != "" || s.mediaConfig == nil || task.Stage != TaskStageDownloading || task.StageStatus != TaskStageStatusBlocked {
		return nil, errors.New("task is not awaiting download file selection")
	}
	p := task.FilePlan()
	s.updateFilePolicy(&p)
	if err := p.Select(indices); err != nil {
		return nil, err
	}
	if err := s.applyFileSelection(ctx, task, &p); err != nil {
		task.setFilePlan(p)
		blockTask(task, TaskStageErrorFileSelection, err.Error(), s.now().UTC())
		_ = s.store.Update(ctx, task)
		return task, err
	}
	task.setFilePlan(p)
	clearTaskStageError(task)
	setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
	task.UpdatedAt = s.now().UTC()
	return task, s.store.Update(ctx, task)
}
func (s *Service) retryFileDownload(ctx context.Context, task *Task) (*Task, error) {
	next := cloneTask(task)
	p := next.FilePlan()
	s.updateFilePolicy(&p)
	p.Applied = false
	next.setFilePlan(p)
	setTaskStage(next, TaskStageDownloading, TaskStageStatusRunning)
	clearTaskStageError(next)
	_, err := s.syncFileSelection(ctx, next)
	if err != nil {
		code := TaskStageErrorFileSelection
		if errors.Is(err, mediafiles.ErrReview) {
			code = TaskStageErrorFileReview
		}
		blockTask(next, code, err.Error(), s.now().UTC())
	}
	if persist := s.store.Update(ctx, next); persist != nil {
		return next, persist
	}
	return next, err
}
func (s *Service) startMediaPreparation(ctx context.Context, task *Task) (*Task, error) {
	s.mediaMu.Lock()
	_, active := s.mediaJobs[task.ID]
	s.mediaMu.Unlock()
	if active {
		return task, nil
	}
	cfg := s.mediaConfig()
	if cfg.Transfer.Action != stashsync.TransferActionCopy && cfg.Transfer.Action != stashsync.TransferActionMove {
		return s.blockMediaPreparation(task, errors.New("clean delivery requires COPY or MOVE"))
	}
	if strings.TrimSpace(cfg.Downloads.MojiRoot) == "" || strings.TrimSpace(cfg.Library.MojiRoot) == "" || strings.TrimSpace(cfg.Library.StashRoot) == "" {
		return s.blockMediaPreparation(task, errors.New("configure Moji download, Moji library, and Stash library roots"))
	}
	relative, err := relativePathWithin(cfg.Downloads.QBRoot, task.SavePath)
	if err != nil {
		return s.blockMediaPreparation(task, err)
	}
	p := task.FilePlan()
	s.updateFilePolicy(&p)
	task.setFilePlan(p)
	if len(p.Files) == 0 || !p.Applied {
		complete, err := s.syncFileSelection(ctx, task)
		if err != nil {
			setTaskStage(task, TaskStageDownloading, TaskStageStatusBlocked)
			code := TaskStageErrorFileSelection
			if errors.Is(err, mediafiles.ErrReview) {
				code = TaskStageErrorFileReview
			}
			blockTask(task, code, err.Error(), s.now().UTC())
			_ = s.store.Update(ctx, task)
			return task, err
		}
		p = task.FilePlan()
		if !complete {
			setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
			task.DownloadCompletedAt = nil
			clearTaskStageError(task)
			return task, s.store.Update(ctx, task)
		}
	}
	if err := p.Validate(); err != nil {
		setTaskStage(task, TaskStageDownloading, TaskStageStatusBlocked)
		blockTask(task, TaskStageErrorFileReview, err.Error(), s.now().UTC())
		_ = s.store.Update(ctx, task)
		return task, err
	}
	s.mediaMu.Lock()
	if _, running := s.mediaJobs[task.ID]; running {
		s.mediaMu.Unlock()
		return task, nil
	}
	select {
	case s.mediaSlots <- struct{}{}:
	default:
		s.mediaMu.Unlock()
		setTaskStage(task, TaskStageTransferring, TaskStageStatusPending)
		clearTaskStageError(task)
		task.UpdatedAt = s.now().UTC()
		return task, s.store.Update(ctx, task)
	}
	workerCtx, cancel := context.WithTimeout(s.mediaContext, 6*time.Hour)
	s.mediaJobs[task.ID] = cancel
	s.mediaWG.Add(1)
	s.mediaMu.Unlock()
	setTaskStage(task, TaskStageTransferring, TaskStageStatusRunning)
	clearTaskStageError(task)
	task.TransferError = ""
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		s.finishMediaJob(task.ID, cancel)
		s.mediaWG.Done()
		<-s.mediaSlots
		return task, err
	}
	snapshot := cloneTask(task)
	request := mediafiles.Request{TaskID: task.ID, Code: task.Code, SourceRoot: filepath.Join(cfg.Downloads.MojiRoot, relative), TargetRoot: cfg.Library.MojiRoot, Action: string(cfg.Transfer.Action), Plan: p}
	go func() {
		defer s.mediaWG.Done()
		// Clear the in-flight marker before releasing the task lock, so an immediate
		// retry cannot observe a finished worker and leave a PENDING task stranded.
		var unlock func()
		defer func() {
			if unlock != nil {
				unlock()
			}
		}()
		defer s.finishMediaJob(snapshot.ID, cancel)
		defer func() { <-s.mediaSlots }()
		var result mediafiles.Result
		var prepErr error
		if request.Action == "MOVE" && snapshot.TorrentHash != "" {
			client, err := s.torrentFiles()
			if err != nil {
				prepErr = err
			} else {
				prepErr = client.PauseTorrents(workerCtx, []string{snapshot.TorrentHash})
			}
		}
		if prepErr == nil {
			result, prepErr = s.mediaPreparer.Prepare(workerCtx, request)
		}
		// During shutdown leave the persisted RUNNING checkpoint for restart recovery.
		if errors.Is(workerCtx.Err(), context.Canceled) {
			return
		}
		if workerCtx.Err() != nil {
			prepErr = workerCtx.Err()
		}
		unlock = s.lockTask(snapshot.ID)
		latest, findErr := s.store.Find(context.Background(), snapshot.ID)
		if findErr != nil || latest.Stage != TaskStageTransferring || latest.Generation != snapshot.Generation {
			return
		}
		if prepErr != nil {
			_, _ = s.blockMediaPreparation(latest, prepErr)
			return
		}
		latest.TransferAction = string(cfg.Transfer.Action)
		latest.MojiSourcePath = request.SourceRoot
		latest.MojiTransferPath = result.Directory
		latest.StashScanPath = filepath.Join(cfg.Library.StashRoot, latest.Code)
		latest.StashIngestPhase = StashIngestScan
		latest.StashScanJobID = ""
		latest.TransferError = ""
		clearTaskStageError(latest)
		setTaskStage(latest, TaskStageScanning, TaskStageStatusPending)
		latest.UpdatedAt = s.now().UTC()
		if err := s.store.Update(context.Background(), latest); err != nil {
			logging.Errorf("taskruntime: persist prepared movie task=%s: %v", latest.ID, err)
		}
	}()
	return task, nil
}
func (s *Service) finishMediaJob(id string, cancel context.CancelFunc) {
	cancel()
	s.mediaMu.Lock()
	delete(s.mediaJobs, id)
	s.mediaMu.Unlock()
}
func (s *Service) blockMediaPreparation(t *Task, err error) (*Task, error) {
	logging.Errorf("taskruntime: media preparation failed task=%s: %v", t.ID, err)
	setTaskStage(t, TaskStageTransferring, TaskStageStatusBlocked)
	t.TransferError = err.Error()
	blockTask(t, TaskStageErrorMediaPreparation, err.Error(), s.now().UTC())
	persist := s.store.Update(context.Background(), t)
	if persist != nil {
		return t, fmt.Errorf("%w (persist: %v)", err, persist)
	}
	return t, err
}
