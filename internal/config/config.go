package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/apptime"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"gopkg.in/yaml.v3"
)

var ErrInvalidSettings = errors.New("invalid settings")

type JackettConfig struct {
	URL      string `yaml:"url"`
	APIKey   string `yaml:"api_key"`
	Password string `yaml:"password"`
}

type TorrentSelectionConfig = resourceselection.Policy
type CandidateSelectionConfig = resourceselection.Policy

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
	Timezone          string                  `yaml:"timezone,omitempty"`
	Language          string                  `yaml:"language"`
	DeveloperMode     bool                    `yaml:"developer_mode"`
	TaskDeletePolicy  TaskDeletePolicy        `yaml:"task_delete_policy"`
	ImageCache        ImageCacheConfig        `yaml:"image_cache"`
	StashBoxDataCache StashBoxDataCacheConfig `yaml:"stash_box_data_cache"`
}

type StashBoxDataCacheConfig struct {
	MaxSizeMB int `yaml:"max_size_mb"`
	TTLHours  int `yaml:"ttl_hours"`
}

func (c StashBoxDataCacheConfig) Normalize() StashBoxDataCacheConfig {
	if c.MaxSizeMB <= 0 {
		c.MaxSizeMB = 512
	}
	if c.MaxSizeMB > 65536 {
		c.MaxSizeMB = 65536
	}
	if c.TTLHours == 0 {
		c.TTLHours = 24
	}
	if c.TTLHours < 12 {
		c.TTLHours = 12
	}
	if c.TTLHours > 360 {
		c.TTLHours = 360
	}
	return c
}

type ImageCacheConfig struct {
	Enabled       *bool `yaml:"enabled,omitempty"`
	MaxSizeMB     int   `yaml:"max_size_mb"`
	RetentionDays int   `yaml:"retention_days"`
}

func DefaultImageCacheConfig() ImageCacheConfig {
	enabled := true
	return ImageCacheConfig{Enabled: &enabled, MaxSizeMB: 1024, RetentionDays: 30}
}
func (c ImageCacheConfig) EffectiveEnabled() bool { return c.Enabled == nil || *c.Enabled }
func (c ImageCacheConfig) Normalize() ImageCacheConfig {
	if c.Enabled == nil {
		v := true
		c.Enabled = &v
	}
	if c.MaxSizeMB == 0 {
		c.MaxSizeMB = 1024
	}
	if c.MaxSizeMB < 64 {
		c.MaxSizeMB = 64
	}
	if c.MaxSizeMB > 20480 {
		c.MaxSizeMB = 20480
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = 30
	}
	if c.RetentionDays < 1 {
		c.RetentionDays = 1
	}
	if c.RetentionDays > 365 {
		c.RetentionDays = 365
	}
	return c
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
	SourcingRetryIntervalHours int                             `yaml:"sourcing_retry_interval_hours"`
	TaskProcessingFrequency    string                          `yaml:"task_processing_frequency"`
	StashBoxEndpoints          []string                        `yaml:"selected_stash_box_endpoints"`
	SubscriptionReleasePolicy  SubscriptionReleasePolicyConfig `yaml:"subscription_release_policy"`
	TorrentSelection           TorrentSelectionConfig          `yaml:"torrent_selection"`
}

// EffectiveTaskProcessingFrequency defaults to the balanced tier.
func (a AutomationConfig) EffectiveTaskProcessingFrequency() string {
	if a.TaskProcessingFrequency == "" {
		return "MEDIUM"
	}
	return a.TaskProcessingFrequency
}

func (a AutomationConfig) TaskProcessingInterval() time.Duration {
	switch a.EffectiveTaskProcessingFrequency() {
	case "HIGH":
		return 30 * time.Second
	case "LOW":
		return 5 * time.Minute
	default:
		return time.Minute
	}
}

func (a AutomationConfig) EffectiveSourcingRetryIntervalHours() int {
	if a.SourcingRetryIntervalHours <= 0 {
		return 6
	}
	if a.SourcingRetryIntervalHours > 168 {
		return 168
	}
	return a.SourcingRetryIntervalHours
}

type SubscriptionReleaseBehavior string

const (
	SubscriptionReleaseBehaviorDownload SubscriptionReleaseBehavior = "DOWNLOAD"
	SubscriptionReleaseBehaviorReview   SubscriptionReleaseBehavior = "REVIEW"
	SubscriptionReleaseBehaviorBlock    SubscriptionReleaseBehavior = "BLOCK"
)

type SubscriptionReleaseDateRange string

const (
	SubscriptionReleaseDateRangeAll        SubscriptionReleaseDateRange = "ALL"
	SubscriptionReleaseDateRangeOneYear    SubscriptionReleaseDateRange = "ONE_YEAR"
	SubscriptionReleaseDateRangeTwoYears   SubscriptionReleaseDateRange = "TWO_YEARS"
	SubscriptionReleaseDateRangeThreeYears SubscriptionReleaseDateRange = "THREE_YEARS"
	SubscriptionReleaseDateRangeFiveYears  SubscriptionReleaseDateRange = "FIVE_YEARS"
)

type SubscriptionReleasePolicyConfig struct {
	SoloBehavior           SubscriptionReleaseBehavior  `yaml:"solo_behavior"`
	GroupBehavior          SubscriptionReleaseBehavior  `yaml:"group_behavior"`
	CompilationBehavior    SubscriptionReleaseBehavior  `yaml:"compilation_behavior"`
	VRBehavior             SubscriptionReleaseBehavior  `yaml:"vr_behavior"`
	MaxGroupPerformerCount int                          `yaml:"max_group_performer_count"`
	ReleaseDateRange       SubscriptionReleaseDateRange `yaml:"release_date_range"`
}

type IngestConfig struct {
	GenerateTranscodes bool                  `yaml:"generate_transcodes"`
	KeepSubtitles      bool                  `yaml:"keep_subtitles"`
	Downloads          DownloadsIngestConfig `yaml:"downloads"`
	Library            LibraryIngestConfig   `yaml:"library"`
	Transfer           TransferIngestConfig  `yaml:"transfer"`
}

type DownloadsIngestConfig struct {
	QBRoot   string `yaml:"qb_root"`
	MojiRoot string `yaml:"moji_root"`
}

type LibraryIngestConfig struct {
	MojiRoot  string `yaml:"moji_root"`
	StashRoot string `yaml:"stash_root"`
}

type TransferIngestConfig struct {
	Action string `yaml:"action"`
}

type LoggingConfig struct {
	Level            string `yaml:"level"`
	MaxEntries       int    `yaml:"max_entries"`
	MaxFileSizeBytes int64  `yaml:"max_file_size_bytes"`
	MaxFileBackups   int    `yaml:"max_file_backups"`
}

type Config struct {
	Downloader DownloaderConfig `yaml:"downloader"`
	Indexers   IndexersConfig   `yaml:"indexers"`
	Connection ConnectionConfig `yaml:"connection"`
	Ingest     IngestConfig     `yaml:"ingest"`
	Automation AutomationConfig `yaml:"automation"`
	System     SystemConfig     `yaml:"system"`
	Logging    LoggingConfig    `yaml:"logging"`

	path string
}

func DefaultTorrentSelectionConfig() TorrentSelectionConfig { return resourceselection.Default() }
func DefaultCandidateSelectionConfig() CandidateSelectionConfig {
	return DefaultTorrentSelectionConfig()
}

func DefaultSubscriptionReleasePolicyConfig() SubscriptionReleasePolicyConfig {
	return SubscriptionReleasePolicyConfig{
		SoloBehavior:           SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          SubscriptionReleaseBehaviorReview,
		CompilationBehavior:    SubscriptionReleaseBehaviorBlock,
		VRBehavior:             SubscriptionReleaseBehaviorReview,
		MaxGroupPerformerCount: 3,
		ReleaseDateRange:       SubscriptionReleaseDateRangeAll,
	}
}

func NormalizeSubscriptionReleaseBehavior(value SubscriptionReleaseBehavior) SubscriptionReleaseBehavior {
	switch SubscriptionReleaseBehavior(strings.TrimSpace(string(value))) {
	case SubscriptionReleaseBehaviorDownload,
		SubscriptionReleaseBehaviorReview,
		SubscriptionReleaseBehaviorBlock:
		return value
	default:
		return SubscriptionReleaseBehaviorReview
	}
}

func NormalizeSubscriptionReleaseDateRange(value SubscriptionReleaseDateRange) SubscriptionReleaseDateRange {
	switch SubscriptionReleaseDateRange(strings.TrimSpace(string(value))) {
	case SubscriptionReleaseDateRangeOneYear,
		SubscriptionReleaseDateRangeTwoYears,
		SubscriptionReleaseDateRangeThreeYears,
		SubscriptionReleaseDateRangeFiveYears:
		return value
	default:
		return SubscriptionReleaseDateRangeAll
	}
}

func NormalizeSubscriptionReleaseMaxGroupPerformerCount(value int) int {
	if value <= 0 {
		return DefaultSubscriptionReleasePolicyConfig().MaxGroupPerformerCount
	}
	return value
}

func (p SubscriptionReleasePolicyConfig) Effective() SubscriptionReleasePolicyConfig {
	return SubscriptionReleasePolicyConfig{
		SoloBehavior:           NormalizeSubscriptionReleaseBehavior(p.SoloBehavior),
		GroupBehavior:          NormalizeSubscriptionReleaseBehavior(p.GroupBehavior),
		CompilationBehavior:    NormalizeSubscriptionReleaseBehavior(p.CompilationBehavior),
		VRBehavior:             NormalizeSubscriptionReleaseBehavior(p.VRBehavior),
		MaxGroupPerformerCount: NormalizeSubscriptionReleaseMaxGroupPerformerCount(p.MaxGroupPerformerCount),
		ReleaseDateRange:       NormalizeSubscriptionReleaseDateRange(p.ReleaseDateRange),
	}
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
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
	if action := config.Ingest.Transfer.Action; action != "" && action != "COPY" && action != "MOVE" {
		return nil, fmt.Errorf("ingest transfer action must be COPY or MOVE")
	}

	config.Connection.Stash.normalize()
	config.System.TaskDeletePolicy = config.System.EffectiveTaskDeletePolicy()
	config.System.ImageCache = config.System.ImageCache.Normalize()
	config.System.StashBoxDataCache = config.System.StashBoxDataCache.Normalize()
	config.Automation.StashBoxEndpoints = cleanStrings(config.Automation.StashBoxEndpoints)
	config.Automation.SubscriptionReleasePolicy = config.Automation.SubscriptionReleasePolicy.Effective()
	if err := config.Automation.TorrentSelection.Validate(); err != nil {
		return nil, err
	}
	config.Automation.TorrentSelection = config.Automation.TorrentSelection.Effective()
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
	return logging.DefaultLogFilePath(c.path)
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

func (s SystemConfig) EffectiveLanguage() string {
	if s.Language == "" {
		return "auto"
	}
	return s.Language
}
func ValidateLanguage(language string) error {
	switch language {
	case "", "auto", "zh-CN", "en":
		return nil
	default:
		return fmt.Errorf("%w: unsupported interface language", ErrInvalidSettings)
	}
}

// EffectiveTimezone resolves the shared application timezone independently of the browser.
func (s SystemConfig) EffectiveTimezone() string {
	return apptime.EffectiveTimezone(s.Timezone)
}
func (s SystemConfig) TimezoneMode() string {
	if strings.TrimSpace(s.Timezone) == "" {
		return "auto"
	}
	return strings.TrimSpace(s.Timezone)
}
