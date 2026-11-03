package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexercheck"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/indexerdownloads"
	"github.com/leothevan2444/moji/internal/indexerlogin"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/torrentsearch"
)

type indexerRuntime struct {
	admin         *indexeradmin.Service
	service       *indexer.Service
	registry      *runtimeIndexerRegistry
	definitions   *indexerdefs.Service
	resources     *indexer.Resources
	sessions      *indexer.Sessions
	network       *indexer.Network
	configuration indexerConfiguration
	events        *indexer.Events
	searchCache   *indexer.SearchCache
	artifactCache *indexer.ArtifactCache
	searches      *torrentsearch.Service
	downloads     *indexerdownloads.Service
	checks        *indexercheck.Service
	logins        *indexerlogin.Service
}

func newIndexerRuntime(ctx context.Context, cfg *config.Config, store *config.Store, manager *storage.Manager, source indexerdefs.Source) (*indexerRuntime, error) {
	configuration := indexerConfiguration{store: store, fallback: cfg}
	validator := &indexerActivationValidator{configuration: configuration}
	if source == nil {
		source = indexerdefs.NewGitHubSource(nil)
	}
	definitions, err := indexerdefs.NewService(manager, source, validator)
	if err != nil {
		return nil, err
	}
	validator.bundles = definitions
	health, err := indexer.NewHealthStore(manager)
	if err != nil {
		return nil, err
	}
	network, err := indexer.NewPersistentNetwork(ctx, manager, func(error) { logging.Warn("indexer network cooldown state unavailable") })
	if err != nil {
		return nil, err
	}
	resources, err := indexer.NewResources(ctx, manager, definitions)
	if err != nil {
		network.Close()
		return nil, err
	}
	sessions, err := indexer.NewSessions(network, manager)
	if err != nil {
		resources.Close()
		network.Close()
		return nil, err
	}
	builtin, err := indexer.NewBuiltinAdapter(definitions, sessions)
	if err != nil {
		resources.Close()
		network.Close()
		return nil, err
	}
	external, err := indexer.NewJackettAdapter(network)
	if err != nil {
		resources.Close()
		network.Close()
		return nil, err
	}
	events := indexer.NewEvents()
	registry := &runtimeIndexerRegistry{configuration: configuration, external: external, events: events}
	service, err := indexer.NewService(registry, resources, builtin, external)
	if err != nil {
		resources.Close()
		network.Close()
		return nil, err
	}
	service.SetEventPublisher(events)
	searchCache, err := indexer.NewSearchCache(ctx)
	if err != nil {
		resources.Close()
		network.Close()
		return nil, err
	}
	service.SetSearchCache(searchCache, func() indexer.SearchCachePolicy {
		cfg, _ := configuration.snapshot()
		policy := cfg.Indexers.SearchCache
		return indexer.SearchCachePolicy{Enabled: policy.Enabled, TTL: time.Duration(policy.TTLSeconds) * time.Second, EmptyTTL: time.Duration(policy.EmptyTTLSeconds) * time.Second, MaxBytes: int64(policy.MaxSizeMB) << 20}
	})
	searchCache.SetEventPublisher(events)
	artifactCache, err := indexer.NewArtifactCache(ctx)
	if err != nil {
		searchCache.Close()
		resources.Close()
		network.Close()
		return nil, err
	}
	service.SetArtifactCache(artifactCache)
	var healthWarningMu sync.Mutex
	var healthWarningAt time.Time
	service.SetHealthRecorder(health, func(err error) {
		if errors.Is(err, context.Canceled) {
			return
		}
		healthWarningMu.Lock()
		defer healthWarningMu.Unlock()
		if time.Since(healthWarningAt) >= time.Minute {
			healthWarningAt = time.Now()
			logging.Warn("indexer observation could not be persisted")
		}
	})
	searches, err := torrentsearch.NewService(ctx, manager, service)
	if err != nil {
		artifactCache.Close()
		searchCache.Close()
		resources.Close()
		network.Close()
		return nil, err
	}
	searches.SetErrorHandler(func(error) { logging.Warn("interactive torrent search state could not be persisted") })
	downloads, err := indexerdownloads.New(ctx, service)
	if err != nil {
		searches.Close()
		artifactCache.Close()
		searchCache.Close()
		resources.Close()
		network.Close()
		return nil, err
	}
	admin := indexeradmin.NewService(store, definitions, registry)
	admin.SetHealthReader(health)
	admin.SetDefinitionJobReader(definitions)
	admin.SetSearchCache(searchCache, events)
	checks, err := indexercheck.New(ctx, admin, indexercheck.AdapterFactory{Network: network, Bundles: definitions})
	if err != nil {
		downloads.Close()
		searches.Close()
		artifactCache.Close()
		searchCache.Close()
		resources.Close()
		network.Close()
		return nil, err
	}
	logins, err := indexerlogin.New(ctx, admin, indexerlogin.AdapterFactory{Sessions: sessions})
	if err != nil {
		checks.Close()
		downloads.Close()
		searches.Close()
		artifactCache.Close()
		searchCache.Close()
		resources.Close()
		sessions.Close()
		network.Close()
		return nil, err
	}
	return &indexerRuntime{admin: admin, service: service, registry: registry, definitions: definitions, resources: resources, sessions: sessions, network: network, configuration: configuration, events: events, searchCache: searchCache, artifactCache: artifactCache, searches: searches, downloads: downloads, checks: checks, logins: logins}, nil
}

// Start does not fetch rules. Definitions already on disk are validated and
// update jobs interrupted by a previous process are recovered before serving.
func (r *indexerRuntime) Start(ctx context.Context) error {
	if err := r.definitions.Start(ctx); err != nil {
		return err
	}
	return r.searches.Start(ctx)
}
func (r *indexerRuntime) Run(ctx context.Context) {
	var workers sync.WaitGroup
	workers.Add(3)
	go func() { defer workers.Done(); r.watchConfiguration(ctx) }()
	go func() { defer workers.Done(); r.searches.Run(ctx) }()
	go func() {
		defer workers.Done()
		r.resources.Maintain(ctx, func(err error) {
			if !errors.Is(err, context.Canceled) {
				logging.Warnf("indexer resource maintenance: %s", indexer.ErrorCode(err))
			}
		})
	}()
	defer workers.Wait()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkDefinitions(ctx)
		}
	}
}

// Configuration delivery is independent of network-bound rule update checks.
func (r *indexerRuntime) watchConfiguration(ctx context.Context) {
	if r.configuration.store == nil {
		return
	}
	changes, unsubscribe, err := r.configuration.store.SubscribeChanges()
	if err != nil {
		logging.Warn("indexer configuration notifications unavailable")
		return
	}
	defer unsubscribe()
	lastConfiguration := ""
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-changes:
			if !ok {
				return
			}
			cfg, _ := r.configuration.snapshot()
			key := indexerFingerprint(struct {
				Indexers   config.IndexersConfig
				Connection config.JackettConfig
			}{cfg.Indexers, cfg.Connection.Jackett})
			if key != lastConfiguration {
				lastConfiguration = key
				r.events.Publish(indexer.EventConfiguration)
			}
		}
	}
}

func (r *indexerRuntime) checkDefinitions(ctx context.Context) {
	status, err := r.definitions.Status(ctx)
	if err != nil || status.State.Active == "" || status.Job != nil && !status.Job.State.Terminal() {
		return
	}
	cfg, _ := r.configuration.snapshot()
	policy := cfg.Indexers.Builtin.Definitions
	if status.State.LastCheckedAt != nil && time.Since(*status.State.LastCheckedAt) < time.Duration(policy.CheckIntervalHours)*time.Hour {
		return
	}
	state, err := r.definitions.Check(ctx)
	if err != nil {
		if ctx.Err() == nil {
			logging.Warn("indexer definitions update check failed")
		}
		return
	}
	cfg, _ = r.configuration.snapshot()
	if cfg.Indexers.Builtin.Definitions.AutoUpdate && state.Available != "" && state.Available != state.Active {
		if _, err := r.definitions.Begin(ctx, "UPDATE", state.Available, state.Active); err != nil && !errors.Is(err, indexerdefs.ErrBusy) && ctx.Err() == nil {
			logging.Warn("indexer definitions automatic update was not accepted")
		}
	}
}

// The caller stops and joins Run and request/task workers before Close. The
// definitions worker must stop before runtime revision pins are released.
func (r *indexerRuntime) Close() {
	if r.logins != nil {
		r.logins.Close()
	}
	if r.checks != nil {
		r.checks.Close()
	}
	if r.downloads != nil {
		r.downloads.Close()
	}
	r.searches.Close()
	r.artifactCache.Close()
	r.searchCache.Close()
	r.events.Close()
	r.definitions.Close()
	r.resources.Close()
	r.sessions.Close()
	r.network.Close()
}
