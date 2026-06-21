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
			Automation:   &model.AutomationSettings{},
			Subscription: &model.SubscriptionSettings{},
		}
	}

	return &model.Settings{
		Stash: &model.StashSettings{
			Configured:            snapshot.Stash.Configured,
			Enabled:               snapshot.Stash.Enabled,
			URL:                   snapshot.Stash.URL,
			APIKeyConfigured:      snapshot.Stash.APIKeyConfigured,
			APIKey:                snapshot.Stash.APIKey,
			Mode:                  snapshot.Stash.Mode,
			LibraryPath:           snapshot.Stash.LibraryPath,
			QbittorrentPathPrefix: snapshot.Stash.QBittorrentPathPrefix,
			StashPathPrefix:       snapshot.Stash.StashPathPrefix,
			TransferAction:        snapshot.Stash.TransferAction,
			TransferTargetPath:    snapshot.Stash.TransferTargetPath,
		},
		Jackett: &model.JackettSettings{
			Configured:       snapshot.Jackett.Configured,
			Enabled:          snapshot.Jackett.Enabled,
			URL:              snapshot.Jackett.URL,
			APIKeyConfigured: snapshot.Jackett.APIKeyConfigured,
			APIKey:           snapshot.Jackett.APIKey,
		},
		Qbittorrent: &model.QBittorrentSettings{
			Configured:         snapshot.QBittorrent.Configured,
			Enabled:            snapshot.QBittorrent.Enabled,
			URL:                snapshot.QBittorrent.URL,
			Username:           snapshot.QBittorrent.Username,
			UsernameConfigured: snapshot.QBittorrent.UsernameConfigured,
			PasswordConfigured: snapshot.QBittorrent.PasswordConfigured,
			Password:           snapshot.QBittorrent.Password,
			DefaultSavePath:    snapshot.QBittorrent.DefaultSavePath,
			Category:           snapshot.QBittorrent.Category,
			Tags:               snapshot.QBittorrent.Tags,
		},
		Automation: &model.AutomationSettings{
			TaskProgressSyncIntervalSeconds: snapshot.Automation.TaskProgressSyncIntervalSeconds,
			SubscriptionPollIntervalSeconds: snapshot.Automation.SubscriptionPollIntervalSeconds,
		},
		Subscription: &model.SubscriptionSettings{
			StashBoxEndpoints: append([]string(nil), snapshot.Subscription.StashBoxEndpoints...),
		},
	}
}

func settingsStatusSnapshotToModel(snapshot *SettingsStatusSnapshot) *model.SettingsStatus {
	if snapshot == nil {
		return &model.SettingsStatus{
			Stash:        &model.ServiceStatus{},
			Jackett:      &model.ServiceStatus{},
			Qbittorrent:  &model.ServiceStatus{},
			Automation:   &model.AutomationStatus{},
			Subscription: &model.SubscriptionStatus{},
		}
	}

	return &model.SettingsStatus{
		Stash: &model.ServiceStatus{
			Configured: snapshot.Stash.Configured,
			Enabled:    snapshot.Stash.Enabled,
		},
		Jackett: &model.ServiceStatus{
			Configured: snapshot.Jackett.Configured,
			Enabled:    snapshot.Jackett.Enabled,
		},
		Qbittorrent: &model.ServiceStatus{
			Configured: snapshot.QBittorrent.Configured,
			Enabled:    snapshot.QBittorrent.Enabled,
		},
		Automation: &model.AutomationStatus{
			TaskProgressSyncIntervalSeconds: snapshot.Automation.TaskProgressSyncIntervalSeconds,
			TaskProgressSyncEnabled:         snapshot.Automation.TaskProgressSyncEnabled,
			SubscriptionPollIntervalSeconds: snapshot.Automation.SubscriptionPollIntervalSeconds,
			SubscriptionPollEnabled:         snapshot.Automation.SubscriptionPollEnabled,
		},
		Subscription: &model.SubscriptionStatus{
			StashBoxes:          subscriptionStashBoxesToModel(snapshot.Subscription.StashBoxes),
			StashBoxesLoaded:    snapshot.Subscription.StashBoxesLoaded,
			StashBoxesLoadError: subscriptionLoadErrorPtr(snapshot.Subscription.StashBoxesLoadError),
		},
	}
}
