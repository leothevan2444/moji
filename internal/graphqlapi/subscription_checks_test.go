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
	return subscription.HomeChecks{Enabled: true, LatestScheduled: &run, ActiveRuns: []subscription.HomeCheckRun{}, RecentRuns: []subscription.HomeCheckRun{run}}, nil
}
func (s *checkAPIService) CheckActivity(_ context.Context, ids []string) (subscription.CheckActivity, error) {
	s.activityIDs = ids
	return subscription.CheckActivity{ActiveCount: 1, Performers: []subscription.CheckActivityPerformer{{PerformerID: "p1", RunID: "active", Status: "RUNNING"}}}, nil
}

func TestHomeSubscriptionCheckGraphQLContract(t *testing.T) {
	now := time.Now().UTC()
	service := &checkAPIService{run: subscription.CheckRun{ID: "check-1", Trigger: "SCHEDULED", Scope: "ALL", Status: "PARTIAL", StartedAt: now, FinishedAt: &now, Items: []subscription.CheckItem{
		{PerformerID: "p1", Name: "One", Status: "SUCCEEDED"},
		{PerformerID: "p2", Name: "Two", Status: "FAILED", Error: "source unavailable"},
		{PerformerID: "p3", Name: "Three", Status: "SKIPPED"},
	}}}
	resolver := NewResolver(nil, nil, nil, nil, "test")
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
	executeGraphQLInto(t, resolver, `{homeSubscriptionChecks { enabled activeCount latestScheduled{id trigger status} activeRuns{id} recentRuns{id status totalCount completedCount succeededCount failedCount skippedCount currentPerformers{performerId name}} } subscriptionCheckActivity(ids:["p1"]){activeCount allRunId performers{performerId runId status}} }`, &result)
	if len(result.Errors) > 0 || !result.Data.HomeSubscriptionChecks.Enabled {
		t.Fatalf("invalid Home query result: %+v", result)
	}
	run := result.Data.HomeSubscriptionChecks.RecentRuns[0]
	if run.TotalCount != 3 || run.CompletedCount != 3 || run.FailedCount != 1 || run.SucceededCount != 1 || run.SkippedCount != 1 {
		t.Fatalf("incorrect counters: %+v", run)
	}
	if len(service.activityIDs) != 1 || service.activityIDs[0] != "p1" || len(result.Data.Activity.Performers) != 1 || result.Data.Activity.Performers[0].RunID != "active" {
		t.Fatalf("incorrect scoped activity: %+v", result)
	}
	executeGraphQLInto(t, resolver, `{subscriptionCheck(id:"check-1"){id startedAt finishedAt items{performerId error}}}`, &result)
	if len(result.Errors) > 0 || result.Data.SubscriptionCheck == nil || result.Data.SubscriptionCheck.Items[1].Error == nil {
		t.Fatalf("detail must load actor results separately: %+v", result)
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
	resolver := NewResolver(nil, nil, nil, nil, "test")
	var result struct {
		Data struct {
			Checks   model.HomeSubscriptionChecks    `json:"homeSubscriptionChecks"`
			Activity model.SubscriptionCheckActivity `json:"subscriptionCheckActivity"`
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `{homeSubscriptionChecks{enabled activeCount activeRuns{id} recentRuns{id}} subscriptionCheckActivity(ids:[]){activeCount performers{performerId}}}`, &result)
	if len(result.Errors) > 0 || result.Data.Checks.Enabled || result.Data.Checks.ActiveRuns == nil || result.Data.Checks.RecentRuns == nil || result.Data.Activity.Performers == nil {
		t.Fatalf("invalid unavailable state: %+v", result)
	}
}
