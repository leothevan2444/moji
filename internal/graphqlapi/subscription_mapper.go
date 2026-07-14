package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/discovery"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	performerdomain "github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/subscription"
)

func stashPerformerToModel(performer performerdomain.Performer) *model.StashPerformer {
	aliases := append([]string(nil), performer.AliasList...)
	return &model.StashPerformer{
		ID:         performer.ID,
		Name:       performer.Name,
		AliasList:  aliases,
		Favorite:   performer.Favorite,
		ImagePath:  nilIfEmpty(performer.ImagePath),
		SceneCount: performer.SceneCount,
		Subscribed: performer.Subscribed,
	}
}

func stashPerformerPageToModel(page StashPerformerPage) *model.StashPerformerConnection {
	items := make([]*model.StashPerformer, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, stashPerformerToModel(item))
	}

	return &model.StashPerformerConnection{
		Items:       items,
		Page:        page.Page,
		PageSize:    page.PageSize,
		TotalCount:  page.TotalCount,
		TotalPages:  page.TotalPages,
		HasPrevPage: page.HasPrevPage,
		HasNextPage: page.HasNextPage,
	}
}

func stashPerformerDetailToModel(detail performerdomain.Detail) *model.StashPerformerDetail {
	return &model.StashPerformerDetail{
		Performer:          stashPerformerToModel(detail.Performer),
		Disambiguation:     nilIfEmpty(detail.Disambiguation),
		Birthdate:          nilIfEmpty(detail.Birthdate),
		Ethnicity:          nilIfEmpty(detail.Ethnicity),
		Country:            nilIfEmpty(detail.Country),
		EyeColor:           nilIfEmpty(detail.EyeColor),
		HeightCm:           detail.HeightCm,
		Rating100:          detail.Rating100,
		Urls:               append([]string(nil), detail.URLs...),
		MatchedStashBox:    matchedStashBoxToModel(detail.MatchedStashBox),
		TotalSceneCount:    detail.TotalSceneCount,
		StashSceneCount:    detail.StashSceneCount,
		StashBoxSceneCount: detail.StashBoxSceneCount,
		DedupedSceneCount:  detail.DedupedSceneCount,
	}
}

func matchedStashBoxToModel(item *performerdomain.MatchedStashBox) *model.MatchedStashBox {
	if item == nil {
		return nil
	}
	return &model.MatchedStashBox{
		Name:          item.Name,
		Endpoint:      item.Endpoint,
		PerformerID:   item.PerformerID,
		PerformerName: item.PerformerName,
	}
}

func discoveredScenePageToModel(page discovery.Page) *model.DiscoverSceneConnection {
	items := make([]*model.DiscoveredScene, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, discoveredSceneToModel(item))
	}
	return &model.DiscoverSceneConnection{
		Items:         items,
		UsedStashBox:  discoveredSourceToModel(page.UsedStashBox),
		FallbackCount: page.FallbackCount,
		SearchedQuery: page.SearchedQuery,
	}
}

func discoveredSourceToModel(item *discovery.MatchedSource) *model.MatchedStashBox {
	if item == nil {
		return nil
	}
	return &model.MatchedStashBox{Name: item.Name, Endpoint: item.Endpoint, PerformerID: item.PerformerID, PerformerName: item.PerformerName}
}

func discoveredSceneToModel(item discovery.Scene) *model.DiscoveredScene {
	return &model.DiscoveredScene{
		Key:              item.Key,
		SceneID:          item.SceneID,
		StashBoxEndpoint: item.StashBoxEndpoint,
		StashBoxName:     item.StashBoxName,
		Title:            item.Title,
		DurationSeconds:  item.DurationSeconds,
		Code:             nilIfEmpty(item.Code),
		Date:             nilIfEmpty(item.Date),
		StudioName:       nilIfEmpty(item.StudioName),
		ImageURL:         nilIfEmpty(item.ImageURL),
		URL:              nilIfEmpty(item.URL),
		PerformerNames:   append([]string(nil), item.PerformerNames...),
		DerivedQuery:     item.DerivedQuery,
	}
}

func performerScenePageToModel(page performerdomain.ScenePage) *model.StashPerformerSceneConnection {
	items := make([]*model.StashPerformerScene, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, performerSceneToModel(item))
	}
	var totalPages *int
	if page.TotalCountExact {
		totalPages = &page.TotalPages
	}
	return &model.StashPerformerSceneConnection{
		Items:               items,
		Page:                page.Page,
		PageSize:            page.PageSize,
		TotalCount:          page.TotalCount,
		TotalPages:          totalPages,
		HasPrevPage:         page.HasPrevPage,
		HasNextPage:         page.HasNextPage,
		StashSceneCount:     page.StashSceneCount,
		StashBoxCount:       page.StashBoxCount,
		DedupedCount:        page.DedupedCount,
		TotalCountExact:     page.TotalCountExact,
		StashBoxRemoteCount: page.StashBoxCount,
		StashBoxLoadedCount: page.StashBoxLoadedCount,
		CacheComplete:       page.CacheComplete,
		CacheUpdatedAt:      formatTimePointer(page.CacheUpdatedAt),
		CacheStale:          page.CacheStale,
	}
}

func performerSceneToModel(item performerdomain.Scene) *model.StashPerformerScene {
	stashIDs := make([]*model.StashSceneID, 0, len(item.StashIDs))
	for _, stashID := range item.StashIDs {
		stashIDs = append(stashIDs, &model.StashSceneID{
			Endpoint: stashID.Endpoint,
			StashID:  stashID.StashID,
		})
	}
	performers := make([]*model.PerformerScenePerson, 0, len(item.Performers))
	for _, performer := range item.Performers {
		performers = append(performers, &model.PerformerScenePerson{ID: performer.ID, Name: performer.Name})
	}
	tags := make([]*model.PerformerSceneTag, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, &model.PerformerSceneTag{ID: tag.ID, Name: tag.Name})
	}
	var mojiTask *model.PerformerSceneTask
	if item.MojiTask != nil {
		mojiTask = &model.PerformerSceneTask{
			ID:               item.MojiTask.ID,
			Stage:            model.TaskStage(item.MojiTask.Stage),
			StageStatus:      model.TaskStageStatus(item.MojiTask.StageStatus),
			StageLabel:       item.MojiTask.StageLabel,
			StageStatusLabel: item.MojiTask.StageStatusLabel,
			Progress:         item.MojiTask.Progress,
		}
	}
	return &model.StashPerformerScene{
		Key:                 item.Key,
		PrimarySource:       model.SceneSource(item.PrimarySource),
		SourceSceneID:       item.SourceSceneID,
		Title:               nilIfEmpty(item.Title),
		Code:                nilIfEmpty(item.Code),
		Date:                nilIfEmpty(item.Date),
		StudioName:          nilIfEmpty(item.StudioName),
		PerformerCount:      item.PerformerCount,
		TagCount:            item.TagCount,
		Performers:          performers,
		Tags:                tags,
		ImageURL:            nilIfEmpty(item.ImageURL),
		URL:                 nilIfEmpty(item.URL),
		InLibrary:           item.InLibrary,
		MatchedStashSceneID: nilIfEmpty(item.MatchedStashSceneID),
		HasStashSource:      item.HasStashSource,
		HasStashBoxSource:   item.HasStashBoxSource,
		StashBoxSceneID:     nilIfEmpty(item.StashBoxSceneID),
		StashBoxEndpoint:    nilIfEmpty(item.StashBoxEndpoint),
		SourceLabels:        append([]string(nil), item.SourceLabels...),
		StashIds:            stashIDs,
		MojiTask:            mojiTask,
	}
}

func subscriptionPerformerToModel(item subscription.SubscribedPerformer) *model.SubscribedPerformer {
	releases := make([]*model.SubscriptionRelease, 0, len(item.RecentReleases))
	for _, release := range item.RecentReleases {
		releases = append(releases, subscriptionReleaseToModel(release))
	}

	return &model.SubscribedPerformer{
		Performer:             stashPerformerToModel(item.Performer),
		LastCheckedAt:         formatTimePointer(item.LastCheckedAt),
		LastError:             nilIfEmpty(item.LastError),
		PendingReleaseCount:   item.PendingReleaseCount,
		ProcessedReleaseCount: item.ProcessedReleaseCount,
		RecentReleases:        releases,
	}
}

func performerBatchPayloadToModel(payload subscription.PerformerBatchPayload) *model.PerformerBatchPayload {
	results := make([]*model.PerformerBatchResult, 0, len(payload.Results))
	for _, item := range payload.Results {
		var performerModel *model.StashPerformer
		if item.Performer != nil {
			performerModel = stashPerformerToModel(*item.Performer)
		}
		var stateModel *model.SubscribedPerformer
		if item.State != nil {
			stateModel = subscriptionPerformerToModel(*item.State)
		}
		results = append(results, &model.PerformerBatchResult{PerformerID: item.PerformerID, Status: model.PerformerBatchStatus(item.Status), ReasonCode: item.ReasonCode, Performer: performerModel, State: stateModel})
	}
	return &model.PerformerBatchPayload{BatchID: payload.BatchID, Summary: &model.PerformerBatchSummary{RequestedCount: payload.Summary.RequestedCount, SucceededCount: payload.Summary.SucceededCount, SkippedCount: payload.Summary.SkippedCount, FailedCount: payload.Summary.FailedCount}, Results: results}
}

func subscriptionReleaseToModel(release subscription.RecordedRelease) *model.SubscriptionRelease {
	return &model.SubscriptionRelease{
		Key:            release.Key,
		Source:         release.Source,
		Title:          release.Title,
		Code:           nilIfEmpty(release.Code),
		Date:           nilIfEmpty(release.Date),
		URL:            nilIfEmpty(release.URL),
		TaskID:         nilIfEmpty(release.TaskID),
		PerformerCount: release.PerformerCount,
		PerformerNames: append([]string(nil), release.PerformerNames...),
		SeenAt:         formatTime(release.SeenAt),
	}
}

func queuePerformerScenesResultToModel(result performerdomain.QueueScenesResult) *model.QueuePerformerScenesPayload {
	queuedTasks := make([]*model.Task, 0, len(result.QueuedTasks))
	for _, task := range result.QueuedTasks {
		queuedTasks = append(queuedTasks, taskToModel(task))
	}
	results := make([]*model.QueuePerformerSceneResult, 0, len(result.Results))
	for _, item := range result.Results {
		results = append(results, queuePerformerSceneResultToModel(item))
	}
	return &model.QueuePerformerScenesPayload{
		QueuedTasks: queuedTasks,
		Results:     results,
		Summary: &model.QueuePerformerScenesSummary{
			RequestedCount: result.Summary.RequestedCount,
			QueuedCount:    result.Summary.QueuedCount,
			SkippedCount:   result.Summary.SkippedCount,
			FailedCount:    result.Summary.FailedCount,
		},
	}
}

func queuePerformerSceneResultToModel(item performerdomain.QueueSceneResult) *model.QueuePerformerSceneResult {
	return &model.QueuePerformerSceneResult{
		Key:          item.Key,
		Status:       model.QueuePerformerSceneStatus(item.Status),
		ReasonCode:   item.ReasonCode,
		Message:      item.Message,
		Task:         taskToModel(item.Task),
		ResolvedCode: nilIfEmpty(item.ResolvedCode),
	}
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatTimePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatTime(*value)
	return &formatted
}
