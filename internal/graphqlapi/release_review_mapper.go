package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/subscription"
	"strings"
)

type ReleaseReviewService interface {
	ListReleaseReviews(context.Context, subscription.ReleaseReviewQuery) (subscription.ReleaseReviewPage, error)
	ProcessReleaseReviews(context.Context, []string, subscription.ReviewAction) (subscription.ReleaseReviewPayload, error)
	SynchronizeReleaseReviewTasks(context.Context) error
}

func releaseReviewToModel(record *subscription.ReleaseReview) *model.ReleaseReview {
	if record == nil {
		return nil
	}
	out := &model.ReleaseReview{Key: record.Key, Code: record.Code, Title: record.Title, Date: nilIfEmpty(record.Date), URL: nilIfEmpty(record.URL), ImageURL: nilIfEmpty(record.ImageURL), Source: record.Source, Vr: strings.Contains(strings.ToUpper(record.Code), "VR"), Status: model.ReleaseReviewStatus(record.Status), ReasonCode: record.DecisionReason, Error: nilIfEmpty(record.Error), TaskID: nilIfEmpty(record.TaskID), SeenAt: formatTime(record.SeenAt), ReviewedAt: formatTimePointer(record.ReviewedAt), Performers: []*model.ReleaseReviewPerformer{}, PerformerNames: append([]string{}, record.PerformerNames...)}
	for _, actor := range record.Performers {
		name := actor.Name
		if name == "" {
			name = actor.ID
		}
		out.Performers = append(out.Performers, &model.ReleaseReviewPerformer{ID: actor.ID, Name: name})
	}
	return out
}

func (r *Resolver) synchronizeReleaseReviews(ctx context.Context) {
	if service, ok := r.PerformerSubscription.(ReleaseReviewService); ok {
		if err := service.SynchronizeReleaseReviewTasks(ctx); err != nil {
			logging.Errorf("subscription: synchronize manually created task: %v", err)
		}
	}
}
