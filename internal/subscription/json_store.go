package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type JSONStore struct {
	mu     sync.RWMutex
	path   string
	states map[string]*PerformerState
}

type jsonStoreFile struct {
	Performers []*PerformerState `json:"performers"`
}

func NewJSONStore(path string) (*JSONStore, error) {
	if path == "" {
		return nil, errors.New("subscription: json store path is required")
	}

	store := &JSONStore{
		path:   path,
		states: make(map[string]*PerformerState),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *JSONStore) Get(_ context.Context, performerID string) (*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[performerID]
	if !ok {
		return nil, nil
	}
	return cloneState(state), nil
}

func (s *JSONStore) Put(ctx context.Context, state *PerformerState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state.PerformerID] = cloneState(state)
	return s.saveLocked(ctx)
}

func (s *JSONStore) Delete(ctx context.Context, performerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, performerID)
	return s.saveLocked(ctx)
}

func (s *JSONStore) List(_ context.Context) ([]*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*PerformerState, 0, len(s.states))
	for _, state := range s.states {
		out = append(out, cloneState(state))
	}
	sortStates(out)
	return out, nil
}

func (s *JSONStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read subscription store %q: %w", s.path, err)
	}
	if len(data) == 0 {
		return nil
	}

	var file jsonStoreFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse subscription store %q: %w", s.path, err)
	}
	for _, state := range file.Performers {
		if state != nil && state.PerformerID != "" {
			s.states[state.PerformerID] = cloneState(state)
		}
	}
	return nil
}

func (s *JSONStore) saveLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	states := make([]*PerformerState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, cloneState(state))
	}
	sortStates(states)

	data, err := json.MarshalIndent(jsonStoreFile{Performers: states}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode subscription store: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create subscription store dir %q: %w", dir, err)
		}
	}

	tmp, err := os.CreateTemp(dir, ".moji-subscription-*.json")
	if err != nil {
		return fmt.Errorf("create subscription store temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write subscription store temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close subscription store temp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace subscription store %q: %w", s.path, err)
	}
	return nil
}
