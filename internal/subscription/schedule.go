package subscription

import (
	"context"
	"errors"
	"time"
)

// recordCheckSchedule runs under the performer lock after every manual or
// scheduled attempt. A confirmed catalog read, even if a task later fails,
// uses the activity interval; upstream errors preserve the previous rating.
func (s *Service) recordCheckSchedule(ctx context.Context, id string, started time.Time, checkErr error) (*PerformerState, error) {
	if errors.Is(checkErr, errCheckPerformerNotFound) {
		return nil, nil
	}
	state, err := s.store.GetCheckState(ctx, id)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &PerformerState{PerformerID: id}
	}
	now := s.now().UTC()
	state.LastCheckedAt = &started
	observed := state.Activity.ObservedAt
	if observed != nil && !observed.Before(started) {
		state.Activity.Failures = 0
		next := now.Add(activityInterval(state.Activity.Rank))
		state.Activity.NextCheckAt = &next
	} else {
		state.Activity.Failures++
		hours := 1 << min(state.Activity.Failures-1, 5)
		next := now.Add(time.Duration(min(hours, 24)) * time.Hour)
		state.Activity.NextCheckAt = &next
	}
	state.LastError = ""
	if checkErr != nil {
		state.LastError = checkErr.Error()
	}
	if err := s.store.SaveCheckOutcome(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

// DispatchDueCheck reconciles subscription truth in Stash, initializes missing
// deadlines once, and admits at most one due actor. Repeated ticks/restarts do
// not move existing deadlines. No missed cycles are replayed.
func (s *Service) DispatchDueCheck(ctx context.Context) error {
	actors, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return err
	}
	states, err := s.store.ListCheckSchedules(ctx)
	if err != nil {
		return err
	}
	byID := make(map[string]CheckSchedule, len(states))
	for _, state := range states {
		byID[state.PerformerID] = state
	}
	reserved := map[string]bool{}
	busy := false
	s.checks.mu.Lock()
	for _, run := range s.checks.runs {
		if run.Status != "RUNNING" {
			continue
		}
		if run.Trigger == "SCHEDULED" {
			busy = true
		}
		for _, item := range run.Items {
			reserved[item.PerformerID] = true
		}
	}
	s.checks.mu.Unlock()
	now := s.now().UTC()
	eligible := map[string]bool{}
	var earliest *time.Time
	for _, raw := range actors {
		actor := performerFromStash(raw, s.customFieldKey)
		if !actor.Subscribed {
			continue
		}
		eligible[actor.ID] = true
		if reserved[actor.ID] {
			continue
		}
		schedule := byID[actor.ID]
		if schedule.NextCheckAt == nil {
			unlock := s.lockPerformerOperation(actor.ID)
			state, readErr := s.store.GetCheckState(ctx, actor.ID)
			next := now
			if readErr == nil && state != nil {
				if state.Activity.NextCheckAt != nil {
					next = *state.Activity.NextCheckAt
				} else if state.LastCheckedAt != nil {
					next = state.LastCheckedAt.Add(activityInterval(state.Activity.Rank))
				}
			}
			if readErr == nil {
				readErr = s.store.InitializeCheckSchedule(ctx, actor.ID, next)
			}
			unlock()
			if readErr != nil {
				return readErr
			}
			schedule.NextCheckAt = &next
		}
		if earliest == nil || schedule.NextCheckAt.Before(*earliest) {
			value := *schedule.NextCheckAt
			earliest = &value
		}
	}
	s.SetCheckSchedule(true, earliest)
	if busy || earliest == nil || earliest.After(now) {
		return nil
	}
	var cursor *CheckSchedule
	for {
		due, err := s.store.ListDueCheckSchedules(ctx, now, cursor, 128)
		if err != nil {
			return err
		}
		for _, item := range due {
			if !eligible[item.PerformerID] || reserved[item.PerformerID] {
				continue
			}
			_, err = s.StartSubscriptionCheck(ctx, []string{item.PerformerID}, "SCHEDULED")
			var conflict *CheckConflictError
			if errors.As(err, &conflict) {
				return nil
			}
			return err
		}
		if len(due) < 128 {
			return nil
		}
		value := due[len(due)-1]
		cursor = &value
	}
}
