package main

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/logging"
)

func runUpcomingWorker(ctx context.Context, service graphqlapi.SubscriptionService, interval func() time.Duration) {
	worker, ok := service.(interface{ ProcessDueUpcoming(context.Context) error })
	if !ok {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if interval() > 0 {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			err := worker.ProcessDueUpcoming(checkCtx)
			cancel()
			if err != nil && ctx.Err() == nil {
				logging.Warnf("upcoming release check: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
