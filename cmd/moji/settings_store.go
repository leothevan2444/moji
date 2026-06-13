package main

import (
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi"
)

type runtimeSettingsEditor struct {
	store              *config.Store
	version            string
	qbittorrentEnabled bool
	downloaderEnabled  bool
	stashEnabled       bool
}

func newRuntimeSettingsEditor(store *config.Store, version string, qbittorrentEnabled bool, downloaderEnabled bool, stashEnabled bool) *runtimeSettingsEditor {
	return &runtimeSettingsEditor{
		store:              store,
		version:            version,
		qbittorrentEnabled: qbittorrentEnabled,
		downloaderEnabled:  downloaderEnabled,
		stashEnabled:       stashEnabled,
	}
}

func (s *runtimeSettingsEditor) Snapshot() *graphqlapi.SettingsSnapshot {
	return buildSettingsSnapshot(s.store.Config(), s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled)
}

func (s *runtimeSettingsEditor) UpdateStashSettings(input graphqlapi.UpdateStashSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateStash(
		strings.TrimSpace(input.URL),
		trimOptionalSecret(input.APIKey),
		strings.TrimSpace(input.LibraryPath),
	)
	if err != nil {
		return nil, err
	}
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateJackettSettings(input graphqlapi.UpdateJackettSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateJackett(
		strings.TrimSpace(input.URL),
		trimOptionalSecret(input.APIKey),
	)
	if err != nil {
		return nil, err
	}
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateQBittorrentSettings(input graphqlapi.UpdateQBittorrentSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateQBittorrent(
		strings.TrimSpace(input.URL),
		strings.TrimSpace(input.Username),
		trimOptionalSecret(input.Password),
		strings.TrimSpace(input.DefaultSavePath),
		strings.TrimSpace(input.Category),
		strings.TrimSpace(input.Tags),
	)
	if err != nil {
		return nil, err
	}
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled), nil
}

func (s *runtimeSettingsEditor) UpdateFollowingSettings(input graphqlapi.UpdateFollowingSettingsInput) (*graphqlapi.SettingsSnapshot, error) {
	cfg, err := s.store.UpdateFollowing(
		strings.TrimSpace(input.Store),
		strings.TrimSpace(input.JSONPath),
		input.PollIntervalSeconds,
		trimOptionalSecret(input.JAVStashAPIKey),
	)
	if err != nil {
		return nil, err
	}
	return buildSettingsSnapshot(cfg, s.version, s.qbittorrentEnabled, s.downloaderEnabled, s.stashEnabled), nil
}

func trimOptionalSecret(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
