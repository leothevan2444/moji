package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

type checkAPIService struct {
	workspaceSubscriptionService
	run         subscription.CheckRun
	ids         []string
	trigger     string
	activityIDs []string
}

func (s *checkAPIService) StartSubscriptionCheck(_ context.Context, ids []string, trigger string) (*subscription.CheckRun, error) {
	s.ids, s.trigger = ids, trigger
	return &s.run, nil
}
func (s *checkAPIService) SubscriptionCheck(_ context.Context, id string) (*subscription.CheckRun, error) {
	if id == s.run.ID {
		return &s.run, nil
	}
	return nil, nil
}
func (s *checkAPIService) HomeChecks(context.Context) (subscription.HomeChecks, error) {
	run := subscription.HomeCheckRun{ID: s.run.ID, Trigger: s.run.Trigger, Scope: s.run.Scope, Status: s.run.Status, StartedAt: s.run.StartedAt, TotalCount: 3, CompletedCount: 3, SucceededCount: 1, FailedCount: 1, SkippedCount: 1}
	return subscription.HomeChecks{Enabled: true, LatestScheduled: &run, LatestFinishedScheduled: &run, Statistics: subscription.HomeCheckStatistics{RunCount: 10, SucceededCount: 81, FailedCount: 5, SkippedCount: 2}, ActiveRuns: []subscription.HomeCheckRun{}, RecentRuns: []subscription.HomeCheckRun{run}}, nil
}
func (s *checkAPIService) CheckActivity(_ context.Context, ids []string) (subscription.CheckActivity, error) {
	s.activityIDs = ids
	return subscription.CheckActivity{ActiveCount: 1, Performers: []subscription.CheckActivityPerformer{{PerformerID: "p1", RunID: "active", Status: "RUNNING"}}}, nil
}

func TestHomeSubscriptionCheckGraphQLContract(t *testing.T) {
	now := time.Now().UTC()
	service := &checkAPIService{run: subscription.CheckRun{ID: "check-1", Trigger: "SCHEDULED", Scope: "ALL", Status: "PARTIAL", StartedAt: now, FinishedAt: &now, Items: []subscription.CheckItem{
		{PerformerID: "p1", Name: "One", Status: "SUCCEEDED", Details: &subscription.CheckDetails{Version: 1, Phase: "FINISHED", CatalogStatus: "SUCCEEDED", CatalogPages: 2, CatalogScenes: 80, RetryTotal: 1, RetryAttempted: 1, RetryFailed: 1, Results: []subscription.CheckReleaseResult{{Key: "film", Code: "T38-043", Phase: "HISTORICAL_RETRY", Outcome: "FAILED", ReasonCode: "solo_behavior_download", ErrorCode: "UPSTREAM_TEMPORARY", Error: "source timeout", NextRetryAt: &now}}}},
		{PerformerID: "p2", Name: "Two", Status: "FAILED", Error: "source unavailable"},
		{PerformerID: "p3", Name: "Three", Status: "SKIPPED"},
	}}}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.PerformerSubscription = service
	var result struct {
		Data struct {
			HomeSubscriptionChecks model.HomeSubscriptionChecks    `json:"homeSubscriptionChecks"`
			SubscriptionCheck      *model.SubscriptionCheckRun     `json:"subscriptionCheck"`
			StartSubscriptionCheck model.SubscriptionCheckRun      `json:"startSubscriptionCheck"`
			Activity               model.SubscriptionCheckActivity `json:"subscriptionCheckActivity"`
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `{homeSubscriptionChecks { enabled activeCount latestScheduled{id trigger status} latestFinishedScheduled{id status} statistics{runCount succeededCount failedCount skippedCount} activeRuns{id} recentRuns{id status totalCount completedCount succeededCount failedCount skippedCount currentPerformers{performerId name}} } subscriptionCheckActivity(ids:["p1"]){activeCount allRunId performers{performerId runId status}} }`, &result)
	if len(result.Errors) > 0 || !result.Data.HomeSubscriptionChecks.Enabled {
		t.Fatalf("invalid Home query result: %+v", result)
	}
	stats := result.Data.HomeSubscriptionChecks.Statistics
	finished := result.Data.HomeSubscriptionChecks.LatestFinishedScheduled
	if stats == nil || stats.RunCount != 10 || stats.SucceededCount != 81 || stats.FailedCount != 5 || stats.SkippedCount != 2 || finished == nil || finished.ID != "check-1" {
		t.Fatalf("incorrect statistics or latest finished result: %+v", result.Data.HomeSubscriptionChecks)
	}
	run := result.Data.HomeSubscriptionChecks.RecentRuns[0]
	if run.TotalCount != 3 || run.CompletedCount != 3 || run.FailedCount != 1 || run.SucceededCount != 1 || run.SkippedCount != 1 {
		t.Fatalf("incorrect counters: %+v", run)
	}
	if len(service.activityIDs) != 1 || service.activityIDs[0] != "p1" || len(result.Data.Activity.Performers) != 1 || result.Data.Activity.Performers[0].RunID != "active" {
		t.Fatalf("incorrect scoped activity: %+v", result)
	}
	executeGraphQLInto(t, resolver, `{subscriptionCheck(id:"check-1"){id startedAt finishedAt items{performerId error details{catalogStatus catalogPages catalogScenes retryTotal retryFailed results{code phase outcome reasonCode errorCode error nextRetryAt}}}}}`, &result)
	if len(result.Errors) > 0 || result.Data.SubscriptionCheck == nil || result.Data.SubscriptionCheck.Items[1].Error == nil {
		t.Fatalf("detail must load actor results separately: %+v", result)
	}
	details := result.Data.SubscriptionCheck.Items[0].Details
	if details == nil || details.CatalogStatus != "SUCCEEDED" || details.RetryFailed != 1 || len(details.Results) != 1 || details.Results[0].ReasonCode != "solo_behavior_download" || details.Results[0].NextRetryAt == nil {
		t.Fatalf("missing independent retry diagnostics: %+v", details)
	}
	executeGraphQLInto(t, resolver, `mutation {startSubscriptionCheck(ids:["p1"]){id status}}`, &result)
	if len(result.Errors) > 0 || len(service.ids) != 1 || service.ids[0] != "p1" || service.trigger != "MANUAL" || result.Data.StartSubscriptionCheck.ID != "check-1" {
		t.Fatalf("invalid start result: %+v", result)
	}
	executeGraphQLInto(t, resolver, `mutation {startSubscriptionCheck{id}}`, &result)
	if len(result.Errors) > 0 || service.ids != nil {
		t.Fatalf("omitted ids must mean all subscriptions: %+v", service.ids)
	}
}

func TestHomeChecksUnavailableReturnsEmptyState(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, "test")
	var result struct {
		Data struct {
			Checks   model.HomeSubscriptionChecks    `json:"homeSubscriptionChecks"`
			Activity model.SubscriptionCheckActivity `json:"subscriptionCheckActivity"`
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `{homeSubscriptionChecks{enabled activeCount activeRuns{id} recentRuns{id} latestFinishedScheduled{id} statistics{runCount succeededCount failedCount skippedCount}} subscriptionCheckActivity(ids:[]){activeCount performers{performerId}}}`, &result)
	if result.Data.Checks.Statistics == nil || result.Data.Checks.Statistics.RunCount != 0 || result.Data.Checks.LatestFinishedScheduled != nil {
		t.Fatalf("unconfigured checks must retain an empty non-null statistics object: %+v", result)
	}
	if len(result.Errors) > 0 || result.Data.Checks.Enabled || result.Data.Checks.ActiveRuns == nil || result.Data.Checks.RecentRuns == nil || result.Data.Activity.Performers == nil {
		t.Fatalf("invalid unavailable state: %+v", result)
	}
}

func (s *checkAPIService) PerformerActivity(context.Context, string) (subscription.ActivityState, error) {
	value, rank := 3.25, 4
	return subscription.ActivityState{Value: &value, Rank: &rank, NextCheckAt: &s.run.StartedAt}, nil
}
func (s *checkAPIService) CheckHistory(ctx context.Context) ([]subscription.HomeCheckRun, error) {
	home, err := s.HomeChecks(ctx)
	return home.RecentRuns, err
}
func TestActivityAndHistoryGraphQLContract(t *testing.T) {
	service := &checkAPIService{run: subscription.CheckRun{ID: "retained", Trigger: "SCHEDULED", Scope: "SINGLE", Status: "SUCCEEDED", StartedAt: time.Now().UTC()}}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.PerformerSubscription = service
	var result struct {
		Data struct {
			Activity model.PerformerActivity           `json:"performerActivity"`
			History  []*model.HomeSubscriptionCheckRun `json:"subscriptionCheckHistory"`
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `{performerActivity(id:"p"){id evaluationStatus value rank lowerBound nextCheckAt} subscriptionCheckHistory{id totalCount failedCount}}`, &result)
	if len(result.Errors) > 0 || result.Data.Activity.ID != "p" || result.Data.Activity.EvaluationStatus != model.ActivityEvaluationStatusReady || result.Data.Activity.Value == nil || *result.Data.Activity.Value != 3.25 || result.Data.Activity.Rank == nil || *result.Data.Activity.Rank != 4 || len(result.Data.History) != 1 || result.Data.History[0].ID != "retained" {
		t.Fatalf("invalid activity/history response: %+v", result)
	}
}
