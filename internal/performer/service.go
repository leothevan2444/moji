package performer

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/taskruntime"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type StashClient interface {
	AllPerformers(context.Context) ([]*stashgraphql.PerformerFragment, error)
	FindPerformerByID(context.Context, string) (*stashgraphql.PerformerFragment, error)
	FindScenes(context.Context, *stashgraphql.SceneFilterType, *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error)
}
type TaskCreator interface {
	QueueDiscoveredScene(context.Context, string, string) (*taskruntime.Task, error)
}
type TaskLister interface {
	ListTasks(context.Context) ([]*taskruntime.Task, error)
}
type StashImageProxy func(context.Context, string) string
type StashBoxImageProxy func(context.Context, string, string) string

type Service struct {
	stash          StashClient
	metadata       *metadata.Service
	taskCreator    TaskCreator
	taskLister     TaskLister
	customFieldKey string
	stashImage     StashImageProxy
	stashBoxImage  StashBoxImageProxy
}

func NewService(stash StashClient, source *metadata.Service, creator TaskCreator, lister TaskLister, stashImage StashImageProxy, stashBoxImage StashBoxImageProxy) (*Service, error) {
	if stash == nil {
		return nil, errors.New("performer: stash client is required")
	}
	if source == nil {
		return nil, errors.New("performer: metadata source is required")
	}
	return &Service{stash: stash, metadata: source, taskCreator: creator, taskLister: lister, customFieldKey: DefaultCustomFieldKey, stashImage: stashImage, stashBoxImage: stashBoxImage}, nil
}

func (s *Service) List(ctx context.Context, search string) ([]Performer, error) {
	raw, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}
	needle := normalize(search)
	out := make([]Performer, 0, len(raw))
	for _, item := range raw {
		performer := performerFromStash(item, s.customFieldKey)
		performer.ImagePath = s.proxyStashImage(ctx, performer.ImagePath)
		if needle == "" || performerMatches(performer, needle) {
			out = append(out, performer)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Subscribed != out[j].Subscribed {
			return out[i].Subscribed
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *Service) proxyStashImage(ctx context.Context, raw string) string {
	if s.stashImage == nil {
		return raw
	}
	return s.stashImage(ctx, raw)
}
func (s *Service) proxyStashBoxImage(ctx context.Context, endpoint, raw string) string {
	if s.stashBoxImage == nil {
		return raw
	}
	return s.stashBoxImage(ctx, endpoint, raw)
}
func performerFromStash(item *stashgraphql.PerformerFragment, key string) Performer {
	if item == nil {
		return Performer{}
	}
	aliases := make([]string, 0, len(item.AliasList))
	for _, alias := range item.AliasList {
		if v := strings.TrimSpace(alias); v != "" {
			aliases = append(aliases, v)
		}
	}
	return Performer{ID: item.ID, Name: item.Name, AliasList: aliases, Favorite: item.Favorite, ImagePath: stringValue(item.ImagePath), SceneCount: item.SceneCount, Subscribed: IsSubscribed(item.CustomFields, key)}
}
func performerMatches(item Performer, needle string) bool {
	if strings.Contains(normalize(item.Name), needle) {
		return true
	}
	for _, alias := range item.AliasList {
		if strings.Contains(normalize(alias), needle) {
			return true
		}
	}
	return false
}
func normalize(v string) string              { return strings.ToLower(strings.TrimSpace(v)) }
func buildReleaseCode(code, _ string) string { return strings.TrimSpace(code) }
func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func endpointKey(v string) string {
	return strings.NewReplacer("/", "_", ":", "_").Replace(strings.ToLower(strings.TrimSpace(v)))
}
