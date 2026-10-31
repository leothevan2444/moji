package config

import (
	"fmt"
	"time"
	_ "time/tzdata"

	"gopkg.in/yaml.v3"
)

// Config's Go fields represent runtime providers. The on-disk v2 layout groups
// values by business ownership; legacy documents are read without rewriting.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	type plain Config
	*c = Config{}
	c.Automation.TorrentSelection = DefaultTorrentSelectionConfig()
	var version struct {
		Version int `yaml:"config_version"`
	}
	if err := node.Decode(&version); err != nil {
		return err
	}
	if version.Version == 0 {
		for _, key := range []string{"indexers", "downloader"} {
			if existingMapValue(node, key) != nil {
				return fmt.Errorf("%s requires config_version: 2", key)
			}
		}
		if stash := existingMapValue(node, "stash"); stash != nil && existingMapValue(stash, "connection") != nil {
			return fmt.Errorf("stash.connection requires config_version: 2")
		}
		if err := node.Decode((*plain)(c)); err != nil {
			return err
		}
		return validateSystemCalendar(c.System)
	}
	if version.Version != 2 {
		return fmt.Errorf("unsupported config_version: %d", version.Version)
	}
	for _, key := range []string{"connection", "automation", "ingest"} {
		if existingMapValue(node, key) != nil {
			return fmt.Errorf("config_version 2 cannot contain legacy %s", key)
		}
	}
	for _, path := range [][]string{{"system", "stash_box_data_cache"}, {"tasks", "selected_stash_box_endpoints"}, {"tasks", "torrent_selection", "fast_rules", "indexer_preference"}} {
		current := node
		for _, key := range path {
			if current == nil {
				break
			}
			current = existingMapValue(current, key)
		}
		if current != nil {
			return fmt.Errorf("config_version 2 contains a legacy field at %v", path)
		}
	}
	if tasks := existingMapValue(node, "tasks"); tasks != nil && existingMapValue(tasks, "task_delete_policy") != nil {
		if system := existingMapValue(node, "system"); system != nil && existingMapValue(system, "task_delete_policy") != nil {
			return fmt.Errorf("task deletion policy appears in both tasks and system")
		}
	}
	raw, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return err
	}
	top := documentNode(&root)
	decanonicalizeConfigNode(top)
	if err := top.Decode((*plain)(c)); err != nil {
		return err
	}
	return validateSystemCalendar(c.System)
}

func moveConfigNode(from *yaml.Node, key string, to *yaml.Node, target string) {
	if value := existingMapValue(from, key); value != nil {
		*mapValue(ensureMapValue(to), target) = *value
		deleteMapKey(from, key)
	}
}

// canonicalizeConfigNode converts an already normalized legacy-shaped tree.
// Moving nodes preserves unmodeled fields and YAML comments within sections.
func canonicalizeConfigNode(top *yaml.Node) {
	conn := ensureMapValue(mapValue(top, "connection"))
	indexers := ensureMapValue(mapValue(top, "indexers"))
	stash := ensureMapValue(mapValue(top, "stash"))
	downloader := ensureMapValue(mapValue(top, "downloader"))
	moveConfigNode(conn, "jackett", indexers, "jackett")
	moveConfigNode(conn, "stash", stash, "connection")
	moveConfigNode(conn, "qbittorrent", downloader, "qbittorrent")
	moveConfigNode(top, "ingest", stash, "ingest")
	automation := ensureMapValue(mapValue(top, "automation"))
	moveConfigNode(automation, "selected_stash_box_endpoints", stash, "metadata_priority")
	system := ensureMapValue(mapValue(top, "system"))
	moveConfigNode(system, "task_delete_policy", automation, "task_delete_policy")
	moveConfigNode(system, "stash_box_data_cache", stash, "data_cache")
	selection := ensureMapValue(mapValue(automation, "torrent_selection"))
	rules := ensureMapValue(mapValue(selection, "fast_rules"))
	if preference := existingMapValue(rules, "indexer_preference"); preference != nil {
		moveConfigNode(preference, "tracker_ids", indexers, "query_order")
		deleteMapKey(rules, "indexer_preference")
	}
	if order := existingMapValue(selection, "fast_rule_order"); order != nil {
		kept := order.Content[:0]
		for _, item := range order.Content {
			if item.Value != "INDEXER_PREFERENCE" {
				kept = append(kept, item)
			}
		}
		order.Content = kept
	}
	moveConfigNode(top, "automation", top, "tasks")
	if len(conn.Content) > 0 {
		*mapValue(top, "legacy_connection") = *conn
	}
	deleteMapKey(top, "connection")
	setIntScalar(top, "config_version", 2)
}

// legacyConfigNode is used solely by Store's existing atomic update machinery.
func legacyConfigNode(root *yaml.Node) error {
	top := documentNode(root)
	version := existingMapValue(top, "config_version")
	if version == nil {
		return nil
	}
	var cfg Config
	if err := top.Decode(&cfg); err != nil {
		return err
	}
	decanonicalizeConfigNode(top)

	return nil
}

// Migrate writes v2 through the same durable, recoverable path as a settings save.
func (s *Store) Migrate() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateConfigNode()
}

func decanonicalizeConfigNode(top *yaml.Node) {
	indexers := ensureMapValue(mapValue(top, "indexers"))
	stash := ensureMapValue(mapValue(top, "stash"))
	downloader := ensureMapValue(mapValue(top, "downloader"))
	conn := ensureMapValue(mapValue(top, "connection"))
	moveConfigNode(indexers, "jackett", conn, "jackett")
	moveConfigNode(stash, "connection", conn, "stash")
	moveConfigNode(downloader, "qbittorrent", conn, "qbittorrent")
	moveConfigNode(stash, "ingest", top, "ingest")
	moveConfigNode(top, "tasks", top, "automation")
	automation := ensureMapValue(mapValue(top, "automation"))
	moveConfigNode(stash, "metadata_priority", automation, "selected_stash_box_endpoints")
	system := ensureMapValue(mapValue(top, "system"))
	moveConfigNode(automation, "task_delete_policy", system, "task_delete_policy")
	moveConfigNode(stash, "data_cache", system, "stash_box_data_cache")
	rules := ensureMapValue(mapValue(ensureMapValue(mapValue(automation, "torrent_selection")), "fast_rules"))
	preference := ensureMapValue(mapValue(rules, "indexer_preference"))
	moveConfigNode(indexers, "query_order", preference, "tracker_ids")
	setBoolScalar(preference, "enabled", true)
	for _, key := range []string{"config_version"} {
		deleteMapKey(top, key)
	}
}

func validateSystemCalendar(system SystemConfig) error {
	if _, err := time.LoadLocation(system.ReleaseTimezoneMode()); system.ReleaseTimezoneMode() != "auto" && err != nil {
		return fmt.Errorf("invalid release timezone: %w", err)
	}
	return ValidateLanguage(system.Language)
}
