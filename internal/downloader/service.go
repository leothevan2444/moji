// Package downloader owns qB monitoring and retention independently of media ingest.
package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	_ "modernc.org/sqlite"
)

var (
	ErrObservationPending = errors.New("command accepted; observation pending")
	ErrInvalidPolicy      = errors.New("invalid retention thresholds")
	ErrInvalidLimit       = errors.New("invalid speed limit")
	ErrNotFound           = errors.New("torrent no longer exists")
	ErrIngestActive       = errors.New("ingest is not complete")
	ErrUnsafeFiles        = errors.New("independent library copy is not verified")
)

type Client interface {
	SetDownloadSpeedLimit(context.Context, bool, int64) error
	GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error)
	GetDownloadServerState(context.Context) (*qbittorrent.DownloadServerState, error)
	PauseTorrents(context.Context, []string) error
	ResumeTorrents(context.Context, []string) error
	RecheckTorrents(context.Context, []string) error
	ReannounceTorrents(context.Context, []string) error
	DeleteTorrents(context.Context, []string, bool) error
}
type Tasks interface {
	ListTasks(context.Context) ([]*taskruntime.Task, error)
}

const SyncInterval = 3 * time.Second

func DefaultPolicy() Policy { return Policy{DeleteFiles: true} }

type Policy struct {
	AfterIngest bool
	Enabled     bool
	Ratio       float64
	SeedHours   float64
	MatchAll    bool
	DeleteFiles bool
}
type Torrent struct {
	Hash           string
	Name           string
	State          string
	Present        bool
	TaskIDs        []string
	TaskCodes      []string
	AllCompleted   bool
	CanDeleteFiles bool
	Progress       float64
	DownloadSpeed  int64
	UploadSpeed    int64
	Size           int64
	CompletedBytes int64
	Downloaded     int64
	Uploaded       int64
	Ratio          float64
	SeedingSeconds int64
	ETA            int64
	Seeds          int64
	Peers          int64
	Availability   float64
	SavePath       string
	Category       string
	Tags           string
	LastSeenAt     string
	RemovedAt      string
	RemovalReason  string
	CleanupError   bool
}
type Snapshot struct {
	CheckedAt        string
	Connected        bool
	StorageAvailable bool
	FreeSpace        *int64
	DownloadSpeed    int64
	UploadSpeed      int64
	DownloadLimit    int64
	UploadLimit      int64
	ConnectionStatus string
	DHTNodes         int64
	CompletedBytes   int64
	RemainingBytes   int64
	Torrents         []Torrent
}
type Service struct {
	mu           sync.Mutex
	stateMu      sync.RWMutex
	snapshot     Snapshot
	observations chan []qbittorrent.Torrent
	groupsFailed bool
	db           *sql.DB
	client       Client
	tasks        Tasks
	now          func() time.Time
}

func New(client Client, tasks Tasks, path string) (*Service, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS downloader_state (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, err
	}
	s := &Service{db: db, client: client, tasks: tasks, now: time.Now, observations: make(chan []qbittorrent.Torrent, 1)}
	if err := s.read(context.Background(), "snapshot", &s.snapshot); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Service) Close() error { return s.db.Close() }
func (s *Service) read(ctx context.Context, key string, out any) error {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM downloader_state WHERE key=?`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}
func (s *Service) write(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO downloader_state(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, string(raw))
	if err == nil && key == "snapshot" {
		s.cacheSnapshot(value.(Snapshot))
	}
	return err
}
func (s *Service) cacheSnapshot(snapshot Snapshot) {
	s.stateMu.Lock()
	s.snapshot = cloneSnapshot(snapshot)
	s.stateMu.Unlock()
}

// Observations remain available even if persisting the restart snapshot fails.
// Mutation intents still use write directly and require durable storage first.
func (s *Service) publishSnapshot(ctx context.Context, snapshot Snapshot) error {
	s.cacheSnapshot(snapshot)
	if err := s.write(ctx, "snapshot", snapshot); err != nil {
		return fmt.Errorf("persist downloader snapshot: %w", err)
	}
	return nil
}

func (s *Service) Policy(ctx context.Context) (Policy, error) {
	p := DefaultPolicy()
	err := s.read(ctx, "policy", &p)
	return p, err
}
func (s *Service) SetPolicy(ctx context.Context, p Policy) error {
	if math.IsNaN(p.Ratio) || math.IsInf(p.Ratio, 0) || math.IsNaN(p.SeedHours) || math.IsInf(p.SeedHours, 0) || p.Ratio < 0 || p.Ratio > 9999 || p.SeedHours < 0 || p.SeedHours > 87600 || (p.Enabled && !p.AfterIngest && p.Ratio == 0 && p.SeedHours == 0) {
		return ErrInvalidPolicy
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.write(ctx, "policy", p)
}

// Snapshot reads the last observation without waiting for network calls or task writes.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return cloneSnapshot(s.snapshot), nil
}
func cloneSnapshot(v Snapshot) Snapshot {
	v.Torrents = append([]Torrent{}, v.Torrents...)
	for i := range v.Torrents {
		v.Torrents[i].TaskIDs = append([]string{}, v.Torrents[i].TaskIDs...)
		v.Torrents[i].TaskCodes = append([]string{}, v.Torrents[i].TaskCodes...)
	}
	if v.FreeSpace != nil {
		space := *v.FreeSpace
		v.FreeSpace = &space
	}
	return v
}
func taskHash(t *taskruntime.Task) string {
	if t.TorrentHash != "" {
		return strings.ToLower(t.TorrentHash)
	}
	return strings.ToLower(t.TorrentIdentityHash)
}
func (s *Service) taskGroups(ctx context.Context) (map[string][]*taskruntime.Task, error) {
	groups := map[string][]*taskruntime.Task{}
	if s.tasks == nil {
		return groups, nil
	}
	tasks, err := s.tasks.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if h := taskHash(t); h != "" {
			groups[h] = append(groups[h], t)
		}
	}
	return groups, nil
}
func independentDelivery(t *taskruntime.Task) bool {
	if t.Stage != taskruntime.TaskStageCompleted || t.StageStatus != taskruntime.TaskStageStatusDone || (!strings.EqualFold(t.TransferAction, "copy") && !strings.EqualFold(t.TransferAction, "move")) || t.MojiSourcePath == "" || t.MojiTransferPath == "" {
		return false
	}
	// Resolve local mounts and symlinks before allowing qB to remove source data.
	source, e1 := filepath.EvalSymlinks(t.MojiSourcePath)
	target, e2 := filepath.EvalSymlinks(t.MojiTransferPath)
	if e1 != nil || e2 != nil {
		return false
	}
	return !within(source, target) && !within(target, source)
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func attach(t *Torrent, tasks []*taskruntime.Task) {
	t.TaskIDs = []string{}
	t.TaskCodes = []string{}
	t.AllCompleted = len(tasks) > 0
	t.CanDeleteFiles = len(tasks) > 0
	for _, task := range tasks {
		t.TaskIDs = append(t.TaskIDs, task.ID)
		t.TaskCodes = append(t.TaskCodes, task.Code)
		t.AllCompleted = t.AllCompleted && task.Stage == taskruntime.TaskStageCompleted && task.StageStatus == taskruntime.TaskStageStatusDone
		t.CanDeleteFiles = t.CanDeleteFiles && independentDelivery(task)
	}
}
func threshold(p Policy, t Torrent) bool {
	if p.AfterIngest {
		return true
	}
	checks := []bool{}
	if p.Ratio > 0 {
		checks = append(checks, t.Ratio >= p.Ratio)
	}
	if p.SeedHours > 0 {
		checks = append(checks, float64(t.SeedingSeconds) >= p.SeedHours*3600)
	}
	if len(checks) == 0 {
		return false
	}
	result := p.MatchAll
	for _, c := range checks {
		if p.MatchAll {
			result = result && c
		} else {
			result = result || c
		}
	}
	return result
}
func (s *Service) refresh(ctx context.Context, cleanup bool) error {
	previous, err := s.Snapshot(ctx)
	if err != nil {
		return err
	}
	torrents, err := s.client.GetTorrentList(ctx, nil)
	if err != nil {
		previous.Connected = false
		if e := s.publishSnapshot(ctx, previous); e != nil {
			return e
		}
		return fmt.Errorf("qB torrent listing: %w", err)
	}
	// Task database work has its own budget and cannot discard a successful qB listing.
	groupCtx, cancelGroups := context.WithTimeout(ctx, time.Second)
	groups, groupErr := s.taskGroups(groupCtx)
	cancelGroups()
	if groupErr != nil && !s.groupsFailed {
		logging.Warnf("downloader: task associations unavailable; cleanup suspended: %v", groupErr)
	}
	if groupErr == nil && s.groupsFailed {
		logging.Infof("downloader: task associations recovered")
	}
	s.groupsFailed = groupErr != nil
	policy := DefaultPolicy()
	policyCtx, cancelPolicy := context.WithTimeout(ctx, time.Second)
	policyErr := s.read(policyCtx, "policy", &policy)
	cancelPolicy()
	if policyErr != nil {
		cleanup = false
	}
	now := s.now().UTC().Format(time.RFC3339)
	out := Snapshot{Connected: true, CheckedAt: now, Torrents: []Torrent{}}
	// Storage failures do not turn a successful torrent listing into mass deletion.
	serverCtx, cancelServer := context.WithTimeout(ctx, 3*time.Second)
	server, serverErr := s.client.GetDownloadServerState(serverCtx)
	cancelServer()
	if serverErr == nil && server != nil {
		out.StorageAvailable = true
		out.FreeSpace = server.FreeSpace
		if out.FreeSpace != nil && *out.FreeSpace < 0 {
			out.FreeSpace = nil
		}
		out.DownloadSpeed = server.DownloadSpeed
		out.UploadSpeed = server.UploadSpeed
		out.DownloadLimit = server.DownloadLimit
		out.UploadLimit = server.UploadLimit
		out.ConnectionStatus = server.ConnectionStatus
		out.DHTNodes = server.DHTNodes
	}
	seen := map[string]bool{}
	old := map[string]Torrent{}
	for _, t := range previous.Torrents {
		old[t.Hash] = t
	}
	for _, raw := range torrents {
		hash := strings.ToLower(raw.Hash)
		seen[hash] = true
		t := Torrent{Hash: hash, Name: raw.Name, State: string(raw.State), Present: true, Progress: raw.Progress, DownloadSpeed: raw.DLSpeed, UploadSpeed: raw.UPSpeed, Size: raw.Size, CompletedBytes: raw.Completed, Downloaded: raw.Downloaded, Uploaded: raw.Uploaded, Ratio: raw.Ratio, SeedingSeconds: raw.SeedingTime, ETA: raw.ETA, Seeds: raw.NumSeeds, Peers: raw.NumLeechs, Availability: raw.Availability, SavePath: raw.SavePath, Category: raw.Category, Tags: raw.Tags, LastSeenAt: now}
		if prev, exists := old[hash]; exists && (prev.RemovalReason == "MANUAL_PENDING" || prev.RemovalReason == "POLICY_PENDING") {
			t.RemovalReason = prev.RemovalReason
		}
		attach(&t, groups[hash])
		if groupErr != nil {
			t.TaskIDs = append([]string{}, old[hash].TaskIDs...)
			t.TaskCodes = append([]string{}, old[hash].TaskCodes...)
		}
		t.CanDeleteFiles = t.CanDeleteFiles && isolatedContent(raw, torrents)
		if serverErr != nil {
			out.DownloadSpeed += raw.DLSpeed
			out.UploadSpeed += raw.UPSpeed
		}
		out.CompletedBytes += raw.Completed
		out.RemainingBytes += raw.AmountLeft
		if cleanup && groupErr == nil && policy.Enabled && t.AllCompleted && (raw.Progress >= 1 || policy.AfterIngest) && threshold(policy, t) && (!policy.DeleteFiles || t.CanDeleteFiles) && (removableState(t.State) || policy.AfterIngest && raw.State == qbittorrent.TorrentStateMissingFiles) {
			// Persist intent before the external mutation. If qB is unavailable after
			// deletion, the following cycle can still distinguish our removal from a missing torrent.
			t.RemovalReason = "POLICY_PENDING"
			previousIntent := t
			previousIntent.CleanupError = false
			old[hash] = previousIntent
			intentSnapshot := previous
			intentSnapshot.Torrents = append([]Torrent{}, out.Torrents...)
			for key, saved := range old {
				if !seen[key] || key == hash {
					intentSnapshot.Torrents = append(intentSnapshot.Torrents, saved)
				}
			}
			if err = s.write(ctx, "snapshot", intentSnapshot); err != nil {
				return err
			}
			if deleteErr := s.client.DeleteTorrents(ctx, []string{hash}, policy.DeleteFiles); deleteErr != nil {
				t.CleanupError = true
				t.RemovalReason = ""
			} else {
				t.RemovalReason = "POLICY_PENDING"
				observed, observeErr := s.client.GetTorrentList(ctx, &qbittorrent.TorrentListOptions{Hashes: []string{hash}})
				if observeErr == nil {
					exists := false
					for _, item := range observed {
						if strings.EqualFold(item.Hash, hash) {
							exists = true
						}
					}
					if !exists {
						t.Present = false
						t.State = "removed"
						t.RemovedAt = now
						t.RemovalReason = "POLICY"
						t.DownloadSpeed = 0
						t.UploadSpeed = 0
					}
				}
			}
		}
		out.Torrents = append(out.Torrents, t)
	}
	for hash, tasks := range groups {
		if seen[hash] {
			continue
		}
		t, ok := old[hash]
		if !ok {
			t = Torrent{Hash: hash, Name: tasks[0].TorrentName}
		}
		attach(&t, tasks)
		t.Present = false
		t.DownloadSpeed = 0
		t.UploadSpeed = 0
		t.State = "missing"
		if t.RemovalReason == "POLICY_PENDING" {
			t.RemovalReason = "POLICY"
			t.RemovedAt = now
		}
		if t.RemovalReason == "MANUAL_PENDING" {
			t.RemovalReason = "MANUAL"
			t.RemovedAt = now
		}
		if t.RemovalReason == "POLICY" || t.RemovalReason == "MANUAL" {
			t.State = "removed"
		}
		out.Torrents = append(out.Torrents, t)
	}
	sort.Slice(out.Torrents, func(i, j int) bool { return out.Torrents[i].Hash < out.Torrents[j].Hash })
	err = s.publishSnapshot(ctx, out)
	// Keep only the latest listing while the task observer is busy.
	select {
	case <-s.observations:
	default:
	}
	select {
	case s.observations <- torrents:
	default:
	}
	return err
}
func (s *Service) runTaskObserver(ctx context.Context) {
	observer, ok := s.tasks.(interface {
		ObserveTorrentStates(context.Context, []qbittorrent.Torrent) error
	})
	if !ok {
		return
	}
	failed := false
	for {
		select {
		case <-ctx.Done():
			return
		case torrents := <-s.observations:
			workCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := observer.ObserveTorrentStates(workCtx, torrents)
			cancel()
			if err != nil && !failed && ctx.Err() == nil {
				logging.Warnf("downloader: task synchronization failed: %v", err)
			}
			if err == nil && failed {
				logging.Infof("downloader: task synchronization recovered")
			}
			failed = err != nil
		}
	}
}
func removableState(state string) bool {
	switch state {
	case "uploading", "stalledUP", "pausedUP", "stoppedUP", "queuedUP", "forcedUP":
		return true
	}
	return false
}
func (s *Service) Refresh(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refresh(ctx, false)
}
func (s *Service) Cycle(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refresh(ctx, true)
}
func (s *Service) Run(ctx context.Context) {
	defer s.Close()
	observerCtx, cancelObserver := context.WithCancel(ctx)
	observerDone := make(chan struct{})
	go func() { defer close(observerDone); s.runTaskObserver(observerCtx) }()
	defer func() { cancelObserver(); <-observerDone }()
	ticker := time.NewTicker(SyncInterval)
	defer ticker.Stop()
	failed := false
	for {
		cycleCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
		err := s.Cycle(cycleCtx)
		if err != nil && !failed {
			logging.Warnf("downloader: monitor failed: %v", err)
		}
		if err == nil && failed {
			logging.Infof("downloader: monitor recovered")
		}
		failed = err != nil
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func validHash(hash string) bool {
	if len(hash) != 40 && len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (s *Service) Action(ctx context.Context, hash, action string) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	hash = strings.ToLower(strings.TrimSpace(hash))
	if !validHash(hash) {
		return errors.New("invalid torrent hash")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Revalidate presence and task ownership at the time of every operation.
	torrents, err := s.client.GetTorrentList(ctx, &qbittorrent.TorrentListOptions{Hashes: []string{hash}})
	if err != nil {
		return err
	}
	found := false
	var current qbittorrent.Torrent
	for _, t := range torrents {
		if strings.EqualFold(t.Hash, hash) {
			found = true
			current = t
		}
	}
	if !found {
		return ErrNotFound
	}
	groups, err := s.taskGroups(ctx)
	if err != nil {
		return err
	}
	switch action {
	case "PAUSE":
		err = s.client.PauseTorrents(ctx, []string{hash})
	case "RESUME":
		err = s.client.ResumeTorrents(ctx, []string{hash})
	case "RECHECK":
		err = s.client.RecheckTorrents(ctx, []string{hash})
	case "REANNOUNCE":
		err = s.client.ReannounceTorrents(ctx, []string{hash})
	case "REMOVE", "REMOVE_FILES":
		t := Torrent{}
		attach(&t, groups[hash])
		if len(t.TaskIDs) > 0 && !t.AllCompleted {
			return ErrIngestActive
		}
		if action == "REMOVE_FILES" {
			all, lookupErr := s.client.GetTorrentList(ctx, nil)
			if lookupErr != nil {
				return lookupErr
			}
			t.CanDeleteFiles = t.CanDeleteFiles && isolatedContent(current, all)
		}
		if action == "REMOVE_FILES" && !t.CanDeleteFiles {
			return ErrUnsafeFiles
		}
		snapshot, snapshotErr := s.Snapshot(ctx)
		if snapshotErr != nil {
			err = snapshotErr
			return err
		}
		marked := false
		for i := range snapshot.Torrents {
			if snapshot.Torrents[i].Hash == hash {
				snapshot.Torrents[i].RemovalReason = "MANUAL_PENDING"
				marked = true
			}
		}
		if !marked {
			t.Hash = hash
			t.Name = current.Name
			t.State = string(current.State)
			t.Present = true
			t.RemovalReason = "MANUAL_PENDING"
			snapshot.Torrents = append(snapshot.Torrents, t)
		}
		if err = s.write(ctx, "snapshot", snapshot); err != nil {
			return err
		}
		err = s.client.DeleteTorrents(ctx, []string{hash}, action == "REMOVE_FILES")
		if err != nil {
			for i := range snapshot.Torrents {
				if snapshot.Torrents[i].Hash == hash {
					snapshot.Torrents[i].RemovalReason = ""
				}
			}
			_ = s.write(ctx, "snapshot", snapshot)
		}
	default:
		return fmt.Errorf("unsupported torrent action %q", action)
	}
	if err != nil {
		return err
	}
	// qB applies commands asynchronously; observed state is only changed by a later read.
	if err := s.refresh(ctx, false); err != nil {
		return fmt.Errorf("%w: %v", ErrObservationPending, err)
	}
	return nil
}

func (s *Service) SpeedLimit(ctx context.Context, upload bool, kib int) error {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if kib < 0 || kib > 10485760 {
		return ErrInvalidLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.client.SetDownloadSpeedLimit(ctx, upload, int64(kib)*1024); err != nil {
		return err
	}
	if err := s.refresh(ctx, false); err != nil {
		return fmt.Errorf("%w: %v", ErrObservationPending, err)
	}
	return nil
}

// Cross-seeded content must stay on disk while any other torrent refers to it.
// qB paths are compared lexically on the qB host, separately from local mount checks.
func isolatedContent(target qbittorrent.Torrent, all []qbittorrent.Torrent) bool {
	if target.ContentPath == "" {
		return false
	}
	for _, other := range all {
		if strings.EqualFold(other.Hash, target.Hash) {
			continue
		}
		if other.ContentPath == "" {
			return false
		}
		if within(target.ContentPath, other.ContentPath) || within(other.ContentPath, target.ContentPath) {
			return false
		}
	}
	return true
}

// Completed tasks may legitimately lose their download source after cleanup.
// Keep those observations visible without treating them as operational errors.
func (t Torrent) NeedsAttention() bool {
	if t.AllCompleted && (t.State == "missing" || t.State == "missingFiles" || t.State == "removed") {
		return false
	}
	return t.CleanupError || t.State == "error" || (!t.AllCompleted && (t.State == "missingFiles" || t.State == "missing"))
}
