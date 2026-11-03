package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/subscription"
)

type upcomingAPIService struct {
	workspaceSubscriptionService
	query subscription.UpcomingQuery
}

func (s *upcomingAPIService) ListUpcomingReleases(_ context.Context, q subscription.UpcomingQuery) (subscription.UpcomingPage, error) {
	s.query = q
	return subscription.UpcomingPage{Items: []subscription.UpcomingRelease{{Release: subscription.Release{Key: "future", Code: "ABC-123", Date: "2026-11-01"}, State: "WAITING", MetadataCheckedAt: time.Now(), Performers: []subscription.ReviewPerformer{{ID: "p1", Name: "Actor"}}}}, Performers: []subscription.ReviewPerformer{{ID: "p1", Name: "Actor"}}, TotalCount: 1, Page: 1, PageSize: 8, TotalPages: 1, SubscribedCount: 1, Timezone: "Asia/Shanghai"}, nil
}
func (s *upcomingAPIService) RefreshUpcomingRelease(ctx context.Context, key string) (*subscription.UpcomingRelease, error) {
	p, _ := s.ListUpcomingReleases(ctx, s.query)
	return &p.Items[0], nil
}
func TestUpcomingGraphQLContract(t *testing.T) {
	service := &upcomingAPIService{}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.PerformerSubscription = service
	var response struct {
		Data struct {
			UpcomingReleases struct {
				TotalCount int
				Timezone   string
				Items      []struct{ Key, Code, Date, State string }
			}
		}
		Errors []any
	}
	executeGraphQLInto(t, resolver, `query{upcomingReleases(input:{days:7,pageSize:8,performerId:"p1"}){totalCount timezone items{key code date state performers{id name}}}}`, &response)
	if len(response.Errors) > 0 || response.Data.UpcomingReleases.TotalCount != 1 || len(response.Data.UpcomingReleases.Items) != 1 || service.query.Days != 7 || service.query.PerformerID != "p1" {
		t.Fatalf("contract %+v query %+v", response, service.query)
	}
	var mutation struct {
		Data   struct{ RefreshUpcomingRelease struct{ Key string } }
		Errors []any
	}
	executeGraphQLInto(t, resolver, `mutation{refreshUpcomingRelease(key:"future"){key state}}`, &mutation)
	if len(mutation.Errors) > 0 || mutation.Data.RefreshUpcomingRelease.Key != "future" {
		t.Fatalf("mutation %+v", mutation)
	}
}

func TestPerformerUpcomingStateGraphQLContract(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.Performer = &fakePerformerService{performerPage: performer.ScenePage{Items: []performer.Scene{{Key: "future", UpcomingState: "WAITING"}, {Key: "ordinary"}}, Page: 1, PageSize: 24}}
	for _, query := range []string{`query{stashPerformerScenes(id:"p1",input:{}){items{key upcomingState}}}`, `mutation{refreshStashPerformerScenes(id:"p1",input:{}){items{key upcomingState}}}`} {
		var response struct {
			Data struct {
				StashPerformerScenes struct {
					Items []struct {
						Key           string
						UpcomingState *string
					}
				}
				RefreshStashPerformerScenes struct {
					Items []struct {
						Key           string
						UpcomingState *string
					}
				}
			}
			Errors []any
		}
		executeGraphQLInto(t, resolver, query, &response)
		items := response.Data.StashPerformerScenes.Items
		if items == nil {
			items = response.Data.RefreshStashPerformerScenes.Items
		}
		if len(response.Errors) > 0 || len(items) != 2 || items[0].UpcomingState == nil || *items[0].UpcomingState != "WAITING" || items[1].UpcomingState != nil {
			t.Fatalf("state contract: %+v", response)
		}
	}
}
