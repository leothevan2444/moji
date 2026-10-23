package subscription

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type ReviewTaskLister interface {
	ListTasks(context.Context) ([]*taskruntime.Task, error)
}

func reviewCode(code string) string {
	if value := mediafiles.ExtractCode(strings.TrimSpace(code)); value != "" {
		return value
	}
	return strings.ToUpper(strings.TrimSpace(code))
}

func reviewStatus(record RecordedRelease, processed bool) ReviewStatus {
	if record.TaskID != "" || processed {
		return ReviewDownloaded
	}
	switch record.Decision {
	case ReleaseDecisionBlocked:
		return ReviewSkipped
	case ReleaseDecisionDownloaded:
		return ReviewFailed
	default:
		return ReviewPending
	}
}

func addReviewPerformer(record *ReleaseReview, id, name string) bool {
	for i, actor := range record.Performers {
		if actor.ID == id {
			if name != "" && name != actor.Name {
				record.Performers[i].Name = name
				return true
			}
			return false
		}
	}
	record.Performers = append(record.Performers, ReviewPerformer{ID: id, Name: name})
	return true
}

// Called under reviewMu. Import only records that still exist; never reconstruct
// truncated history or create tasks as a side effect of migration.
func (s *Service) reviewRecordsLocked(ctx context.Context) (map[string]ReleaseReview, error) {
	rows, err := s.reviews.ListReleaseReviews(ctx)
	if err != nil {
		return nil, err
	}
	records := make(map[string]ReleaseReview, len(rows))
	persisted := make(map[string]bool, len(rows))
	for _, record := range rows {
		records[record.Key] = record
		persisted[record.Key] = true
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	changed := make(map[string]ReleaseReview)
	for _, state := range states {
		if state == nil {
			continue
		}
		for _, group := range []struct {
			records   []RecordedRelease
			processed bool
		}{{state.ProcessedReleases, true}, {state.PendingReleases, false}} {
			for _, old := range group.records {
				record, exists := records[old.Key]
				promote := exists && !persisted[old.Key] && record.TaskID == "" && (old.TaskID != "" || (group.processed && record.Status != ReviewDownloaded))
				if promote {
					record.RecordedRelease = old
					record.Status = reviewStatus(old, group.processed)
					record.Error = ""
				}
				if !exists {
					record = ReleaseReview{RecordedRelease: old, Status: reviewStatus(old, group.processed)}
					if record.Status == ReviewFailed {
						record.Error = state.LastError
					}
				}
				if addReviewPerformer(&record, state.PerformerID, "") || !exists || promote {
					records[old.Key] = record
					changed[old.Key] = record
				}
			}
		}
	}
	updates := make([]ReleaseReview, 0, len(changed))
	for _, record := range changed {
		updates = append(updates, record)
	}
	if err := s.reviews.PutReleaseReviews(ctx, updates); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Service) reviewTaskMap(ctx context.Context) (map[string]*taskruntime.Task, error) {
	out := make(map[string]*taskruntime.Task)
	if s.reviewTasks == nil {
		return out, nil
	}
	tasks, err := s.reviewTasks.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if task != nil && task.ID != "" && task.Code != "" {
			out[reviewCode(task.Code)] = task
		}
	}
	return out, nil
}

func (s *Service) reconcileReviewTasksLocked(ctx context.Context, records map[string]ReleaseReview) error {
	tasks, err := s.reviewTaskMap(ctx)
	if err != nil {
		return err
	}
	changed := make([]ReleaseReview, 0)
	for key, record := range records {
		if task := tasks[reviewCode(record.Code)]; task != nil && (record.TaskID != task.ID || record.Status != ReviewDownloaded) {
			record.TaskID, record.Status, record.Error = task.ID, ReviewDownloaded, ""
			record.Decision = ReleaseDecisionDownloaded
			records[key] = record
			changed = append(changed, record)
		}
	}
	return s.reviews.PutReleaseReviews(ctx, changed)
}

// SynchronizeReleaseReviewTasks also covers downloads initiated from other pages.
func (s *Service) SynchronizeReleaseReviewTasks(ctx context.Context) error {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	records, err := s.reviewRecordsLocked(ctx)
	if err != nil {
		return err
	}
	if err := s.reconcileReviewTasksLocked(ctx, records); err != nil {
		return err
	}
	for _, state := range s.reviewPerformerSnapshots(ctx, records) {
		copy := state
		s.publishEvent(PerformerSubscriptionEventUpdated, state.Performer.ID, &copy)
	}
	return nil
}

func (s *Service) ListReleaseReviews(ctx context.Context, query ReleaseReviewQuery) (ReleaseReviewPage, error) {
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	records, err := s.reviewRecordsLocked(ctx)
	if err != nil {
		return ReleaseReviewPage{}, err
	}
	if err := s.reconcileReviewTasksLocked(ctx, records); err != nil {
		return ReleaseReviewPage{}, err
	}
	actors, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return ReleaseReviewPage{}, err
	}
	names := make(map[string]string)
	for _, actor := range actors {
		if actor != nil {
			names[actor.ID] = actor.Name
		}
	}
	page := ReleaseReviewPage{Page: query.Page, PageSize: query.PageSize, Items: []ReleaseReview{}, Performers: []ReviewPerformer{}}
	if page.Page < 1 {
		page.Page = 1
	}
	if page.PageSize < 1 {
		page.PageSize = 24
	}
	if page.PageSize > 100 {
		page.PageSize = 100
	}
	seenActors := make(map[string]bool)
	actorCounts := make(map[string]int)
	items := make([]ReleaseReview, 0)
	for _, record := range records {
		matchesActor := query.PerformerID == ""
		for i, actor := range record.Performers {
			if record.Status == ReviewPending {
				actorCounts[actor.ID]++
			}
			if name := names[actor.ID]; name != "" {
				actor.Name = name
			}
			if actor.Name == "" {
				actor.Name = actor.ID
			}
			record.Performers[i] = actor
			if !seenActors[actor.ID] {
				page.Performers = append(page.Performers, actor)
				seenActors[actor.ID] = true
			}
			if actor.ID == query.PerformerID {
				matchesActor = true
			}
		}
		if !matchesActor {
			continue
		}
		if record.Status == ReviewPending {
			page.PendingCount++
		}
		if record.Status == ReviewFailed {
			page.FailedCount++
		}
		if query.Status != "ALL" && record.Status != query.Status && !(query.Status == "" && record.Status == ReviewPending) {
			continue
		}
		if query.VR != nil && strings.Contains(strings.ToUpper(record.Code), "VR") != *query.VR {
			continue
		}
		if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" && !strings.Contains(strings.ToLower(record.Code+" "+record.Title+" "+strings.Join(record.PerformerNames, " ")), search) {
			continue
		}
		items = append(items, record)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SeenAt.Equal(items[j].SeenAt) {
			return items[i].Key < items[j].Key
		}
		return items[i].SeenAt.After(items[j].SeenAt)
	})
	sort.Slice(page.Performers, func(i, j int) bool { return page.Performers[i].Name < page.Performers[j].Name })
	for i := range page.Performers {
		page.Performers[i].PendingCount = actorCounts[page.Performers[i].ID]
	}
	page.TotalCount = len(items)
	page.TotalPages = (len(items) + page.PageSize - 1) / page.PageSize
	if page.TotalPages > 0 && page.Page > page.TotalPages {
		page.Page = page.TotalPages
	}
	start := min((page.Page-1)*page.PageSize, len(items))
	end := min(start+page.PageSize, len(items))
	page.Items = items[start:end]
	return page, nil
}

func (s *Service) downloadReviewLocked(ctx context.Context, record *ReleaseReview) error {
	if strings.TrimSpace(record.Code) == "" {
		return errors.New("subscription: release code is missing")
	}
	tasks, err := s.reviewTaskMap(ctx)
	if err != nil {
		return err
	}
	if task := tasks[reviewCode(record.Code)]; task != nil {
		record.TaskID = task.ID
		record.Status = ReviewDownloaded
		record.Error = ""
		return nil
	}
	sceneID := record.SceneID
	if sceneID == "" {
		prefix := "stashbox:" + endpointKey(strings.TrimPrefix(record.Source, "stash-box:")) + ":"
		sceneID = strings.TrimPrefix(record.Key, prefix)
		if sceneID == record.Key {
			sceneID = ""
		}
	}
	inLibrary, err := s.stashSceneExistsForRelease(ctx, Release{Key: record.Key, Source: record.Source, SceneID: sceneID})
	if err != nil {
		return err
	}
	if inLibrary {
		record.Status = ReviewInLibrary
		record.Error = ""
		return nil
	}
	if s.taskCreator == nil {
		return errors.New("subscription: task creation is not configured")
	}
	task, err := s.taskCreator.QueueSubscriptionRelease(ctx, record.Code, record.Title)
	if task != nil {
		record.TaskID = task.ID
		record.Status = ReviewDownloaded
		record.Decision = ReleaseDecisionDownloaded
		record.Error = ""
		return nil
	}
	if errors.Is(err, taskruntime.ErrDuplicateLibraryCode) {
		record.Status = ReviewInLibrary
		record.Error = ""
		return nil
	}
	if errors.Is(err, taskruntime.ErrDuplicateCodeTask) {
		tasks, listErr := s.reviewTaskMap(ctx)
		if listErr != nil {
			return listErr
		}
		if existing := tasks[reviewCode(record.Code)]; existing != nil {
			record.TaskID = existing.ID
			record.Status = ReviewDownloaded
			record.Error = ""
			return nil
		}
	}
	if err == nil {
		err = errors.New("subscription: task creation returned no task")
	}
	return err
}

func (s *Service) ProcessReleaseReviews(ctx context.Context, keys []string, action ReviewAction) (ReleaseReviewPayload, error) {
	if len(keys) == 0 || len(keys) > 100 {
		return ReleaseReviewPayload{}, errors.New("subscription: select between 1 and 100 releases")
	}
	if action != ReviewDownload && action != ReviewReject && action != ReviewReconsider {
		return ReleaseReviewPayload{}, errors.New("subscription: unknown review action")
	}
	s.reviewMu.Lock()
	defer s.reviewMu.Unlock()
	records, err := s.reviewRecordsLocked(ctx)
	if err != nil {
		return ReleaseReviewPayload{}, err
	}
	if err := s.reconcileReviewTasksLocked(ctx, records); err != nil {
		return ReleaseReviewPayload{}, err
	}
	payload := ReleaseReviewPayload{Results: []ReleaseReviewResult{}}
	seen := make(map[string]bool)
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		result := ReleaseReviewResult{Key: key, Outcome: "SKIPPED", ReasonCode: "ALREADY_HANDLED"}
		record, exists := records[key]
		if !exists {
			result.Outcome = "FAILED"
			result.ReasonCode = "NOT_FOUND"
		} else if ctx.Err() != nil {
			result.Outcome = "FAILED"
			result.ReasonCode = "CANCELLED"
		} else {
			allowed := record.Status == ReviewPending || record.Status == ReviewFailed
			if action == ReviewReconsider {
				allowed = record.Status == ReviewRejected || record.Status == ReviewSkipped
			}
			if allowed {
				now := s.now().UTC()
				record.ReviewedAt = &now
				record.Error = ""
				switch action {
				case ReviewReconsider:
					record.Status = ReviewPending
					record.Decision = ReleaseDecisionQueued
					record.DecisionReason = "user_reconsidered"
				case ReviewReject:
					record.Status = ReviewRejected
					record.DecisionReason = "user_rejected"
				case ReviewDownload:
					if err := s.downloadReviewLocked(ctx, &record); err != nil {
						record.Status = ReviewFailed
						record.Error = err.Error()
					}
				}
				result.Outcome = "SUCCEEDED"
				result.ReasonCode = "UPDATED"
				switch record.Status {
				case ReviewDownloaded:
					result.ReasonCode = "DOWNLOAD_READY"
				case ReviewInLibrary:
					result.ReasonCode = "ALREADY_IN_LIBRARY"
				case ReviewRejected:
					result.ReasonCode = "REJECTED"
				case ReviewPending:
					result.ReasonCode = "RECONSIDERED"
				}
				if record.Status == ReviewFailed {
					result.Outcome = "FAILED"
					result.ReasonCode = "TASK_CREATION_FAILED"
				}
				if err := s.reviews.PutReleaseReviews(ctx, []ReleaseReview{record}); err != nil {
					result.Outcome = "FAILED"
					result.ReasonCode = "SAVE_FAILED"
					record.Error = err.Error()
				} else {
					records[key] = record
				}
			}
			result.Release = &record
		}
		switch result.Outcome {
		case "SUCCEEDED":
			payload.SucceededCount++
		case "SKIPPED":
			payload.SkippedCount++
		default:
			payload.FailedCount++
		}
		payload.Results = append(payload.Results, result)
	}
	// Publish after decisions are durable. Actor pages re-read their exact counts.
	for _, state := range s.reviewPerformerSnapshots(ctx, records) {
		copy := state
		s.publishEvent(PerformerSubscriptionEventUpdated, state.Performer.ID, &copy)
	}
	return payload, nil
}

func applyReviewSummary(item *SubscribedPerformer, records map[string]ReleaseReview) {
	item.ReviewReleaseCount = 0
	item.PendingReleaseCount = 0
	item.ProcessedReleaseCount = 0
	recent := make([]RecordedRelease, 0)
	for _, record := range records {
		for _, actor := range record.Performers {
			if actor.ID == item.Performer.ID {
				if record.Status == ReviewPending {
					item.ReviewReleaseCount++
				}
				if record.Status == ReviewDownloaded || record.Status == ReviewInLibrary {
					item.ProcessedReleaseCount++
				} else {
					item.PendingReleaseCount++
				}
				recent = append(recent, record.RecordedRelease)
				break
			}
		}
	}
	sort.Slice(recent, func(i, j int) bool {
		if recent[i].SeenAt.Equal(recent[j].SeenAt) {
			return recent[i].Key < recent[j].Key
		}
		return recent[i].SeenAt.After(recent[j].SeenAt)
	})
	item.RecentReleases = trimRecordedReleases(recent, 10)
}

func (s *Service) applyStoredReviewSummary(ctx context.Context, item *SubscribedPerformer) {
	rows, err := s.reviews.ListReleaseReviews(ctx)
	if err != nil {
		return
	}
	records := make(map[string]ReleaseReview, len(rows))
	for _, row := range rows {
		records[row.Key] = row
	}
	if len(records) > 0 {
		applyReviewSummary(item, records)
	}
}

func (s *Service) reviewPerformerSnapshots(ctx context.Context, records map[string]ReleaseReview) []SubscribedPerformer {
	actors, err := s.stash.AllPerformers(ctx)
	if err != nil {
		return nil
	}
	out := make([]SubscribedPerformer, 0)
	for _, actor := range actors {
		item := performerFromStash(actor, s.customFieldKey)
		item.ImagePath = s.proxyStashImage(ctx, item.ImagePath)
		if !item.Subscribed {
			continue
		}
		state, err := s.store.Get(ctx, item.ID)
		if err != nil {
			continue
		}
		snapshot := buildSubscribedPerformer(item, state)
		applyReviewSummary(&snapshot, records)
		out = append(out, snapshot)
	}
	return out
}
