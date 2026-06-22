package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type JackettConfig struct {
	URL      string `yaml:"url"`
	APIKey   string `yaml:"api_key"`
	Password string `yaml:"password"`
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
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

func (s *StashConfig) normalize() {}

func (s StashConfig) GraphQLEndpoint() string {
	return buildStashGraphQLEndpoint(s.URL)
}

type AutomationConfig struct {
	TaskProgressSyncIntervalSeconds int `yaml:"task_progress_sync_interval_seconds"`
	SubscriptionPollIntervalSeconds int `yaml:"subscription_poll_interval_seconds"`
}

type IngestConfig struct {
	Mode          string                    `yaml:"mode"`
	SharedStorage SharedStorageIngestConfig `yaml:"shared_storage"`
	FileTransfer  FileTransferIngestConfig  `yaml:"file_transfer"`
	LibraryScan   LibraryScanIngestConfig   `yaml:"library_scan"`
}

type SharedStorageIngestConfig struct {
	QBittorrentPathPrefix string `yaml:"qbittorrent_path_prefix"`
	StashPathPrefix       string `yaml:"stash_path_prefix"`
}

type FileTransferIngestConfig struct {
	Action     string `yaml:"action"`
	TargetPath string `yaml:"target_path"`
}

type LibraryScanIngestConfig struct {
	LibraryPath string `yaml:"library_path"`
}

type SubscriptionConfig struct {
	StashBoxEndpoints []string `yaml:"selected_stash_box_endpoints"`
}

type LoggingConfig struct {
	Level            string `yaml:"level"`
	FilePath         string `yaml:"file_path"`
	MaxEntries       int    `yaml:"max_entries"`
	MaxFileSizeBytes int64  `yaml:"max_file_size_bytes"`
	MaxFileBackups   int    `yaml:"max_file_backups"`
}

type Config struct {
	// Jackett is the configuration for the Jackett service
	Jackett      JackettConfig      `yaml:"jackett"`
	QBittorrent  QBittorrentConfig  `yaml:"qbittorrent"`
	Stash        StashConfig        `yaml:"stash"`
	Ingest       IngestConfig       `yaml:"ingest"`
	Automation   AutomationConfig   `yaml:"automation"`
	Subscription SubscriptionConfig `yaml:"subscription"`
	Logging      LoggingConfig      `yaml:"logging"`

	path string
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
	config.path = path

	return &config, nil
}

func (c Config) Path() string {
	return c.path
}

func (c Config) EffectiveLogLevel() string {
	level := strings.TrimSpace(c.Logging.Level)
	if level == "" {
		return "info"
	}
	return level
}

func (c Config) EffectiveLogFilePath() string {
	if strings.TrimSpace(c.Logging.FilePath) != "" {
		return strings.TrimSpace(c.Logging.FilePath)
	}
	return defaultLogFilePathForConfig(c.path)
}

func (c Config) EffectiveLogMaxEntries() int {
	if c.Logging.MaxEntries > 0 {
		return c.Logging.MaxEntries
	}
	return 500
}

func (c Config) EffectiveLogMaxFileSizeBytes() int64 {
	if c.Logging.MaxFileSizeBytes > 0 {
		return c.Logging.MaxFileSizeBytes
	}
	return 10 * 1024 * 1024
}

func (c Config) EffectiveLogMaxFileBackups() int {
	if c.Logging.MaxFileBackups > 0 {
		return c.Logging.MaxFileBackups
	}
	return 5
}

func defaultLogFilePathForConfig(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "moji.log"
	}
	dir := filepath.Dir(trimmed)
	if dir == "" || dir == "." {
		return "moji.log"
	}
	return filepath.Join(dir, "moji.log")
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
