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

type TorrentSelectionRuleType string

const (
	TorrentSelectionRuleTypeIndexerPreference    TorrentSelectionRuleType = "INDEXER_PREFERENCE"
	TorrentSelectionRuleTypeTitleMatch           TorrentSelectionRuleType = "TITLE_MATCH"
	TorrentSelectionRuleTypePublishDate          TorrentSelectionRuleType = "PUBLISH_DATE"
	TorrentSelectionRuleTypeTitleSimilarity      TorrentSelectionRuleType = "TITLE_SIMILARITY"
	TorrentSelectionRuleTypeSeeders              TorrentSelectionRuleType = "SEEDERS"
	TorrentSelectionRuleTypeSize                 TorrentSelectionRuleType = "SIZE"
	TorrentSelectionRuleTypeTorrentSingleVideo   TorrentSelectionRuleType = "TORRENT_SINGLE_VIDEO"
	TorrentSelectionRuleTypeTorrentFileNameMatch TorrentSelectionRuleType = "TORRENT_FILE_NAME_MATCH"
)

const (
	CandidateSelectionRuleTypeIndexerPreference    = TorrentSelectionRuleTypeIndexerPreference
	CandidateSelectionRuleTypeTitleMatch           = TorrentSelectionRuleTypeTitleMatch
	CandidateSelectionRuleTypePublishDate          = TorrentSelectionRuleTypePublishDate
	CandidateSelectionRuleTypeTitleSimilarity      = TorrentSelectionRuleTypeTitleSimilarity
	CandidateSelectionRuleTypeSeeders              = TorrentSelectionRuleTypeSeeders
	CandidateSelectionRuleTypeSize                 = TorrentSelectionRuleTypeSize
	CandidateSelectionRuleTypeTorrentSingleVideo   = TorrentSelectionRuleTypeTorrentSingleVideo
	CandidateSelectionRuleTypeTorrentFileNameMatch = TorrentSelectionRuleTypeTorrentFileNameMatch
)

type TorrentSelectionDirection string

const (
	TorrentSelectionDirectionAsc  TorrentSelectionDirection = "ASC"
	TorrentSelectionDirectionDesc TorrentSelectionDirection = "DESC"
)

const (
	CandidateSelectionDirectionAsc  = TorrentSelectionDirectionAsc
	CandidateSelectionDirectionDesc = TorrentSelectionDirectionDesc
)

type TitleMatchPatternMode string

const (
	TitleMatchPatternModePlain TitleMatchPatternMode = "PLAIN"
	TitleMatchPatternModeRegex TitleMatchPatternMode = "REGEX"
)

type TitleMatchEffect string

const (
	TitleMatchEffectPrefer TitleMatchEffect = "PREFER"
	TitleMatchEffectAvoid  TitleMatchEffect = "AVOID"
)

type TorrentFileMatchEffect string

const (
	TorrentFileMatchEffectPrefer TorrentFileMatchEffect = "PREFER"
	TorrentFileMatchEffectAvoid  TorrentFileMatchEffect = "AVOID"
	TorrentFileMatchEffectLock   TorrentFileMatchEffect = "LOCK"
)

type TorrentSelectionConfig struct {
	Enabled                  bool                   `yaml:"enabled"`
	InspectionCandidateLimit int                    `yaml:"inspection_candidate_limit"`
	Rules                    []TorrentSelectionRule `yaml:"rules"`
}

type TorrentSelectionRule struct {
	ID                   string                         `yaml:"id"`
	Name                 string                         `yaml:"name"`
	Type                 TorrentSelectionRuleType       `yaml:"type"`
	Enabled              bool                           `yaml:"enabled"`
	Direction            TorrentSelectionDirection      `yaml:"direction"`
	IndexerPreference    IndexerPreferenceRuleConfig    `yaml:"indexer_preference"`
	TitleMatch           TitleMatchRuleConfig           `yaml:"title_match"`
	TorrentFileNameMatch TorrentFileNameMatchRuleConfig `yaml:"torrent_file_name_match"`
}

type CandidateSelectionRuleType = TorrentSelectionRuleType
type CandidateSelectionDirection = TorrentSelectionDirection
type CandidateSelectionConfig = TorrentSelectionConfig
type CandidateSelectionRule = TorrentSelectionRule

type IndexerPreferenceRuleConfig struct {
	TrackerIDs []string `yaml:"tracker_ids"`
}

type TitleMatchRuleConfig struct {
	Clauses []TitleMatchClause `yaml:"clauses"`
}

type TitleMatchClause struct {
	Pattern     string                `yaml:"pattern"`
	PatternMode TitleMatchPatternMode `yaml:"pattern_mode"`
	Effect      TitleMatchEffect      `yaml:"effect"`
}

type TorrentFileNameMatchRuleConfig struct {
	Clauses []TorrentFileNameMatchClause `yaml:"clauses"`
}

type TorrentFileNameMatchClause struct {
	Pattern     string                 `yaml:"pattern"`
	PatternMode TitleMatchPatternMode  `yaml:"pattern_mode"`
	Effect      TorrentFileMatchEffect `yaml:"effect"`
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
	TaskProgressSyncIntervalSeconds int                    `yaml:"task_progress_sync_interval_seconds"`
	SubscriptionPollIntervalHours   int                    `yaml:"subscription_poll_interval_hours"`
	StashBoxEndpoints               []string               `yaml:"selected_stash_box_endpoints"`
	TorrentSelection                TorrentSelectionConfig `yaml:"torrent_selection"`
}

type IngestConfig struct {
	DeliveryMode string                `yaml:"delivery_mode"`
	Downloads    DownloadsIngestConfig `yaml:"downloads"`
	Library      LibraryIngestConfig   `yaml:"library"`
	Transfer     TransferIngestConfig  `yaml:"transfer"`
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
	FilePath         string `yaml:"file_path"`
	MaxEntries       int    `yaml:"max_entries"`
	MaxFileSizeBytes int64  `yaml:"max_file_size_bytes"`
	MaxFileBackups   int    `yaml:"max_file_backups"`
}

type Config struct {
	Connection ConnectionConfig `yaml:"connection"`
	Ingest     IngestConfig     `yaml:"ingest"`
	Automation AutomationConfig `yaml:"automation"`
	System     SystemConfig     `yaml:"system"`
	Logging    LoggingConfig    `yaml:"logging"`

	path string
}

func DefaultTorrentSelectionConfig() TorrentSelectionConfig {
	return TorrentSelectionConfig{
		Enabled:                  true,
		InspectionCandidateLimit: 5,
		Rules: []TorrentSelectionRule{
			{
				ID:        "default-seeders",
				Name:      "Seeders",
				Type:      TorrentSelectionRuleTypeSeeders,
				Enabled:   true,
				Direction: TorrentSelectionDirectionDesc,
			},
			{
				ID:        "default-size",
				Name:      "Size",
				Type:      TorrentSelectionRuleTypeSize,
				Enabled:   true,
				Direction: TorrentSelectionDirectionDesc,
			},
		},
	}
}

func DefaultCandidateSelectionConfig() CandidateSelectionConfig {
	return DefaultTorrentSelectionConfig()
}

func NormalizeTorrentInspectionCandidateLimit(value int) int {
	if value <= 0 {
		return DefaultTorrentSelectionConfig().InspectionCandidateLimit
	}
	return value
}

func NormalizeTorrentSelectionDirection(value TorrentSelectionDirection) TorrentSelectionDirection {
	switch TorrentSelectionDirection(strings.TrimSpace(string(value))) {
	case TorrentSelectionDirectionAsc:
		return TorrentSelectionDirectionAsc
	default:
		return TorrentSelectionDirectionDesc
	}
}

func NormalizeCandidateSelectionDirection(value CandidateSelectionDirection) CandidateSelectionDirection {
	return NormalizeTorrentSelectionDirection(value)
}

func NormalizeTitleMatchPatternMode(value TitleMatchPatternMode) TitleMatchPatternMode {
	switch TitleMatchPatternMode(strings.TrimSpace(string(value))) {
	case TitleMatchPatternModeRegex:
		return TitleMatchPatternModeRegex
	default:
		return TitleMatchPatternModePlain
	}
}

func NormalizeTitleMatchEffect(value TitleMatchEffect) TitleMatchEffect {
	switch TitleMatchEffect(strings.TrimSpace(string(value))) {
	case TitleMatchEffectAvoid:
		return TitleMatchEffectAvoid
	default:
		return TitleMatchEffectPrefer
	}
}

func NormalizeTorrentFileMatchEffect(value TorrentFileMatchEffect) TorrentFileMatchEffect {
	switch TorrentFileMatchEffect(strings.TrimSpace(string(value))) {
	case TorrentFileMatchEffectAvoid:
		return TorrentFileMatchEffectAvoid
	case TorrentFileMatchEffectLock:
		return TorrentFileMatchEffectLock
	default:
		return TorrentFileMatchEffectPrefer
	}
}

func NormalizeTorrentSelectionRuleType(value TorrentSelectionRuleType) TorrentSelectionRuleType {
	switch TorrentSelectionRuleType(strings.TrimSpace(string(value))) {
	case TorrentSelectionRuleTypeIndexerPreference,
		TorrentSelectionRuleTypeTitleMatch,
		TorrentSelectionRuleTypePublishDate,
		TorrentSelectionRuleTypeTitleSimilarity,
		TorrentSelectionRuleTypeSeeders,
		TorrentSelectionRuleTypeSize,
		TorrentSelectionRuleTypeTorrentSingleVideo,
		TorrentSelectionRuleTypeTorrentFileNameMatch:
		return value
	default:
		return TorrentSelectionRuleTypeSeeders
	}
}

func DefaultTorrentSelectionRuleName(ruleType TorrentSelectionRuleType, index int) string {
	switch NormalizeTorrentSelectionRuleType(ruleType) {
	case TorrentSelectionRuleTypeIndexerPreference:
		return "Indexer Preference"
	case TorrentSelectionRuleTypeTitleMatch:
		return "Title Match"
	case TorrentSelectionRuleTypePublishDate:
		return "Publish Date"
	case TorrentSelectionRuleTypeTitleSimilarity:
		return "Title Similarity"
	case TorrentSelectionRuleTypeSeeders:
		return "Seeders"
	case TorrentSelectionRuleTypeSize:
		return "Size"
	case TorrentSelectionRuleTypeTorrentSingleVideo:
		return "Single Video"
	case TorrentSelectionRuleTypeTorrentFileNameMatch:
		return "Torrent File Name Match"
	default:
		return fmt.Sprintf("Rule %d", index+1)
	}
}

func NormalizeCandidateSelectionRuleType(value CandidateSelectionRuleType) CandidateSelectionRuleType {
	return NormalizeTorrentSelectionRuleType(value)
}

func (c TorrentSelectionConfig) Effective() TorrentSelectionConfig {
	if len(c.Rules) == 0 {
		return DefaultTorrentSelectionConfig()
	}

	normalized := TorrentSelectionConfig{
		Enabled:                  c.Enabled,
		InspectionCandidateLimit: NormalizeTorrentInspectionCandidateLimit(c.InspectionCandidateLimit),
		Rules:                    make([]TorrentSelectionRule, 0, len(c.Rules)),
	}
	for i, rule := range c.Rules {
		next := rule.normalized(i)
		if next.Type == "" {
			continue
		}
		normalized.Rules = append(normalized.Rules, next)
	}
	if len(normalized.Rules) == 0 {
		return DefaultTorrentSelectionConfig()
	}
	return normalized
}

func (r TorrentSelectionRule) normalized(index int) TorrentSelectionRule {
	r.ID = strings.TrimSpace(r.ID)
	if r.ID == "" {
		r.ID = fmt.Sprintf("rule-%d", index+1)
	}
	r.Type = NormalizeTorrentSelectionRuleType(r.Type)
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		r.Name = DefaultTorrentSelectionRuleName(r.Type, index)
	}
	r.Direction = NormalizeTorrentSelectionDirection(r.Direction)
	r.IndexerPreference.TrackerIDs = cleanStrings(r.IndexerPreference.TrackerIDs)

	if r.Type == TorrentSelectionRuleTypeTitleMatch {
		clauses := make([]TitleMatchClause, 0, len(r.TitleMatch.Clauses))
		for _, clause := range r.TitleMatch.Clauses {
			clause.Pattern = strings.TrimSpace(clause.Pattern)
			if clause.Pattern == "" {
				continue
			}
			clause.PatternMode = NormalizeTitleMatchPatternMode(clause.PatternMode)
			clause.Effect = NormalizeTitleMatchEffect(clause.Effect)
			clauses = append(clauses, clause)
		}
		r.TitleMatch.Clauses = clauses
	}
	if r.Type == TorrentSelectionRuleTypeTorrentFileNameMatch {
		clauses := make([]TorrentFileNameMatchClause, 0, len(r.TorrentFileNameMatch.Clauses))
		for _, clause := range r.TorrentFileNameMatch.Clauses {
			clause.Pattern = strings.TrimSpace(clause.Pattern)
			if clause.Pattern == "" {
				continue
			}
			clause.PatternMode = NormalizeTitleMatchPatternMode(clause.PatternMode)
			clause.Effect = NormalizeTorrentFileMatchEffect(clause.Effect)
			clauses = append(clauses, clause)
		}
		r.TorrentFileNameMatch.Clauses = clauses
	}
	return r
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
	config.Connection.Stash.normalize()
	config.System.TaskDeletePolicy = config.System.EffectiveTaskDeletePolicy()
	config.Automation.StashBoxEndpoints = cleanStrings(config.Automation.StashBoxEndpoints)
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
