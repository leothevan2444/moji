// Package discovery owns interactive metadata discovery and candidate queuing.
package discovery

import (
	"context"

	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type Backend interface {
	SearchPreferredStashBoxScenes(context.Context, string, int, subscription.DiscoverSort) (subscription.DiscoverScenePage, error)
	QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error)
}

type Service struct{ backend Backend }

func NewService(backend Backend) *Service { return &Service{backend: backend} }

func (s *Service) SearchPreferredStashBoxScenes(ctx context.Context, query string, limit int, sort subscription.DiscoverSort) (subscription.DiscoverScenePage, error) {
	return s.backend.SearchPreferredStashBoxScenes(ctx, query, limit, sort)
}

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID, endpoint string) (*taskruntime.Task, error) {
	return s.backend.QueueDiscoveredScene(ctx, sceneID, endpoint)
}
