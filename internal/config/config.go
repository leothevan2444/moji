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
	Enabled                  bool                          `yaml:"enabled"`
	InspectionCandidateLimit int                           `yaml:"inspection_candidate_limit"`
	FastRuleOrder            []TorrentSelectionRuleType    `yaml:"fast_rule_order"`
	FastRules                FastTorrentSelectionRules     `yaml:"fast_rules"`
	TorrentRules             TorrentInspectionRuleSettings `yaml:"torrent_rules"`
}

type TorrentSelectionRule struct {
	Type                 TorrentSelectionRuleType `yaml:"-"`
	Enabled              bool                     `yaml:"-"`
	IndexerPreference    IndexerPreferenceRuleConfig
	TitleMatch           TitleMatchRuleConfig
	PublishDate          PublishDateRuleConfig
	Seeders              SeedersRuleConfig
	Size                 SizeRuleConfig
	TorrentFileNameMatch TorrentFileNameMatchRuleConfig
}

type FastTorrentSelectionRules struct {
	IndexerPreference IndexerPreferenceRuleSettings `yaml:"indexer_preference"`
	TitleMatch        TitleMatchRuleSettings        `yaml:"title_match"`
	PublishDate       DirectionRuleSettings         `yaml:"publish_date"`
	TitleSimilarity   ToggleRuleSettings            `yaml:"title_similarity"`
	Seeders           DirectionRuleSettings         `yaml:"seeders"`
	Size              DirectionRuleSettings         `yaml:"size"`
}

type TorrentInspectionRuleSettings struct {
	TorrentSingleVideo   ToggleRuleSettings               `yaml:"torrent_single_video"`
	TorrentFileNameMatch TorrentFileNameMatchRuleSettings `yaml:"torrent_file_name_match"`
}

type CandidateSelectionRuleType = TorrentSelectionRuleType
type CandidateSelectionDirection = TorrentSelectionDirection
type CandidateSelectionConfig = TorrentSelectionConfig
type CandidateSelectionRule = TorrentSelectionRule

type IndexerPreferenceRuleConfig struct {
	TrackerIDs []string `yaml:"tracker_ids"`
}

type IndexerPreferenceRuleSettings struct {
	Enabled    bool     `yaml:"enabled"`
	TrackerIDs []string `yaml:"tracker_ids"`
}

type TitleMatchRuleConfig struct {
	Clauses []TitleMatchClause `yaml:"clauses"`
}

type TitleMatchRuleSettings struct {
	Enabled bool               `yaml:"enabled"`
	Clauses []TitleMatchClause `yaml:"clauses"`
}

type PublishDateRuleConfig struct {
	Direction TorrentSelectionDirection `yaml:"direction"`
}

type DirectionRuleSettings struct {
	Enabled   bool                      `yaml:"enabled"`
	Direction TorrentSelectionDirection `yaml:"direction,omitempty"`
}

type ToggleRuleSettings struct {
	Enabled bool `yaml:"enabled"`
}

type SeedersRuleConfig struct {
	Direction TorrentSelectionDirection `yaml:"direction"`
}

type SizeRuleConfig struct {
	Direction TorrentSelectionDirection `yaml:"direction"`
}

type TitleMatchClause struct {
	Pattern     string                `yaml:"pattern"`
	PatternMode TitleMatchPatternMode `yaml:"pattern_mode"`
	Effect      TitleMatchEffect      `yaml:"effect"`
}

type TorrentFileNameMatchRuleConfig struct {
	Clauses []TorrentFileNameMatchClause `yaml:"clauses"`
}

type TorrentFileNameMatchRuleSettings struct {
	Enabled bool                         `yaml:"enabled"`
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
	TaskDeletePolicy  TaskDeletePolicy        `yaml:"task_delete_policy"`
	ImageCache        ImageCacheConfig        `yaml:"image_cache"`
	StashBoxDataCache StashBoxDataCacheConfig `yaml:"stash_box_data_cache"`
}

type StashBoxDataCacheConfig struct {
	TTLHours int `yaml:"ttl_hours"`
}

func (c StashBoxDataCacheConfig) Normalize() StashBoxDataCacheConfig {
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
	TaskProgressSyncIntervalSeconds int                             `yaml:"task_progress_sync_interval_seconds"`
	SubscriptionPollIntervalHours   int                             `yaml:"subscription_poll_interval_hours"`
	StashBoxEndpoints               []string                        `yaml:"selected_stash_box_endpoints"`
	SubscriptionReleasePolicy       SubscriptionReleasePolicyConfig `yaml:"subscription_release_policy"`
	TorrentSelection                TorrentSelectionConfig          `yaml:"torrent_selection"`
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
	MaxGroupPerformerCount int                          `yaml:"max_group_performer_count"`
	ReleaseDateRange       SubscriptionReleaseDateRange `yaml:"release_date_range"`
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
	cfg := TorrentSelectionConfig{
		Enabled:                  true,
		InspectionCandidateLimit: 5,
	}
	cfg.FastRuleOrder = append([]TorrentSelectionRuleType(nil), defaultFastTorrentSelectionRuleTypes()...)
	cfg.FastRules, cfg.TorrentRules = torrentSelectionRulesFromOrdered(defaultTorrentSelectionRules())
	return cfg
}

func DefaultCandidateSelectionConfig() CandidateSelectionConfig {
	return DefaultTorrentSelectionConfig()
}

func DefaultSubscriptionReleasePolicyConfig() SubscriptionReleasePolicyConfig {
	return SubscriptionReleasePolicyConfig{
		SoloBehavior:           SubscriptionReleaseBehaviorDownload,
		GroupBehavior:          SubscriptionReleaseBehaviorReview,
		CompilationBehavior:    SubscriptionReleaseBehaviorBlock,
		MaxGroupPerformerCount: 3,
		ReleaseDateRange:       SubscriptionReleaseDateRangeAll,
	}
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

func NormalizeCandidateSelectionRuleType(value CandidateSelectionRuleType) CandidateSelectionRuleType {
	return NormalizeTorrentSelectionRuleType(value)
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

func (c TorrentSelectionConfig) Effective() TorrentSelectionConfig {
	normalized := TorrentSelectionConfig{
		Enabled:                  c.Enabled,
		InspectionCandidateLimit: NormalizeTorrentInspectionCandidateLimit(c.InspectionCandidateLimit),
		FastRuleOrder:            normalizeFastRuleOrder(c.FastRuleOrder),
		FastRules:                c.FastRules.normalized(),
		TorrentRules:             c.TorrentRules.normalized(),
	}
	if len(normalized.FastRuleOrder) == 0 {
		normalized.FastRuleOrder = append([]TorrentSelectionRuleType(nil), defaultFastTorrentSelectionRuleTypes()...)
	}
	return normalized
}

func (p SubscriptionReleasePolicyConfig) Effective() SubscriptionReleasePolicyConfig {
	return SubscriptionReleasePolicyConfig{
		SoloBehavior:           NormalizeSubscriptionReleaseBehavior(p.SoloBehavior),
		GroupBehavior:          NormalizeSubscriptionReleaseBehavior(p.GroupBehavior),
		CompilationBehavior:    NormalizeSubscriptionReleaseBehavior(p.CompilationBehavior),
		MaxGroupPerformerCount: NormalizeSubscriptionReleaseMaxGroupPerformerCount(p.MaxGroupPerformerCount),
		ReleaseDateRange:       NormalizeSubscriptionReleaseDateRange(p.ReleaseDateRange),
	}
}

func (r TorrentSelectionRule) normalized() TorrentSelectionRule {
	r.Type = NormalizeTorrentSelectionRuleType(r.Type)
	r.IndexerPreference.TrackerIDs = cleanStrings(r.IndexerPreference.TrackerIDs)
	r.PublishDate.Direction = NormalizeTorrentSelectionDirection(r.PublishDate.Direction)
	if r.PublishDate.Direction == "" {
		r.PublishDate.Direction = TorrentSelectionDirectionDesc
	}
	r.Seeders.Direction = NormalizeTorrentSelectionDirection(r.Seeders.Direction)
	if r.Seeders.Direction == "" {
		r.Seeders.Direction = TorrentSelectionDirectionDesc
	}
	r.Size.Direction = NormalizeTorrentSelectionDirection(r.Size.Direction)
	if r.Size.Direction == "" {
		r.Size.Direction = TorrentSelectionDirectionDesc
	}

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

func (c TorrentSelectionConfig) Validate() error {
	seen := make(map[TorrentSelectionRuleType]struct{}, len(c.FastRuleOrder))
	for _, ruleType := range c.FastRuleOrder {
		normalizedType := NormalizeTorrentSelectionRuleType(ruleType)
		if isTorrentInspectionRuleType(normalizedType) {
			return fmt.Errorf("automation.torrent_selection.fast_rule_order contains non-fast rule type %s", normalizedType)
		}
		if _, exists := seen[normalizedType]; exists {
			return fmt.Errorf("automation.torrent_selection.fast_rule_order contains duplicate type %s", normalizedType)
		}
		seen[normalizedType] = struct{}{}
	}
	return nil
}

func (c TorrentSelectionConfig) rulesByType() map[TorrentSelectionRuleType]TorrentSelectionRule {
	out := c.FastRules.toMap()
	for ruleType, rule := range c.TorrentRules.toMap() {
		out[ruleType] = rule
	}
	return out
}

func (c TorrentSelectionConfig) OrderedRules() []TorrentSelectionRule {
	effective := c.Effective()
	ruleMap := effective.rulesByType()
	out := make([]TorrentSelectionRule, 0, len(defaultTorrentSelectionRules()))
	for _, ruleType := range effective.FastRuleOrder {
		out = append(out, ruleMap[ruleType])
	}
	for _, ruleType := range defaultTorrentInspectionRuleTypes() {
		out = append(out, ruleMap[ruleType])
	}
	return out
}

func NewTorrentSelectionConfig(enabled bool, inspectionCandidateLimit int, orderedRules []TorrentSelectionRule) TorrentSelectionConfig {
	cfg := DefaultTorrentSelectionConfig()
	cfg.Enabled = enabled
	cfg.InspectionCandidateLimit = inspectionCandidateLimit
	cfg.FastRuleOrder = make([]TorrentSelectionRuleType, 0, len(defaultFastTorrentSelectionRuleTypes()))
	for _, rule := range orderedRules {
		normalizedType := NormalizeTorrentSelectionRuleType(rule.Type)
		if isTorrentInspectionRuleType(normalizedType) {
			continue
		}
		cfg.FastRuleOrder = append(cfg.FastRuleOrder, normalizedType)
	}
	cfg.FastRules, cfg.TorrentRules = torrentSelectionRulesFromOrdered(orderedRules)
	return cfg
}

func defaultTorrentSelectionRules() []TorrentSelectionRule {
	ruleTypes := append(defaultFastTorrentSelectionRuleTypes(), defaultTorrentInspectionRuleTypes()...)
	rules := make([]TorrentSelectionRule, 0, len(ruleTypes))
	for _, ruleType := range ruleTypes {
		rules = append(rules, defaultTorrentSelectionRule(ruleType))
	}
	return rules
}

func defaultFastTorrentSelectionRuleTypes() []TorrentSelectionRuleType {
	return []TorrentSelectionRuleType{
		TorrentSelectionRuleTypeIndexerPreference,
		TorrentSelectionRuleTypeTitleMatch,
		TorrentSelectionRuleTypePublishDate,
		TorrentSelectionRuleTypeTitleSimilarity,
		TorrentSelectionRuleTypeSeeders,
		TorrentSelectionRuleTypeSize,
	}
}

func defaultTorrentInspectionRuleTypes() []TorrentSelectionRuleType {
	return []TorrentSelectionRuleType{
		TorrentSelectionRuleTypeTorrentSingleVideo,
		TorrentSelectionRuleTypeTorrentFileNameMatch,
	}
}

func isTorrentInspectionRuleType(ruleType TorrentSelectionRuleType) bool {
	switch ruleType {
	case TorrentSelectionRuleTypeTorrentSingleVideo, TorrentSelectionRuleTypeTorrentFileNameMatch:
		return true
	default:
		return false
	}
}

func defaultTorrentSelectionRule(ruleType TorrentSelectionRuleType) TorrentSelectionRule {
	rule := TorrentSelectionRule{
		Type:    ruleType,
		Enabled: true,
		PublishDate: PublishDateRuleConfig{
			Direction: TorrentSelectionDirectionDesc,
		},
		Seeders: SeedersRuleConfig{
			Direction: TorrentSelectionDirectionDesc,
		},
		Size: SizeRuleConfig{
			Direction: TorrentSelectionDirectionDesc,
		},
	}
	return rule
}

func missingDefaultRuleTypes(current []TorrentSelectionRuleType, defaults []TorrentSelectionRuleType) []TorrentSelectionRuleType {
	seen := make(map[TorrentSelectionRuleType]struct{}, len(current))
	for _, ruleType := range current {
		seen[ruleType] = struct{}{}
	}
	missing := make([]TorrentSelectionRuleType, 0, len(defaults))
	for _, ruleType := range defaults {
		if _, exists := seen[ruleType]; exists {
			continue
		}
		missing = append(missing, ruleType)
	}
	return missing
}

func chooseDefaultTorrentSelectionRule(ruleType TorrentSelectionRuleType, seen map[TorrentSelectionRuleType]TorrentSelectionRule) TorrentSelectionRule {
	if rule, exists := seen[ruleType]; exists {
		return rule
	}
	return defaultTorrentSelectionRule(ruleType)
}

func normalizeFastRuleOrder(order []TorrentSelectionRuleType) []TorrentSelectionRuleType {
	seen := make(map[TorrentSelectionRuleType]struct{}, len(order))
	out := make([]TorrentSelectionRuleType, 0, len(defaultFastTorrentSelectionRuleTypes()))
	for _, ruleType := range order {
		normalized := NormalizeTorrentSelectionRuleType(ruleType)
		if isTorrentInspectionRuleType(normalized) {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return append(out, missingDefaultRuleTypes(out, defaultFastTorrentSelectionRuleTypes())...)
}

func (r FastTorrentSelectionRules) normalized() FastTorrentSelectionRules {
	out := FastTorrentSelectionRules{}
	for _, ruleType := range defaultFastTorrentSelectionRuleTypes() {
		out.setRule(defaultTorrentSelectionRule(ruleType).merge(r.ruleForType(ruleType)).normalized())
	}
	return out
}

func (r FastTorrentSelectionRules) toMap() map[TorrentSelectionRuleType]TorrentSelectionRule {
	return map[TorrentSelectionRuleType]TorrentSelectionRule{
		TorrentSelectionRuleTypeIndexerPreference: r.ruleForType(TorrentSelectionRuleTypeIndexerPreference),
		TorrentSelectionRuleTypeTitleMatch:        r.ruleForType(TorrentSelectionRuleTypeTitleMatch),
		TorrentSelectionRuleTypePublishDate:       r.ruleForType(TorrentSelectionRuleTypePublishDate),
		TorrentSelectionRuleTypeTitleSimilarity:   r.ruleForType(TorrentSelectionRuleTypeTitleSimilarity),
		TorrentSelectionRuleTypeSeeders:           r.ruleForType(TorrentSelectionRuleTypeSeeders),
		TorrentSelectionRuleTypeSize:              r.ruleForType(TorrentSelectionRuleTypeSize),
	}
}

func (r *FastTorrentSelectionRules) setRule(rule TorrentSelectionRule) {
	switch rule.Type {
	case TorrentSelectionRuleTypeIndexerPreference:
		r.IndexerPreference = IndexerPreferenceRuleSettings{
			Enabled:    rule.Enabled,
			TrackerIDs: append([]string(nil), rule.IndexerPreference.TrackerIDs...),
		}
	case TorrentSelectionRuleTypeTitleMatch:
		r.TitleMatch = TitleMatchRuleSettings{
			Enabled: rule.Enabled,
			Clauses: append([]TitleMatchClause(nil), rule.TitleMatch.Clauses...),
		}
	case TorrentSelectionRuleTypePublishDate:
		r.PublishDate = DirectionRuleSettings{
			Enabled:   rule.Enabled,
			Direction: rule.PublishDate.Direction,
		}
	case TorrentSelectionRuleTypeTitleSimilarity:
		r.TitleSimilarity = ToggleRuleSettings{Enabled: rule.Enabled}
	case TorrentSelectionRuleTypeSeeders:
		r.Seeders = DirectionRuleSettings{
			Enabled:   rule.Enabled,
			Direction: rule.Seeders.Direction,
		}
	case TorrentSelectionRuleTypeSize:
		r.Size = DirectionRuleSettings{
			Enabled:   rule.Enabled,
			Direction: rule.Size.Direction,
		}
	}
}

func (r FastTorrentSelectionRules) ruleForType(ruleType TorrentSelectionRuleType) TorrentSelectionRule {
	switch ruleType {
	case TorrentSelectionRuleTypeIndexerPreference:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.IndexerPreference.Enabled,
			IndexerPreference: IndexerPreferenceRuleConfig{
				TrackerIDs: append([]string(nil), r.IndexerPreference.TrackerIDs...),
			},
		}
	case TorrentSelectionRuleTypeTitleMatch:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.TitleMatch.Enabled,
			TitleMatch: TitleMatchRuleConfig{
				Clauses: append([]TitleMatchClause(nil), r.TitleMatch.Clauses...),
			},
		}
	case TorrentSelectionRuleTypePublishDate:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.PublishDate.Enabled,
			PublishDate: PublishDateRuleConfig{
				Direction: r.PublishDate.Direction,
			},
		}
	case TorrentSelectionRuleTypeTitleSimilarity:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.TitleSimilarity.Enabled,
		}
	case TorrentSelectionRuleTypeSeeders:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.Seeders.Enabled,
			Seeders: SeedersRuleConfig{
				Direction: r.Seeders.Direction,
			},
		}
	case TorrentSelectionRuleTypeSize:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.Size.Enabled,
			Size: SizeRuleConfig{
				Direction: r.Size.Direction,
			},
		}
	default:
		return TorrentSelectionRule{}
	}
}

func (r TorrentInspectionRuleSettings) normalized() TorrentInspectionRuleSettings {
	out := TorrentInspectionRuleSettings{}
	for _, ruleType := range defaultTorrentInspectionRuleTypes() {
		out.setRule(defaultTorrentSelectionRule(ruleType).merge(r.ruleForType(ruleType)).normalized())
	}
	return out
}

func (r TorrentInspectionRuleSettings) toMap() map[TorrentSelectionRuleType]TorrentSelectionRule {
	return map[TorrentSelectionRuleType]TorrentSelectionRule{
		TorrentSelectionRuleTypeTorrentSingleVideo:   r.ruleForType(TorrentSelectionRuleTypeTorrentSingleVideo),
		TorrentSelectionRuleTypeTorrentFileNameMatch: r.ruleForType(TorrentSelectionRuleTypeTorrentFileNameMatch),
	}
}

func (r *TorrentInspectionRuleSettings) setRule(rule TorrentSelectionRule) {
	switch rule.Type {
	case TorrentSelectionRuleTypeTorrentSingleVideo:
		r.TorrentSingleVideo = ToggleRuleSettings{Enabled: rule.Enabled}
	case TorrentSelectionRuleTypeTorrentFileNameMatch:
		r.TorrentFileNameMatch = TorrentFileNameMatchRuleSettings{
			Enabled: rule.Enabled,
			Clauses: append([]TorrentFileNameMatchClause(nil), rule.TorrentFileNameMatch.Clauses...),
		}
	}
}

func (r TorrentInspectionRuleSettings) ruleForType(ruleType TorrentSelectionRuleType) TorrentSelectionRule {
	switch ruleType {
	case TorrentSelectionRuleTypeTorrentSingleVideo:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.TorrentSingleVideo.Enabled,
		}
	case TorrentSelectionRuleTypeTorrentFileNameMatch:
		return TorrentSelectionRule{
			Type:    ruleType,
			Enabled: r.TorrentFileNameMatch.Enabled,
			TorrentFileNameMatch: TorrentFileNameMatchRuleConfig{
				Clauses: append([]TorrentFileNameMatchClause(nil), r.TorrentFileNameMatch.Clauses...),
			},
		}
	default:
		return TorrentSelectionRule{}
	}
}

func torrentSelectionRulesFromOrdered(rules []TorrentSelectionRule) (FastTorrentSelectionRules, TorrentInspectionRuleSettings) {
	fastRules := FastTorrentSelectionRules{}
	torrentRules := TorrentInspectionRuleSettings{}
	for _, rule := range rules {
		rule.Type = NormalizeTorrentSelectionRuleType(rule.Type)
		if isTorrentInspectionRuleType(rule.Type) {
			torrentRules.setRule(rule)
			continue
		}
		fastRules.setRule(rule)
	}
	return fastRules, torrentRules
}

func (r TorrentSelectionRule) merge(other TorrentSelectionRule) TorrentSelectionRule {
	r.Enabled = other.Enabled
	if len(other.IndexerPreference.TrackerIDs) > 0 {
		r.IndexerPreference = other.IndexerPreference
	}
	if len(other.TitleMatch.Clauses) > 0 {
		r.TitleMatch = other.TitleMatch
	}
	if other.PublishDate.Direction != "" {
		r.PublishDate = other.PublishDate
	}
	if other.Seeders.Direction != "" {
		r.Seeders = other.Seeders
	}
	if other.Size.Direction != "" {
		r.Size = other.Size
	}
	if len(other.TorrentFileNameMatch.Clauses) > 0 {
		r.TorrentFileNameMatch = other.TorrentFileNameMatch
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
