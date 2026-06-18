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
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService)
}

func (s *runtimeSettingsEditor) UpdateStashSettings(input graphqlapi.UpdateStashSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateStash(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.APIKey),
		strings.TrimSpace(input.LibraryPath),
	)
	if err != nil {
		logging.Errorf("settings: save stash settings failed: %v", err)
		return nil, err
	}
	logging.Infof("settings: stash settings saved for url=%s library_path=%s", cfg.Stash.URL, cfg.Stash.LibraryPath)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService), nil
}

func (s *runtimeSettingsEditor) UpdateJackettSettings(input graphqlapi.UpdateJackettSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateJackett(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.APIKey),
	)
	if err != nil {
		logging.Errorf("settings: save jackett settings failed: %v", err)
		return nil, err
	}
	logging.Infof("settings: jackett settings saved for url=%s", cfg.Jackett.URL)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService), nil
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
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService), nil
}

func (s *runtimeSettingsEditor) UpdateSubscriptionSettings(input graphqlapi.UpdateSubscriptionSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateSubscription(
		strings.TrimSpace(input.Store),
		strings.TrimSpace(input.DBPath),
		input.PollIntervalSeconds,
		input.StashBoxEndpoints,
	)
	if err != nil {
		logging.Errorf("settings: save subscription settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: subscription settings saved for store=%s db_path=%s poll_interval=%d selected_endpoints=%d",
		cfg.Subscription.Store,
		cfg.Subscription.DBPath,
		cfg.Subscription.PollIntervalSeconds,
		len(cfg.Subscription.StashBoxEndpoints),
	)
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService), nil
}

func (s *runtimeSettingsEditor) UpdateLoggingSettings(input graphqlapi.UpdateLoggingSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateLogging(
		strings.TrimSpace(input.Level),
		strings.TrimSpace(input.FilePath),
		input.MaxEntries,
		input.MaxFileSizeBytes,
		input.MaxFileBackups,
	)
	if err != nil {
		logging.Errorf("settings: save logging settings failed: %v", err)
		return nil, err
	}
	if _, err := logging.ConfigureDefault(logging.Options{
		Level:            cfg.EffectiveLogLevel(),
		FilePath:         cfg.EffectiveLogFilePath(),
		MaxEntries:       cfg.EffectiveLogMaxEntries(),
		MaxFileSizeBytes: cfg.EffectiveLogMaxFileSizeBytes(),
		MaxFileBackups:   cfg.EffectiveLogMaxFileBackups(),
	}); err != nil {
		logging.Errorf("settings: hot-reload logger failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: logging settings saved level=%s file=%s max_entries=%d max_size=%d backups=%d",
		cfg.EffectiveLogLevel(),
		cfg.EffectiveLogFilePath(),
		cfg.EffectiveLogMaxEntries(),
		cfg.EffectiveLogMaxFileSizeBytes(),
		cfg.EffectiveLogMaxFileBackups(),
	)
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled, s.subscriptionService), nil
}
