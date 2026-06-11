package graphqlapi

import "github.com/leothevan2444/moji/internal/graphqlapi/model"

func settingsSnapshotToModel(snapshot *SettingsSnapshot, appVersion string) *model.Settings {
	if snapshot == nil {
		return &model.Settings{
			Stash:       &model.StashSettings{},
			Jackett:     &model.JackettSettings{},
			Qbittorrent: &model.QBittorrentSettings{},
			Tasks:       &model.TaskSettings{},
			System:      &model.SystemSettings{AppVersion: appVersion},
		}
	}

	return &model.Settings{
		Stash: &model.StashSettings{
			Configured:       snapshot.Stash.Configured,
			Enabled:          snapshot.Stash.Enabled,
			GraphqlURL:       snapshot.Stash.GraphQLURL,
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
			JSONPath:                    snapshot.Tasks.JSONPath,
			ProgressSyncIntervalSeconds: snapshot.Tasks.ProgressSyncIntervalSeconds,
			ProgressSyncEnabled:         snapshot.Tasks.ProgressSyncEnabled,
		},
		System: &model.SystemSettings{
			AppVersion: snapshot.System.AppVersion,
		},
	}
}
