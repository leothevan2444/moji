package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

func checkTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	text := value.UTC().Format(time.RFC3339Nano)
	return &text
}

func subscriptionCheckToModel(run *subscription.CheckRun) *model.SubscriptionCheckRun {
	if run == nil {
		return nil
	}
	out := &model.SubscriptionCheckRun{ID: run.ID, Trigger: model.SubscriptionCheckTrigger(run.Trigger), Scope: model.SubscriptionCheckScope(run.Scope), Status: model.SubscriptionCheckStatus(run.Status), StartedAt: run.StartedAt.UTC().Format(time.RFC3339Nano), FinishedAt: checkTime(run.FinishedAt), Error: nilIfEmpty(run.Error), TotalCount: len(run.Items), Items: []*model.SubscriptionCheckItem{}}
	for _, item := range run.Items {
		out.Items = append(out.Items, &model.SubscriptionCheckItem{PerformerID: item.PerformerID, Name: item.Name, Status: model.SubscriptionCheckStatus(item.Status), Error: nilIfEmpty(item.Error), StartedAt: checkTime(item.StartedAt), FinishedAt: checkTime(item.FinishedAt)})
		switch item.Status {
		case "SUCCEEDED":
			out.SucceededCount++
			out.CompletedCount++
		case "SKIPPED":
			out.SkippedCount++
			out.CompletedCount++
		case "FAILED", "INTERRUPTED":
			out.FailedCount++
			out.CompletedCount++
		}
	}
	return out
}
