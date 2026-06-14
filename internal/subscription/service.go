package subscription

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/logging"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
	stashboxgraphql "github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

type StashClient interface {
	AllPerformers(ctx context.Context) ([]*stashgraphql.PerformerFragment, error)
	FindPerformerByID(ctx context.Context, id string) (*stashgraphql.PerformerFragment, error)
	UpdatePerformerCustomFields(ctx context.Context, id string, partial map[string]any, remove []string) (*stashgraphql.PerformerFragment, error)
}

type StashboxClient interface {
	FindPerformerByID(ctx context.Context, id string) (*stashboxgraphql.PerformerFragment, error)
	SearchPerformer(ctx context.Context, term string) ([]*stashboxgraphql.PerformerFragment, error)
	QueryScenes(ctx context.Context, input stashboxgraphql.SceneQueryInput) ([]*stashboxgraphql.SceneFragment, error)
}

type Downloader interface {
	DownloadMediaContext(ctx context.Context, req downloader.DownloadRequest) (*downloader.Task, error)
}

type Service struct {
	stash          StashClient
	stashbox       StashboxClient
	downloader     Downloader
	store          Store
	customFieldKey string
	now            func() time.Time
}

func NewService(stash StashClient, stashbox StashboxClient, downloader Downloader, store Store) (*Service, error) {
	if stash == nil {
		return nil, errors.New("subscription: stash client is required")
	}
	if store == nil {
		store = NewMemoryStore()
	}

	return &Service{
		stash:          stash,
		stashbox:       stashbox,
		downloader:     downloader,
		store:          store,
		customFieldKey: DefaultCustomFieldKey,
		now:            time.Now,
	}, nil
}

func (s *Service) ListStashPerformers(ctx context.Context, search string) ([]Performer, error) {
	performers, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}

	needle := normalize(search)
	out := make([]Performer, 0, len(performers))
	for _, performer := range performers {
		item := performerFromStash(performer, s.customFieldKey)
		if needle != "" && !performerMatches(item, needle) {
			continue
		}
		out = append(out, item)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Subscribed != out[j].Subscribed {
			return out[i].Subscribed
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *Service) ListSubscribedPerformers(ctx context.Context) ([]SubscribedPerformer, error) {
	performers, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]SubscribedPerformer, 0)
	for _, performer := range performers {
		item := performerFromStash(performer, s.customFieldKey)
		if !item.Subscribed {
			continue
		}
		state, err := s.store.Get(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, buildSubscribedPerformer(item, state))
	}

	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		if left.LastCheckedAt == nil || right.LastCheckedAt == nil {
			return left.LastCheckedAt != nil
		}
		return left.LastCheckedAt.After(*right.LastCheckedAt)
	})
	return out, nil
}

func (s *Service) SubscribePerformer(ctx context.Context, performerID string) (SubscribedPerformer, error) {
	performer, err := s.stash.UpdatePerformerCustomFields(ctx, performerID, map[string]any{s.customFieldKey: true}, nil)
	if err != nil {
		logging.Errorf("subscription: subscribe performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}

	item := performerFromStash(performer, s.customFieldKey)
	state, err := s.store.Get(ctx, item.ID)
	if err != nil {
		logging.Errorf("subscription: load state for performer %s after subscribe failed: %v", item.ID, err)
		return SubscribedPerformer{}, err
	}
	logging.Infof("subscription: subscribed performer %s (%s)", item.ID, item.Name)

	return buildSubscribedPerformer(item, state), nil
}

func (s *Service) UnsubscribePerformer(ctx context.Context, performerID string) error {
	if _, err := s.stash.UpdatePerformerCustomFields(ctx, performerID, nil, []string{s.customFieldKey}); err != nil {
		logging.Errorf("subscription: unsubscribe performer %s failed: %v", performerID, err)
		return err
	}
	if err := s.store.Delete(ctx, performerID); err != nil {
		logging.Errorf("subscription: delete state for performer %s failed: %v", performerID, err)
		return err
	}
	logging.Infof("subscription: unsubscribed performer %s", performerID)
	return nil
}

func (s *Service) RefreshSubscribedPerformer(ctx context.Context, performerID string) (SubscribedPerformer, error) {
	logging.Infof("subscription: refresh started for performer %s", performerID)
	performer, err := s.stash.FindPerformerByID(ctx, performerID)
	if err != nil {
		logging.Errorf("subscription: load performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	if performer == nil {
		return SubscribedPerformer{}, fmt.Errorf("subscription: performer %q not found", performerID)
	}

	item := performerFromStash(performer, s.customFieldKey)
	if !item.Subscribed {
		return SubscribedPerformer{}, fmt.Errorf("subscription: performer %q is not subscribed", performerID)
	}

	state, err := s.store.Get(ctx, performerID)
	if err != nil {
		logging.Errorf("subscription: load state for performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	if state == nil {
		state = &PerformerState{PerformerID: performerID}
	}

	now := s.now().UTC()
	state.LastCheckedAt = &now
	state.LastError = ""

	releases, err := s.fetchReleases(ctx, performer)
	if err != nil {
		state.LastError = err.Error()
		if putErr := s.store.Put(ctx, state); putErr != nil {
			logging.Errorf("subscription: persist error state for performer %s failed: %v", performerID, putErr)
			return SubscribedPerformer{}, putErr
		}
		logging.Errorf("subscription: refresh failed for performer %s (%s): %v", performerID, performer.Name, err)
		return buildSubscribedPerformer(item, state), err
	}
	logging.Infof("subscription: refresh fetched %d releases for performer %s (%s)", len(releases), performerID, performer.Name)

	processed := make(map[string]RecordedRelease, len(state.ProcessedReleases))
	for _, release := range state.ProcessedReleases {
		processed[release.Key] = release
	}

	pending := make([]RecordedRelease, 0)
	for _, release := range releases {
		if _, exists := processed[release.Key]; exists {
			continue
		}
		record := RecordedRelease{
			Key:    release.Key,
			Source: release.Source,
			Title:  release.Title,
			Code:   release.Code,
			Date:   release.Date,
			URL:    release.URL,
			Query:  release.Query,
			SeenAt: now,
		}
		pending = append(pending, record)
	}
	if len(pending) > 0 {
		logging.Infof("subscription: detected %d new releases for performer %s (%s)", len(pending), performerID, performer.Name)
	}

	if s.downloader != nil {
		for i := range pending {
			task, err := s.downloader.DownloadMediaContext(ctx, downloader.DownloadRequest{Query: pending[i].Query})
			if err != nil {
				state.LastError = err.Error()
				logging.Errorf("subscription: auto-download failed for performer %s release %q: %v", performerID, pending[i].Query, err)
				continue
			}
			if task != nil {
				pending[i].TaskID = task.ID
				logging.Infof("subscription: auto-download created task %s for performer %s release %q", task.ID, performerID, pending[i].Query)
			}
			state.ProcessedReleases = append([]RecordedRelease{pending[i]}, state.ProcessedReleases...)
		}
		state.PendingReleases = nil
	} else {
		state.PendingReleases = pending
		if len(pending) > 0 {
			logging.Infof("subscription: queued %d pending releases for performer %s (%s)", len(pending), performerID, performer.Name)
		}
	}

	state.ProcessedReleases = trimRecordedReleases(state.ProcessedReleases, 25)
	state.PendingReleases = trimRecordedReleases(state.PendingReleases, 25)
	if err := s.store.Put(ctx, state); err != nil {
		logging.Errorf("subscription: persist state for performer %s failed: %v", performerID, err)
		return SubscribedPerformer{}, err
	}
	logging.Infof(
		"subscription: refresh completed for performer %s (%s), processed=%d pending=%d",
		performerID,
		performer.Name,
		len(state.ProcessedReleases),
		len(state.PendingReleases),
	)

	return buildSubscribedPerformer(item, state), nil
}

func (s *Service) RefreshAll(ctx context.Context) ([]SubscribedPerformer, error) {
	items, err := s.ListSubscribedPerformers(ctx)
	if err != nil {
		logging.Errorf("subscription: list subscribed performers failed: %v", err)
		return nil, err
	}
	logging.Infof("subscription: refresh all started for %d performers", len(items))

	out := make([]SubscribedPerformer, 0, len(items))
	for _, item := range items {
		refreshed, refreshErr := s.RefreshSubscribedPerformer(ctx, item.Performer.ID)
		if refreshErr != nil {
			out = append(out, refreshed)
			continue
		}
		out = append(out, refreshed)
	}
	logging.Infof("subscription: refresh all completed for %d performers", len(out))
	return out, nil
}

func (s *Service) fetchReleases(ctx context.Context, performer *stashgraphql.PerformerFragment) ([]Release, error) {
	if s.stashbox == nil {
		return nil, errors.New("subscription: javstash client is not configured")
	}

	target, err := s.resolveStashboxPerformer(ctx, performer)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("subscription: no javstash performer match for %q", performer.Name)
	}

	scenes, err := s.stashbox.QueryScenes(ctx, stashboxgraphql.SceneQueryInput{
		Performers: &stashboxgraphql.MultiIDCriterionInput{Value: []string{target.ID}},
		Page:       1,
		PerPage:    12,
		Direction:  stashboxgraphql.SortDirectionEnumDesc,
		Sort:       stashboxgraphql.SceneSortEnumDate,
	})
	if err != nil {
		return nil, err
	}

	releases := make([]Release, 0, len(scenes))
	for _, scene := range scenes {
		if scene == nil {
			continue
		}
		release := Release{
			Key:    "javstash:" + scene.ID,
			Source: "javstash",
			Title:  stringValue(scene.Title),
			Code:   stringValue(scene.Code),
			Date:   stringValue(scene.Date),
			Query:  buildReleaseQuery(stringValue(scene.Code), stringValue(scene.Title)),
		}
		if len(scene.Urls) > 0 && scene.Urls[0] != nil {
			release.URL = scene.Urls[0].URL
		}
		releases = append(releases, release)
	}

	return releases, nil
}

func (s *Service) resolveStashboxPerformer(ctx context.Context, performer *stashgraphql.PerformerFragment) (*stashboxgraphql.PerformerFragment, error) {
	for _, stashID := range performer.StashIds {
		if stashID == nil {
			continue
		}
		if strings.Contains(strings.ToLower(stashID.Endpoint), "javstash.org") && strings.TrimSpace(stashID.StashID) != "" {
			return s.stashbox.FindPerformerByID(ctx, stashID.StashID)
		}
	}

	candidates, err := s.stashbox.SearchPerformer(ctx, performer.Name)
	if err != nil {
		return nil, err
	}

	targetNames := make([]string, 0, 1+len(performer.AliasList))
	targetNames = append(targetNames, performer.Name)
	targetNames = append(targetNames, performer.AliasList...)
	normalizedTargets := make(map[string]struct{}, len(targetNames))
	for _, name := range targetNames {
		if normalized := normalize(name); normalized != "" {
			normalizedTargets[normalized] = struct{}{}
		}
	}

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if _, ok := normalizedTargets[normalize(candidate.Name)]; ok {
			return candidate, nil
		}
		for _, alias := range candidate.Aliases {
			if _, ok := normalizedTargets[normalize(alias)]; ok {
				return candidate, nil
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0], nil
	}
	return nil, nil
}

func performerFromStash(performer *stashgraphql.PerformerFragment, customFieldKey string) Performer {
	if performer == nil {
		return Performer{}
	}

	return Performer{
		ID:         performer.ID,
		Name:       performer.Name,
		AliasList:  append([]string(nil), performer.AliasList...),
		Favorite:   performer.Favorite,
		ImagePath:  derefString(performer.ImagePath),
		SceneCount: performer.SceneCount,
		Subscribed: customFieldTruthy(performer.CustomFields, customFieldKey),
	}
}

func buildSubscribedPerformer(performer Performer, state *PerformerState) SubscribedPerformer {
	item := SubscribedPerformer{
		Performer: performer,
	}
	if state == nil {
		return item
	}

	item.LastCheckedAt = state.LastCheckedAt
	item.LastError = state.LastError
	item.PendingReleaseCount = len(state.PendingReleases)
	item.ProcessedReleaseCount = len(state.ProcessedReleases)
	item.RecentReleases = append([]RecordedRelease(nil), state.PendingReleases...)
	item.RecentReleases = append(item.RecentReleases, state.ProcessedReleases...)
	item.RecentReleases = trimRecordedReleases(item.RecentReleases, 10)
	return item
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func performerMatches(performer Performer, needle string) bool {
	if strings.Contains(normalize(performer.Name), needle) {
		return true
	}
	for _, alias := range performer.AliasList {
		if strings.Contains(normalize(alias), needle) {
			return true
		}
	}
	return false
}

func customFieldTruthy(fields map[string]any, key string) bool {
	value, ok := fields[key]
	if !ok {
		return false
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		}
	case float64:
		return typed != 0
	}

	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildReleaseQuery(code, title string) string {
	code = strings.TrimSpace(code)
	if code != "" {
		return code
	}
	return strings.TrimSpace(title)
}

func trimRecordedReleases(items []RecordedRelease, limit int) []RecordedRelease {
	if len(items) <= limit {
		return items
	}
	return append([]RecordedRelease(nil), items[:limit]...)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
