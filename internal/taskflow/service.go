package taskflow

import (
	"context"
	"errors"
	"strings"

	"github.com/leothevan2444/moji/internal/downloader"
)

// Service is Moji's application-layer task creation seam. Callers hand it
// business intent such as "manual torrent", "known code", or "known scene",
// and it translates that intent into standard downloader requests.
type Downloader interface {
	AddTorrentContext(ctx context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error)
	DownloadMediaContext(ctx context.Context, req downloader.DownloadRequest) (*downloader.Task, error)
}

type DiscoveredSceneResolver interface {
	ResolveDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (ResolvedScene, error)
}

type ResolvedScene struct {
	Code  string
	Title string
}

type Service struct {
	downloader              Downloader
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

func NewService(downloader Downloader) *Service {
	return &Service{downloader: downloader}
}

func (s *Service) SetDiscoveredSceneResolver(resolver DiscoveredSceneResolver) {
	if s == nil {
		return
	}
	s.discoveredSceneResolver = resolver
}

func (s *Service) CreateFromManualTorrent(ctx context.Context, input CreateFromManualTorrentInput) (*downloader.Task, error) {
	if s == nil || s.downloader == nil {
		return nil, errors.New("taskflow: downloader is not configured")
	}

	return s.downloader.AddTorrentContext(ctx, downloader.AddTorrentRequest{
		Source:   downloader.TaskSourceManual,
		URL:      input.URL,
		Paused:   input.Paused,
		SavePath: input.SavePath,
		Category: input.Category,
		Tags:     input.Tags,
	})
}

func (s *Service) CreateFromSearchCode(ctx context.Context, input CreateFromSearchCodeInput) (*downloader.Task, error) {
	if s == nil || s.downloader == nil {
		return nil, errors.New("taskflow: downloader is not configured")
	}

	return s.downloader.DownloadMediaContext(ctx, downloader.DownloadRequest{
		Source:     downloader.TaskSourceManual,
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

func (s *Service) CreateFromDiscoveredScene(ctx context.Context, input CreateFromDiscoveredSceneInput) (*downloader.Task, error) {
	return s.createFromCode(ctx, downloader.TaskSourceSearch, input.Code, input.Title)
}

func (s *Service) CreateFromDiscoveredSceneRef(ctx context.Context, input CreateFromDiscoveredSceneRefInput) (*downloader.Task, error) {
	if s == nil || s.downloader == nil {
		return nil, errors.New("taskflow: downloader is not configured")
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

func (s *Service) QueueDiscoveredScene(ctx context.Context, sceneID string, stashBoxEndpoint string) (*downloader.Task, error) {
	task, err := s.CreateFromDiscoveredSceneRef(ctx, CreateFromDiscoveredSceneRefInput{
		SceneID:          sceneID,
		StashBoxEndpoint: stashBoxEndpoint,
	})
	return task, err
}

func (s *Service) CreateFromSubscriptionRelease(ctx context.Context, input CreateFromSubscriptionReleaseInput) (*downloader.Task, error) {
	return s.createFromCode(ctx, downloader.TaskSourceSubscription, input.Code, input.Title)
}

func (s *Service) QueueSubscriptionRelease(ctx context.Context, code, title string) (*downloader.Task, error) {
	return s.CreateFromSubscriptionRelease(ctx, CreateFromSubscriptionReleaseInput{
		Code:  code,
		Title: title,
	})
}

func (s *Service) createFromCode(ctx context.Context, source downloader.TaskSource, code, title string) (*downloader.Task, error) {
	if s == nil || s.downloader == nil {
		return nil, errors.New("taskflow: downloader is not configured")
	}

	resolvedCode := buildReleaseCode(code, title)
	if resolvedCode == "" {
		return nil, errors.New("task code is required")
	}

	task, err := s.downloader.DownloadMediaContext(ctx, downloader.DownloadRequest{
		Source: source,
		Code:   resolvedCode,
	})
	return task, err
}

func buildReleaseCode(code, _ string) string {
	return strings.TrimSpace(code)
}
