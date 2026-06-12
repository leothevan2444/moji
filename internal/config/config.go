package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type JackettConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

type QBittorrentConfig struct {
	URL             string `yaml:"url"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	DefaultSavePath string `yaml:"default_save_path"`
	Category        string `yaml:"category"`
	Tags            string `yaml:"tags"`
}

type StashConfig struct {
	URL              string `yaml:"url"`
	LegacyGraphQLURL string `yaml:"graphql_url"`
	APIKey           string `yaml:"api_key"`
	LibraryPath      string `yaml:"library_path"`
}

func (s *StashConfig) normalize() {
	if s.URL == "" && s.LegacyGraphQLURL != "" {
		s.URL = trimGraphQLSuffix(s.LegacyGraphQLURL)
	}
}

func (s StashConfig) GraphQLEndpoint() string {
	return buildStashGraphQLEndpoint(s.URL)
}

type TaskConfig struct {
	Store                       string `yaml:"store"`
	JSONPath                    string `yaml:"json_path"`
	ProgressSyncIntervalSeconds int    `yaml:"progress_sync_interval_seconds"`
}

type FollowingConfig struct {
	Store               string `yaml:"store"`
	JSONPath            string `yaml:"json_path"`
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	JAVStashAPIKey      string `yaml:"javstash_api_key"`
}

type Config struct {
	// Jackett is the configuration for the Jackett service
	Jackett     JackettConfig     `yaml:"jackett"`
	QBittorrent QBittorrentConfig `yaml:"qbittorrent"`
	Stash       StashConfig       `yaml:"stash"`
	Tasks       TaskConfig        `yaml:"tasks"`
	Following   FollowingConfig   `yaml:"following"`
}

func Load() (*Config, error) {
	path := DefaultPath()
	return LoadFromPath(path)
}

func LoadFromPath(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	config.Stash.normalize()

	return &config, nil
}

func buildStashGraphQLEndpoint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	base := strings.TrimRight(trimmed, "/")
	if strings.HasSuffix(base, "/graphql") {
		return base
	}
	return base + "/graphql"
}

func trimGraphQLSuffix(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	base := strings.TrimRight(trimmed, "/")
	if strings.HasSuffix(base, "/graphql") {
		return strings.TrimSuffix(base, "/graphql")
	}
	return base
}
