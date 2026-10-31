package main

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"testing"
	"time"
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
func TestUpcomingWorkerStartupAndDisabledSchedule(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		service := &upcomingWorkerService{cancel: cancel}
		runUpcomingWorker(ctx, service, func() time.Duration {
			if !enabled {
				cancel()
				return 0
			}
			return time.Hour
		})
		cancel()
		want := 0
		if enabled {
			want = 1
		}
		if service.calls != want {
			t.Fatalf("enabled=%v calls=%d", enabled, service.calls)
		}
	}
}
