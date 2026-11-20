package main

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/logging"
)

func runUpcomingWorker(ctx context.Context, service graphqlapi.SubscriptionService) {
	worker, ok := service.(interface{ ProcessDueUpcoming(context.Context) error })
	if !ok {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		err := worker.ProcessDueUpcoming(checkCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logging.Warnf("upcoming release check: %v", err)
		}
		if retries, ok := service.(interface{ ProcessDueReviewRetries(context.Context) error }); ok && ctx.Err() == nil {
			retryCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			err := retries.ProcessDueReviewRetries(retryCtx)
			cancel()
			if err != nil && ctx.Err() == nil {
				logging.Warnf("release task retry: %v", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
