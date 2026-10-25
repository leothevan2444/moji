package main

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/subscription"
	"sync/atomic"
)

type checkWorkerService struct {
	graphqlapi.SubscriptionService
	configured context.Context
	schedules  chan *time.Time
	calls      chan string
	startErr   error
}

func (s *checkWorkerService) ConfigureCheckRuntime(ctx context.Context) { s.configured = ctx }
func (s *checkWorkerService) SetCheckSchedule(_ bool, next *time.Time)  { s.schedules <- next }
func (s *checkWorkerService) StartSubscriptionCheck(_ context.Context, _ []string, trigger string) (*subscription.CheckRun, error) {
	s.calls <- trigger
	if s.startErr != nil {
		return nil, s.startErr
	}
	return &subscription.CheckRun{ID: "scheduled"}, nil
}

func TestSubscriptionWorkerPublishesDeadlineAndUsesTrackedBackgroundCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &checkWorkerService{schedules: make(chan *time.Time, 10), calls: make(chan string, 10)}
	before := time.Now()
	startSubscriptionWorker(ctx, service, func() time.Duration { return 20 * time.Millisecond })
	if service.configured != ctx {
		t.Fatal("background checks must use runtime lifetime")
	}
	select {
	case next := <-service.schedules:
		if next == nil || !next.After(before) {
			t.Fatal("missing actual schedule deadline")
		}
	case <-time.After(time.Second):
		t.Fatal("missing schedule")
	}
	select {
	case trigger := <-service.calls:
		if trigger != "SCHEDULED" {
			t.Fatalf("wrong origin: %s", trigger)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduled check did not dispatch")
	}
	cancel()
	for {
		select {
		case next := <-service.schedules:
			if next == nil {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("worker did not clear schedule on shutdown")
		}
	}
}

func TestSubscriptionWorkerCanEnablePollingWithoutRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &checkWorkerService{schedules: make(chan *time.Time, 10), calls: make(chan string, 10)}
	var interval atomic.Int64
	startSubscriptionWorker(ctx, service, func() time.Duration { return time.Duration(interval.Load()) })
	select {
	case next := <-service.schedules:
		if next != nil {
			t.Fatal("disabled worker must not report a deadline")
		}
	case <-time.After(time.Second):
		t.Fatal("missing disabled state")
	}
	interval.Store(int64(20 * time.Millisecond))
	select {
	case trigger := <-service.calls:
		if trigger != "SCHEDULED" {
			t.Fatal("missing scheduled trigger")
		}
	case <-time.After(6 * time.Second):
		t.Fatal("enabling polling did not start the worker")
	}
}

func TestUnavailableSubscriptionServiceDoesNotStartWorker(t *testing.T) {
	runtime := newHTTPRuntime(testConfig(), "test", nil)
	if runtime.subscriptionService != nil {
		t.Fatal("unconfigured subscription service must remain a nil interface")
	}
	startSubscriptionWorker(context.Background(), runtime.subscriptionService, func() time.Duration { return time.Millisecond })
}

func TestSubscriptionWorkerDefersConflictingScheduledCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &checkWorkerService{schedules: make(chan *time.Time, 10), calls: make(chan string, 10), startErr: &subscription.CheckConflictError{RunID: "manual-active"}}
	var reads atomic.Int32
	startSubscriptionWorker(ctx, service, func() time.Duration {
		if reads.Add(1) == 1 {
			return 20 * time.Millisecond
		}
		return time.Hour
	})
	<-service.schedules
	select {
	case next := <-service.schedules:
		if next == nil || time.Until(*next) > 31*time.Second || time.Until(*next) < 29*time.Second {
			t.Fatalf("conflicting scheduled check must retry soon: %v", next)
		}
	case <-time.After(time.Second):
		t.Fatal("missing retry deadline")
	}
}
