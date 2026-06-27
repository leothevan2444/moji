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

type ConnectionConfig struct {
	Stash       StashConfig       `yaml:"stash"`
	Jackett     JackettConfig     `yaml:"jackett"`
	QBittorrent QBittorrentConfig `yaml:"qbittorrent"`
}

type TaskDeletePolicy string

const (
	TaskDeletePolicyKeepOnly              TaskDeletePolicy = "KEEP_ONLY"
	TaskDeletePolicyRemoveTorrent         TaskDeletePolicy = "REMOVE_TORRENT"
	TaskDeletePolicyRemoveTorrentAndFiles TaskDeletePolicy = "REMOVE_TORRENT_AND_FILES"
)

func NormalizeTaskDeletePolicy(value string) TaskDeletePolicy {
	switch TaskDeletePolicy(strings.TrimSpace(value)) {
	case TaskDeletePolicyRemoveTorrent:
		return TaskDeletePolicyRemoveTorrent
	case TaskDeletePolicyRemoveTorrentAndFiles:
		return TaskDeletePolicyRemoveTorrentAndFiles
	default:
		return TaskDeletePolicyKeepOnly
	}
}

type SystemConfig struct {
	TaskDeletePolicy TaskDeletePolicy `yaml:"task_delete_policy"`
}

func (s SystemConfig) EffectiveTaskDeletePolicy() TaskDeletePolicy {
	return NormalizeTaskDeletePolicy(string(s.TaskDeletePolicy))
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
	SubscriptionPollIntervalHours   int `yaml:"subscription_poll_interval_hours"`
}

type IngestConfig struct {
	DeliveryMode     string               `yaml:"delivery_mode"`
	StashLibraryPath string               `yaml:"stash_library_path"`
	Transfer         TransferIngestConfig `yaml:"transfer"`
}

type TransferIngestConfig struct {
	Action         string `yaml:"action"`
	MojiSourceRoot string `yaml:"moji_source_root"`
	MojiTargetRoot string `yaml:"moji_target_root"`
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
	Connection   ConnectionConfig   `yaml:"connection"`
	Ingest       IngestConfig       `yaml:"ingest"`
	Automation   AutomationConfig   `yaml:"automation"`
	System       SystemConfig       `yaml:"system"`
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
	config.Connection.Stash.normalize()
	config.System.TaskDeletePolicy = config.System.EffectiveTaskDeletePolicy()
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
