package main

import (
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/pkg/stash"
)

type runtimeSettingsEditor struct {
	store               *config.Store
	version             string
	downloaderEnabled   bool
	stashEnabled        bool
	stashClient         *stash.Client
	subscriptionService graphqlapi.SubscriptionService
}

func newRuntimeSettingsEditor(store *config.Store, version string, downloaderEnabled bool, stashEnabled bool, stashClient *stash.Client, subscriptionService graphqlapi.SubscriptionService) *runtimeSettingsEditor {
	return &runtimeSettingsEditor{
		store:               store,
		version:             version,
		downloaderEnabled:   downloaderEnabled,
		stashEnabled:        stashEnabled,
		stashClient:         stashClient,
		subscriptionService: subscriptionService,
	}
}

func (s *runtimeSettingsEditor) Snapshot() *graphqlapi.SettingsSnapshot {
	cfg := s.store.Config()
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsSnapshot(cfg, s.version)
}

func (s *runtimeSettingsEditor) StatusSnapshot() *graphqlapi.SettingsStatusSnapshot {
	cfg := s.store.Config()
	applySubscriptionOrder(cfg, s.subscriptionService)
	return buildSettingsStatusSnapshot(cfg, s.version, s.downloaderEnabled, s.stashEnabled, s.stashClient, s.subscriptionService)
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
	return buildSettingsSnapshot(cfg, s.version), nil
}

func (s *runtimeSettingsEditor) UpdateIngestSettings(input graphqlapi.UpdateIngestSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateIngest(
		strings.TrimSpace(input.DeliveryMode),
		strings.TrimSpace(input.StashLibraryPath),
		config.TransferIngestConfig{
			Action:         strings.TrimSpace(input.Transfer.Action),
			MojiSourceRoot: strings.TrimSpace(input.Transfer.MojiSourceRoot),
			MojiTargetRoot: strings.TrimSpace(input.Transfer.MojiTargetRoot),
		},
	)
	if err != nil {
		logging.Errorf("settings: save ingest settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: ingest settings saved delivery_mode=%s stash_library=%s moji_target_root=%s",
		cfg.Ingest.DeliveryMode,
		cfg.Ingest.StashLibraryPath,
		cfg.Ingest.Transfer.MojiTargetRoot,
	)
	return buildSettingsSnapshot(cfg, s.version), nil
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
	return buildSettingsSnapshot(cfg, s.version), nil
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
	return buildSettingsSnapshot(cfg, s.version), nil
}

func (s *runtimeSettingsEditor) UpdateAutomationSettings(input graphqlapi.UpdateAutomationSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateAutomation(
		input.TaskProgressSyncIntervalSeconds,
		input.SubscriptionPollIntervalHours,
	)
	if err != nil {
		logging.Errorf("settings: save automation settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: automation settings saved task_sync_interval=%d subscription_poll_interval_hours=%d",
		cfg.Automation.TaskProgressSyncIntervalSeconds,
		cfg.Automation.SubscriptionPollIntervalHours,
	)
	return buildSettingsSnapshot(cfg, s.version), nil
}

func (s *runtimeSettingsEditor) UpdateSystemSettings(input graphqlapi.UpdateSystemSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	policy := config.NormalizeTaskDeletePolicy(input.TaskDeletePolicy)
	cfg, err := s.store.UpdateSystem(policy)
	if err != nil {
		logging.Errorf("settings: save system settings failed: %v", err)
		return nil, err
	}
	logging.Infof("settings: system settings saved task_delete_policy=%s", cfg.System.EffectiveTaskDeletePolicy())
	return buildSettingsSnapshot(cfg, s.version), nil
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
	return buildSettingsSnapshot(cfg, s.version), nil
}
