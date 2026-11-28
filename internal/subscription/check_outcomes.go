package subscription

import (
	"context"
	"time"
)

// applyLatestCheckOutcomes reconciles local outcomes with retained run results.
// Both the Home count and actor workspace must include recovered interruptions.
func (s *Service) applyLatestCheckOutcomes(states map[string]*PerformerState) {
	// Recovered interruptions can be newer than the last saved check outcome.
	// Pending/running checks and skipped items do not replace the last outcome.
	if s.checks != nil {
		s.checks.mu.Lock()
		for _, run := range s.checks.runs {
			for _, item := range run.Items {
				state := states[item.PerformerID]
				if state == nil || (item.Status != "SUCCEEDED" && item.Status != "FAILED" && item.Status != "INTERRUPTED") {
					continue
				}
				started := run.StartedAt
				if item.StartedAt != nil {
					started = *item.StartedAt
				}
				if state.LastCheckedAt != nil && !started.After(*state.LastCheckedAt) {
					continue
				}
				state.LastCheckedAt = copyTime(&started)
				state.LastError = ""
				if item.Status != "SUCCEEDED" {
					state.LastError = item.Error
					if state.LastError == "" {
						state.LastError = run.Error
					}
					if state.LastError == "" {
						state.LastError = item.Status
					}
				}
			}
		}
		s.checks.mu.Unlock()
	}
}

// PerformerCheckTimes includes retained history for actors no longer subscribed.
func (s *Service) PerformerCheckTimes(ctx context.Context, ids []string) (map[string]time.Time, error) {
	records, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	states := make(map[string]*PerformerState, len(ids))
	for _, id := range ids {
		states[id] = &PerformerState{PerformerID: id}
	}
	for _, record := range records {
		if record != nil && states[record.PerformerID] != nil {
			copy := *record
			states[record.PerformerID] = &copy
		}
	}
	s.applyLatestCheckOutcomes(states)
	times := make(map[string]time.Time)
	for id, state := range states {
		if state.LastCheckedAt != nil {
			times[id] = *state.LastCheckedAt
		}
	}
	return times, nil
}
