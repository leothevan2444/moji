package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var ErrConfigRevision = errors.New("CONFIG_REVISION_CONFLICT")

type IndexerSettingValue struct {
	String  *string
	Boolean *bool
	Number  *float64
}

func (v *IndexerSettingValue) UnmarshalYAML(node *yaml.Node) error {
	*v = IndexerSettingValue{}
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("indexer setting must be a scalar")
	}
	switch node.Tag {
	case "!!str":
		value := node.Value
		v.String = &value
	case "!!bool":
		var value bool
		if err := node.Decode(&value); err != nil {
			return err
		}
		v.Boolean = &value
	case "!!int", "!!float":
		var value float64
		if err := node.Decode(&value); err != nil {
			return err
		}
		v.Number = &value
	default:
		return fmt.Errorf("invalid indexer setting type")
	}
	return v.Validate()
}
func (v IndexerSettingValue) Validate() error {
	count := 0
	if v.String != nil {
		count++
		if len(*v.String) > 65536 {
			return fmt.Errorf("indexer setting exceeds length budget")
		}
	}
	if v.Boolean != nil {
		count++
	}
	if v.Number != nil {
		count++
		if math.IsNaN(*v.Number) || math.IsInf(*v.Number, 0) {
			return fmt.Errorf("invalid indexer number")
		}
	}
	if count != 1 {
		return fmt.Errorf("indexer setting requires one value")
	}
	return nil
}
func (v IndexerSettingValue) MarshalYAML() (any, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	if v.String != nil {
		return *v.String, nil
	}
	if v.Boolean != nil {
		return *v.Boolean, nil
	}
	return *v.Number, nil
}
func (v IndexerSettingValue) clone() IndexerSettingValue {
	out := v
	if v.String != nil {
		x := *v.String
		out.String = &x
	}
	if v.Boolean != nil {
		x := *v.Boolean
		out.Boolean = &x
	}
	if v.Number != nil {
		x := *v.Number
		out.Number = &x
	}
	return out
}

type IndexerProxyConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
type IndexerSolverConfig struct {
	MaxTimeoutMS int    `yaml:"max_timeout_ms"`
	Enabled      bool   `yaml:"enabled"`
	URL          string `yaml:"url"`
}
type IndexerNetworkConfig struct {
	Proxy        IndexerProxyConfig  `yaml:"proxy"`
	FlareSolverr IndexerSolverConfig `yaml:"flaresolverr"`
}

func (n IndexerNetworkConfig) ProxyURL() string {
	if !n.Proxy.Enabled {
		return ""
	}
	u, err := url.Parse(n.Proxy.URL)
	if err != nil {
		return ""
	}
	if n.Proxy.Username != "" || n.Proxy.Password != "" {
		u.User = url.UserPassword(n.Proxy.Username, n.Proxy.Password)
	}
	return u.String()
}
func (n IndexerNetworkConfig) SolverURL() string {
	if n.FlareSolverr.Enabled {
		return n.FlareSolverr.URL
	}
	return ""
}

type BuiltinIndexerInstance struct {
	Definition     string                         `yaml:"definition"`
	Name           string                         `yaml:"name"`
	Enabled        bool                           `yaml:"enabled"`
	BaseURL        string                         `yaml:"base_url"`
	Settings       map[string]IndexerSettingValue `yaml:"settings"`
	TrustedOrigins []string                       `yaml:"trusted_origins,omitempty"`
}
type IndexerDefinitionsConfig struct {
	Source             string `yaml:"source"`
	SchemaVersion      int    `yaml:"schema_version"`
	CheckIntervalHours int    `yaml:"check_interval_hours"`
	AutoUpdate         bool   `yaml:"auto_update"`
}
type BuiltinIndexersConfig struct {
	Definitions           IndexerDefinitionsConfig          `yaml:"definitions"`
	Instances             map[string]BuiltinIndexerInstance `yaml:"instances"`
	Network               IndexerNetworkConfig              `yaml:"network"`
	RequestTimeoutSeconds int                               `yaml:"request_timeout_seconds"`
	SiteTimeoutSeconds    int                               `yaml:"site_timeout_seconds"`
}
type IndexerSearchCacheConfig struct {
	Enabled         bool `yaml:"enabled"`
	TTLSeconds      int  `yaml:"ttl_seconds"`
	EmptyTTLSeconds int  `yaml:"empty_ttl_seconds"`
	MaxSizeMB       int  `yaml:"max_size_mb"`
}

// Jackett connection and query order retain their current Go ownership views;
// the YAML layout still owns them under indexers.
type IndexersConfig struct {
	Jackett     IndexerJackettOptions    `yaml:"jackett"`
	QueryOrder  []string                 `yaml:"query_order"`
	Builtin     BuiltinIndexersConfig    `yaml:"builtin"`
	SearchCache IndexerSearchCacheConfig `yaml:"search_cache"`
}

type IndexerJackettOptions struct {
	Enabled          bool     `yaml:"enabled"`
	ExcludedIndexers []string `yaml:"excluded_indexers,omitempty"`
}

func DefaultIndexersConfig() IndexersConfig {
	return IndexersConfig{Builtin: BuiltinIndexersConfig{Definitions: IndexerDefinitionsConfig{Source: "prowlarr", SchemaVersion: 11, CheckIntervalHours: 24}, Instances: map[string]BuiltinIndexerInstance{}, Network: IndexerNetworkConfig{FlareSolverr: IndexerSolverConfig{MaxTimeoutMS: 90000}}, RequestTimeoutSeconds: 30, SiteTimeoutSeconds: 120}, SearchCache: IndexerSearchCacheConfig{Enabled: true, TTLSeconds: 300, EmptyTTLSeconds: 30, MaxSizeMB: 32}}
}
func (c IndexersConfig) Clone() IndexersConfig {
	out := c
	out.QueryOrder = append([]string(nil), c.QueryOrder...)
	out.Jackett.ExcludedIndexers = append([]string(nil), c.Jackett.ExcludedIndexers...)
	out.Builtin.Instances = map[string]BuiltinIndexerInstance{}
	for id, instance := range c.Builtin.Instances {
		copy := instance
		copy.Settings = map[string]IndexerSettingValue{}
		for name, value := range instance.Settings {
			copy.Settings[name] = value.clone()
		}
		copy.TrustedOrigins = append([]string(nil), instance.TrustedOrigins...)
		out.Builtin.Instances[id] = copy
	}
	return out
}

func (c IndexersConfig) Validate() error {
	b := c.Builtin
	d := b.Definitions
	if d.Source != "prowlarr" || d.SchemaVersion != 11 || d.CheckIntervalHours < 1 || d.CheckIntervalHours > 168 {
		return fmt.Errorf("invalid definition library configuration")
	}
	if b.RequestTimeoutSeconds < 1 || b.RequestTimeoutSeconds > 120 || b.SiteTimeoutSeconds < 10 || b.SiteTimeoutSeconds > 600 {
		return fmt.Errorf("invalid indexer timeout")
	}
	if c.SearchCache.TTLSeconds < 1 || c.SearchCache.TTLSeconds > 3600 || c.SearchCache.EmptyTTLSeconds < 1 || c.SearchCache.EmptyTTLSeconds > 300 || c.SearchCache.MaxSizeMB < 1 || c.SearchCache.MaxSizeMB > 256 {
		return fmt.Errorf("invalid indexer cache budget")
	}
	if len(b.Instances) > 256 {
		return fmt.Errorf("indexer configuration exceeds capacity")
	}
	if len(c.QueryOrder) > 1024 || len(c.Jackett.ExcludedIndexers) > 1024 {
		return fmt.Errorf("too many ordered indexers")
	}
	seen := map[string]bool{}
	for _, id := range c.QueryOrder {
		if id == "" || len(id) > 256 || seen[id] {
			return fmt.Errorf("invalid indexer order")
		}
		seen[id] = true
		if strings.HasPrefix(id, "jackett/") && len(id) > len("jackett/") {
			continue
		}
		if _, ok := b.Instances[id]; !ok {
			return fmt.Errorf("unknown builtin instance in query order")
		}
	}
	if b.Network.FlareSolverr.MaxTimeoutMS < 1000 || b.Network.FlareSolverr.MaxTimeoutMS > 300000 {
		return fmt.Errorf("invalid solver max timeout")
	}
	if b.Network.Proxy.Enabled && b.Network.Proxy.URL == "" || b.Network.FlareSolverr.Enabled && b.Network.FlareSolverr.URL == "" {
		return fmt.Errorf("enabled network service requires a URL")
	}
	if len(b.Network.Proxy.Username) > 1024 || len(b.Network.Proxy.Password) > 4096 || strings.ContainsAny(b.Network.Proxy.Username+b.Network.Proxy.Password, "\r\n") {
		return fmt.Errorf("invalid proxy credentials")
	}
	for kind, raw := range []string{b.Network.Proxy.URL, b.Network.FlareSolverr.URL} {
		if raw == "" {
			continue
		}
		target, err := url.Parse(raw)
		if err != nil || len(raw) > 4096 || target.Hostname() == "" || target.User != nil || target.Fragment != "" || target.RawQuery != "" {
			return fmt.Errorf("invalid network endpoint")
		}
		if kind == 1 {
			if target.Scheme != "http" && target.Scheme != "https" {
				return fmt.Errorf("invalid solver endpoint")
			}
			continue
		}
		if target.Path != "" && target.Path != "/" {
			return fmt.Errorf("invalid proxy path")
		}
		switch target.Scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return fmt.Errorf("invalid proxy scheme")
		}
	}
	for id, instance := range b.Instances {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("invalid builtin instance ID")
		}
		if !indexerParameterName.MatchString(instance.Definition) || len(instance.Name) > 256 || len(instance.Settings) > 128 {
			return fmt.Errorf("invalid builtin instance")
		}
		if instance.BaseURL != "" {
			if err := validIndexerURL(instance.BaseURL); err != nil {
				return err
			}
		}
		for key, value := range instance.Settings {
			if len(key) > 128 {
				return fmt.Errorf("invalid setting name")
			}
			if err := value.Validate(); err != nil {
				return err
			}
		}
		if len(instance.TrustedOrigins) > 16 {
			return fmt.Errorf("too many trusted origins")
		}
		for _, origin := range instance.TrustedOrigins {
			if err := validIndexerOrigin(origin); err != nil {
				return err
			}
		}
	}
	return nil
}
func validIndexerOrigin(raw string) error {
	if err := validIndexerURL(raw); err != nil {
		return err
	}
	target, _ := url.Parse(raw)
	if strings.Contains(target.Hostname(), "*") || target.ForceQuery || strings.Contains(raw, "#") || target.Path != "" && target.Path != "/" {
		return fmt.Errorf("invalid trusted origin")
	}
	return nil
}
func validIndexerURL(raw string) error {
	target, err := url.Parse(raw)
	if err != nil || len(raw) > 4096 || target.User != nil || target.Hostname() == "" || target.Fragment != "" || target.RawQuery != "" || target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("invalid indexer URL")
	}
	return nil
}

func (s *Store) Revision() string     { s.mu.RLock(); defer s.mu.RUnlock(); return configDigest(s.saved) }
func configDigest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func (s *Store) WithRevision(ctx context.Context, revision string, commit func() error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if configDigest(s.saved) != revision {
		return ErrConfigRevision
	}
	return commit()
}

func cloneConfig(cfg *Config) *Config {
	copy := *cfg
	copy.Indexers = cfg.Indexers.Clone()
	return &copy
}

func (s *Store) Snapshot() (*Config, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.cfg), configDigest(s.saved)
}

var indexerParameterName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)
