package config

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

type DownloaderConfig struct {
	Cleanup DownloaderCleanupConfig `yaml:"cleanup"`
}

type DownloaderCleanupConfig struct {
	Enabled       bool    `yaml:"enabled"`
	Trigger       string  `yaml:"trigger"`
	DeleteFiles   bool    `yaml:"delete_files"`
	MinFreeGB     float64 `yaml:"min_free_gb"`
	CapacityOrder string  `yaml:"capacity_order"`
}

func DefaultDownloaderCleanup() DownloaderCleanupConfig {
	return DownloaderCleanupConfig{Trigger: "AFTER_INGEST", DeleteFiles: true, MinFreeGB: 50, CapacityOrder: "DISK_USAGE"}
}

func (c *DownloaderCleanupConfig) UnmarshalYAML(node *yaml.Node) error {
	type plain DownloaderCleanupConfig
	*c = DefaultDownloaderCleanup()
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("downloader.cleanup must be a mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "enabled", "trigger", "delete_files", "min_free_gb", "capacity_order":
		default:
			return fmt.Errorf("unknown downloader.cleanup field: %s", node.Content[i].Value)
		}
	}
	if err := node.Decode((*plain)(c)); err != nil {
		return err
	}
	if (c.Trigger != "AFTER_INGEST" && c.Trigger != "CAPACITY") || math.IsNaN(c.MinFreeGB) || math.IsInf(c.MinFreeGB, 0) || c.MinFreeGB <= 0 || c.MinFreeGB > 1e8 || (c.CapacityOrder != "DISK_USAGE" && c.CapacityOrder != "RATIO" && c.CapacityOrder != "SEED_TIME") || (c.Trigger == "CAPACITY" && !c.DeleteFiles) {
		return fmt.Errorf("invalid downloader.cleanup settings")
	}
	return nil
}

func (s *Store) DownloaderCleanup() DownloaderCleanupConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Downloader.Cleanup
}

func (s *Store) UpdateDownloaderCleanup(c DownloaderCleanupConfig) error {
	_, err := s.ApplyPatch(map[string]any{"downloader": map[string]any{"cleanup": c}})
	return err
}
