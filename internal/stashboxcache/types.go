package stashboxcache

import (
	"context"
	"time"

	stashboxpkg "github.com/leothevan2444/moji/pkg/stashbox"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

const (
	PageSize              = 40
	DefaultTTL            = 24 * time.Hour
	DefaultStaleRetention = 30 * 24 * time.Hour
)

type Config struct {
	TTL            time.Duration
	StaleRetention time.Duration
}

func (c Config) normalize() Config {
	if c.TTL <= 0 {
		c.TTL = DefaultTTL
	}
	if c.StaleRetention <= 0 {
		c.StaleRetention = DefaultStaleRetention
	}
	return c
}

type ConfigProvider func() Config

type Client interface {
	FindPerformerByID(ctx context.Context, id string) (*stashboxgraphql.PerformerFragment, error)
	QueryScenesPage(ctx context.Context, input stashboxgraphql.SceneQueryInput) (stashboxpkg.ScenePage, error)
}

type FreshnessPolicy string

const (
	CachePreferred FreshnessPolicy = "CACHE_PREFERRED"
	RequireFresh   FreshnessPolicy = "REQUIRE_FRESH"
	ForceRefresh   FreshnessPolicy = "FORCE_REFRESH"
	RefreshHead    FreshnessPolicy = "REFRESH_HEAD"
)

type PerformerKey struct {
	Endpoint    string
	PerformerID string
}

type Result struct {
	Scenes      []*stashboxgraphql.SceneFragment
	RemoteCount int
	LoadedCount int
	Complete    bool
	UpdatedAt   time.Time
	Stale       bool
	Generation  int64
}

type Status struct {
	UsedBytes      int64
	SceneCount     int
	PerformerCount int
	SnapshotCount  int
	DatabasePath   string
	LastCleanupAt  *time.Time
	LastError      string
}

type performerDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type urlDTO struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

type imageDTO struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type namedDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sceneDTO struct {
	ID         string         `json:"id"`
	Title      *string        `json:"title,omitempty"`
	Code       *string        `json:"code,omitempty"`
	Details    *string        `json:"details,omitempty"`
	Duration   *int           `json:"duration,omitempty"`
	Date       *string        `json:"date,omitempty"`
	URLs       []urlDTO       `json:"urls,omitempty"`
	Images     []imageDTO     `json:"images,omitempty"`
	Studio     *namedDTO      `json:"studio,omitempty"`
	Tags       []namedDTO     `json:"tags,omitempty"`
	Performers []performerDTO `json:"performers,omitempty"`
}

type cachedPage struct {
	Number    int
	FetchedAt time.Time
	SceneIDs  []string
}

type snapshot struct {
	Key          PerformerKey
	Generation   int64
	RemoteCount  int
	Complete     bool
	UpdatedAt    time.Time
	LastAccessed time.Time
	Pages        []cachedPage
	Scenes       map[string]sceneDTO
}

func (s *snapshot) loadedCount() int {
	total := 0
	for _, page := range s.Pages {
		total += len(page.SceneIDs)
	}
	return total
}

func (s *snapshot) orderedScenes() []*stashboxgraphql.SceneFragment {
	if s == nil {
		return nil
	}
	out := make([]*stashboxgraphql.SceneFragment, 0, s.loadedCount())
	for _, page := range s.Pages {
		for _, id := range page.SceneIDs {
			if scene, ok := s.Scenes[id]; ok {
				out = append(out, scene.toGraphQL())
			}
		}
	}
	return out
}

func performerFromGraphQL(value *stashboxgraphql.PerformerFragment) performerDTO {
	if value == nil {
		return performerDTO{}
	}
	return performerDTO{ID: value.ID, Name: value.Name}
}

func (p performerDTO) toGraphQL() *stashboxgraphql.PerformerFragment {
	return &stashboxgraphql.PerformerFragment{ID: p.ID, Name: p.Name}
}

func sceneFromGraphQL(value *stashboxgraphql.SceneFragment) sceneDTO {
	if value == nil {
		return sceneDTO{}
	}
	out := sceneDTO{ID: value.ID, Title: value.Title, Code: value.Code, Details: value.Details, Duration: value.Duration, Date: value.Date}
	for _, item := range value.Urls {
		if item != nil {
			out.URLs = append(out.URLs, urlDTO{URL: item.URL, Type: item.Type})
		}
	}
	for _, item := range value.Images {
		if item != nil {
			out.Images = append(out.Images, imageDTO{ID: item.ID, URL: item.URL, Width: item.Width, Height: item.Height})
		}
	}
	if value.Studio != nil {
		out.Studio = &namedDTO{ID: value.Studio.ID, Name: value.Studio.Name}
	}
	for _, item := range value.Tags {
		if item != nil {
			out.Tags = append(out.Tags, namedDTO{ID: item.ID, Name: item.Name})
		}
	}
	for _, item := range value.Performers {
		if item != nil && item.Performer != nil {
			out.Performers = append(out.Performers, performerFromGraphQL(item.Performer))
		}
	}
	return out
}

func (s sceneDTO) toGraphQL() *stashboxgraphql.SceneFragment {
	out := &stashboxgraphql.SceneFragment{ID: s.ID, Title: s.Title, Code: s.Code, Details: s.Details, Duration: s.Duration, Date: s.Date}
	for _, item := range s.URLs {
		out.Urls = append(out.Urls, &stashboxgraphql.URLFragment{URL: item.URL, Type: item.Type})
	}
	for _, item := range s.Images {
		out.Images = append(out.Images, &stashboxgraphql.ImageFragment{ID: item.ID, URL: item.URL, Width: item.Width, Height: item.Height})
	}
	if s.Studio != nil {
		out.Studio = &stashboxgraphql.StudioFragment{ID: s.Studio.ID, Name: s.Studio.Name}
	}
	for _, item := range s.Tags {
		out.Tags = append(out.Tags, &stashboxgraphql.TagFragment{ID: item.ID, Name: item.Name})
	}
	for _, item := range s.Performers {
		out.Performers = append(out.Performers, &stashboxgraphql.PerformerAppearanceFragment{Performer: item.toGraphQL()})
	}
	return out
}
