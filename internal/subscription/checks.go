package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/logging"
)

const checkHistoryLimit = 100
const checkTimeout = 30 * time.Minute

type CheckItem struct {
	PerformerID string
	Name        string
	Status      string
	Error       string
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

type CheckRun struct {
	ID         string
	Trigger    string
	Scope      string
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
	Items      []CheckItem
}

type CheckRunStore interface {
	LoadCheckRuns(context.Context) ([]CheckRun, error)
	SaveCheckRun(context.Context, CheckRun) error
}

type checkTracker struct {
	wg        sync.WaitGroup
	mu        sync.Mutex
	runs      []CheckRun
	store     CheckRunStore
	ctx       context.Context
	enabled   bool
	nextRunAt *time.Time
}

var errCheckPerformerNotFound = errors.New("subscription: performer not found")

type checkRunContextKey struct{}

type CheckConflictError struct{ RunID string }

func (e *CheckConflictError) Error() string {
	return fmt.Sprintf("subscription check already running: %s", e.RunID)
}

func (s *Service) initCheckTracking() error {
	tracker := &checkTracker{ctx: context.Background()}
	tracker.store, _ = s.store.(CheckRunStore)
	if tracker.store != nil {
		runs, err := tracker.store.LoadCheckRuns(context.Background())
		if err != nil {
			return err
		}
		tracker.runs = runs
		for i := range tracker.runs {
			run := &tracker.runs[i]
			if run.Status != "RUNNING" {
				continue
			}
			now := s.now().UTC()
			run.Status, run.Error, run.FinishedAt = "INTERRUPTED", "SERVICE_RESTARTED", &now
			for j := range run.Items {
				if run.Items[j].Status == "RUNNING" || run.Items[j].Status == "PENDING" {
					run.Items[j].Status = "INTERRUPTED"
					run.Items[j].FinishedAt = &now
				}
			}
			if err := tracker.store.SaveCheckRun(context.Background(), *run); err != nil {
				return err
			}
		}
	}
	s.checks = tracker
	return nil
}

// ConfigureCheckRuntime binds background checks to the application's lifetime.
func (s *Service) ConfigureCheckRuntime(ctx context.Context) {
	s.checks.mu.Lock()
	defer s.checks.mu.Unlock()
	s.checks.ctx = ctx
}

// SetCheckSchedule publishes the worker's actual next deadline, including disabled state.
func (s *Service) SetCheckSchedule(enabled bool, next *time.Time) {
	s.checks.mu.Lock()
	defer s.checks.mu.Unlock()
	s.checks.enabled = enabled
	s.checks.nextRunAt = copyTime(next)
}

func copyTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	value := *t
	return &value
}
func cloneCheckRun(run CheckRun) CheckRun {
	run.FinishedAt = copyTime(run.FinishedAt)
	run.Items = append([]CheckItem{}, run.Items...)
	for i := range run.Items {
		run.Items[i].StartedAt = copyTime(run.Items[i].StartedAt)
		run.Items[i].FinishedAt = copyTime(run.Items[i].FinishedAt)
	}
	return run
}

func (s *Service) SubscriptionCheck(_ context.Context, id string) (*CheckRun, error) {
	s.checks.mu.Lock()
	defer s.checks.mu.Unlock()
	for _, run := range s.checks.runs {
		if run.ID == id {
			value := cloneCheckRun(run)
			return &value, nil
		}
	}
	return nil, nil
}

func checkCovers(run CheckRun, scope string, ids []string) bool {
	if run.Scope == "ALL" {
		return true
	}
	if scope == "ALL" {
		return false
	}
	set := map[string]bool{}
	for _, item := range run.Items {
		set[item.PerformerID] = true
	}
	for _, id := range ids {
		if !set[id] {
			return false
		}
	}
	return true
}

func (s *Service) beginCheck(ctx context.Context, scope string, ids []string, trigger string) (context.Context, *CheckRun, bool, error) {
	if id, ok := ctx.Value(checkRunContextKey{}).(string); ok {
		run, err := s.SubscriptionCheck(ctx, id)
		if err != nil || run == nil {
			return ctx, nil, false, errors.New("subscription check context is invalid")
		}
		return ctx, run, false, nil
	}
	t := s.checks
	t.mu.Lock()
	defer t.mu.Unlock()
	wanted := map[string]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	for _, active := range t.runs {
		if active.Status != "RUNNING" {
			continue
		}
		overlap := scope == "ALL" || active.Scope == "ALL"
		for _, item := range active.Items {
			overlap = overlap || wanted[item.PerformerID]
		}
		if overlap {
			value := cloneCheckRun(active)
			return ctx, &value, false, &CheckConflictError{RunID: active.ID}
		}
	}
	run := CheckRun{ID: "check-" + uuid.NewString(), Trigger: trigger, Scope: scope, Status: "RUNNING", StartedAt: s.now().UTC(), Items: []CheckItem{}}
	for _, id := range ids {
		run.Items = append(run.Items, CheckItem{PerformerID: id, Status: "PENDING"})
	}
	if t.store != nil {
		if err := t.store.SaveCheckRun(ctx, run); err != nil {
			return ctx, nil, false, err
		}
	}
	t.runs = append([]CheckRun{run}, t.runs...)
	s.trimCheckHistoryLocked()
	logging.Infof("subscription: check started run_id=%s trigger=%s scope=%s", run.ID, trigger, scope)
	value := cloneCheckRun(run)
	return context.WithValue(ctx, checkRunContextKey{}, run.ID), &value, true, nil
}

func (s *Service) trimCheckHistoryLocked() {
	kept := make([]CheckRun, 0, len(s.checks.runs))
	scheduledKept := false
	for _, run := range s.checks.runs {
		keep := len(kept) < checkHistoryLimit || run.Status == "RUNNING" || (run.Trigger == "SCHEDULED" && !scheduledKept)
		if run.Trigger == "SCHEDULED" {
			scheduledKept = true
		}
		if keep {
			kept = append(kept, run)
		}
	}
	s.checks.runs = kept
}

func (s *Service) updateCheck(ctx context.Context, update func(*CheckRun)) error {
	id, _ := ctx.Value(checkRunContextKey{}).(string)
	if id == "" {
		return nil
	}
	t := s.checks
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := range t.runs {
		if t.runs[i].ID != id {
			continue
		}
		update(&t.runs[i])
		if t.store != nil {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := t.store.SaveCheckRun(saveCtx, t.runs[i]); err != nil {
				logging.Errorf("subscription: save check run_id=%s: %v", id, err)
				return err
			}
		}
		return nil
	}
	return errors.New("subscription check not found")
}

func (s *Service) checkItems(ctx context.Context, items []SubscribedPerformer) error {
	return s.updateCheck(ctx, func(run *CheckRun) {
		run.Items = make([]CheckItem, 0, len(items))
		for _, item := range items {
			run.Items = append(run.Items, CheckItem{PerformerID: item.Performer.ID, Name: item.Performer.Name, Status: "PENDING"})
		}
	})
}

func (s *Service) startCheckItem(ctx context.Context, id string) error {
	return s.updateCheck(ctx, func(run *CheckRun) {
		for i := range run.Items {
			if run.Items[i].PerformerID == id {
				now := s.now().UTC()
				run.Items[i].Status = "RUNNING"
				run.Items[i].StartedAt = &now
			}
		}
	})
}

func (s *Service) finishCheckItem(ctx context.Context, id, name, status, message string) error {
	return s.updateCheck(ctx, func(run *CheckRun) {
		for i := range run.Items {
			if run.Items[i].PerformerID == id {
				now := s.now().UTC()
				item := &run.Items[i]
				if name != "" {
					item.Name = name
				}
				item.Status, item.Error, item.FinishedAt = status, message, &now
			}
		}
	})
}

func (s *Service) finishCheck(ctx context.Context, runErr error) {
	_ = s.updateCheck(ctx, func(run *CheckRun) {
		now := s.now().UTC()
		run.FinishedAt = &now
		if runErr != nil {
			run.Error = runErr.Error()
		}
		succeeded, failed := 0, 0
		interrupted := errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || ctx.Err() != nil
		if interrupted {
			run.Error = "CHECK_CANCELLED"
			if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				run.Error = "CHECK_TIMED_OUT"
			}
		}
		for i := range run.Items {
			item := &run.Items[i]
			if item.Status == "PENDING" || item.Status == "RUNNING" {
				item.Status = "FAILED"
				if interrupted {
					item.Status = "INTERRUPTED"
				}
				item.Error = run.Error
				item.FinishedAt = &now
			}
			if item.Status == "SUCCEEDED" {
				succeeded++
			}
			if item.Status == "FAILED" || item.Status == "INTERRUPTED" {
				failed++
			}
		}
		run.Status = "SUCCEEDED"
		if failed > 0 || runErr != nil {
			run.Status = "FAILED"
			if succeeded > 0 {
				run.Status = "PARTIAL"
			}
		}
		if interrupted {
			run.Status = "INTERRUPTED"
		}
		logging.Infof("subscription: check finished run_id=%s status=%s succeeded=%d failed=%d", run.ID, run.Status, succeeded, failed)
	})
}

// StartSubscriptionCheck persists and reserves a run before returning; work survives HTTP disconnects.
// A repeat request covered by an active run returns that run instead of queueing duplicate work.
func (s *Service) StartSubscriptionCheck(ctx context.Context, ids []string, trigger string) (*CheckRun, error) {
	scope := "ALL"
	if ids != nil {
		var err error
		ids, err = normalizePerformerBatchIDs(ids)
		if err != nil {
			return nil, err
		}
		scope = "SELECTED"
		if len(ids) == 1 {
			scope = "SINGLE"
		}
	}
	if trigger != "SCHEDULED" {
		trigger = "MANUAL"
	}
	// Capture eligibility once, before actors enter the run. Manual explicit IDs
	// may include unsubscribed actors; automatic work and check-all never do.
	if scope == "ALL" || trigger == "SCHEDULED" {
		performers, err := s.ListSubscribedPerformers(ctx)
		if err != nil {
			return nil, err
		}
		eligible := make(map[string]bool, len(performers))
		for _, item := range performers {
			eligible[item.Performer.ID] = true
		}
		if scope == "ALL" {
			ids = make([]string, 0, len(performers))
			for _, item := range performers {
				ids = append(ids, item.Performer.ID)
			}
		} else {
			selected := make([]string, 0, len(ids))
			for _, id := range ids {
				if eligible[id] {
					selected = append(selected, id)
				}
			}
			ids = selected
		}
	}
	_, run, _, err := s.beginCheck(ctx, scope, ids, trigger)
	if err != nil {
		var conflict *CheckConflictError
		if errors.As(err, &conflict) && run != nil && checkCovers(*run, scope, ids) {
			return run, nil
		}
		return nil, err
	}
	s.checks.mu.Lock()
	parent := s.checks.ctx
	s.checks.wg.Add(1)
	s.checks.mu.Unlock()
	go func() {
		defer s.checks.wg.Done()
		workCtx, cancel := context.WithTimeout(parent, checkTimeout)
		defer cancel()
		workCtx = context.WithValue(workCtx, checkRunContextKey{}, run.ID)
		var err error
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("check panicked: %v", recovered)
			}
			s.finishCheck(workCtx, err)
		}()
		if len(ids) > 0 {
			_, err = s.RefreshSubscribedPerformers(workCtx, ids)
		}
	}()
	return run, nil
}

// stable IDs simplify conflict detection and keep invalid single requests out of history.
func normalizeCheckID(id string) ([]string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrPerformerBatchEmpty
	}
	return []string{id}, nil
}

func sortCheckRuns(runs []CheckRun) {
	sort.Slice(runs, func(i, j int) bool { return runs[i].StartedAt.After(runs[j].StartedAt) })
}

// WaitBackground joins checks after the runtime context and HTTP admission stop.
func (s *Service) WaitBackground() {
	if s.checks != nil {
		s.checks.wg.Wait()
	}
}
