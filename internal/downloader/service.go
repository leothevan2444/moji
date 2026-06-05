package downloader

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentAdder interface {
	AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error
}

type TaskStore interface {
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Find(ctx context.Context, id string) (*Task, error)
}

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusAdded   TaskStatus = "added"
	TaskStatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID         string
	Query      string
	Status     TaskStatus
	Candidate  Candidate
	TorrentURL string
	SavePath   string
	Category   string
	Tags       string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
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

type Service struct {
	tracker tracker.Tracker
	qbt     TorrentAdder
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

func NewService(tr tracker.Tracker, qbt TorrentAdder, store TaskStore, options ...Option) (*Service, error) {
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
		return nil, fmt.Errorf("search torrents: %w", err)
	}

	result, err := selectCandidate(results)
	if err != nil {
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
		return nil, fmt.Errorf("create task: %w", err)
	}

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
		return task, fmt.Errorf("add torrent: %w", err)
	}

	task.Status = TaskStatusAdded
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		return task, fmt.Errorf("update task: %w", err)
	}

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

func cloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cp := *task
	return &cp
}
