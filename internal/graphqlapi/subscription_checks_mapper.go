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
		out.Items = append(out.Items, &model.SubscriptionCheckItem{PerformerID: item.PerformerID, Name: item.Name, Status: model.SubscriptionCheckStatus(item.Status), Error: nilIfEmpty(item.Error), StartedAt: checkTime(item.StartedAt), FinishedAt: checkTime(item.FinishedAt), Details: checkDetailsToModel(item.Details)})
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

func checkDetailsToModel(d *subscription.CheckDetails) *model.SubscriptionCheckDetails {
	if d == nil {
		return nil
	}
	out := &model.SubscriptionCheckDetails{
		Phase: d.Phase, CatalogStatus: d.CatalogStatus, CatalogErrorCode: d.CatalogErrorCode, CatalogSource: d.CatalogSource, CatalogMode: d.CatalogMode, CatalogStopReason: d.CatalogStopReason,
		CatalogPages: d.CatalogPages, CatalogPagesWithResults: d.CatalogPagesWithResults, CatalogScenes: d.CatalogScenes, CatalogReleases: d.CatalogReleases,
		NewCount: d.NewCount, KnownCount: d.KnownCount, UpcomingCount: d.UpcomingCount, InLibraryCount: d.InLibraryCount, QueuedCount: d.QueuedCount, ReviewCount: d.ReviewCount, SkippedCount: d.SkippedCount, CreationFailedCount: d.CreationFailedCount,
		RetryTotal: d.RetryTotal, RetryAttempted: d.RetryAttempted, RetrySucceeded: d.RetrySucceeded, RetryFailed: d.RetryFailed, RetryDeferred: d.RetryDeferred, OmittedResults: d.OmittedResults, Results: []*model.SubscriptionCheckReleaseResult{},
	}
	for _, r := range d.Results {
		out.Results = append(out.Results, &model.SubscriptionCheckReleaseResult{Key: r.Key, Code: r.Code, Title: r.Title, Phase: r.Phase, Outcome: r.Outcome, ReasonCode: r.ReasonCode, ErrorCode: r.ErrorCode, Error: r.Error, TaskID: nilIfEmpty(r.TaskID), NextRetryAt: checkTime(r.NextRetryAt)})
	}
	return out
}
