package graphqlapi

import (
	"time"

	"github.com/leothevan2444/moji/internal/following"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

func stashPerformerToModel(performer following.Performer) *model.StashPerformer {
	aliases := append([]string(nil), performer.AliasList...)
	return &model.StashPerformer{
		ID:         performer.ID,
		Name:       performer.Name,
		AliasList:  aliases,
		Favorite:   performer.Favorite,
		ImagePath:  nilIfEmpty(performer.ImagePath),
		SceneCount: performer.SceneCount,
		Followed:   performer.Followed,
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

func followingPerformerToModel(item following.FollowingPerformer) *model.FollowingPerformer {
	releases := make([]*model.FollowingRelease, 0, len(item.RecentReleases))
	for _, release := range item.RecentReleases {
		releases = append(releases, followingReleaseToModel(release))
	}

	return &model.FollowingPerformer{
		Performer:             stashPerformerToModel(item.Performer),
		LastCheckedAt:         formatTimePointer(item.LastCheckedAt),
		LastError:             nilIfEmpty(item.LastError),
		PendingReleaseCount:   item.PendingReleaseCount,
		ProcessedReleaseCount: item.ProcessedReleaseCount,
		RecentReleases:        releases,
	}
}

func followingReleaseToModel(release following.RecordedRelease) *model.FollowingRelease {
	return &model.FollowingRelease{
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
