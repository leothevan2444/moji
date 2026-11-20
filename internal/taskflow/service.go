package taskflow

import (
	"context"
	"github.com/leothevan2444/moji/internal/apptime"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

// Service is Moji's application-layer task creation seam. Callers hand it
// a verified StashBox film and an optional explicitly chosen resource,
// and it translates that intent into standard taskruntime requests.
type TaskRuntime interface {
	AddTorrentContext(ctx context.Context, req taskruntime.AddTorrentRequest) (*taskruntime.Task, error)
	QueueDownloadContext(ctx context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error)
}

type DiscoveredSceneResolver interface {
	ResolveDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (ResolvedScene, error)
}

type ResolvedScene = film.Snapshot

type Service struct {
	timezoneProvider        func() string
	taskRuntime             TaskRuntime
	discoveredSceneResolver DiscoveredSceneResolver
	filmImageProxy          func(context.Context, string, string) string
}

func NewService(taskRuntime TaskRuntime) *Service {
	return &Service{taskRuntime: taskRuntime}
}

func (s *Service) SetDiscoveredSceneResolver(resolver DiscoveredSceneResolver) {
	if s == nil {
		return
	}
	s.discoveredSceneResolver = resolver
}

func (s *Service) ListTasks(ctx context.Context) ([]*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, nil
	}
	if lister, ok := s.taskRuntime.(interface {
		ListTasks(context.Context) ([]*taskruntime.Task, error)
	}); ok {
		return lister.ListTasks(ctx)
	}
	return nil, nil
}

// QueueResolvedScene reuses a fresh server observation, or resolves a reference.
func (s *Service) QueueResolvedScene(ctx context.Context, target film.Snapshot) (*taskruntime.Task, error) {
	resolved, err := s.resolveTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return s.queueFilm(ctx, taskruntime.TaskSourceSearch, &resolved)
}

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID, endpoint string) (*taskruntime.Task, error) {
	return s.QueueResolvedScene(ctx, film.Snapshot{Endpoint: endpoint, SceneID: sceneID})
}

func (s *Service) QueueSubscriptionRelease(ctx context.Context, target film.Snapshot) (*taskruntime.Task, error) {
	resolved, err := s.resolveTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return s.queueFilm(ctx, taskruntime.TaskSourceSubscription, &resolved)
}

func (s *Service) SetFilmImageProxy(proxy func(context.Context, string, string) string) {
	s.filmImageProxy = proxy
}

func (s *Service) resolveTarget(ctx context.Context, target film.Snapshot) (film.Snapshot, error) {
	// Only trusted in-process catalog callers can supply an observation. Browser
	// admission always uses ConfirmFilm and revalidates the selected identity.
	if !target.ConfirmedAt.IsZero() && target.Validate() == nil {
		return target.Clone(), nil
	}
	return s.ResolveFilm(ctx, target.Endpoint, target.SceneID)
}

// ListReviewTasks exposes the narrow identity projection used by subscription
// reconciliation; it never needs file plans, checkpoints or attachments.
func (s *Service) ListReviewTasks(ctx context.Context) ([]*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, nil
	}
	if provider, ok := s.taskRuntime.(interface {
		ListReviewTasks(context.Context) ([]*taskruntime.Task, error)
	}); ok {
		return provider.ListReviewTasks(ctx)
	}
	return s.ListTasks(ctx)
}

func (s *Service) SetTimezone(provider func() string) { s.timezoneProvider = provider }
func (s *Service) timezone() string {
	if s.timezoneProvider != nil {
		return s.timezoneProvider()
	}
	return apptime.DefaultTimezone
}
