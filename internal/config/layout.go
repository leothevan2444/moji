package config

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Config's Go fields represent runtime providers. The on-disk v2 layout groups
// values by business ownership; legacy documents are read without rewriting.
func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	for _, section := range []string{"tasks", "automation"} {
		if group := existingMapValue(node, section); group != nil {
			if selection := existingMapValue(group, "torrent_selection"); selection != nil {
				if selection.Kind != yaml.MappingNode {
					return fmt.Errorf("%s.torrent_selection must be a versioned resource policy", section)
				}
				for i := 0; i < len(selection.Content); i += 2 {
					key := selection.Content[i].Value
					switch key {
					case "version", "active_group", "groups", "rules", "enabled", "inspection_candidate_limit", "fast_rule_order", "fast_rules", "torrent_rules":
					default:
						return fmt.Errorf("unknown %s.torrent_selection field %s", section, key)
					}
				}

				for _, key := range []string{"enabled", "inspection_candidate_limit", "fast_rule_order", "fast_rules", "torrent_rules"} {
					if existingMapValue(selection, key) != nil {
						return fmt.Errorf("%s.torrent_selection.%s is retired; back up and manually replace resource selection with version: 7 positive-only rules and groups; existing filters must not be silently discarded", section, key)
					}
				}
				raw, err := yaml.Marshal(selection)
				if err != nil {
					return err
				}
				decoder := yaml.NewDecoder(bytes.NewReader(raw))
				decoder.KnownFields(true)
				var policy TorrentSelectionConfig
				if err := decoder.Decode(&policy); err != nil {
					return fmt.Errorf("invalid %s.torrent_selection: %w", section, err)
				}

			}

			for _, retired := range []string{"subscription_poll_interval_hours", "subscription_check_enabled", "task_progress_sync_interval_seconds"} {
				if existingMapValue(group, retired) != nil {
					return fmt.Errorf("%s.%s is no longer supported; remove subscription switches and use task_processing_frequency (HIGH, MEDIUM or LOW)", section, retired)
				}
			}
			if value := existingMapValue(group, "task_processing_frequency"); value != nil && value.Value != "HIGH" && value.Value != "MEDIUM" && value.Value != "LOW" {
				return fmt.Errorf("task_processing_frequency must be HIGH, MEDIUM or LOW")
			}
		}
	}
	if system := existingMapValue(node, "system"); system != nil && existingMapValue(system, "release_timezone") != nil {
		return fmt.Errorf("system.release_timezone is no longer supported; rename it to system.timezone")
	}
	type plain Config
	*c = Config{}
	c.Downloader.Cleanup = DefaultDownloaderCleanup()
	c.Automation.TaskProcessingFrequency = "MEDIUM"
	c.Automation.TorrentSelection = DefaultTorrentSelectionConfig()
	c.Indexers = DefaultIndexersConfig()
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
		c.Indexers.Jackett.Enabled = c.Connection.Jackett.URL != ""
		return validateSystemCalendar(c.System)
	}
	if version.Version != 2 && version.Version != 3 && version.Version != 4 {
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
	if err := validateIndexerNetworkLayout(top); err != nil {
		return err
	}
	if idx := existingMapValue(top, "indexers"); idx != nil {
		if err := idx.Decode(&c.Indexers); err != nil {
			return err
		}
	}
	decanonicalizeConfigNode(top)
	if err := top.Decode((*plain)(c)); err != nil {
		return err
	}
	if version.Version < 3 {
		c.Indexers.Jackett.Enabled = c.Connection.Jackett.URL != ""
	}
	if version.Version < 3 {
		for i, id := range c.Indexers.QueryOrder {
			if !strings.HasPrefix(id, "jackett/") {
				c.Indexers.QueryOrder[i] = "jackett/" + id
			}
		}
	}
	if err := c.Indexers.Validate(); err != nil {
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
func canonicalizeConfigNode(top *yaml.Node) error {
	// Log files have a fixed location; drop the retired override on save.
	if logs := existingMapValue(top, "logging"); logs != nil {
		deleteMapKey(logs, "file_path")
	}
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
	moveConfigNode(top, "automation", top, "tasks")
	if len(conn.Content) > 0 {
		*mapValue(top, "legacy_connection") = *conn
	}
	deleteMapKey(top, "connection")
	return normalizeIndexerNode(top)
}

func normalizeIndexerNode(top *yaml.Node) error {
	if err := validateIndexerNetworkLayout(top); err != nil {
		return err
	}
	version := existingMapValue(top, "config_version")
	legacyIDs := version == nil || version.Value == "2"
	indexers := ensureMapValue(mapValue(top, "indexers"))
	jackett := ensureMapValue(mapValue(indexers, "jackett"))
	if existingMapValue(jackett, "enabled") == nil {
		url := existingMapValue(jackett, "url")
		setBoolScalar(jackett, "enabled", url != nil && url.Value != "")
	}
	if order := existingMapValue(indexers, "query_order"); order != nil {
		for _, item := range order.Content {
			if item.Value != "" && !strings.HasPrefix(item.Value, "jackett/") {
				if _, err := uuid.Parse(item.Value); legacyIDs || err != nil {
					item.Value = "jackett/" + item.Value
				}
			}
		}
	}
	setIntScalar(top, "config_version", 4)
	return nil
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
	deleteMapKey(top, "config_version")
}

func validateSystemCalendar(system SystemConfig) error {
	if _, err := time.LoadLocation(system.TimezoneMode()); system.TimezoneMode() != "auto" && err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	return ValidateLanguage(system.Language)
}

// Removed profile keys are rejected rather than silently ignored.
func validateIndexerNetworkLayout(top *yaml.Node) error {
	idx := existingMapValue(top, "indexers")
	if idx == nil {
		return nil
	}
	builtin := existingMapValue(idx, "builtin")
	if builtin == nil {
		return nil
	}
	if existingMapValue(builtin, "challenge_timeout_seconds") != nil {
		return fmt.Errorf("challenge_timeout_seconds was removed; use builtin.network.flaresolverr.max_timeout_ms")
	}
	if existingMapValue(builtin, "network_profiles") != nil {
		return fmt.Errorf("network_profiles was removed; use builtin.network")
	}
	instances := existingMapValue(builtin, "instances")
	if instances != nil {
		for i := 1; i < len(instances.Content); i += 2 {
			if existingMapValue(instances.Content[i], "network_profile") != nil {
				return fmt.Errorf("network_profile was removed; all builtin sites use builtin.network")
			}
		}
	}
	return nil
}
