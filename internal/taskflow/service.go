package taskflow

import (
	"context"
	"errors"
	"strings"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

// Service is Moji's application-layer task creation seam. Callers hand it
// business intent such as "manual torrent", "known code", or "known scene",
// and it translates that intent into standard taskruntime requests.
type TaskRuntime interface {
	AddTorrentContext(ctx context.Context, req taskruntime.AddTorrentRequest) (*taskruntime.Task, error)
	DownloadMediaContext(ctx context.Context, req taskruntime.DownloadRequest) (*taskruntime.Task, error)
}

type DiscoveredSceneResolver interface {
	ResolveDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (ResolvedScene, error)
}

type ResolvedScene struct {
	Code  string
	Title string
}

type Service struct {
	taskRuntime             TaskRuntime
	discoveredSceneResolver DiscoveredSceneResolver
}

type CreateFromManualTorrentInput struct {
	URL      string
	Paused   *bool
	SavePath string
	Category string
	Tags     string
}

type CreateFromSearchCodeInput struct {
	Code       string
	Trackers   []string
	Categories []int
	Limit      int
	Paused     *bool
	SavePath   string
	Category   string
	Tags       string
}

type CreateFromDiscoveredSceneInput struct {
	Code  string
	Title string
}

type CreateFromDiscoveredSceneRefInput struct {
	SceneID          string
	StashBoxEndpoint string
}

type CreateFromSubscriptionReleaseInput struct {
	Code  string
	Title string
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

func (s *Service) CreateFromManualTorrent(ctx context.Context, input CreateFromManualTorrentInput) (*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, errors.New("taskflow: task runtime is not configured")
	}

	return s.taskRuntime.AddTorrentContext(ctx, taskruntime.AddTorrentRequest{
		Source:   taskruntime.TaskSourceManual,
		URL:      input.URL,
		Paused:   input.Paused,
		SavePath: input.SavePath,
		Category: input.Category,
		Tags:     input.Tags,
	})
}

func (s *Service) CreateFromSearchCode(ctx context.Context, input CreateFromSearchCodeInput) (*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, errors.New("taskflow: task runtime is not configured")
	}

	return s.taskRuntime.DownloadMediaContext(ctx, taskruntime.DownloadRequest{
		Source:     taskruntime.TaskSourceManual,
		Code:       input.Code,
		Trackers:   input.Trackers,
		Categories: input.Categories,
		Limit:      input.Limit,
		Paused:     input.Paused,
		SavePath:   input.SavePath,
		Category:   input.Category,
		Tags:       input.Tags,
	})
}

func (s *Service) CreateFromDiscoveredScene(ctx context.Context, input CreateFromDiscoveredSceneInput) (*taskruntime.Task, error) {
	return s.createFromCode(ctx, taskruntime.TaskSourceSearch, input.Code, input.Title)
}

func (s *Service) CreateFromDiscoveredSceneRef(ctx context.Context, input CreateFromDiscoveredSceneRefInput) (*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, errors.New("taskflow: task runtime is not configured")
	}
	if s.discoveredSceneResolver == nil {
		return nil, errors.New("taskflow: discovered scene resolver is not configured")
	}

	resolved, err := s.discoveredSceneResolver.ResolveDiscoveredScene(ctx, input.SceneID, input.StashBoxEndpoint)
	if err != nil {
		return nil, err
	}
	return s.CreateFromDiscoveredScene(ctx, CreateFromDiscoveredSceneInput{
		Code:  resolved.Code,
		Title: resolved.Title,
	})
}

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*taskruntime.Task, error) {
	task, err := s.CreateFromDiscoveredSceneRef(ctx, CreateFromDiscoveredSceneRefInput{
		SceneID:          sceneID,
		StashBoxEndpoint: stashBoxEndpoint,
	})
	return task, err
}

func (s *Service) CreateFromSubscriptionRelease(ctx context.Context, input CreateFromSubscriptionReleaseInput) (*taskruntime.Task, error) {
	return s.createFromCode(ctx, taskruntime.TaskSourceSubscription, input.Code, input.Title)
}

func (s *Service) QueueSubscriptionRelease(ctx context.Context, code, title string) (*taskruntime.Task, error) {
	return s.CreateFromSubscriptionRelease(ctx, CreateFromSubscriptionReleaseInput{
		Code:  code,
		Title: title,
	})
}

func (s *Service) createFromCode(ctx context.Context, source taskruntime.TaskSource, code, title string) (*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil {
		return nil, errors.New("taskflow: task runtime is not configured")
	}

	resolvedCode := buildReleaseCode(code, title)
	if resolvedCode == "" {
		return nil, errors.New("task code is required")
	}

	task, err := s.taskRuntime.DownloadMediaContext(ctx, taskruntime.DownloadRequest{
		Source: source,
		Code:   resolvedCode,
	})
	return task, err
}

func buildReleaseCode(code, _ string) string {
	return strings.TrimSpace(code)
}
