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
	cfg.Stash.normalize()
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

func (s *Store) UpdateStash(url string, apiKey *string, libraryPath string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Stash.URL = trimGraphQLSuffix(url)
	s.cfg.Stash.LegacyGraphQLURL = ""
	if apiKey != nil {
		s.cfg.Stash.APIKey = *apiKey
	}
	s.cfg.Stash.LibraryPath = libraryPath

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateJackett(url string, apiKey *string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Jackett.URL = url
	if apiKey != nil {
		s.cfg.Jackett.APIKey = *apiKey
	}

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateQBittorrent(url, username string, password *string, defaultSavePath, category, tags string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.QBittorrent.URL = url
	s.cfg.QBittorrent.Username = username
	if password != nil {
		s.cfg.QBittorrent.Password = *password
	}
	s.cfg.QBittorrent.DefaultSavePath = defaultSavePath
	s.cfg.QBittorrent.Category = category
	s.cfg.QBittorrent.Tags = tags

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) UpdateSubscription(store, dbPath string, pollIntervalSeconds int, selectedEndpoints []string) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Subscription.Store = store
	s.cfg.Subscription.DBPath = dbPath
	s.cfg.Subscription.PollIntervalSeconds = pollIntervalSeconds
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

func (s *Store) UpdateLogging(level, filePath string, maxEntries int, maxFileSizeBytes int64, maxFileBackups int) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Logging.Level = level
	s.cfg.Logging.FilePath = filePath
	s.cfg.Logging.MaxEntries = maxEntries
	s.cfg.Logging.MaxFileSizeBytes = maxFileSizeBytes
	s.cfg.Logging.MaxFileBackups = maxFileBackups

	if err := s.updateConfigNode(); err != nil {
		return nil, err
	}
	clone := *s.cfg
	return &clone, nil
}

func (s *Store) updateConfigNode() error {
	doc := documentNode(&s.root)
	top := ensureMapValue(doc)

	setMapString(top, "jackett", map[string]string{
		"url":     s.cfg.Jackett.URL,
		"api_key": s.cfg.Jackett.APIKey,
	})
	setMapString(top, "stash", map[string]string{
		"url":          s.cfg.Stash.URL,
		"api_key":      s.cfg.Stash.APIKey,
		"library_path": s.cfg.Stash.LibraryPath,
	})
	deleteMapKey(mapValue(top, "stash"), "graphql_url")
	setMapString(top, "qbittorrent", map[string]string{
		"url":               s.cfg.QBittorrent.URL,
		"username":          s.cfg.QBittorrent.Username,
		"password":          s.cfg.QBittorrent.Password,
		"default_save_path": s.cfg.QBittorrent.DefaultSavePath,
		"category":          s.cfg.QBittorrent.Category,
		"tags":              s.cfg.QBittorrent.Tags,
	})
	setMapString(top, "tasks", map[string]string{
		"store":   s.cfg.Tasks.Store,
		"db_path": s.cfg.Tasks.DBPath,
	})
	setIntScalar(mapValue(top, "tasks"), "progress_sync_interval_seconds", s.cfg.Tasks.ProgressSyncIntervalSeconds)
	setMapString(top, "subscription", map[string]string{
		"store":   s.cfg.Subscription.Store,
		"db_path": s.cfg.Subscription.DBPath,
	})
	setIntScalar(mapValue(top, "subscription"), "poll_interval_seconds", s.cfg.Subscription.PollIntervalSeconds)
	setStringList(mapValue(top, "subscription"), "selected_stash_box_endpoints", s.cfg.Subscription.StashBoxEndpoints)
	setMapString(top, "logging", map[string]string{
		"level":     s.cfg.Logging.Level,
		"file_path": s.cfg.Logging.FilePath,
	})
	setIntScalar(mapValue(top, "logging"), "max_entries", s.cfg.Logging.MaxEntries)
	setInt64Scalar(mapValue(top, "logging"), "max_file_size_bytes", s.cfg.Logging.MaxFileSizeBytes)
	setIntScalar(mapValue(top, "logging"), "max_file_backups", s.cfg.Logging.MaxFileBackups)

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
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!str"
	target.Value = value
	target.Style = 0
}

func setIntScalar(parent *yaml.Node, key string, value int) {
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!int"
	target.Value = strconv.Itoa(value)
	target.Style = 0
}

func setInt64Scalar(parent *yaml.Node, key string, value int64) {
	target := mapValue(parent, key)
	target.Kind = yaml.ScalarNode
	target.Tag = "!!int"
	target.Value = strconv.FormatInt(value, 10)
	target.Style = 0
}

func mapValue(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i+1]
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return valueNode
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
