package main

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi"
)

type checkWorkerService struct {
	graphqlapi.SubscriptionService
	configured context.Context
	schedules  chan *time.Time
	calls      chan bool
}

func (s *checkWorkerService) ConfigureCheckRuntime(ctx context.Context)  { s.configured = ctx }
func (s *checkWorkerService) SetCheckSchedule(_ bool, next *time.Time)   { s.schedules <- next }
func (s *checkWorkerService) DispatchDueCheck(ctx context.Context) error { s.calls <- true; return nil }

func TestSubscriptionWorkerDispatchesOnStartupAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &checkWorkerService{schedules: make(chan *time.Time, 10), calls: make(chan bool, 10)}
	done := startSubscriptionWorker(ctx, s)
	if s.configured != ctx {
		t.Fatal("checks must use runtime lifetime")
	}
	select {
	case <-s.calls:
	case <-time.After(time.Second):
		t.Fatal("startup did not reconcile durable schedules")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	<-s.schedules
	if next := <-s.schedules; next != nil {
		t.Fatal("shutdown must clear displayed deadline")
	}
}
func TestUnavailableSubscriptionServiceDoesNotStartWorker(t *testing.T) {
	done := startSubscriptionWorker(context.Background(), nil)
	select {
	case <-done:
	default:
		t.Fatal("unavailable service started worker")
	}
}
