package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Jackett is the configuration for the Jackett service
	Jackett struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"api_key"`
	} `yaml:"jackett"`

	QBittorrent struct {
		URL             string `yaml:"url"`
		Username        string `yaml:"username"`
		Password        string `yaml:"password"`
		DefaultSavePath string `yaml:"default_save_path"`
		Category        string `yaml:"category"`
		Tags            string `yaml:"tags"`
	} `yaml:"qbittorrent"`

	Stash struct {
		GraphQLURL  string `yaml:"graphql_url"`
		APIKey      string `yaml:"api_key"`
		LibraryPath string `yaml:"library_path"`
	} `yaml:"stash"`

	Tasks struct {
		Store                       string `yaml:"store"`
		JSONPath                    string `yaml:"json_path"`
		ProgressSyncIntervalSeconds int    `yaml:"progress_sync_interval_seconds"`
	} `yaml:"tasks"`
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

	return &config, nil
}
