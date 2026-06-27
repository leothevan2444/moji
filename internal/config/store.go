package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Store struct {
	mu   sync.RWMutex
	path string
	cfg  *Config
	root yaml.Node
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

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.Connection.Stash.normalize()
	cfg.System.TaskDeletePolicy = cfg.System.EffectiveTaskDeletePolicy()
	cfg.path = path

	var root yaml.Node
	if err := yaml.Unmarshal(file, &root); err != nil {
		return nil, fmt.Errorf("parse config node %q: %w", path, err)
	}

	return &Store{
		path: path,
		cfg:  &cfg,
		root: root,
	}, nil
}

func (s *Store) Config() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clone := *s.cfg
	return &clone
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
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateIngest(
	deliveryMode string,
	stashLibraryPath string,
	transfer TransferIngestConfig,
) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Ingest.DeliveryMode = deliveryMode
	s.cfg.Ingest.StashLibraryPath = stashLibraryPath
	s.cfg.Ingest.Transfer = transfer

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
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
	clone := *s.cfg
	return &clone, nil
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
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateAutomation(taskProgressSyncIntervalSeconds, subscriptionPollIntervalHours int) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Automation.TaskProgressSyncIntervalSeconds = taskProgressSyncIntervalSeconds
	s.cfg.Automation.SubscriptionPollIntervalHours = subscriptionPollIntervalHours

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateSystem(taskDeletePolicy TaskDeletePolicy) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.System.TaskDeletePolicy = NormalizeTaskDeletePolicy(string(taskDeletePolicy))

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateSubscription(selectedEndpoints []string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleaned := make([]string, 0, len(selectedEndpoints))
	for _, endpoint := range selectedEndpoints {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}
		cleaned = append(cleaned, endpoint)
	}
	s.cfg.Subscription.StashBoxEndpoints = cleaned

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) updateConfigNode() error {
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
	setMapString(top, "ingest", map[string]string{
		"delivery_mode":      s.cfg.Ingest.DeliveryMode,
		"stash_library_path": s.cfg.Ingest.StashLibraryPath,
	})
	setMapString(mapValue(top, "ingest"), "transfer", map[string]string{
		"action":           s.cfg.Ingest.Transfer.Action,
		"moji_source_root": s.cfg.Ingest.Transfer.MojiSourceRoot,
		"moji_target_root": s.cfg.Ingest.Transfer.MojiTargetRoot,
	})
	setMapString(connection, "qbittorrent", map[string]string{
		"url":               s.cfg.Connection.QBittorrent.URL,
		"username":          s.cfg.Connection.QBittorrent.Username,
		"password":          s.cfg.Connection.QBittorrent.Password,
		"default_save_path": s.cfg.Connection.QBittorrent.DefaultSavePath,
		"category":          s.cfg.Connection.QBittorrent.Category,
		"tags":              s.cfg.Connection.QBittorrent.Tags,
	})
	setIntScalar(mapValue(top, "automation"), "task_progress_sync_interval_seconds", s.cfg.Automation.TaskProgressSyncIntervalSeconds)
	setIntScalar(mapValue(top, "automation"), "subscription_poll_interval_hours", s.cfg.Automation.SubscriptionPollIntervalHours)
	setScalar(mapValue(top, "system"), "task_delete_policy", string(s.cfg.System.EffectiveTaskDeletePolicy()))
	setStringList(mapValue(top, "subscription"), "selected_stash_box_endpoints", s.cfg.Subscription.StashBoxEndpoints)
	deleteMapKey(top, "stash")
	deleteMapKey(top, "jackett")
	deleteMapKey(top, "qbittorrent")
	deleteMapKey(top, "tasks")
	deleteMapKey(top, "logging")
	if stash := existingMapValue(connection, "stash"); stash != nil {
		deleteMapKey(stash, "graphql_url")
		deleteMapKey(stash, "mode")
		deleteMapKey(stash, "library_path")
		deleteMapKey(stash, "qbittorrent_path_prefix")
		deleteMapKey(stash, "stash_path_prefix")
		deleteMapKey(stash, "transfer_action")
		deleteMapKey(stash, "transfer_target_path")
	}
	if ingest := existingMapValue(top, "ingest"); ingest != nil {
		deleteMapKey(ingest, "mode")
		deleteMapKey(ingest, "shared_storage")
		deleteMapKey(ingest, "file_transfer")
		deleteMapKey(ingest, "library_scan")
		deleteMapKey(ingest, "scan_scope")
		deleteMapKey(ingest, "path_map")
		deleteMapKey(ingest, "library_path")
		deleteMapKey(ingest, "library")
		deleteMapKey(ingest, "qbittorrent_path_prefix")
		deleteMapKey(ingest, "stash_path_prefix")
		deleteMapKey(ingest, "transfer_action")
		deleteMapKey(ingest, "transfer_target_path")
		deleteMapKey(ingest, "target_path")
	}

	data, err := yaml.Marshal(&s.root)
	if err != nil {
		return fmt.Errorf("marshal config %q: %w", s.path, err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", s.path, err)
	}
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
