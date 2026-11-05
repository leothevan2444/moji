package indexeradmin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/url"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type Definitions interface {
	Acquire(context.Context, string) (*indexerdefs.Lease, error)
	RequireActive(context.Context, string) error
}
type Registry interface {
	CachedSnapshot(context.Context) (indexer.Registry, error)
	Refresh(context.Context) (indexer.Registry, error)
}
type Service struct {
	connectionKey  []byte
	definitionJobs DefinitionJobReader
	store          *config.Store
	definitions    Definitions
	registry       Registry
	health         HealthReader
	cache          *indexer.SearchCache
	events         indexer.EventPublisher
}

type HealthReader interface {
	Snapshot(context.Context, []indexer.HealthKey) (map[indexer.HealthKey]indexer.HealthState, error)
}

// SetHealthReader is installed by application assembly before requests start.
func (s *Service) SetHealthReader(health HealthReader) { s.health = health }

func NewService(store *config.Store, definitions Definitions, registry Registry) *Service {
	key := uuid.New()
	return &Service{connectionKey: key[:], store: store, definitions: definitions, registry: registry}
}

func (s *Service) Definitions(ctx context.Context, filter DefinitionFilter) (DefinitionPage, error) {
	if s.definitions == nil {
		return DefinitionPage{}, indexerdefs.ErrUnavailable
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit < 1 || filter.Limit > 100 || filter.Offset < 0 || filter.Offset > 100000 || len(filter.Query) > 256 || len(filter.Language) > 64 || len(filter.Privacy) > 64 || len(filter.Capability) > 64 {
		return DefinitionPage{}, indexer.ErrInvalidQuery
	}
	lease, err := s.definitions.Acquire(ctx, "")
	if err != nil {
		return DefinitionPage{}, err
	}
	defer lease.Release()
	page := DefinitionPage{Revision: lease.Bundle.Manifest.Revision, Offset: filter.Offset, Limit: filter.Limit, Items: []Definition{}}
	ids := make([]string, 0, len(lease.Bundle.Plans))
	for id := range lease.Bundle.Plans {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := lease.Bundle.Plans[ids[i]].Definition.Name, lease.Bundle.Plans[ids[j]].Definition.Name
		if a == b {
			return ids[i] < ids[j]
		}
		return a < b
	})
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return DefinitionPage{}, err
		}
		plan := lease.Bundle.Plans[id]
		d := plan.Definition
		if query != "" && !strings.Contains(strings.ToLower(d.ID+" "+d.Name+" "+d.Description), query) || filter.Language != "" && !strings.EqualFold(filter.Language, d.Language) || filter.Privacy != "" && !strings.EqualFold(filter.Privacy, d.Type) {
			continue
		}
		if filter.Capability != "" {
			if _, ok := d.Caps.Modes[filter.Capability]; !ok {
				continue
			}
		}
		page.Total++
		if page.Total <= filter.Offset || len(page.Items) >= filter.Limit {
			continue
		}
		page.Items = append(page.Items, definitionProjection(plan))
	}
	return page, nil
}
func definitionProjection(plan *cardigann.Plan) Definition {
	d := plan.Definition
	result := Definition{ID: d.ID, Name: d.Name, Description: d.Description, Language: d.Language, Privacy: d.Type, Digest: plan.Digest, Links: append([]string(nil), d.Links...), Modes: []string{}, Warnings: []string{}, Settings: settingProjection(plan, nil)}
	result.LoginSupported = d.Login != nil
	categories := map[string]bool{}
	for _, category := range d.Caps.Categories {
		if name := strings.TrimSpace(string(category.Value)); name != "" {
			categories[name] = true
		}
	}
	for _, mapping := range d.Caps.CategoryMappings {
		if name := strings.TrimSpace(mapping.Cat); name != "" {
			categories[name] = true
		}
	}
	result.Categories = []string{}
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}
	sort.Strings(result.Categories)
	for mode := range d.Caps.Modes {
		result.Modes = append(result.Modes, mode)
	}
	sort.Strings(result.Modes)
	for _, warning := range plan.Warnings {
		result.Warnings = append(result.Warnings, warning.Error())
	}
	return result
}
func (s *Service) Instances(ctx context.Context) (InstancePage, error) {
	if s.store == nil {
		return InstancePage{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	for attempt := 0; attempt < 3; attempt++ {
		page, err := s.instances(ctx, cfg, revision)
		if !errors.Is(err, config.ErrConfigRevision) {
			return page, err
		}
		cfg, revision = s.store.Snapshot()
	}
	return InstancePage{}, config.ErrConfigRevision
}
func (s *Service) instances(ctx context.Context, cfg *config.Config, revision string) (InstancePage, error) {
	result := InstancePage{Revision: revision, JackettEnabled: cfg.Indexers.Jackett.Enabled, Order: append([]string(nil), cfg.Indexers.QueryOrder...), NetworkProfiles: []string{}, Items: []Instance{}}
	result.JackettConfigured = cfg.Connection.Jackett.URL != "" && cfg.Connection.Jackett.APIKey != ""
	for id := range cfg.Indexers.Builtin.NetworkProfiles {
		result.NetworkProfiles = append(result.NetworkProfiles, id)
	}
	sort.Strings(result.NetworkProfiles)
	var bundle *indexerdefs.Bundle
	if s.definitions != nil {
		lease, err := s.definitions.Acquire(ctx, "")
		if err == nil {
			defer lease.Release()
			bundle = lease.Bundle
			result.RuleRevision = bundle.Manifest.Revision
		} else {
			result.RuleErrorCode = indexer.ErrorCode(err)
		}
	}
	ids := make([]string, 0, len(cfg.Indexers.Builtin.Instances))
	for id := range cfg.Indexers.Builtin.Instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		source := cfg.Indexers.Builtin.Instances[id]
		item := Instance{ID: id, Kind: indexer.Builtin, Name: source.Name, Enabled: source.Enabled, DefinitionID: source.Definition, BaseURL: source.BaseURL, NetworkProfile: source.NetworkProfile, TrustedOrigins: append([]string(nil), source.TrustedOrigins...), Settings: []Setting{}, Modes: []string{}}
		if item.Name == "" {
			item.Name = source.Definition
		}
		if item.NetworkProfile == "" {
			item.NetworkProfile = "direct"
		}
		var plan *cardigann.Plan
		if bundle != nil {
			plan = bundle.Plans[source.Definition]
		}
		if plan != nil {
			item.LoginSupported = plan.Definition.Login != nil
			item.Description = plan.Definition.Description
			item.RuleDigest = plan.Digest
			item.SiteURL = source.BaseURL
			if item.SiteURL == "" && len(plan.Definition.Links) > 0 {
				item.SiteURL = plan.Definition.Links[0]
			}
			item.Privacy, item.Language = plan.Definition.Type, plan.Definition.Language
			item.Settings = settingProjection(plan, source.Settings)
			for mode := range plan.Definition.Caps.Modes {
				item.Modes = append(item.Modes, mode)
			}
			sort.Strings(item.Modes)
			if err := ValidateSettings(plan, nil, source); err != nil {
				item.ErrorCode = indexer.ErrInvalidSettings.Error()
			}
		} else {
			item.ErrorCode = result.RuleErrorCode
			if item.ErrorCode == "" {
				item.ErrorCode = indexer.ErrUnsupported.Error()
			}
			for name, value := range source.Settings {
				item.Settings = append(item.Settings, Setting{Name: name, Type: "unknown", Secret: true, Configured: configured(value)})
			}
			sort.Slice(item.Settings, func(i, j int) bool { return item.Settings[i].Name < item.Settings[j].Name })
		}
		item.ConnectionRevision = s.connectionRevision(BuiltinSnapshot(cfg, id, source).ConfigRevision, item.RuleDigest)
		result.Items = append(result.Items, item)
	}
	if s.registry != nil {
		registry, err := s.registry.CachedSnapshot(ctx)
		if err != nil {
			return InstancePage{}, err
		}
		if registry.Revision != revision {
			return InstancePage{}, config.ErrConfigRevision
		}
		result.JackettCatalog = registry.JackettCatalog
		for _, diagnostic := range registry.Diagnostics {
			if result.CatalogErrorCode == "" {
				result.CatalogErrorCode = diagnostic.Code
			}
		}
		for _, source := range registry.Instances {
			if cfg.Indexers.Jackett.Enabled && source.Kind == indexer.Jackett {
				result.Items = append(result.Items, Instance{ConnectionRevision: s.connectionRevision(source.ConfigRevision, ""), SiteURL: source.SiteURL, Privacy: source.SiteType, Language: source.Language, ID: source.ID, Name: source.Name, Kind: source.Kind, Enabled: source.Enabled, Settings: []Setting{}, Modes: []string{"search"}})
			}
		}
		if s.health != nil {
			keys := make([]indexer.HealthKey, 0, len(registry.Instances))
			byID := map[string]indexer.HealthKey{}
			for _, source := range registry.Instances {
				if source.Kind == indexer.Builtin {
					source.BundleRevision = result.RuleRevision
				}
				key := indexer.HealthScope(source)
				keys = append(keys, key)
				byID[source.ID] = key
			}
			states, err := s.health.Snapshot(ctx, keys)
			if err != nil {
				result.HealthErrorCode = "INDEXER_STATUS_UNAVAILABLE"
			} else {
				for i := range result.Items {
					result.Items[i].Health = states[byID[result.Items[i].ID]]
				}
			}
		}
	}
	byID := map[string]Instance{}
	for _, item := range result.Items {
		byID[item.ID] = item
	}
	ordered := make([]Instance, 0, len(result.Items))
	for _, id := range result.Order {
		if item, ok := byID[id]; ok {
			ordered = append(ordered, item)
			delete(byID, id)
		}
	}
	for _, item := range result.Items {
		if _, ok := byID[item.ID]; ok {
			ordered = append(ordered, item)
			delete(byID, item.ID)
		}
	}
	result.Items = ordered
	for i := range result.Items {
		result.Items[i].SiteURL = publicSiteURL(result.Items[i].SiteURL)
	}
	return result, ctx.Err()
}

func publicSiteURL(raw string) string {
	target, err := url.Parse(raw)
	if err != nil || target.Hostname() == "" || target.Scheme != "http" && target.Scheme != "https" {
		return ""
	}
	target.User = nil
	target.RawQuery = ""
	target.Fragment = ""
	return target.String()
}
func (s *Service) SaveBuiltin(ctx context.Context, id, expected string, patch BuiltinPatch) (SaveResult, error) {
	if s.store == nil || s.definitions == nil {
		return SaveResult{}, indexer.ErrNotConfigured
	}
	cfg, revision := s.store.Snapshot()
	if revision != expected {
		return SaveResult{}, config.ErrConfigRevision
	}
	current, exists := cfg.Indexers.Builtin.Instances[id]
	if id != "" && !exists {
		return SaveResult{}, indexer.ErrInvalidScope
	}
	candidate, err := config.MergeBuiltinInstance(current, patch)
	if err != nil {
		return SaveResult{}, err
	}
	// Disabling a broken source must remain possible while the library is absent.
	needsRules := id == "" || candidate.Enabled || patch.Settings != nil || patch.Definition != nil
	var guard func() error
	if needsRules {
		lease, err := s.definitions.Acquire(ctx, "")
		if err != nil {
			return SaveResult{}, err
		}
		defer lease.Release()
		plan := lease.Bundle.Plans[candidate.Definition]
		if plan == nil {
			return SaveResult{}, indexer.ErrUnsupported
		}
		if err := ValidateSettings(plan, nil, candidate); err != nil {
			return SaveResult{}, fmt.Errorf("%w: invalid definition settings", config.ErrInvalidSettings)
		}
		guard = func() error { return s.definitions.RequireActive(ctx, lease.Bundle.Manifest.Revision) }
	}
	savedID, _, err := s.store.UpdateBuiltinInstanceGuarded(ctx, id, expected, patch, guard)
	if err != nil {
		return SaveResult{}, err
	}
	page, err := s.Instances(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	for _, item := range page.Items {
		if item.ID == savedID {
			return SaveResult{Revision: page.Revision, Instance: item}, nil
		}
	}
	return SaveResult{}, config.ErrConfigRevision
}
func (s *Service) RemoveBuiltin(ctx context.Context, id, expected string) (InstancePage, error) {
	if s.store == nil {
		return InstancePage{}, indexer.ErrNotConfigured
	}
	if _, err := s.store.RemoveBuiltinInstance(ctx, id, expected); err != nil {
		return InstancePage{}, err
	}
	return s.Instances(ctx)
}
func (s *Service) SetEnabled(ctx context.Context, id, expected string, enabled bool) (InstancePage, error) {
	if strings.HasPrefix(id, "jackett/") || id == "jackett" {
		if s.store == nil {
			return InstancePage{}, indexer.ErrNotConfigured
		}
		cfg, revision := s.store.Snapshot()
		if revision != expected {
			return InstancePage{}, config.ErrConfigRevision
		}
		if id == "jackett" && enabled && (cfg.Connection.Jackett.URL == "" || cfg.Connection.Jackett.APIKey == "") {
			return InstancePage{}, indexer.ErrNotConfigured
		}
		if id != "jackett" {
			if s.registry == nil {
				return InstancePage{}, indexer.ErrInvalidScope
			}
			snapshot, err := s.registry.CachedSnapshot(ctx)
			if err != nil {
				return InstancePage{}, err
			}
			if snapshot.Revision != expected {
				return InstancePage{}, config.ErrConfigRevision
			}
			known := false
			for _, source := range snapshot.Instances {
				if source.ID == id && source.Kind == indexer.Jackett {
					known = true
					break
				}
			}
			if !known {
				return InstancePage{}, indexer.ErrInvalidScope
			}
		}
		if _, err := s.store.SetJackettIndexerEnabled(ctx, expected, id, enabled); err != nil {
			return InstancePage{}, err
		}
		return s.Instances(ctx)
	}
	if _, err := s.SaveBuiltin(ctx, id, expected, BuiltinPatch{Enabled: &enabled}); err != nil {
		return InstancePage{}, err
	}
	return s.Instances(ctx)
}
func (s *Service) Reorder(ctx context.Context, expected string, ids []string) (InstancePage, error) {
	if s.store == nil {
		return InstancePage{}, indexer.ErrNotConfigured
	}
	if _, err := s.store.ReorderIndexers(ctx, expected, ids); err != nil {
		return InstancePage{}, err
	}
	return s.Instances(ctx)
}
func (s *Service) RefreshJackett(ctx context.Context) (InstancePage, error) {
	if s.registry == nil {
		return InstancePage{}, indexer.ErrNotConfigured
	}
	if _, err := s.registry.Refresh(ctx); err != nil {
		return InstancePage{}, err
	}
	return s.Instances(ctx)
}

func secretSetting(setting cardigann.Setting) bool {
	if setting.Type == "password" {
		return true
	}
	name := strings.NewReplacer("-", "", "_", "").Replace(strings.ToLower(setting.Name))
	for _, part := range []string{"password", "passkey", "cookie", "apikey", "token", "authorization", "secret"} {
		if strings.Contains(name, part) {
			return true
		}
	}
	return name == "username" || name == "user" || name == "email" || name == "uid" || name == "key"
}
func configured(value config.IndexerSettingValue) bool {
	return value.String != nil && *value.String != "" || value.Boolean != nil || value.Number != nil
}
func publicValue(value config.IndexerSettingValue) *Value {
	return &Value{String: value.String, Boolean: value.Boolean, Number: value.Number}
}
func settingProjection(plan *cardigann.Plan, settings map[string]config.IndexerSettingValue) []Setting {
	result := make([]Setting, 0, len(plan.Definition.Settings))
	known := map[string]bool{}
	for _, field := range plan.Definition.Settings {
		known[field.Name] = true
		setting := Setting{Name: field.Name, Type: field.Type, Label: field.Label, Secret: secretSetting(field), Options: []Option{}}
		if value, ok := settings[field.Name]; ok {
			setting.Configured = configured(value)
			if !setting.Secret {
				setting.Value = publicValue(value)
			}
		}
		keys := make([]string, 0, len(field.Options))
		for key := range field.Options {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			setting.Options = append(setting.Options, Option{Value: key, Label: string(field.Options[key])})
		}
		if !setting.Secret {
			value := string(field.Default)
			if field.Type == "checkbox" {
				boolean := strings.EqualFold(value, "true")
				setting.Default = &Value{Boolean: &boolean}
			} else if field.Type == "text" || field.Type == "select" {
				setting.Default = &Value{String: &value}
			}
		}
		result = append(result, setting)
	}
	unknown := []string{}
	for name := range settings {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	for _, name := range unknown {
		result = append(result, Setting{Name: name, Label: name, Type: "unknown", Secret: true, Configured: configured(settings[name]), Options: []Option{}})
	}
	return result
}

// connectionRevision does not reveal a guessable hash of account credentials.
func (s *Service) connectionRevision(configRevision, ruleDigest string) string {
	mac := hmac.New(sha256.New, s.connectionKey)
	mac.Write([]byte(configRevision + "\x00" + ruleDigest))
	return hex.EncodeToString(mac.Sum(nil))
}
