package subscription

import (
	"context"
	"sort"
	"sync"
)

type MemoryStore struct {
	mu     sync.RWMutex
	states map[string]*PerformerState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{states: make(map[string]*PerformerState)}
}

func (s *MemoryStore) Get(_ context.Context, performerID string) (*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[performerID]
	if !ok {
		return nil, nil
	}
	return cloneState(state), nil
}

func (s *MemoryStore) Put(_ context.Context, state *PerformerState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state.PerformerID] = cloneState(state)
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, performerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, performerID)
	return nil
}

func (s *MemoryStore) List(_ context.Context) ([]*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*PerformerState, 0, len(s.states))
	for _, state := range s.states {
		out = append(out, cloneState(state))
	}
	sortStates(out)
	return out, nil
}

func cloneState(state *PerformerState) *PerformerState {
	if state == nil {
		return nil
	}

	cloned := *state
	cloned.ProcessedReleases = append([]RecordedRelease(nil), state.ProcessedReleases...)
	cloned.PendingReleases = append([]RecordedRelease(nil), state.PendingReleases...)
	return &cloned
}

func sortStates(states []*PerformerState) {
	sort.Slice(states, func(i, j int) bool {
		left := states[i]
		right := states[j]
		if left == nil || right == nil {
			return left != nil
		}
		if left.LastCheckedAt == nil || right.LastCheckedAt == nil {
			return left.LastCheckedAt != nil
		}
		return left.LastCheckedAt.After(*right.LastCheckedAt)
	})
}
