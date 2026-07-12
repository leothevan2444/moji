// Package performer owns the performer catalog application boundary.
//
// The catalog is intentionally expressed through a narrow backend contract so
// callers do not depend on subscription polling or StashBox administration.
package performer

import (
	"context"

	"github.com/leothevan2444/moji/internal/subscription"
)

type Catalog interface {
	ListStashPerformers(context.Context, string) ([]subscription.Performer, error)
	GetPerformerDetail(context.Context, string) (subscription.PerformerDetail, error)
	ListPerformerScenes(context.Context, string, subscription.PerformerSceneQuery) (subscription.PerformerScenePage, error)
	QueuePerformerScenes(context.Context, string, []subscription.QueuePerformerSceneSelection) (subscription.QueuePerformerScenesResult, error)
}

type Service struct{ catalog Catalog }

func NewService(catalog Catalog) *Service { return &Service{catalog: catalog} }

func (s *Service) ListStashPerformers(ctx context.Context, search string) ([]subscription.Performer, error) {
	return s.catalog.ListStashPerformers(ctx, search)
}

func (s *Service) GetPerformerDetail(ctx context.Context, id string) (subscription.PerformerDetail, error) {
	return s.catalog.GetPerformerDetail(ctx, id)
}

func (s *Service) ListPerformerScenes(ctx context.Context, id string, query subscription.PerformerSceneQuery) (subscription.PerformerScenePage, error) {
	return s.catalog.ListPerformerScenes(ctx, id, query)
}

func (s *Service) QueuePerformerScenes(ctx context.Context, id string, scenes []subscription.QueuePerformerSceneSelection) (subscription.QueuePerformerScenesResult, error) {
	return s.catalog.QueuePerformerScenes(ctx, id, scenes)
}
