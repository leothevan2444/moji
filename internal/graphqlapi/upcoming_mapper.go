package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
	"time"
)

type upcomingService interface {
	ListUpcomingReleases(context.Context, subscription.UpcomingQuery) (subscription.UpcomingPage, error)
	RefreshUpcomingRelease(context.Context, string) (*subscription.UpcomingRelease, error)
}

func upcomingToModel(r subscription.UpcomingRelease) *model.UpcomingRelease {
	out := &model.UpcomingRelease{Key: r.Release.Key, Code: r.Release.Code, Title: r.Release.Title, Date: r.Release.Date, ImageURL: r.Release.ImageURL, SourceURL: r.Release.URL, State: r.State, Decision: string(r.Release.Decision), ErrorCode: r.LastErrorCode, MetadataCheckedAt: r.MetadataCheckedAt.Format(time.RFC3339), Performers: []*model.UpcomingPerformer{}}
	for _, a := range r.Performers {
		out.Performers = append(out.Performers, &model.UpcomingPerformer{ID: a.ID, Name: a.Name})
	}
	return out
}
