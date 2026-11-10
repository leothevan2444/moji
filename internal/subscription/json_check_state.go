package subscription

import (
	"context"
	"time"
)

func (s *JSONStore) GetCheckState(_ context.Context, id string) (*PerformerState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneCheckState(s.states[id]), nil
}
func (s *JSONStore) SaveActivity(ctx context.Context, id string, activity ActivityState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saveActivityState(s.states, id, activity)
	return s.saveLocked(ctx)
}
func (s *JSONStore) SaveCheckOutcome(ctx context.Context, state *PerformerState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saveOutcomeState(s.states, state)
	return s.saveLocked(ctx)
}
func (s *JSONStore) InitializeCheckSchedule(ctx context.Context, id string, next time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	initializeScheduleState(s.states, id, next)
	return s.saveLocked(ctx)
}
func (s *JSONStore) ListCheckSchedules(context.Context) ([]CheckSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return checkSchedules(s.states), nil
}
func (s *JSONStore) ListDueCheckSchedules(_ context.Context, now time.Time, after *CheckSchedule, limit int) ([]CheckSchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dueCheckSchedules(s.states, now, after, limit), nil
}

func (s *JSONStore) GetCheckStates(_ context.Context, ids []string) ([]*PerformerState, error) {
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
