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
	logging.Infof("settings: stash settings saved for url=%s", cfg.Connection.Stash.URL)
	return buildSettingsSnapshot(cfg, s.version), nil
}

func (s *runtimeSettingsEditor) UpdateIngestSettings(input graphqlapi.UpdateIngestSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateIngest(
		strings.TrimSpace(input.DeliveryMode),
		config.DownloadsIngestConfig{
			QBRoot:   strings.TrimSpace(input.Downloads.QBRoot),
			MojiRoot: strings.TrimSpace(input.Downloads.MojiRoot),
		},
		config.LibraryIngestConfig{
			MojiRoot:  strings.TrimSpace(input.Library.MojiRoot),
			StashRoot: strings.TrimSpace(input.Library.StashRoot),
		},
		config.TransferIngestConfig{
			Action: strings.TrimSpace(input.Transfer.Action),
		},
	)
	if err != nil {
		logging.Errorf("settings: save ingest settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: ingest settings saved delivery_mode=%s qb_root=%s moji_download_root=%s stash_library_root=%s",
		cfg.Ingest.DeliveryMode,
		cfg.Ingest.Downloads.QBRoot,
		cfg.Ingest.Downloads.MojiRoot,
		cfg.Ingest.Library.StashRoot,
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
	logging.Infof("settings: jackett settings saved for url=%s", cfg.Connection.Jackett.URL)
	return buildSettingsSnapshot(cfg, s.version), nil
}

func torrentSelectionConfigRules(rules []graphqlapi.TorrentSelectionRuleSnapshot) []config.TorrentSelectionRule {
	out := make([]config.TorrentSelectionRule, 0, len(rules))
	for _, rule := range rules {
		item := config.TorrentSelectionRule{
			Type:    config.TorrentSelectionRuleType(strings.TrimSpace(rule.Type)),
			Enabled: rule.Enabled,
			IndexerPreference: config.IndexerPreferenceRuleConfig{
				TrackerIDs: append([]string(nil), rule.IndexerPreference.TrackerIDs...),
			},
			PublishDate: config.PublishDateRuleConfig{
				Direction: config.TorrentSelectionDirection(strings.TrimSpace(rule.PublishDate.Direction)),
			},
			Seeders: config.SeedersRuleConfig{
				Direction: config.TorrentSelectionDirection(strings.TrimSpace(rule.Seeders.Direction)),
			},
			Size: config.SizeRuleConfig{
				Direction: config.TorrentSelectionDirection(strings.TrimSpace(rule.Size.Direction)),
			},
		}
		if len(rule.TitleMatch.Clauses) > 0 {
			item.TitleMatch.Clauses = make([]config.TitleMatchClause, 0, len(rule.TitleMatch.Clauses))
			for _, clause := range rule.TitleMatch.Clauses {
				item.TitleMatch.Clauses = append(item.TitleMatch.Clauses, config.TitleMatchClause{
					Pattern:     strings.TrimSpace(clause.Pattern),
					PatternMode: config.TitleMatchPatternMode(strings.TrimSpace(clause.PatternMode)),
					Effect:      config.TitleMatchEffect(strings.TrimSpace(clause.Effect)),
				})
			}
		}
		if len(rule.TorrentFileNameMatch.Clauses) > 0 {
			item.TorrentFileNameMatch.Clauses = make([]config.TorrentFileNameMatchClause, 0, len(rule.TorrentFileNameMatch.Clauses))
			for _, clause := range rule.TorrentFileNameMatch.Clauses {
				item.TorrentFileNameMatch.Clauses = append(item.TorrentFileNameMatch.Clauses, config.TorrentFileNameMatchClause{
					Pattern:     strings.TrimSpace(clause.Pattern),
					PatternMode: config.TitleMatchPatternMode(strings.TrimSpace(clause.PatternMode)),
					Effect:      config.TorrentFileMatchEffect(strings.TrimSpace(clause.Effect)),
				})
			}
		}
		out = append(out, item)
	}
	return out
}

func torrentSelectionConfigRulesFromSnapshot(snapshot graphqlapi.TorrentSelectionSettingsSnapshot) []config.TorrentSelectionRule {
	rules := append([]graphqlapi.TorrentSelectionRuleSnapshot(nil), snapshot.FastRules...)
	rules = append(rules, snapshot.TorrentRules...)
	return torrentSelectionConfigRules(rules)
}

func subscriptionReleasePolicyConfigFromSnapshot(snapshot graphqlapi.SubscriptionReleasePolicySnapshot) config.SubscriptionReleasePolicyConfig {
	return config.SubscriptionReleasePolicyConfig{
		SoloBehavior:           config.SubscriptionReleaseBehavior(strings.TrimSpace(snapshot.SoloBehavior)),
		GroupBehavior:          config.SubscriptionReleaseBehavior(strings.TrimSpace(snapshot.GroupBehavior)),
		CompilationBehavior:    config.SubscriptionReleaseBehavior(strings.TrimSpace(snapshot.CompilationBehavior)),
		MaxGroupPerformerCount: snapshot.MaxGroupPerformerCount,
		ReleaseDateRange:       config.SubscriptionReleaseDateRange(strings.TrimSpace(snapshot.ReleaseDateRange)),
	}
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
		cfg.Connection.QBittorrent.URL,
		cfg.Connection.QBittorrent.Username,
		cfg.Connection.QBittorrent.DefaultSavePath,
		cfg.Connection.QBittorrent.Category,
	)
	return buildSettingsSnapshot(cfg, s.version), nil
}

func (s *runtimeSettingsEditor) UpdateAutomationSettings(input graphqlapi.UpdateAutomationSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateAutomation(
		input.TaskProgressSyncIntervalSeconds,
		input.SubscriptionPollIntervalHours,
		input.StashBoxEndpoints,
		subscriptionReleasePolicyConfigFromSnapshot(input.SubscriptionReleasePolicy),
		config.NewTorrentSelectionConfig(
			input.TorrentSelection.Enabled,
			input.TorrentSelection.InspectionCandidateLimit,
			torrentSelectionConfigRulesFromSnapshot(input.TorrentSelection),
		),
	)
	if err != nil {
		logging.Errorf("settings: save automation settings failed: %v", err)
		return nil, err
	}
	logging.Infof(
		"settings: automation settings saved task_sync_interval=%d subscription_poll_interval_hours=%d selected_endpoints=%d",
		cfg.Automation.TaskProgressSyncIntervalSeconds,
		cfg.Automation.SubscriptionPollIntervalHours,
		len(cfg.Automation.StashBoxEndpoints),
	)
	applySubscriptionOrder(cfg, s.subscriptionService)
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
