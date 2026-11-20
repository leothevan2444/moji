package main

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/logging"
)

// The timer only wakes the dispatcher. Durable per-actor deadlines belong to
// subscription, and are never recomputed from process startup time.
func startSubscriptionWorker(ctx context.Context, service graphqlapi.SubscriptionService) <-chan struct{} {
	done := make(chan struct{})
	tracked, ok := service.(interface {
		ConfigureCheckRuntime(context.Context)
		SetCheckSchedule(bool, *time.Time)
		DispatchDueCheck(context.Context) error
	})
	if !ok {
		close(done)
		return done
	}
	tracked.ConfigureCheckRuntime(ctx)
	go func() {
		defer close(done)
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		var nextDispatch time.Time
		tracked.SetCheckSchedule(true, nil)
		dispatch := func() {
			if time.Now().Before(nextDispatch) {
				return
			}
			workCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
			defer cancel()
			if err := tracked.DispatchDueCheck(workCtx); err != nil && ctx.Err() == nil {
				logging.Errorf("subscription schedule: %v", err)
			}
			nextDispatch = time.Now().Add(30 * time.Second)
		}
		dispatch()
		for {
			select {
			case <-ctx.Done():
				tracked.SetCheckSchedule(false, nil)
				return
			case <-tick.C:
				dispatch()
			}
		}
	}()
	return done
}
