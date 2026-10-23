package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

type reviewAPIService struct {
	workspaceSubscriptionService
	query  subscription.ReleaseReviewQuery
	keys   []string
	action subscription.ReviewAction
}

func (s *reviewAPIService) ListReleaseReviews(_ context.Context, q subscription.ReleaseReviewQuery) (subscription.ReleaseReviewPage, error) {
	s.query = q
	return subscription.ReleaseReviewPage{Items: []subscription.ReleaseReview{{RecordedRelease: subscription.RecordedRelease{Key: "r1", Code: "ABCVR-123", Title: "Movie", SeenAt: time.Now(), DecisionReason: "vr_behavior_review"}, Status: subscription.ReviewPending, Performers: []subscription.ReviewPerformer{{ID: "p1", Name: "Actor"}}}}, Page: 2, PageSize: 10, TotalPages: 3, TotalCount: 21, PendingCount: 8, FailedCount: 1}, nil
}
func (s *reviewAPIService) ProcessReleaseReviews(_ context.Context, keys []string, action subscription.ReviewAction) (subscription.ReleaseReviewPayload, error) {
	s.keys = keys
	s.action = action
	return subscription.ReleaseReviewPayload{SucceededCount: 1, Results: []subscription.ReleaseReviewResult{{Key: "r1", Outcome: "SUCCEEDED", ReasonCode: "UPDATED", Release: &subscription.ReleaseReview{RecordedRelease: subscription.RecordedRelease{Key: "r1", Code: "ABCVR-123", SeenAt: time.Now()}, Status: subscription.ReviewRejected}}}}, nil
}
func (s *reviewAPIService) SynchronizeReleaseReviewTasks(context.Context) error { return nil }

func TestReleaseReviewGraphQLContract(t *testing.T) {
	service := &reviewAPIService{}
	resolver := NewResolver(nil, nil, nil, nil, "test")
	resolver.PerformerSubscription = service
	var result struct {
		Data struct {
			ReleaseReviews        model.ReleaseReviewConnection `json:"releaseReviews"`
			ProcessReleaseReviews model.ReleaseReviewPayload    `json:"processReleaseReviews"`
		}
		Errors []struct{ Message string }
	}
	executeGraphQLInto(t, resolver, `query { releaseReviews(input:{performerId:"p1",vr:true,status:PENDING,page:2,pageSize:10,search:"abc"}) { pendingCount failedCount page totalCount items {key code vr status performers{id name} } } }`, &result)
	if len(result.Errors) > 0 || !result.Data.ReleaseReviews.Items[0].Vr || service.query.PerformerID != "p1" || service.query.VR == nil || !*service.query.VR || service.query.Page != 2 || service.query.PageSize != 10 || service.query.Search != "abc" {
		t.Fatalf("incorrect query contract: %+v %+v", result, service.query)
	}
	executeGraphQLInto(t, resolver, `mutation { processReleaseReviews(input:{keys:["r1"],action:REJECT}) { succeededCount results { key outcome release { key status } } } }`, &result)
	if len(result.Errors) > 0 || service.action != subscription.ReviewReject || len(service.keys) != 1 || result.Data.ProcessReleaseReviews.Results[0].Release.Status != model.ReleaseReviewStatusRejected {
		t.Fatalf("incorrect action contract: %+v", result)
	}
}
