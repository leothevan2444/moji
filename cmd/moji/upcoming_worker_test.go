package main

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"testing"
)

type upcomingWorkerService struct {
	graphqlapi.SubscriptionService
	calls  int
	cancel context.CancelFunc
}

func (s *upcomingWorkerService) ProcessDueUpcoming(context.Context) error {
	s.calls++
	s.cancel()
	return nil
}
func TestUpcomingWorkerRunsAutomaticallyOnStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &upcomingWorkerService{cancel: cancel}
	runUpcomingWorker(ctx, service)
	if service.calls != 1 {
		t.Fatalf("automatic upcoming check calls=%d", service.calls)
	}
}
