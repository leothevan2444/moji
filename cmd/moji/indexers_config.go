package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type indexerConfiguration struct {
	store    *config.Store
	fallback *config.Config
}

func (c indexerConfiguration) snapshot() (*config.Config, string) {
	if c.store != nil {
		return c.store.Snapshot()
	}
	cfg := *c.fallback
	cfg.Indexers = cfg.Indexers.Clone()
	// Literal configurations used by embedders/tests precede the v3 defaults.
	if cfg.Indexers.Builtin.Definitions.Source == "" {
		cfg.Indexers = config.DefaultIndexersConfig()
		cfg.Indexers.Jackett.Enabled = cfg.Connection.Jackett.URL != ""
		for _, id := range cfg.Automation.TorrentSelection.FastRules.IndexerPreference.TrackerIDs {
			cfg.Indexers.QueryOrder = append(cfg.Indexers.QueryOrder, "jackett/"+id)
		}
	}
	return &cfg, indexerFingerprint(cfg)
}
func indexerFingerprint(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type externalIndexerCatalog interface {
	Catalog(context.Context, indexer.InstanceSnapshot) ([]indexer.InstanceSnapshot, error)
}
type remoteIndexer struct{ ID, Name, SiteURL, SiteType, Language string }
type runtimeIndexerRegistry struct {
	configuration      indexerConfiguration
	external           externalIndexerCatalog
	mu                 sync.Mutex
	catalogKey         string
	catalog            []remoteIndexer
	catalogAt          time.Time
	catalogConfirmedAt time.Time
	attemptKey         string
	attemptAt          time.Time
	lastError          error
	events             indexer.EventPublisher
	loading            chan struct{}
}

// Snapshot freezes account settings before any site is searched. A failed
// catalog refresh retains only the last confirmed sites for this connection,
// and reports incompleteness rather than turning it into an empty result.
func (p *runtimeIndexerRegistry) Snapshot(ctx context.Context) (indexer.Registry, error) {
	return p.snapshot(ctx, true)
}

func (p *runtimeIndexerRegistry) CachedSnapshot(ctx context.Context) (indexer.Registry, error) {
	return p.snapshot(ctx, false)
}

// ReferenceSnapshot checks local admission policy without refreshing Jackett.
// Persisted search results remain usable after restart when that connection is
// unchanged, even before its optional remote directory has been loaded again.
func (p *runtimeIndexerRegistry) ReferenceSnapshot(ctx context.Context, id string) (indexer.InstanceSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return indexer.InstanceSnapshot{}, err
	}
	cfg, _ := p.configuration.snapshot()
	if source, ok := cfg.Indexers.Builtin.Instances[id]; ok && source.Enabled {
		return indexeradmin.BuiltinSnapshot(cfg, id, source), nil
	}
	if !cfg.Indexers.Jackett.Enabled || !strings.HasPrefix(id, "jackett/") || len(id) <= len("jackett/") || len(id) > 256 {
		return indexer.InstanceSnapshot{}, indexer.ErrReference
	}
	remoteID := strings.TrimPrefix(id, "jackett/")
	for _, excluded := range cfg.Indexers.Jackett.ExcludedIndexers {
		if excluded == id || excluded == remoteID {
			return indexer.InstanceSnapshot{}, indexer.ErrReference
		}
	}
	connection := jackettConnectionSnapshot(cfg)
	if connection.BaseURL == "" || connection.JackettAPIKey == "" {
		return indexer.InstanceSnapshot{}, indexer.ErrNotConfigured
	}
	connection.ID, connection.RemoteID = id, remoteID
	return connection, nil
}

func jackettConnectionSnapshot(cfg *config.Config) indexer.InstanceSnapshot {
	connection := indexer.InstanceSnapshot{ID: "jackett", Kind: indexer.Jackett, Enabled: true, BaseURL: cfg.Connection.Jackett.URL, JackettAPIKey: cfg.Connection.Jackett.APIKey, RequestTimeout: time.Duration(cfg.Indexers.Builtin.RequestTimeoutSeconds) * time.Second, SiteTimeout: time.Duration(cfg.Indexers.Builtin.SiteTimeoutSeconds) * time.Second}
	connection.ConfigRevision = indexerFingerprint(connection)
	return connection
}
func (p *runtimeIndexerRegistry) Refresh(ctx context.Context) (indexer.Registry, error) {
	p.mu.Lock()
	if time.Since(p.attemptAt) >= 30*time.Second {
		p.catalogAt = time.Time{}
	}
	p.mu.Unlock()
	return p.snapshot(ctx, true)
}
func (p *runtimeIndexerRegistry) snapshot(ctx context.Context, refresh bool) (indexer.Registry, error) {
	cfg, revision := p.configuration.snapshot()
	registry := indexer.Registry{Revision: revision, Order: append([]string(nil), cfg.Indexers.QueryOrder...)}
	ids := make([]string, 0, len(cfg.Indexers.Builtin.Instances))
	for id := range cfg.Indexers.Builtin.Instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		source := cfg.Indexers.Builtin.Instances[id]
		instance := indexeradmin.BuiltinSnapshot(cfg, id, source)
		registry.Instances = append(registry.Instances, instance)
	}
	if !cfg.Indexers.Jackett.Enabled {
		return registry, ctx.Err()
	}
	connection := jackettConnectionSnapshot(cfg)
	if connection.BaseURL == "" || connection.JackettAPIKey == "" || p.external == nil {
		registry.Diagnostics = append(registry.Diagnostics, indexer.SiteReport{InstanceID: "jackett", Code: indexer.ErrNotConfigured.Error(), Partial: true, CorrelationID: uuid.NewString()})
		return registry, ctx.Err()
	}
	var sites []remoteIndexer
	var err error
	if refresh {
		sites, err = p.remoteSites(ctx, connection)
	} else {
		p.mu.Lock()
		if p.catalogKey == connection.ConfigRevision {
			sites = append(sites, p.catalog...)
		} else {
			err = indexer.ErrCatalogNotLoaded
		}
		if p.attemptKey == connection.ConfigRevision && p.lastError != nil {
			err = p.lastError
		}
		p.mu.Unlock()
	}
	if ctx.Err() != nil {
		return indexer.Registry{}, ctx.Err()
	}
	p.mu.Lock()
	if p.catalogKey == connection.ConfigRevision && !p.catalogConfirmedAt.IsZero() {
		confirmed := p.catalogConfirmedAt
		registry.JackettCatalog = indexer.CatalogStatus{Known: true, UpdatedAt: &confirmed, Stale: err != nil || time.Since(p.catalogAt) >= 5*time.Minute}
	}
	p.mu.Unlock()
	if err != nil {
		registry.Diagnostics = append(registry.Diagnostics, indexer.SiteReport{InstanceID: "jackett", Code: indexer.ErrorCode(err), Partial: true, CorrelationID: uuid.NewString()})
	}
	excluded := map[string]bool{}
	for _, id := range cfg.Indexers.Jackett.ExcludedIndexers {
		excluded[id] = true
	}
	for _, site := range sites {
		instance := connection
		instance.ID = site.ID
		instance.RemoteID = site.ID[len("jackett/"):]
		instance.Name = site.Name
		instance.SiteURL, instance.SiteType, instance.Language = site.SiteURL, site.SiteType, site.Language
		instance.Enabled = !excluded[site.ID] && !excluded[instance.RemoteID]
		registry.Instances = append(registry.Instances, instance)
	}
	return registry, nil
}
func (p *runtimeIndexerRegistry) remoteSites(ctx context.Context, connection indexer.InstanceSnapshot) ([]remoteIndexer, error) {
	key := connection.ConfigRevision
	for {
		p.mu.Lock()
		if p.catalogKey == key && time.Since(p.catalogAt) < 5*time.Minute {
			sites := append([]remoteIndexer(nil), p.catalog...)
			p.mu.Unlock()
			return sites, nil
		}
		if p.attemptKey == key && p.lastError != nil && time.Since(p.attemptAt) < 30*time.Second {
			var sites []remoteIndexer
			if p.catalogKey == key {
				sites = append(sites, p.catalog...)
			}
			err := p.lastError
			p.mu.Unlock()
			return sites, err
		}
		if p.loading != nil {
			wait := p.loading
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-wait:
				continue
			}
		}
		p.loading = make(chan struct{})
		p.mu.Unlock()
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		rows, err := p.external.Catalog(callCtx, connection)
		cancel()
		sites := make([]remoteIndexer, 0, len(rows))
		if err == nil {
			if len(rows) > 1024 {
				err = indexer.ErrResponseBudget
			} else {
				seen := map[string]bool{}
				for _, row := range rows {
					if row.ID != "jackett/"+row.RemoteID || row.RemoteID == "" || seen[row.ID] {
						err = indexer.ErrInvalidSettings
						break
					}
					seen[row.ID] = true
					sites = append(sites, remoteIndexer{ID: row.ID, Name: row.Name, SiteURL: row.SiteURL, SiteType: row.SiteType, Language: row.Language})
				}
			}
		}
		p.mu.Lock()
		// A validated success is safe to share even if its initiating caller
		// leaves. Do not attach the preceding failure to that confirmed cache.
		if ctx.Err() == nil || err == nil {
			p.attemptKey, p.attemptAt, p.lastError = key, time.Now(), err
		}
		if err == nil {
			p.catalogKey, p.catalog, p.catalogAt = key, sites, time.Now()
			p.catalogConfirmedAt = p.catalogAt
		} else if p.catalogKey == key {
			sites = append([]remoteIndexer(nil), p.catalog...)
		} else {
			sites = nil
		}
		close(p.loading)
		p.loading = nil
		p.mu.Unlock()
		if p.events != nil && (ctx.Err() == nil || err == nil) {
			p.events.Publish(indexer.EventCatalog)
		}
		return sites, err
	}
}

type indexerActivationValidator struct {
	configuration indexerConfiguration
	bundles       indexer.BundleProvider
}
type indexerActivationGuard struct {
	configuration indexerConfiguration
	revision      string
}

func (g indexerActivationGuard) WithCurrent(ctx context.Context, commit func() error) error {
	if g.configuration.store != nil {
		err := g.configuration.store.WithRevision(ctx, g.revision, commit)
		if errors.Is(err, config.ErrConfigRevision) {
			return indexerdefs.ErrConflict
		}
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, revision := g.configuration.snapshot()
	if revision != g.revision {
		return indexerdefs.ErrConflict
	}
	return commit()
}
func (v *indexerActivationValidator) Validate(ctx context.Context, bundle *indexerdefs.Bundle) (indexerdefs.ActivationGuard, error) {
	cfg, revision := v.configuration.snapshot()
	var previous *indexerdefs.Bundle
	if v.bundles != nil {
		if lease, err := v.bundles.Acquire(ctx, ""); err == nil {
			defer lease.Release()
			previous = lease.Bundle
		}
	}
	for id, instance := range cfg.Indexers.Builtin.Instances {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !instance.Enabled {
			continue
		}
		plan := bundle.Plans[instance.Definition]
		if plan == nil {
			return nil, fmt.Errorf("%w: instance %s definition unavailable", indexerdefs.ErrIncompatible, id)
		}
		var old *cardigann.Plan
		if previous != nil {
			old = previous.Plans[instance.Definition]
		}
		if err := indexeradmin.ValidateSettings(plan, old, instance); err != nil {
			return nil, fmt.Errorf("%w: instance %s: %v", indexerdefs.ErrIncompatible, id, err)
		}
	}
	return indexerActivationGuard{configuration: v.configuration, revision: revision}, ctx.Err()
}
