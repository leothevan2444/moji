package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
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

type TorrentRemover interface {
	DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error
}

type TorrentClient interface {
	TorrentAdder
	TorrentLister
	TorrentRemover
}

type TaskStore interface {
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Find(ctx context.Context, id string) (*Task, error)
	FindByCode(ctx context.Context, code string) (*Task, error)
	FindByTorrentIdentity(ctx context.Context, infoHash string, magnetURI string) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
	Delete(ctx context.Context, id string) (*Task, error)
}

type TaskStatus string

const (
	TaskStatusPending     TaskStatus = "pending"
	TaskStatusAdded       TaskStatus = "added"
	TaskStatusDownloading TaskStatus = "downloading"
	TaskStatusCompleted   TaskStatus = "completed"
	TaskStatusFailed      TaskStatus = "failed"
)

type TaskSource string

const (
	TaskSourceManual       TaskSource = "MANUAL"
	TaskSourceSearch       TaskSource = "SEARCH"
	TaskSourceSubscription TaskSource = "SUBSCRIPTION"
)

type Task struct {
	ID                    string
	Source                TaskSource
	Query                 string
	Code                  string
	Status                TaskStatus
	Candidate             Candidate
	TorrentURL            string
	SavePath              string
	Category              string
	Tags                  string
	TorrentIdentityHash   string
	TorrentIdentityMagnet string
	TorrentHash           string
	TorrentName           string
	Progress              float64
	QBittorrentState      string
	ContentPath           string
	CompletedAt           *time.Time
	StashMode             string
	StashSourcePath       string
	StashTransferAction   string
	StashTransferPath     string
	StashTransferStatus   string
	StashTransferError    string
	StashJobID            string
	StashScanPath         string
	StashScanStatus       string
	StashScanError        string
	StashScanHint         string
	StashScanStartedAt    *time.Time
	Error                 string
	CreatedAt             time.Time
	UpdatedAt             time.Time
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
	Source     TaskSource
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
	Source   TaskSource
	URL      string
	SavePath string
	Category string
	Tags     string
	Paused   *bool
}

var (
	ErrDuplicateTorrentTask = errors.New("duplicate torrent task")
	ErrDuplicateCodeTask    = errors.New("duplicate code task")
	ErrTaskCodeRequired     = errors.New("task code is required")
)

type Service struct {
	tracker            tracker.Tracker
	qbt                TorrentClient
	store              TaskStore
	httpClient         *http.Client
	selector           CandidateSelector
	fileOps            FileOperator
	candidateSelection func() config.CandidateSelectionConfig
	taskDeletePolicy   func() config.TaskDeletePolicy
	now                func() time.Time
	newID              func() string
}

type Option func(*Service)

type FileOperator interface {
	Transfer(ctx context.Context, sourcePath string, action stashsync.TransferAction, targetPath string) error
}

type CandidateSelector interface {
	Select(query string, results []jackett.SearchResult, cfg config.CandidateSelectionConfig) (jackett.SearchResult, error)
}

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
		tracker:            tr,
		qbt:                qbt,
		store:              store,
		httpClient:         &http.Client{Timeout: 15 * time.Second},
		selector:           defaultCandidateSelector{},
		fileOps:            osFileOperator{},
		candidateSelection: config.DefaultCandidateSelectionConfig,
		taskDeletePolicy: func() config.TaskDeletePolicy {
			return config.TaskDeletePolicyKeepOnly
		},
		now:   time.Now,
		newID: newTaskID,
	}
	for _, option := range options {
		option(s)
	}
	return s, nil
}

func WithCandidateSelectionProvider(provider func() config.CandidateSelectionConfig) Option {
	return func(s *Service) {
		if provider != nil {
			s.candidateSelection = provider
		}
	}
}

func WithCandidateSelector(selector CandidateSelector) Option {
	return func(s *Service) {
		if selector != nil {
			s.selector = selector
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(s *Service) {
		if client != nil {
			s.httpClient = client
		}
	}
}

func WithFileOperator(fileOps FileOperator) Option {
	return func(s *Service) {
		if fileOps != nil {
			s.fileOps = fileOps
		}
	}
}

func WithTaskDeletePolicyProvider(provider func() config.TaskDeletePolicy) Option {
	return func(s *Service) {
		if provider != nil {
			s.taskDeletePolicy = provider
		}
	}
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

func (s *Service) DeleteTask(ctx context.Context, id string) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("downloader: task id is required")
	}

	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	policy := config.TaskDeletePolicyKeepOnly
	if s.taskDeletePolicy != nil {
		policy = config.NormalizeTaskDeletePolicy(string(s.taskDeletePolicy()))
	}

	if hash := strings.TrimSpace(task.TorrentHash); hash != "" && policy != config.TaskDeletePolicyKeepOnly {
		deleteFiles := policy == config.TaskDeletePolicyRemoveTorrentAndFiles
		if err := s.qbt.DeleteTorrents(ctx, []string{hash}, deleteFiles); err != nil {
			return nil, fmt.Errorf("delete qBittorrent torrent %q with policy %s: %w", hash, policy, err)
		}
	}

	return s.store.Delete(ctx, id)
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

	selectionConfig := config.DefaultCandidateSelectionConfig()
	if s.candidateSelection != nil {
		selectionConfig = s.candidateSelection().Effective()
	}
	selector := s.selector
	if selector == nil {
		selector = defaultCandidateSelector{}
	}
	result, err := selector.Select(query, results, selectionConfig)
	if err != nil {
		logging.Errorf("downloader: select candidate failed for query %q: %v", query, err)
		return nil, err
	}

	candidate := candidateFromSearchResult(result)
	torrentURL := preferredTorrentURL(result)
	identity := torrentIdentityFromCandidate(candidate, torrentURL)
	code := extractCode(candidate.Title, candidate.Link, candidate.MagnetURI)
	if err := s.ensureTaskCanBeCreated(ctx, identity, code); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	source := req.Source
	if source == "" {
		source = TaskSourceManual
	}
	task := &Task{
		ID:                    s.newID(),
		Source:                source,
		Query:                 query,
		Code:                  code,
		Status:                TaskStatusPending,
		Candidate:             candidate,
		TorrentURL:            torrentURL,
		SavePath:              req.SavePath,
		Category:              req.Category,
		Tags:                  req.Tags,
		TorrentIdentityHash:   identity.InfoHash,
		TorrentIdentityMagnet: identity.MagnetURI,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("downloader: create task failed for query %q: %v", query, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof(
		"downloader: created %s task %s for query %q using tracker=%s title=%q",
		strings.ToLower(string(source)),
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
	logging.Infof("downloader: %s task %s added to qBittorrent for query %q", strings.ToLower(string(source)), task.ID, query)

	return task, nil
}

func (s *Service) AddTorrentContext(ctx context.Context, req AddTorrentRequest) (*Task, error) {
	torrentURL := strings.TrimSpace(req.URL)
	if torrentURL == "" {
		return nil, errors.New("downloader: torrent url is required")
	}

	candidate, code, identity, err := s.resolveManualTorrent(ctx, torrentURL)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTaskCanBeCreated(ctx, identity, code); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	source := req.Source
	if source == "" {
		source = TaskSourceManual
	}
	task := &Task{
		ID:                    s.newID(),
		Source:                source,
		Query:                 torrentURL,
		Code:                  code,
		Status:                TaskStatusPending,
		Candidate:             candidate,
		TorrentURL:            torrentURL,
		SavePath:              req.SavePath,
		Category:              req.Category,
		Tags:                  req.Tags,
		TorrentIdentityHash:   identity.InfoHash,
		TorrentIdentityMagnet: identity.MagnetURI,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("downloader: create manual task failed for url %q: %v", torrentURL, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof("downloader: created %s task %s for url %q", strings.ToLower(string(source)), task.ID, torrentURL)

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
	logging.Infof("downloader: %s task %s added to qBittorrent", strings.ToLower(string(source)), task.ID)

	return task, nil
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
	if task.TorrentIdentityHash != "" {
		for _, torrent := range torrents {
			if strings.EqualFold(torrent.Hash, task.TorrentIdentityHash) {
				return torrent, true
			}
		}
	}

	needle := normalizeMagnetURI(task.TorrentURL)
	for _, torrent := range torrents {
		if needle != "" && normalizeMagnetURI(torrent.MagnetURI) == needle {
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
	task.TorrentIdentityHash = normalizeInfoHash(torrent.Hash)
	task.TorrentName = torrent.Name
	task.Progress = torrent.Progress
	task.QBittorrentState = string(torrent.State)
	task.ContentPath = torrent.ContentPath
	task.SavePath = torrent.SavePath
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

func (s *MemoryTaskStore) FindByCode(_ context.Context, code string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	for _, task := range s.tasks {
		if strings.EqualFold(strings.TrimSpace(task.Code), code) {
			return cloneTask(task), nil
		}
	}
	return nil, nil
}

func (s *MemoryTaskStore) FindByTorrentIdentity(_ context.Context, infoHash string, magnetURI string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infoHash = normalizeInfoHash(infoHash)
	magnetURI = normalizeMagnetURI(magnetURI)
	if infoHash == "" && magnetURI == "" {
		return nil, nil
	}
	for _, task := range s.tasks {
		if infoHash != "" && normalizeInfoHash(task.TorrentHash) == infoHash {
			return cloneTask(task), nil
		}
		if infoHash != "" && normalizeInfoHash(task.TorrentIdentityHash) == infoHash {
			return cloneTask(task), nil
		}
		if infoHash != "" && normalizeInfoHash(task.Candidate.InfoHash) == infoHash {
			return cloneTask(task), nil
		}
		if magnetURI != "" {
			if normalizeMagnetURI(task.TorrentIdentityMagnet) == magnetURI ||
				normalizeMagnetURI(task.TorrentURL) == magnetURI ||
				normalizeMagnetURI(task.Candidate.MagnetURI) == magnetURI {
				return cloneTask(task), nil
			}
		}
	}
	return nil, nil
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

func (s *MemoryTaskStore) Delete(_ context.Context, id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("downloader: task %q not found", id)
	}
	delete(s.tasks, id)
	return cloneTask(task), nil
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

type osFileOperator struct{}

func (osFileOperator) Transfer(ctx context.Context, sourcePath string, action stashsync.TransferAction, targetPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(sourcePath) == "" {
		return errors.New("downloader: source path is required for file transfer")
	}
	if strings.TrimSpace(targetPath) == "" {
		return errors.New("downloader: target path is required for file transfer")
	}
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("downloader: transfer target already exists: %s", targetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("downloader: stat transfer target %q: %w", targetPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("downloader: create transfer target dir for %q: %w", targetPath, err)
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("downloader: stat transfer source %q: %w", sourcePath, err)
	}

	switch action {
	case stashsync.TransferActionCopy:
		if sourceInfo.IsDir() {
			return copyDir(ctx, sourcePath, targetPath)
		}
		return copyFile(ctx, sourcePath, targetPath)
	case stashsync.TransferActionMove:
		if err := os.Rename(sourcePath, targetPath); err == nil {
			return nil
		} else if !isCrossDeviceError(err) {
			return fmt.Errorf("downloader: move %q -> %q: %w", sourcePath, targetPath, err)
		}
		if sourceInfo.IsDir() {
			if err := copyDir(ctx, sourcePath, targetPath); err != nil {
				return err
			}
			if err := os.RemoveAll(sourcePath); err != nil {
				return fmt.Errorf("downloader: remove transferred source %q: %w", sourcePath, err)
			}
			return nil
		}
		if err := copyFile(ctx, sourcePath, targetPath); err != nil {
			return err
		}
		if err := os.Remove(sourcePath); err != nil {
			return fmt.Errorf("downloader: remove transferred source %q: %w", sourcePath, err)
		}
		return nil
	case stashsync.TransferActionSymlink:
		if err := os.Symlink(sourcePath, targetPath); err != nil {
			return fmt.Errorf("downloader: symlink %q -> %q: %w", targetPath, sourcePath, err)
		}
		return nil
	default:
		return fmt.Errorf("downloader: unsupported transfer action %q", action)
	}
}

func copyDir(ctx context.Context, sourcePath string, targetPath string) error {
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return fmt.Errorf("downloader: create transfer target dir %q: %w", targetPath, err)
	}
	return filepath.Walk(sourcePath, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(sourcePath, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		destination := filepath.Join(targetPath, relative)
		if info.IsDir() {
			if err := os.MkdirAll(destination, info.Mode()); err != nil {
				return fmt.Errorf("downloader: create transfer target dir %q: %w", destination, err)
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("downloader: create transfer target dir for %q: %w", destination, err)
		}
		return copyFile(ctx, current, destination)
	})
}

func copyFile(ctx context.Context, sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("downloader: open transfer source %q: %w", sourcePath, err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("downloader: create transfer target %q: %w", targetPath, err)
	}
	defer target.Close()

	if _, err := io.Copy(target, &contextReader{ctx: ctx, reader: source}); err != nil {
		return fmt.Errorf("downloader: copy %q -> %q: %w", sourcePath, targetPath, err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("downloader: finalize transfer target %q: %w", targetPath, err)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func isCrossDeviceError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "cross-device link")
}
