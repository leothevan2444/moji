package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

func homeCheckRunToModel(run subscription.HomeCheckRun) *model.HomeSubscriptionCheckRun {
	out := &model.HomeSubscriptionCheckRun{ID: run.ID, Trigger: model.SubscriptionCheckTrigger(run.Trigger), Scope: model.SubscriptionCheckScope(run.Scope), Status: model.SubscriptionCheckStatus(run.Status), StartedAt: run.StartedAt.UTC().Format(time.RFC3339Nano), FinishedAt: checkTime(run.FinishedAt), TotalCount: run.TotalCount, CompletedCount: run.CompletedCount, SucceededCount: run.SucceededCount, FailedCount: run.FailedCount, SkippedCount: run.SkippedCount, CurrentPerformers: []*model.HomeCheckPerformer{}}
	for _, actor := range run.CurrentPerformers {
		out.CurrentPerformers = append(out.CurrentPerformers, &model.HomeCheckPerformer{PerformerID: actor.PerformerID, Name: actor.Name})
	}
	return out
}
