package graphqlapi

import "github.com/leothevan2444/moji/internal/graphqlapi/model"

func subscriptionStashBoxesToModel(items []StashBoxEndpointSnapshot) []*model.StashBoxEndpoint {
	if len(items) == 0 {
		return []*model.StashBoxEndpoint{}
	}
	out := make([]*model.StashBoxEndpoint, 0, len(items))
	for _, item := range items {
		out = append(out, &model.StashBoxEndpoint{
			Name:             item.Name,
			Endpoint:         item.Endpoint,
			APIKeyConfigured: item.APIKeyConfigured,
		})
	}
	return out
}

func subscriptionLoadErrorPtr(message string) *string {
	if message == "" {
		return nil
	}
	copy := message
	return &copy
}

func settingsSnapshotToModel(snapshot *SettingsSnapshot, appVersion string) *model.Settings {
	if snapshot == nil {
		return &model.Settings{
			Stash:        &model.StashSettings{},
			Jackett:      &model.JackettSettings{},
			Qbittorrent:  &model.QBittorrentSettings{},
			Tasks:        &model.TaskSettings{},
			Subscription: &model.SubscriptionSettings{},
			Logging:      &model.LoggingSettings{},
			System:       &model.SystemSettings{AppVersion: appVersion},
		}
	}

	return &model.Settings{
		Stash: &model.StashSettings{
			Configured:       snapshot.Stash.Configured,
			Enabled:          snapshot.Stash.Enabled,
			URL:              snapshot.Stash.URL,
			APIKeyConfigured: snapshot.Stash.APIKeyConfigured,
			LibraryPath:      snapshot.Stash.LibraryPath,
		},
		Jackett: &model.JackettSettings{
			Configured:       snapshot.Jackett.Configured,
			Enabled:          snapshot.Jackett.Enabled,
			URL:              snapshot.Jackett.URL,
			APIKeyConfigured: snapshot.Jackett.APIKeyConfigured,
		},
		Qbittorrent: &model.QBittorrentSettings{
			Configured:         snapshot.QBittorrent.Configured,
			Enabled:            snapshot.QBittorrent.Enabled,
			URL:                snapshot.QBittorrent.URL,
			Username:           snapshot.QBittorrent.Username,
			UsernameConfigured: snapshot.QBittorrent.UsernameConfigured,
			PasswordConfigured: snapshot.QBittorrent.PasswordConfigured,
			DefaultSavePath:    snapshot.QBittorrent.DefaultSavePath,
			Category:           snapshot.QBittorrent.Category,
			Tags:               snapshot.QBittorrent.Tags,
		},
		Tasks: &model.TaskSettings{
			Store:                       snapshot.Tasks.Store,
			DbPath:                      snapshot.Tasks.DBPath,
			ProgressSyncIntervalSeconds: snapshot.Tasks.ProgressSyncIntervalSeconds,
			ProgressSyncEnabled:         snapshot.Tasks.ProgressSyncEnabled,
		},
		Subscription: &model.SubscriptionSettings{
			Store:                       snapshot.Subscription.Store,
			DbPath:                      snapshot.Subscription.DBPath,
			PollIntervalSeconds:         snapshot.Subscription.PollIntervalSeconds,
			PollEnabled:                 snapshot.Subscription.PollEnabled,
			StashBoxes:                  subscriptionStashBoxesToModel(snapshot.Subscription.StashBoxes),
			SelectedStashBoxEndpoints:   append([]string(nil), snapshot.Subscription.SelectedStashBoxEndpoints...),
			StashBoxesLoaded:            snapshot.Subscription.StashBoxesLoaded,
			StashBoxesLoadError:         subscriptionLoadErrorPtr(snapshot.Subscription.StashBoxesLoadError),
		},
		Logging: &model.LoggingSettings{
			Level:            snapshot.Logging.Level,
			FilePath:         snapshot.Logging.FilePath,
			MaxEntries:       snapshot.Logging.MaxEntries,
			MaxFileSizeBytes: int(snapshot.Logging.MaxFileSizeBytes),
			MaxFileBackups:   snapshot.Logging.MaxFileBackups,
		},
		System: &model.SystemSettings{
			AppVersion: snapshot.System.AppVersion,
		},
	}
}
