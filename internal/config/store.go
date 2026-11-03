package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/leothevan2444/moji/internal/durablefile"
	"gopkg.in/yaml.v3"
)

type Store struct {
	mu      sync.RWMutex
	path    string
	cfg     *Config
	root    yaml.Node
	saved   []byte
	changes map[chan struct{}]struct{}
}

func DefaultPath() string {
	path := os.Getenv("MOJI_CONFIG")
	if path == "" {
		path = "config.yaml"
	}
	return path
}

func OpenStore(path string) (*Store, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return parseStore(path, file)
}
func parseStore(path string, file []byte) (*Store, error) {

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.Connection.Stash.normalize()
	cfg.System.TaskDeletePolicy = cfg.System.EffectiveTaskDeletePolicy()
	cfg.Automation.StashBoxEndpoints = cleanStrings(cfg.Automation.StashBoxEndpoints)
	cfg.Automation.SubscriptionReleasePolicy = cfg.Automation.SubscriptionReleasePolicy.Effective()
	if err := cfg.Automation.TorrentSelection.Validate(); err != nil {
		return nil, err
	}
	cfg.Automation.TorrentSelection = cfg.Automation.TorrentSelection.Effective()
	cfg.path = path

	var root yaml.Node
	if err := yaml.Unmarshal(file, &root); err != nil {
		return nil, fmt.Errorf("parse config node %q: %w", path, err)
	}

	return &Store{
		path:  path,
		cfg:   &cfg,
		root:  root,
		saved: append([]byte(nil), file...),
	}, nil
}

func (s *Store) Config() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneConfig(s.cfg)
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) UpdateStash(url, apiKey string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Connection.Stash.URL = trimGraphQLSuffix(url)
	s.cfg.Connection.Stash.APIKey = apiKey

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) UpdateIngest(
	downloads DownloadsIngestConfig,
	library LibraryIngestConfig,
	transfer TransferIngestConfig,
	keepSubtitles, generateTranscodes bool,
) (*Config, error) {
	if transfer.Action != "COPY" && transfer.Action != "MOVE" {
		return nil, fmt.Errorf("ingest transfer action must be COPY or MOVE")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Ingest.Downloads = downloads
	s.cfg.Ingest.Library = library
	s.cfg.Ingest.Transfer = transfer
	s.cfg.Ingest.KeepSubtitles = keepSubtitles
	s.cfg.Ingest.GenerateTranscodes = generateTranscodes

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) UpdateJackett(url, apiKey, password string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Connection.Jackett.URL = url
	s.cfg.Connection.Jackett.APIKey = apiKey
	s.cfg.Connection.Jackett.Password = password

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) UpdateQBittorrent(url, username, password, defaultSavePath, category, tags string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Connection.QBittorrent.URL = url
	s.cfg.Connection.QBittorrent.Username = username
	s.cfg.Connection.QBittorrent.Password = password
	s.cfg.Connection.QBittorrent.DefaultSavePath = defaultSavePath
	s.cfg.Connection.QBittorrent.Category = category
	s.cfg.Connection.QBittorrent.Tags = tags

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) UpdateAutomation(taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours int, stashBoxEndpoints []string, subscriptionReleasePolicy SubscriptionReleasePolicyConfig, torrentSelection TorrentSelectionConfig, sourcingRetryIntervalHours ...int) (*Config, error) {
	return s.UpdateAutomationProcessing(nil, taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours, stashBoxEndpoints, subscriptionReleasePolicy, torrentSelection, sourcingRetryIntervalHours...)
}

func (s *Store) UpdateAutomationProcessing(enabled *bool, taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours int, stashBoxEndpoints []string, subscriptionReleasePolicy SubscriptionReleasePolicyConfig, torrentSelection TorrentSelectionConfig, sourcingRetryIntervalHours ...int) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.updateAutomationLocked(enabled, taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours, stashBoxEndpoints, subscriptionReleasePolicy, torrentSelection, sourcingRetryIntervalHours...)
}

func (s *Store) updateAutomationLocked(enabled *bool, taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours int, stashBoxEndpoints []string, subscriptionReleasePolicy SubscriptionReleasePolicyConfig, torrentSelection TorrentSelectionConfig, sourcingRetryIntervalHours ...int) (*Config, error) {
	if enabled != nil {
		switch taskProgressSyncIntervalSeconds {
		case 30, 60, 120, 300:
		default:
			return nil, fmt.Errorf("task processing interval must be 30, 60, 120 or 300 seconds")
		}
	}

	if err := torrentSelection.Validate(); err != nil {
		return nil, err
	}
	if len(sourcingRetryIntervalHours) > 0 {
		if sourcingRetryIntervalHours[0] < 1 || sourcingRetryIntervalHours[0] > 168 {
			return nil, fmt.Errorf("resource search interval must be between 1 and 168 hours")
		}
		s.cfg.Automation.SourcingRetryIntervalHours = sourcingRetryIntervalHours[0]
	}

	if enabled != nil {
		value := *enabled
		s.cfg.Automation.TaskProcessingEnabled = &value
	}
	s.cfg.Automation.TaskProgressSyncIntervalSeconds = taskProgressSyncIntervalSeconds
	s.cfg.Automation.SubscriptionPollIntervalHours = subscriptionPollIntervalHours
	s.cfg.Automation.StashBoxEndpoints = cleanStrings(stashBoxEndpoints)
	s.cfg.Automation.SubscriptionReleasePolicy = subscriptionReleasePolicy.Effective()
	s.cfg.Automation.TorrentSelection = torrentSelection.Effective()

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) UpdateSystem(taskDeletePolicy TaskDeletePolicy, imageCache ...ImageCacheConfig) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stashBoxCache := s.cfg.System.StashBoxDataCache.Normalize()
	return s.updateSystemLocked(taskDeletePolicy, firstImageCache(s.cfg.System.ImageCache, imageCache), stashBoxCache)
}

func firstImageCache(current ImageCacheConfig, values []ImageCacheConfig) ImageCacheConfig {
	if len(values) > 0 {
		return values[0]
	}
	return current
}

func (s *Store) UpdateSystemWithDataCache(taskDeletePolicy TaskDeletePolicy, imageCache ImageCacheConfig, stashBoxCache StashBoxDataCacheConfig, developerMode ...bool) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateSystemLocked(taskDeletePolicy, imageCache, stashBoxCache, developerMode...)
}

func (s *Store) updateSystemLocked(taskDeletePolicy TaskDeletePolicy, imageCache ImageCacheConfig, stashBoxCache StashBoxDataCacheConfig, developerMode ...bool) (*Config, error) {
	if len(developerMode) > 0 {
		s.cfg.System.DeveloperMode = developerMode[0]
	}
	s.cfg.System.TaskDeletePolicy = NormalizeTaskDeletePolicy(string(taskDeletePolicy))
	s.cfg.System.ImageCache = imageCache.Normalize()
	if stashBoxCache.MaxSizeMB == 0 {
		stashBoxCache.MaxSizeMB = s.cfg.System.StashBoxDataCache.Normalize().MaxSizeMB
	}
	s.cfg.System.StashBoxDataCache = stashBoxCache.Normalize()

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}

func (s *Store) updateConfigNode() (resultErr error) {
	defer func() {
		if resultErr == nil {
			return
		}
		// A failed pre-publication write must not activate unpersisted settings.
		// If rename succeeded but directory fsync failed, match the actual valid
		// file; the returned I/O error still reports uncertain crash durability.
		previous, _ := parseStore(s.path, s.saved)
		if raw, err := os.ReadFile(s.path); err == nil {
			if actual, err := parseStore(s.path, raw); err == nil {
				previous = actual
			}
		}
		if previous != nil {
			s.cfg, s.root, s.saved = previous.cfg, previous.root, previous.saved
			s.changedLocked()
		}
	}()

	if err := legacyConfigNode(&s.root); err != nil {
		return err
	}
	doc := documentNode(&s.root)
	top := ensureMapValue(doc)
	connection := mapValue(top, "connection")
	ensureMapValue(connection)

	setMapString(connection, "jackett", map[string]string{
		"url":      s.cfg.Connection.Jackett.URL,
		"api_key":  s.cfg.Connection.Jackett.APIKey,
		"password": s.cfg.Connection.Jackett.Password,
	})
	setMapString(connection, "stash", map[string]string{
		"url":     s.cfg.Connection.Stash.URL,
		"api_key": s.cfg.Connection.Stash.APIKey,
	})
	ensureMapValue(mapValue(top, "ingest"))
	setBoolScalar(mapValue(top, "ingest"), "keep_subtitles", s.cfg.Ingest.KeepSubtitles)
	setBoolScalar(mapValue(top, "ingest"), "generate_transcodes", s.cfg.Ingest.GenerateTranscodes)
	setMapString(mapValue(top, "ingest"), "downloads", map[string]string{
		"qb_root":   s.cfg.Ingest.Downloads.QBRoot,
		"moji_root": s.cfg.Ingest.Downloads.MojiRoot,
	})
	setMapString(mapValue(top, "ingest"), "library", map[string]string{
		"moji_root":  s.cfg.Ingest.Library.MojiRoot,
		"stash_root": s.cfg.Ingest.Library.StashRoot,
	})
	setMapString(mapValue(top, "ingest"), "transfer", map[string]string{
		"action": s.cfg.Ingest.Transfer.Action,
	})
	setMapString(connection, "qbittorrent", map[string]string{
		"url":               s.cfg.Connection.QBittorrent.URL,
		"username":          s.cfg.Connection.QBittorrent.Username,
		"password":          s.cfg.Connection.QBittorrent.Password,
		"default_save_path": s.cfg.Connection.QBittorrent.DefaultSavePath,
		"category":          s.cfg.Connection.QBittorrent.Category,
		"tags":              s.cfg.Connection.QBittorrent.Tags,
	})
	if s.cfg.Automation.TaskProcessingEnabled != nil {
		setBoolScalar(mapValue(top, "automation"), "task_processing_enabled", *s.cfg.Automation.TaskProcessingEnabled)
	}
	setIntScalar(mapValue(top, "automation"), "task_progress_sync_interval_seconds", s.cfg.Automation.TaskProgressSyncIntervalSeconds)
	setIntScalar(mapValue(top, "automation"), "sourcing_retry_interval_hours", s.cfg.Automation.EffectiveSourcingRetryIntervalHours())
	setIntScalar(mapValue(top, "automation"), "subscription_poll_interval_hours", s.cfg.Automation.SubscriptionPollIntervalHours)
	setStringList(mapValue(top, "automation"), "selected_stash_box_endpoints", s.cfg.Automation.StashBoxEndpoints)
	setNodeValue(mapValue(top, "automation"), "subscription_release_policy", s.cfg.Automation.SubscriptionReleasePolicy.Effective())
	setNodeValue(mapValue(top, "automation"), "torrent_selection", s.cfg.Automation.TorrentSelection.Effective())
	system := mapValue(top, "system")
	setScalar(system, "language", s.cfg.System.EffectiveLanguage())
	setBoolScalar(system, "developer_mode", s.cfg.System.DeveloperMode)
	setScalar(system, "task_delete_policy", string(s.cfg.System.EffectiveTaskDeletePolicy()))
	imageCache := mapValue(system, "image_cache")
	setBoolScalar(imageCache, "enabled", s.cfg.System.ImageCache.EffectiveEnabled())
	setIntScalar(imageCache, "max_size_mb", s.cfg.System.ImageCache.Normalize().MaxSizeMB)
	setIntScalar(imageCache, "retention_days", s.cfg.System.ImageCache.Normalize().RetentionDays)
	stashBoxDataCache := mapValue(system, "stash_box_data_cache")
	setIntScalar(stashBoxDataCache, "ttl_hours", s.cfg.System.StashBoxDataCache.Normalize().TTLHours)
	setIntScalar(stashBoxDataCache, "max_size_mb", s.cfg.System.StashBoxDataCache.Normalize().MaxSizeMB)
	deleteMapKey(top, "jackett")
	deleteMapKey(top, "qbittorrent")
	deleteMapKey(top, "tasks")
	deleteMapKey(top, "subscription")
	if stash := existingMapValue(connection, "stash"); stash != nil {
		deleteMapKey(stash, "graphql_url")
		deleteMapKey(stash, "mode")
		deleteMapKey(stash, "library_path")
		deleteMapKey(stash, "qbittorrent_path_prefix")
		deleteMapKey(stash, "stash_path_prefix")
		deleteMapKey(stash, "transfer_action")
		deleteMapKey(stash, "transfer_target_path")
	}
	if jackett := existingMapValue(connection, "jackett"); jackett != nil {
		deleteMapKey(jackett, "candidate_selection")
	}

	canonicalizeConfigNode(top)
	indexers := ensureMapValue(mapValue(top, "indexers"))
	setNodeValue(indexers, "builtin", s.cfg.Indexers.Builtin)
	setNodeValue(indexers, "search_cache", s.cfg.Indexers.SearchCache)
	setBoolScalar(mapValue(indexers, "jackett"), "enabled", s.cfg.Indexers.Jackett.Enabled)
	setNodeValue(mapValue(indexers, "jackett"), "excluded_indexers", s.cfg.Indexers.Jackett.ExcludedIndexers)
	data, err := yaml.Marshal(&s.root)
	if err != nil {
		return fmt.Errorf("marshal config %q: %w", s.path, err)
	}
	next, err := parseStore(s.path, data)
	if err != nil {
		return err
	}
	if err := durablefile.Write(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", s.path, err)
	}
	s.saved = append(s.saved[:0], data...)
	s.cfg, s.root = next.cfg, next.root
	s.changedLocked()
	return nil
}

func documentNode(root *yaml.Node) *yaml.Node {
	if root.Kind == yaml.DocumentNode {
		if len(root.Content) == 0 {
			root.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
		}
		return root.Content[0]
	}
	return root
}

func ensureMapValue(node *yaml.Node) *yaml.Node {
	if node.Kind == 0 {
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
	}
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
		node.Content = nil
	}
	return node
}

func setMapString(parent *yaml.Node, key string, values map[string]string) {
	section := mapValue(parent, key)
	ensureMapValue(section)
	for field, value := range values {
		setScalar(section, field, value)
	}
}

func setNodeValue(parent *yaml.Node, key string, value any) {
	ensureMapValue(parent)
	target := mapValue(parent, key)

	var node yaml.Node
	if err := node.Encode(value); err != nil {
		target.Kind = yaml.MappingNode
		target.Tag = "!!map"
		target.Content = nil
		return
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		*target = *node.Content[0]
		return
	}
	*target = node
}

func setStringList(parent *yaml.Node, key string, values []string) {
	ensureMapValue(parent)
	target := mapValue(parent, key)
	if len(values) == 0 {
		target.Kind = yaml.SequenceNode
		target.Tag = "!!seq"
		target.Style = 0
		target.Content = nil
		return
	}
	target.Kind = yaml.SequenceNode
	target.Tag = "!!seq"
	target.Style = yaml.FlowStyle
	target.Content = make([]*yaml.Node, 0, len(values))
	for _, value := range values {
		target.Content = append(target.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: value,
		})
	}
}

func setScalar(parent *yaml.Node, key, value string) {
	ensureMapValue(parent)
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!str"
	target.Value = value
	target.Style = 0
}

func setIntScalar(parent *yaml.Node, key string, value int) {
	ensureMapValue(parent)
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!int"
	target.Value = strconv.Itoa(value)
	target.Style = 0
}

func setBoolScalar(parent *yaml.Node, key string, value bool) {
	ensureMapValue(parent)
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!bool"
	target.Value = strconv.FormatBool(value)
	target.Style = 0
}

func setInt64Scalar(parent *yaml.Node, key string, value int64) {
	ensureMapValue(parent)
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!int"
	target.Value = strconv.FormatInt(value, 10)
	target.Style = 0
}

func mapValue(parent *yaml.Node, key string) *yaml.Node {
	if existing := existingMapValue(parent, key); existing != nil {
		return existing
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return valueNode
}

func existingMapValue(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i+1]
		}
	}
	return nil
}

func deleteMapKey(parent *yaml.Node, key string) {
	if parent == nil || parent.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
			return
		}
	}
}

// UpdateAutomationSection merges under the store lock: stale fields from other
// tabs never replace a more recently persisted section.
func (s *Store) UpdateAutomationSection(section string, update AutomationConfig) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.cfg.Automation
	switch section {
	case "", "ALL":
		current = update
	case "TASKS":
		current.TaskProcessingEnabled = update.TaskProcessingEnabled
		current.TaskProgressSyncIntervalSeconds = update.TaskProgressSyncIntervalSeconds
		current.SourcingRetryIntervalHours = update.SourcingRetryIntervalHours
		current.SubscriptionPollIntervalHours = update.SubscriptionPollIntervalHours
		current.SubscriptionReleasePolicy = update.SubscriptionReleasePolicy
	case "RESOURCES":
		preference := current.TorrentSelection.FastRules.IndexerPreference
		current.TorrentSelection = update.TorrentSelection
		current.TorrentSelection.FastRules.IndexerPreference = preference
	case "METADATA":
		current.StashBoxEndpoints = update.StashBoxEndpoints
	case "INDEXERS":
		current.TorrentSelection.FastRules.IndexerPreference = update.TorrentSelection.FastRules.IndexerPreference
	default:
		return nil, fmt.Errorf("unknown settings section %q", section)
	}
	return s.updateAutomationLocked(current.TaskProcessingEnabled, current.TaskProgressSyncIntervalSeconds, current.SubscriptionPollIntervalHours, current.StashBoxEndpoints, current.SubscriptionReleasePolicy, current.TorrentSelection, current.EffectiveSourcingRetryIntervalHours())
}

func (s *Store) UpdateStashBoxCache(ttl int) (*Config, error) {
	if ttl < 12 || ttl > 360 {
		return nil, fmt.Errorf("StashBox data cache ttlHours must be between 12 and 360")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.System.StashBoxDataCache.TTLHours = ttl
	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}
func (s *Store) UpdateSystemSection(policy TaskDeletePolicy, image ImageCacheConfig, cache *StashBoxDataCacheConfig, developer bool) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.cfg.System.StashBoxDataCache
	if cache != nil {
		current = *cache
	}
	return s.updateSystemLocked(policy, image, current, developer)
}

func (s *Store) UpdateTaskDeletePolicy(policy TaskDeletePolicy) (*Config, error) {
	switch policy {
	case TaskDeletePolicyKeepOnly, TaskDeletePolicyRemoveTorrent, TaskDeletePolicyRemoveTorrentAndFiles:
	default:
		return nil, ErrInvalidSettings
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.System.TaskDeletePolicy = policy
	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	return cloneConfig(s.cfg), nil
}
func (s *Store) UpdateSystemPreferences(policy TaskDeletePolicy, image ImageCacheConfig, cache *StashBoxDataCacheConfig, developer bool, language *string, preservePolicy bool) (*Config, error) {
	if language != nil {
		if err := ValidateLanguage(*language); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.cfg.System.StashBoxDataCache
	if cache != nil {
		current = *cache
	}
	if preservePolicy {
		policy = s.cfg.System.EffectiveTaskDeletePolicy()
	}
	if language != nil {
		s.cfg.System.Language = *language
	}
	return s.updateSystemLocked(policy, image, current, developer)
}
