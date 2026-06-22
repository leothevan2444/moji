package main

import (
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/logging"
)

type runtimeSettingsEditor struct {
	store               *config.Store
	version             string
	qbittorrentEnabled  bool
	downloaderEnabled   bool
	stashEnabled        bool
	subscriptionService graphqlapi.SubscriptionService
}

func newRuntimeSettingsEditor(store *config.Store, version string, qbittorrentEnabled bool, downloaderEnabled bool, stashEnabled bool, subscriptionService graphqlapi.SubscriptionService) *runtimeSettingsEditor {
	return &runtimeSettingsEditor{
		store:               store,
		version:             version,
		qbittorrentEnabled:  qbittorrentEnabled,
		downloaderEnabled:   downloaderEnabled,
		stashEnabled:        stashEnabled,
		subscriptionService: subscriptionService,
	}
}

func (s *runtimeSettingsEditor) Snapshot() *graphqlapi.SettingsSnapshot {
	cfg := s.store.Config()
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled)
}

func (s *runtimeSettingsEditor) StatusSnapshot() *graphqlapi.SettingsStatusSnapshot {
	cfg := s.store.Config()
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsStatusSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService)
}

func (s *runtimeSettingsEditor) UpdateStashSettings(input graphqlapi.UpdateStashSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateStash(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.APIKey),
	)
	if err != nil {
		logging.Errorf("settings: save stash settings failed: %v", err)
		return nil, err
	}
	logging.Infof("settings: stash settings saved for url=%s", cfg.Stash.URL)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateIngestSettings(input graphqlapi.UpdateIngestSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateIngest(
		strings.TrimSpace(input.Mode),
		config.SharedStorageIngestConfig{
			QBittorrentPathPrefix: strings.TrimSpace(input.SharedStorage.QBittorrentPathPrefix),
			StashPathPrefix:       strings.TrimSpace(input.SharedStorage.StashPathPrefix),
		},
		config.FileTransferIngestConfig{
			Action:     strings.TrimSpace(input.FileTransfer.Action),
			TargetPath: strings.TrimSpace(input.FileTransfer.TargetPath),
		},
		config.LibraryScanIngestConfig{
			LibraryPath: strings.TrimSpace(input.LibraryScan.LibraryPath),
		},
	)
	if err != nil {
		logging.Errorf("settings: save ingest settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: ingest settings saved mode=%s library_path=%s transfer_target=%s",
		cfg.Ingest.Mode,
		cfg.Ingest.LibraryScan.LibraryPath,
		cfg.Ingest.FileTransfer.TargetPath,
	)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateJackettSettings(input graphqlapi.UpdateJackettSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateJackett(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.APIKey),
		strings.TrimSpace(input.Password),
	)
	if err != nil {
		logging.Errorf("settings: save jackett settings failed: %v", err)
		return nil, err
	}
	logging.Infof("settings: jackett settings saved for url=%s", cfg.Jackett.URL)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateQBittorrentSettings(input graphqlapi.UpdateQBittorrentSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateQBittorrent(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.Username),
		strings.TrimSpace(input.Password),
		strings.TrimSpace(input.DefaultSavePath),
		strings.TrimSpace(input.Category),
		strings.TrimSpace(input.Tags),
	)
	if err != nil {
		logging.Errorf("settings: save qBittorrent settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: qBittorrent settings saved for url=%s username=%s save_path=%s category=%s",
		cfg.QBittorrent.URL,
		cfg.QBittorrent.Username,
		cfg.QBittorrent.DefaultSavePath,
		cfg.QBittorrent.Category,
	)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateAutomationSettings(input graphqlapi.UpdateAutomationSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateAutomation(
		input.TaskProgressSyncIntervalSeconds,
		input.SubscriptionPollIntervalSeconds,
	)
	if err != nil {
		logging.Errorf("settings: save automation settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: automation settings saved task_sync_interval=%d subscription_poll_interval=%d",
		cfg.Automation.TaskProgressSyncIntervalSeconds,
		cfg.Automation.SubscriptionPollIntervalSeconds,
	)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateSubscriptionSettings(input graphqlapi.UpdateSubscriptionSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateSubscription(input.StashBoxEndpoints)
	if err != nil {
		logging.Errorf("settings: save subscription settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: subscription settings saved selected_endpoints=%d",
		len(cfg.Subscription.StashBoxEndpoints),
	)
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled), nil
}
