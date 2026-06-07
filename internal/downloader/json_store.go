package downloader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONTaskStore struct {
	mu    sync.RWMutex
	path  string
	tasks map[string]*Task
}

type jsonTaskFile struct {
	Tasks []*Task `json:"tasks"`
}

func NewJSONTaskStore(path string) (*JSONTaskStore, error) {
	if path == "" {
		return nil, errors.New("downloader: json task store path is required")
	}

	store := &JSONTaskStore{
		path:  path,
		tasks: make(map[string]*Task),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *JSONTaskStore) Create(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("downloader: task %q already exists", task.ID)
	}
	s.tasks[task.ID] = cloneTask(task)
	return s.saveLocked(ctx)
}

func (s *JSONTaskStore) Update(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("downloader: task is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("downloader: task %q not found", task.ID)
	}
	s.tasks[task.ID] = cloneTask(task)
	return s.saveLocked(ctx)
}

func (s *JSONTaskStore) Find(_ context.Context, id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("downloader: task %q not found", id)
	}
	return cloneTask(task), nil
}

func (s *JSONTaskStore) List(_ context.Context) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	sortTasks(tasks)
	return tasks, nil
}

func (s *JSONTaskStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read task store %q: %w", s.path, err)
	}
	if len(data) == 0 {
		return nil
	}

	var file jsonTaskFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse task store %q: %w", s.path, err)
	}
	for _, task := range file.Tasks {
		if task != nil && task.ID != "" {
			s.tasks[task.ID] = cloneTask(task)
		}
	}
	return nil
}

func (s *JSONTaskStore) saveLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	sortTasks(tasks)

	data, err := json.MarshalIndent(jsonTaskFile{Tasks: tasks}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode task store: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create task store dir %q: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, ".moji-tasks-*.json")
	if err != nil {
		return fmt.Errorf("create task store temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write task store temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close task store temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace task store %q: %w", s.path, err)
	}
	return nil
}
