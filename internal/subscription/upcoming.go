package subscription

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/releasedate"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func (s *Service) SetReleaseTimezone(provider func() string) { s.timezone = provider }
func (s *Service) releaseTimezone() string {
	if s.timezone != nil {
		return s.timezone()
	}
	return releasedate.EffectiveTimezone("")
}
func (s *Service) upcomingStore() (UpcomingStore, error) {
	store, ok := s.store.(UpcomingStore)
	if !ok {
		return nil, errors.New("subscription: upcoming storage unavailable")
	}
	return store, nil
}
func (s *Service) captureUpcoming(ctx context.Context, release Release, performerID, name string) (bool, error) {
	s.upcomingMu.Lock()
	defer s.upcomingMu.Unlock()
	store, err := s.upcomingStore()
	if err != nil {
		return false, err
	}
	record, err := store.GetUpcoming(ctx, release.Key)
	if err != nil {
		return false, err
	}
	future := errors.Is(releasedate.Check(release.Date, s.now(), s.releaseTimezone()), releasedate.ErrNotReleased)
	if record == nil && !future {
		return false, nil
	}
	if record != nil && record.State == "HANDED_OFF" {
		return false, nil
	}
	now := s.now().UTC()
	if record == nil {
		record = &UpcomingRelease{FirstSeenAt: now, State: "WAITING", Performers: []ReviewPerformer{}}
	}
	record.OriginalDate = release.Date
	if date, ok := releasedate.Parse(release.Date); ok {
		release.Date = date
	}
	record.Release = release
	record.Endpoint = strings.TrimPrefix(release.Source, "stash-box:")
	record.MetadataCheckedAt = now
	record.LastErrorCode = ""
	if _, ok := releasedate.Parse(release.Date); !ok {
		record.State = "UNCONFIRMED"
	} else {
		record.State = "WAITING"
	}
	found := false
	for i, a := range record.Performers {
		if a.ID == performerID {
			record.Performers[i].Name = name
			found = true
		}
	}
	if !found {
		record.Performers = append(record.Performers, ReviewPerformer{ID: performerID, Name: name})
	}
	record.NextCheckAt = s.nextUpcomingCheck(release.Date, now)
	if !future {
		record.NextCheckAt = now
	}
	if err = store.PutUpcoming(ctx, record); err != nil {
		return false, err
	}
	s.publishEvent(PerformerSubscriptionEventUpdated, performerID, nil)
	return true, nil
}
func (s *Service) nextUpcomingCheck(date string, now time.Time) time.Time {
	next := now.Add(24 * time.Hour)
	if parsed, ok := releasedate.Parse(date); ok {
		midnight := releasedate.Midnight(parsed, s.releaseTimezone())
		if midnight.After(now) && midnight.Before(next) {
			next = midnight
		}
	}
	return next
}
func (s *Service) activeUpcomingPerformers(ctx context.Context) (map[string]*stashgraphql.PerformerFragment, error) {
	actors, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil, err
	}
	active := map[string]*stashgraphql.PerformerFragment{}
	for _, actor := range actors {
		if actor != nil && performer.IsSubscribed(actor.CustomFields, s.customFieldKey) {
			active[actor.ID] = actor
		}
	}
	return active, nil
}
func (s *Service) ListUpcomingReleases(ctx context.Context, q UpcomingQuery) (UpcomingPage, error) {
	store, err := s.upcomingStore()
	if err != nil {
		return UpcomingPage{}, err
	}
	active, err := s.activeUpcomingPerformers(ctx)
	if err != nil {
		return UpcomingPage{}, err
	}
	ids := make([]string, 0, len(active))
	for id := range active {
		ids = append(ids, id)
	}
	if q.Days > 0 {
		q.Before = s.now().In(releasedate.Location(s.releaseTimezone())).AddDate(0, 0, q.Days).Format("2006-01-02")
	}
	page, err := store.QueryUpcoming(ctx, q, ids)
	// Display current names once per performer, without merging release identities.
	names := map[string]ReviewPerformer{}
	for _, a := range page.Performers {
		if actor := active[a.ID]; actor != nil {
			a.Name = actor.Name
			names[a.ID] = a
		}
	}
	page.Performers = nil
	for _, a := range names {
		page.Performers = append(page.Performers, a)
	}
	sort.Slice(page.Performers, func(i, j int) bool { return page.Performers[i].Name < page.Performers[j].Name })
	for i := range page.Items {
		for j, a := range page.Items[i].Performers {
			if actor := active[a.ID]; actor != nil {
				page.Items[i].Performers[j].Name = actor.Name
			}
		}
	}
	page.SubscribedCount = len(ids)
	page.Timezone = s.releaseTimezone()
	page.Today = s.now().In(releasedate.Location(page.Timezone)).Format("2006-01-02")
	return page, err
}
func (s *Service) RefreshUpcomingRelease(ctx context.Context, key string) (*UpcomingRelease, error) {
	s.upcomingMu.Lock()
	defer s.upcomingMu.Unlock()
	active, err := s.activeUpcomingPerformers(ctx)
	if err != nil {
		return nil, err
	}
	return s.refreshUpcomingLocked(ctx, key, active)
}

// ProcessDueUpcoming scans durable records independently of bounded actor catalog pages.
func (s *Service) ProcessDueUpcoming(ctx context.Context) error {
	s.upcomingMu.Lock()
	defer s.upcomingMu.Unlock()
	store, err := s.upcomingStore()
	if err != nil {
		return err
	}
	due, err := store.DueUpcoming(ctx, s.now().UTC(), 32)
	if err != nil || len(due) == 0 {
		return err
	}
	active, err := s.activeUpcomingPerformers(ctx)
	if err != nil {
		return err
	}
	for _, record := range due {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if _, err = s.refreshUpcomingLocked(ctx, record.Release.Key, active); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) refreshUpcomingLocked(ctx context.Context, key string, active map[string]*stashgraphql.PerformerFragment) (*UpcomingRelease, error) {
	store, err := s.upcomingStore()
	if err != nil {
		return nil, err
	}
	record, err := store.GetUpcoming(ctx, key)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("subscription: upcoming release not found")
	}
	if record.State == "HANDED_OFF" {
		return record, nil
	}
	now := s.now().UTC()
	defer func() {
		for _, a := range record.Performers {
			s.publishEvent(PerformerSubscriptionEventUpdated, a.ID, nil)
		}
	}()
	record.NextCheckAt = now.Add(time.Hour)
	saveFailure := func(code string) (*UpcomingRelease, error) {
		record.LastErrorCode = code
		record.State = "UNCONFIRMED"
		if err := store.PutUpcoming(ctx, record); err != nil {
			return nil, err
		}
		return record, nil
	}
	client, ok := s.metadata.Get(record.Endpoint)
	if !ok {
		return saveFailure("SOURCE_UNAVAILABLE")
	}
	hasActive := false
	for _, a := range record.Performers {
		if active[a.ID] != nil {
			hasActive = true
		}
	}
	if !hasActive {
		record.NextCheckAt = now.Add(24 * time.Hour)
		return record, store.PutUpcoming(ctx, record)
	}
	scene, err := client.FindSceneByID(ctx, record.Release.SceneID)
	if err != nil {
		return saveFailure("METADATA_UNAVAILABLE")
	}
	if scene == nil {
		return saveFailure("SCENE_NOT_FOUND")
	}
	var evaluation ReleaseEvaluation
	matched := false
	for _, a := range record.Performers {
		actor := active[a.ID]
		if actor == nil {
			continue
		}
		// Resolve against the recorded source even after source priorities change.
		sourceActor := *actor
		sourceActor.StashIds = nil
		for _, id := range actor.StashIds {
			if id != nil && metadata.NormalizeEndpoint(id.Endpoint) == metadata.NormalizeEndpoint(record.Endpoint) {
				sourceActor.StashIds = append(sourceActor.StashIds, id)
			}
		}
		target, err := s.metadata.ResolvePerformerWithPolicy(ctx, &sourceActor, metadata.RequireFresh)
		if err != nil {
			return saveFailure("PERFORMER_UNAVAILABLE")
		}
		if target == nil || metadata.NormalizeEndpoint(target.Endpoint) != metadata.NormalizeEndpoint(record.Endpoint) {
			continue
		}
		if result, ok := evaluateReleasePolicy(s.currentReleasePolicy(), now, target.Performer, scene); ok {
			evaluation = result
			matched = true
			break
		}
	}
	if !matched {
		return saveFailure("PERFORMER_MISMATCH")
	}
	record.Release.Code = stringValue(scene.Code)
	record.Release.Title = stringValue(scene.Title)
	record.OriginalDate = stringValue(scene.Date)
	record.Release.Date = record.OriginalDate
	if date, ok := releasedate.Parse(record.OriginalDate); ok {
		record.Release.Date = date
	}
	record.Release.Classification = evaluation.Classification
	record.Release.PerformerCount = evaluation.PerformerCount
	record.Release.PerformerNames = evaluation.PerformerNames
	record.Release.Decision = evaluation.Decision
	record.Release.DecisionReason = evaluation.DecisionReason
	if len(scene.Images) > 0 && scene.Images[0] != nil {
		record.Release.ImageURL = s.proxyImage(ctx, imagecache.SourceStashBox, record.Endpoint, scene.Images[0].URL, s.metadata.APIKey(record.Endpoint))
	}
	if len(scene.Urls) > 0 && scene.Urls[0] != nil {
		record.Release.URL = scene.Urls[0].URL
	}
	record.MetadataCheckedAt = now
	record.LastErrorCode = ""
	gate := releasedate.Check(record.Release.Date, now, s.releaseTimezone())
	if errors.Is(gate, releasedate.ErrNotReleased) {
		record.State = "WAITING"
		record.NextCheckAt = s.nextUpcomingCheck(record.Release.Date, now)
		return record, store.PutUpcoming(ctx, record)
	}
	if errors.Is(gate, releasedate.ErrUnknown) {
		record.Release.Decision = ReleaseDecisionQueued
		record.Release.DecisionReason = "release_date_unknown_review"
	}
	current, err := s.activeUpcomingPerformers(ctx)
	if err != nil {
		return saveFailure("SUBSCRIPTION_UNAVAILABLE")
	}
	stillActive := false
	for _, actor := range record.Performers {
		if current[actor.ID] != nil {
			stillActive = true
		}
	}
	if !stillActive {
		record.NextCheckAt = now.Add(24 * time.Hour)
		return record, store.PutUpcoming(ctx, record)
	}
	record.State = "HANDOFF_PENDING"
	if err = store.PutUpcoming(ctx, record); err != nil {
		return nil, err
	}
	// Persist the handoff before invoking taskflow; retries reconcile an already-created task.
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	reviews, err := s.reviewRecordsLocked(ctx)
	if err != nil {
		return nil, err
	}
	review, exists := reviews[key]
	if !exists {
		r := record.Release
		review = ReleaseReview{RecordedRelease: RecordedRelease{Key: r.Key, Source: r.Source, Title: r.Title, Code: r.Code, Date: r.Date, URL: r.URL, SeenAt: record.FirstSeenAt, PerformerCount: r.PerformerCount, PerformerNames: r.PerformerNames, Classification: r.Classification, Decision: r.Decision, DecisionReason: r.DecisionReason}, SceneID: r.SceneID, ImageURL: r.ImageURL, Performers: record.Performers}
		review.Status = reviewStatus(review.RecordedRelease, false)
		inLibrary, err := s.stashSceneExistsForRelease(ctx, r)
		if err != nil {
			return saveFailure("LIBRARY_UNAVAILABLE")
		}
		if inLibrary {
			review.Status = ReviewInLibrary
		}
		if err = store.PutUpcomingReview(ctx, record, review); err != nil {
			return nil, err
		}
	}
	if exists {
		review.Date = record.Release.Date
		review.Title = record.Release.Title
		review.ImageURL = record.Release.ImageURL
		if review.Status == ReviewFailed && review.ReviewedAt == nil {
			review.Decision = record.Release.Decision
			review.DecisionReason = record.Release.DecisionReason
			review.Status = reviewStatus(review.RecordedRelease, false)
		}
	}
	// Recheck after waiting on review reconciliation and library lookups. A
	// cancellation during those reads must not admit a new automatic task.
	current, err = s.activeUpcomingPerformers(ctx)
	if err != nil {
		return saveFailure("SUBSCRIPTION_UNAVAILABLE")
	}
	stillActive = false
	for _, actor := range record.Performers {
		if current[actor.ID] != nil {
			stillActive = true
		}
	}
	if !stillActive {
		record.State = "WAITING"
		record.NextCheckAt = now.Add(24 * time.Hour)
		return record, store.PutUpcoming(ctx, record)
	}
	// A saved rejection, review requirement, policy skip or linked task remains authoritative.
	if review.Decision == ReleaseDecisionDownloaded && review.Status == ReviewFailed {
		if err = s.downloadReviewLocked(ctx, &review); err != nil {
			review.Error = err.Error()
			if saveErr := store.PutUpcomingReview(ctx, record, review); saveErr != nil {
				return nil, saveErr
			}
			return saveFailure("TASK_CREATION_FAILED")
		}
		if err = store.PutUpcomingReview(ctx, record, review); err != nil {
			return nil, err
		}
	}
	record.State = "HANDED_OFF"
	record.LastErrorCode = ""
	return record, store.PutUpcomingReview(ctx, record, review)
}

// UpcomingReleaseStates projects persisted lifecycle state without fetching metadata
// or enumerating all subscriptions. Callers provide their bounded visible scene keys.
func (s *Service) UpcomingReleaseStates(ctx context.Context, keys []string) (map[string]string, error) {
	store, err := s.upcomingStore()
	if err != nil {
		return nil, err
	}
	return store.UpcomingReleaseStates(ctx, keys)
}
