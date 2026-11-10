package subscription

import (
	"context"
	"time"
)

func (s *MemoryStore) GetCheckState(_ context.Context, id string) (*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCheckState(s.states[id]), nil
}
func (s *MemoryStore) SaveActivity(ctx context.Context, id string, activity ActivityState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saveActivityState(s.states, id, activity)
	return nil
}
func (s *MemoryStore) SaveCheckOutcome(ctx context.Context, state *PerformerState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saveOutcomeState(s.states, state)
	return nil
}
func (s *MemoryStore) InitializeCheckSchedule(ctx context.Context, id string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	initializeScheduleState(s.states, id, next)
	return nil
}
func (s *MemoryStore) ListCheckSchedules(context.Context) ([]CheckSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return checkSchedules(s.states), nil
}
func (s *MemoryStore) ListDueCheckSchedules(_ context.Context, now time.Time, after *CheckSchedule, limit int) ([]CheckSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dueCheckSchedules(s.states, now, after, limit), nil
}

func (s *MemoryStore) GetCheckStates(_ context.Context, ids []string) ([]*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make([]*PerformerState, 0, len(ids))
	for _, id := range ids {
		if state := s.states[id]; state != nil {
			states = append(states, cloneCheckState(state))
		}
	}
	return states, nil
}
