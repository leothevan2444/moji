package taskflow

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/releasedate"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func (s *Service) ResolveFilm(ctx context.Context, endpoint, id string) (film.Snapshot, error) {
	if s == nil || s.discoveredSceneResolver == nil {
		return film.Snapshot{}, film.ErrRequired
	}
	v, err := s.discoveredSceneResolver.ResolveDiscoveredScene(ctx, id, endpoint)
	if err != nil {
		return v, err
	}
	v.Endpoint = film.Endpoint(v.Endpoint)
	if v.Endpoint != film.Endpoint(endpoint) || v.SceneID != id {
		return v, film.ErrRequired
	}
	if err = v.Validate(); err != nil {
		return v, err
	}
	s.proxyFilm(ctx, &v)
	return v, nil
}

func (s *Service) proxyFilm(ctx context.Context, v *film.Snapshot) {
	if s.filmImageProxy != nil && v.ImageURL != "" && !strings.HasPrefix(v.ImageURL, "/api/images/") {
		v.ImageURL = s.filmImageProxy(ctx, v.Endpoint, v.ImageURL)
	}
}
func (s *Service) queueFilm(ctx context.Context, source taskruntime.TaskSource, v *film.Snapshot) (*taskruntime.Task, error) {
	if s == nil || s.taskRuntime == nil || v == nil || v.Validate() != nil {
		return nil, film.ErrRequired
	}
	if err := releasedate.Check(v.Date, time.Now(), s.timezone()); err == releasedate.ErrNotReleased {
		return nil, err
	}
	copy := v.Clone()
	s.proxyFilm(ctx, &copy)
	copy.ConfirmedAt = time.Now().UTC()
	return s.taskRuntime.QueueDownloadContext(ctx, taskruntime.DownloadRequest{Source: source, Code: copy.Code, Film: &copy})
}

type ConfirmFilmInput struct {
	Endpoint, SceneID, Digest string
	ResourceID, URL           string
	SavePath, Category, Tags  string
	Paused                    *bool
}

// ConfirmFilm is the only public admission operation for untrusted clients.
// Both identities are revalidated before the durable admission transaction.
func (s *Service) ConfirmFilm(ctx context.Context, input ConfirmFilmInput) (*taskruntime.Task, error) {
	v, err := s.ResolveFilm(ctx, input.Endpoint, input.SceneID)
	if err != nil {
		return nil, err
	}
	if input.Digest == "" || input.Digest != v.Digest() {
		return nil, film.ErrChanged
	}
	if err = releasedate.Check(v.Date, time.Now(), s.timezone()); err == releasedate.ErrNotReleased {
		return nil, err
	}
	v.ConfirmedAt = time.Now().UTC()
	if reader, ok := s.taskRuntime.(interface {
		ExistingFilmTask(context.Context, film.Snapshot) (*taskruntime.Task, error)
	}); ok {
		if existing, err := reader.ExistingFilmTask(ctx, v); err != nil || existing != nil {
			return existing, err
		}
	}
	if input.ResourceID != "" && input.URL != "" {
		return nil, errors.New("RESOURCE_INPUT_INVALID")
	}
	if input.ResourceID != "" {
		r, ok := s.taskRuntime.(interface {
			QueueIndexedResourceContext(context.Context, taskruntime.QueueIndexedResourceRequest) (*taskruntime.Task, error)
		})
		if !ok {
			return nil, errors.New("resource admission unavailable")
		}
		return r.QueueIndexedResourceContext(ctx, taskruntime.QueueIndexedResourceRequest{Film: &v, Code: v.Code, ResourceID: input.ResourceID, SavePath: input.SavePath, Category: input.Category, Tags: input.Tags, Paused: input.Paused})
	}
	if input.URL != "" {
		return s.taskRuntime.AddTorrentContext(ctx, taskruntime.AddTorrentRequest{Film: &v, URL: input.URL, SavePath: input.SavePath, Category: input.Category, Tags: input.Tags, Paused: input.Paused})
	}
	return s.taskRuntime.QueueDownloadContext(ctx, taskruntime.DownloadRequest{Film: &v, Code: v.Code, SavePath: input.SavePath, Category: input.Category, Tags: input.Tags, Paused: input.Paused})
}

// ResourceQueries is deliberately the only caller of the legacy extractor.
func ResourceQueries(title string) []string {
	queries := []string{}
	if code := suggestResourceCode(title); code != "" {
		queries = append(queries, code)
	}
	title = cleanResourceName(title)
	if title != "" && (len(queries) == 0 || queries[0] != title) {
		queries = append(queries, title)
	}
	return queries
}
