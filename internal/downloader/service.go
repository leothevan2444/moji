package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentAdder interface {
	AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error
}

type TorrentLister interface {
	GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error)
}

type TorrentClient interface {
	TorrentAdder
	TorrentLister
}

type TaskStore interface {
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Find(ctx context.Context, id string) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
}

type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusAdded       TaskStatus = "added"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
)

type Task struct {
	ID                 string
	Query              string
	Status             TaskStatus
	Candidate          Candidate
	TorrentURL         string
	SavePath           string
	Category           string
	Tags               string
	TorrentHash        string
	TorrentName        string
	Progress           float64
	QBittorrentState   string
	ContentPath        string
	CompletedAt        *time.Time
	StashJobID         string
	StashScanStatus    string
	StashScanError     string
	StashScanStartedAt *time.Time
	Error              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Candidate struct {
	Title     string
	Tracker   string
	InfoHash  string
	Link      string
	MagnetURI string
	Size      int64
	Seeders   int
	Peers     int
}

type DownloadRequest struct {
	Query      string
	Trackers   []string
	Categories []int
	Limit      int
	SavePath   string
	Category   string
	Tags       string
	Paused     *bool
}

type AddTorrentRequest struct {
	URL      string
	SavePath string
	Category string
	Tags     string
	Paused   *bool
}

type Service struct {
	tracker tracker.Tracker
	qbt     TorrentClient
	store   TaskStore
	now     func() time.Time
	newID   func() string
}

type Option func(*Service)

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

func WithIDGenerator(newID func() string) Option {
	return func(s *Service) {
		if newID != nil {
			s.newID = newID
		}
	}
}

func NewService(tr tracker.Tracker, qbt TorrentClient, store TaskStore, options ...Option) (*Service, error) {
	if tr == nil {
		return nil, errors.New("downloader: tracker is required")
	}
	if qbt == nil {
		return nil, errors.New("downloader: qBittorrent client is required")
	}
	if store == nil {
		store = NewMemoryTaskStore()
	}

	s := &Service{
		tracker: tr,
		qbt:     qbt,
		store:   store,
		now:     time.Now,
		newID:   newTaskID,
	}
	for _, option := range options {
		option(s)
	}
	return s, nil
}

func (s *Service) DownloadMedia(query string) (*Task, error) {
	return s.DownloadMediaContext(context.Background(), DownloadRequest{Query: query})
}

func (s *Service) FindTask(ctx context.Context, id string) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("downloader: task id is required")
	}
	return s.store.Find(ctx, id)
}

func (s *Service) ListTasks(ctx context.Context) ([]*Task, error) {
	return s.store.List(ctx)
}

func (s *Service) SyncProgress(ctx context.Context) ([]*Task, error) {
	tasks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	torrents, err := s.qbt.GetTorrentList(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list torrents: %w", err)
	}

	updated := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		if task == nil || task.Status == TaskStatusFailed || task.Status == TaskStatusCompleted {
			updated = append(updated, task)
			continue
		}

		torrent, ok := matchTaskTorrent(task, torrents)
		if !ok {
			updated = append(updated, task)
			continue
		}

		next := cloneTask(task)
		prevStatus := next.Status
		prevProgress := next.Progress
		applyTorrentProgress(next, torrent, s.now().UTC())
		if err := s.store.Update(ctx, next); err != nil {
			return updated, fmt.Errorf("update task %q: %w", next.ID, err)
		}
		if next.Status != prevStatus {
			logging.Infof(
				"downloader: task %s status %s -> %s (%s %.1f%%)",
				next.ID,
				prevStatus,
				next.Status,
				next.Candidate.Title,
				next.Progress*100,
			)
		} else if next.Status == TaskStatusCompleted && next.Progress != prevProgress {
			logging.Infof("downloader: task %s completed with content path %s", next.ID, next.ContentPath)
		}
		updated = append(updated, next)
	}

	return updated, nil
}

func (s *Service) DownloadMediaContext(ctx context.Context, req DownloadRequest) (*Task, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, errors.New("downloader: query is required")
	}

	searchOptions := []tracker.SearchOption{
		tracker.WithTrackers(req.Trackers),
		tracker.WithCategories(req.Categories),
	}
	if req.Limit > 0 {
		searchOptions = append(searchOptions, tracker.WithLimit(req.Limit))
	}

	results, err := s.tracker.Search(query, searchOptions...)
	if err != nil {
		logging.Errorf("downloader: search failed for query %q: %v", query, err)
		return nil, fmt.Errorf("search torrents: %w", err)
	}
	logging.Infof("downloader: search returned %d results for query %q", len(results), query)

	result, err := selectCandidate(results)
	if err != nil {
		logging.Errorf("downloader: select candidate failed for query %q: %v", query, err)
		return nil, err
	}

	candidate := candidateFromSearchResult(result)
	torrentURL := preferredTorrentURL(result)
	now := s.now().UTC()
	task := &Task{
		ID:         s.newID(),
		Query:      query,
		Status:     TaskStatusPending,
		Candidate:  candidate,
		TorrentURL: torrentURL,
		SavePath:   req.SavePath,
		Category:   req.Category,
		Tags:       req.Tags,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("downloader: create task failed for query %q: %v", query, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof(
		"downloader: created task %s for query %q using tracker=%s title=%q",
		task.ID,
		query,
		candidate.Tracker,
		candidate.Title,
	)

	addOptions := qbittorrent.AddTorrentOptions{
		URLs: []string{torrentURL},
	}
	if req.SavePath != "" {
		addOptions.SavePath = &req.SavePath
	}
	if req.Category != "" {
		addOptions.Category = &req.Category
	}
	if req.Tags != "" {
		addOptions.Tags = &req.Tags
	}
	if req.Paused != nil {
		addOptions.Paused = req.Paused
	}

	if err := s.qbt.AddNewTorrent(ctx, addOptions); err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		task.UpdatedAt = s.now().UTC()
		_ = s.store.Update(ctx, task)
		logging.Errorf("downloader: add torrent failed for task %s query %q: %v", task.ID, query, err)
		return task, fmt.Errorf("add torrent: %w", err)
	}

	task.Status = TaskStatusAdded
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		logging.Errorf("downloader: persist added task %s failed: %v", task.ID, err)
		return task, fmt.Errorf("update task: %w", err)
	}
	logging.Infof("downloader: task %s added to qBittorrent for query %q", task.ID, query)

	return task, nil
}

func (s *Service) AddTorrentContext(ctx context.Context, req AddTorrentRequest) (*Task, error) {
	torrentURL := strings.TrimSpace(req.URL)
	if torrentURL == "" {
		return nil, errors.New("downloader: torrent url is required")
	}

	now := s.now().UTC()
	task := &Task{
		ID:         s.newID(),
		Query:      torrentURL,
		Status:     TaskStatusPending,
		Candidate:  candidateFromTorrentURL(torrentURL),
		TorrentURL: torrentURL,
		SavePath:   req.SavePath,
		Category:   req.Category,
		Tags:       req.Tags,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("downloader: create manual task failed for url %q: %v", torrentURL, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof("downloader: created manual task %s for url %q", task.ID, torrentURL)

	addOptions := qbittorrent.AddTorrentOptions{
		URLs: []string{torrentURL},
	}
	if req.SavePath != "" {
		addOptions.SavePath = &req.SavePath
	}
	if req.Category != "" {
		addOptions.Category = &req.Category
	}
	if req.Tags != "" {
		addOptions.Tags = &req.Tags
	}
	if req.Paused != nil {
		addOptions.Paused = req.Paused
	}

	if err := s.qbt.AddNewTorrent(ctx, addOptions); err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		task.UpdatedAt = s.now().UTC()
		_ = s.store.Update(ctx, task)
		logging.Errorf("downloader: add manual torrent failed for task %s: %v", task.ID, err)
		return task, fmt.Errorf("add torrent: %w", err)
	}

	task.Status = TaskStatusAdded
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		logging.Errorf("downloader: persist manual task %s failed: %v", task.ID, err)
		return task, fmt.Errorf("update task: %w", err)
	}
	logging.Infof("downloader: manual task %s added to qBittorrent", task.ID)

	return task, nil
}

func selectCandidate(results []jackett.SearchResult) (jackett.SearchResult, error) {
	var best jackett.SearchResult
	found := false
	for _, result := range results {
		if preferredTorrentURL(result) == "" {
			continue
		}
		if !found || result.Seeders > best.Seeders || result.Seeders == best.Seeders && result.Size > best.Size {
			best = result
			found = true
		}
	}
	if !found {
		return jackett.SearchResult{}, errors.New("downloader: no downloadable torrent candidate found")
	}
	return best, nil
}

func preferredTorrentURL(result jackett.SearchResult) string {
	if result.MagnetURI != "" {
		return result.MagnetURI
	}
	return result.Link
}

func candidateFromSearchResult(result jackett.SearchResult) Candidate {
	return Candidate{
		Title:     result.Title,
		Tracker:   result.Tracker,
		InfoHash:  result.InfoHash,
		Link:      result.Link,
		MagnetURI: result.MagnetURI,
		Size:      result.Size,
		Seeders:   result.Seeders,
		Peers:     result.Peers,
	}
}

func candidateFromTorrentURL(torrentURL string) Candidate {
	return Candidate{
		Title:     torrentURL,
		InfoHash:  infoHashFromMagnet(torrentURL),
		Link:      torrentURL,
		MagnetURI: magnetURI(torrentURL),
	}
}

func magnetURI(torrentURL string) string {
	if strings.HasPrefix(strings.ToLower(torrentURL), "magnet:") {
		return torrentURL
	}
	return ""
}

func infoHashFromMagnet(torrentURL string) string {
	if !strings.HasPrefix(strings.ToLower(torrentURL), "magnet:") {
		return ""
	}

	parsed, err := url.Parse(torrentURL)
	if err != nil {
		return ""
	}
	for _, xt := range parsed.Query()["xt"] {
		parts := strings.Split(xt, ":")
		if len(parts) >= 3 && strings.EqualFold(parts[0], "urn") && strings.EqualFold(parts[1], "btih") {
			return parts[2]
		}
	}
	return ""
}

func matchTaskTorrent(task *Task, torrents []qbittorrent.Torrent) (qbittorrent.Torrent, bool) {
	if task.TorrentHash != "" {
		for _, torrent := range torrents {
			if strings.EqualFold(torrent.Hash, task.TorrentHash) {
				return torrent, true
			}
		}
	}

	needle := strings.TrimSpace(task.TorrentURL)
	for _, torrent := range torrents {
		if needle != "" && torrent.MagnetURI == needle {
			return torrent, true
		}
		if needle != "" && strings.Contains(torrent.Name, needle) {
			return torrent, true
		}
		if task.Candidate.Title != "" && torrent.Name == task.Candidate.Title {
			return torrent, true
		}
		if task.Candidate.InfoHash != "" && strings.EqualFold(torrent.Hash, task.Candidate.InfoHash) {
			return torrent, true
		}
	}

	return qbittorrent.Torrent{}, false
}

func applyTorrentProgress(task *Task, torrent qbittorrent.Torrent, now time.Time) {
	task.TorrentHash = torrent.Hash
	task.TorrentName = torrent.Name
	task.Progress = torrent.Progress
	task.QBittorrentState = string(torrent.State)
	task.ContentPath = torrent.ContentPath
	task.UpdatedAt = now

	if torrent.Progress >= 1 || torrent.CompletionOn > 0 || isCompletedTorrentState(torrent.State) {
		task.Status = TaskStatusCompleted
		completedAt := now
		if torrent.CompletionOn > 0 {
			completedAt = time.Unix(torrent.CompletionOn, 0).UTC()
		}
		task.CompletedAt = &completedAt
		return
	}

	if torrent.Progress > 0 || isDownloadingTorrentState(torrent.State) {
		task.Status = TaskStatusDownloading
		return
	}

	task.Status = TaskStatusAdded
}

func isCompletedTorrentState(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.TorrentStateUploading,
		qbittorrent.TorrentStatePausedUP,
		qbittorrent.TorrentStateQueuedUP,
		qbittorrent.TorrentStateStalledUP,
		qbittorrent.TorrentStateCheckingUP,
		qbittorrent.TorrentStateForcedUP:
		return true
	default:
		return false
	}
}

func isDownloadingTorrentState(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.TorrentStateAllocating,
		qbittorrent.TorrentStateDownloading,
		qbittorrent.TorrentStateMetaDL,
		qbittorrent.TorrentStatePausedDL,
		qbittorrent.TorrentStateQueuedDL,
		qbittorrent.TorrentStateStalledDL,
		qbittorrent.TorrentStateCheckingDL,
		qbittorrent.TorrentStateForcedDL,
		qbittorrent.TorrentStateCheckingResumeData,
		qbittorrent.TorrentStateMoving:
		return true
	default:
		return false
	}
}

func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return "task-" + hex.EncodeToString(b[:])
}

type MemoryTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{tasks: make(map[string]*Task)}
}

func (s *MemoryTaskStore) Create(_ context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("downloader: task %q already exists", task.ID)
	}
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryTaskStore) Update(_ context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("downloader: task %q not found", task.ID)
	}
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryTaskStore) Find(_ context.Context, id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("downloader: task %q not found", id)
	}
	return cloneTask(task), nil
}

func (s *MemoryTaskStore) List(_ context.Context) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	sortTasks(tasks)
	return tasks, nil
}

func sortTasks(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
}

func cloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cp := *task
	cp.CompletedAt = cloneTime(task.CompletedAt)
	cp.StashScanStartedAt = cloneTime(task.StashScanStartedAt)
	return &cp
}
