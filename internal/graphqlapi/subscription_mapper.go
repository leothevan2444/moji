package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/subscription"
)

func stashPerformerToModel(performer subscription.Performer) *model.StashPerformer {
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

func subscriptionReleaseToModel(release subscription.RecordedRelease) *model.SubscriptionRelease {
	return &model.SubscriptionRelease{
		Key:    release.Key,
		Source: release.Source,
		Title:  release.Title,
		Code:   nilIfEmpty(release.Code),
		Date:   nilIfEmpty(release.Date),
		URL:    nilIfEmpty(release.URL),
		Query:  release.Query,
		TaskID: nilIfEmpty(release.TaskID),
		SeenAt: formatTime(release.SeenAt),
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
