package discovery

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type Sort string

const (
	SortRelevance    Sort = "RELEVANCE"
	SortDateDesc     Sort = "DATE_DESC"
	SortDateAsc      Sort = "DATE_ASC"
	SortDurationDesc Sort = "DURATION_DESC"
	SortTitleAsc     Sort = "TITLE_ASC"
)

type MatchedSource struct{ Name, Endpoint, PerformerID, PerformerName string }
type Scene struct {
	Key, SceneID, StashBoxEndpoint, StashBoxName, Title string
	DurationSeconds                                     *int
	Code, Date, StudioName, ImageURL, URL               string
	PerformerNames                                      []string
	DerivedQuery                                        string
}
type Page struct {
	Items         []Scene
	UsedStashBox  *MatchedSource
	FallbackCount int
	SearchedQuery string
}

type TaskCreator interface {
	QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error)
}
type ImageProxy func(context.Context, string, string) string

type Service struct {
	metadata *metadata.Service
	tasks    TaskCreator
	images   ImageProxy
}

func NewService(source *metadata.Service, tasks TaskCreator, images ImageProxy) *Service {
	return &Service{metadata: source, tasks: tasks, images: images}
}

func (s *Service) Search(ctx context.Context, query string, limit int, sortBy Sort) (Page, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Page{}, errors.New("discovery: query is required")
	}
	if s.metadata == nil || len(s.metadata.Endpoints()) == 0 {
		return Page{}, errors.New("discovery: no stash-box endpoints configured")
	}
	if limit <= 0 {
		limit = 50
	}
	if sortBy == "" {
		sortBy = SortRelevance
	}
	var lastErr error
	for index, box := range s.metadata.Endpoints() {
		client, ok := s.metadata.Get(box.Endpoint)
		if !ok || client == nil {
			continue
		}
		items, err := client.SearchScene(ctx, query)
		if err != nil {
			lastErr = err
			logging.Warnf("discovery: search failed endpoint=%s query=%q: %v", box.Endpoint, query, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		page := Page{Items: make([]Scene, 0, min(limit, len(items))), UsedStashBox: &MatchedSource{Name: box.Name, Endpoint: box.Endpoint}, FallbackCount: index, SearchedQuery: query}
		for _, raw := range items {
			if raw == nil {
				continue
			}
			item := sceneFromStashBox(raw, box)
			if s.images != nil {
				item.ImageURL = s.images(ctx, box.Endpoint, item.ImageURL)
			}
			page.Items = append(page.Items, item)
			if len(page.Items) >= limit {
				break
			}
		}
		sortScenes(page.Items, sortBy)
		return page, nil
	}
	if lastErr != nil {
		return Page{Items: []Scene{}, SearchedQuery: query}, lastErr
	}
	return Page{Items: []Scene{}, SearchedQuery: query}, nil
}
func (s *Service) Queue(ctx context.Context, sceneID, endpoint string) (*taskruntime.Task, error) {
	if s.tasks == nil {
		return nil, errors.New("discovery: task creator is not configured")
	}
	return s.tasks.QueueDiscoveredScene(ctx, sceneID, endpoint)
}

func sceneFromStashBox(scene *stashboxgraphql.SceneFragment, box metadata.StashBoxEndpoint) Scene {
	names := make([]string, 0, len(scene.Performers))
	for _, a := range scene.Performers {
		if a != nil && a.Performer != nil && strings.TrimSpace(a.Performer.Name) != "" {
			names = append(names, strings.TrimSpace(a.Performer.Name))
		}
	}
	studio := ""
	if scene.Studio != nil {
		studio = strings.TrimSpace(scene.Studio.Name)
	}
	return Scene{Key: "stashbox-search:" + endpointKey(box.Endpoint) + ":" + scene.ID, SceneID: scene.ID, StashBoxEndpoint: box.Endpoint, StashBoxName: box.Name, Title: value(scene.Title), DurationSeconds: scene.Duration, Code: value(scene.Code), Date: value(scene.Date), StudioName: studio, ImageURL: sceneImage(scene), URL: sceneURL(scene), PerformerNames: names, DerivedQuery: value(scene.Code)}
}
func value(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
func endpointKey(v string) string {
	return strings.NewReplacer("/", "_", ":", "_").Replace(strings.ToLower(strings.TrimSpace(v)))
}
func sceneImage(s *stashboxgraphql.SceneFragment) string {
	if s == nil || len(s.Images) == 0 || s.Images[0] == nil {
		return ""
	}
	return s.Images[0].URL
}
func sceneURL(s *stashboxgraphql.SceneFragment) string {
	if s == nil {
		return ""
	}
	if len(s.Urls) == 0 || s.Urls[0] == nil {
		return ""
	}
	return s.Urls[0].URL
}
func sortScenes(items []Scene, by Sort) {
	switch by {
	case SortDateDesc:
		sort.SliceStable(items, func(i, j int) bool { return desc(items[i].Date, items[j].Date) })
	case SortDateAsc:
		sort.SliceStable(items, func(i, j int) bool { return desc(items[j].Date, items[i].Date) })
	case SortDurationDesc:
		sort.SliceStable(items, func(i, j int) bool {
			a, b := items[i].DurationSeconds, items[j].DurationSeconds
			return a != nil && (b == nil || *a > *b)
		})
	case SortTitleAsc:
		sort.SliceStable(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
	}
}
func desc(a, b string) bool { return a != "" && (b == "" || a > b) }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
