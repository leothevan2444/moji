package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/storage"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/leothevan2444/moji"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/controller/api"
	"github.com/leothevan2444/moji/internal/discovery"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/metadata"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/stashboxcache"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/stats"
	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/taskflow"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/internal/webui"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"github.com/leothevan2444/moji/pkg/stash"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

var runtimeCacheSequence atomic.Uint64

func main() {
	version := moji.Version()
	var (
		configPath      = flag.String("config", "", "path to config yaml (or set MOJI_CONFIG)")
		addr            = flag.String("addr", ":10000", "http listen address")
		backupDirectory = flag.String("backup-dir", "", "backup directory (default: data-dir/backups; may be an independently mounted backup volume)")
		dataDirectory   = flag.String("data-dir", storage.DefaultDirectory, "directory containing Moji databases and recovery generations")
		showVersion     = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()
	if *showVersion {
		fmt.Printf("moji %s\n", version)
		return
	}

	// Config
	var (
		cfg         *config.Config
		configStore *config.Store
		err         error
	)
	path := config.DefaultPath()
	if *configPath != "" {
		path = *configPath
	}
	if _, err := logging.ConfigureDefault(logging.Options{
		Level:    "info",
		FilePath: logging.DefaultLogFilePath(path),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "configure logger: %v\n", err)
		os.Exit(1)
	}
	configStore, err = config.OpenStore(path)
	if err == nil {
		cfg = configStore.Config()
	}
	if err != nil {
		logging.Fatalf("load config: %v", err)
	}
	if _, err := logging.ConfigureDefault(logging.Options{
		Level:            cfg.EffectiveLogLevel(),
		FilePath:         cfg.EffectiveLogFilePath(),
		MaxEntries:       cfg.EffectiveLogMaxEntries(),
		MaxFileSizeBytes: cfg.EffectiveLogMaxFileSizeBytes(),
		MaxFileBackups:   cfg.EffectiveLogMaxFileBackups(),
	}); err != nil {
		logging.Fatalf("reconfigure logger: %v", err)
	}

	if cfg.Connection.Jackett.URL == "" {
		logging.Fatalf("invalid config: connection.jackett.url is required")
	}
	if cfg.Connection.Jackett.APIKey == "" {
		logging.Fatalf("invalid config: connection.jackett.api_key is required")
	}

	// Dependencies: the storage owner is acquired before any background work.
	managerOptions := storage.DefaultManagerOptions(*dataDirectory)
	managerOptions.BackupDirectory = *backupDirectory
	managerOptions.Supplemental = func(ctx context.Context) ([]storage.BackupFile, error) {
		return storage.SupplementalFiles(ctx, configStore.Path(), configStore.Config().Ingest.Library.MojiRoot)
	}
	manager, err := storage.OpenManager(context.Background(), managerOptions)
	if err != nil {
		logging.Fatalf("open storage (legacy installations require explicit migration): %v", err)
	}
	defer func() {
		if err := manager.Close(); err != nil {
			logging.Errorf("close storage: %v", err)
		}
	}()
	runtime := newHTTPRuntime(cfg, version, configStore, manager)
	defer runtime.stashBoxCacheService.Close()

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var requests sync.WaitGroup
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		defer requests.Done()
		runtime.handler.ServeHTTP(w, r)
	})
	server := &http.Server{
		Addr:              *addr,
		Handler:           requestHandler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		logging.Fatalf("listen %s: %v", server.Addr, err)
	}

	logging.Infof("moji listening on %s", ln.Addr().String())

	var workers sync.WaitGroup
	launch := func(work func()) { workers.Add(1); go func() { defer workers.Done(); work() }() }
	launch(func() { manager.MonitorPerformance(ctx) })
	launch(func() {
		manager.Maintain(ctx, func(result storage.MaintenanceResult) {
			if result.Err != nil && !errors.Is(result.Err, context.Canceled) {
				logging.Errorf("storage maintenance: %v", result.Err)
			}
		})
	})
	defer workers.Wait()
	if preparer, ok := runtime.taskRuntimeService.(interface{ SetMediaPreparationContext(context.Context) }); ok {
		preparer.SetMediaPreparationContext(ctx)
	}
	if worker, ok := runtime.taskRuntimeService.(interface{ RunSourcingWorker(context.Context) }); ok {
		launch(func() { worker.RunSourcingWorker(ctx) })
	}
	if runtime.downloader != nil {
		launch(func() { runtime.downloader.Run(ctx) })
	}
	taskSyncDone := startTaskSyncWorker(ctx, runtime.taskRuntimeService, runtime.stashService, configureProgressSyncIntervalProvider(configStore, cfg))
	launch(func() { <-taskSyncDone })
	subscriptionDone := startSubscriptionWorker(ctx, runtime.subscriptionService, configureSubscriptionPollIntervalProvider(configStore, cfg))
	launch(func() { <-subscriptionDone })
	if runtime.stashBoxCacheService != nil {
		runtime.stashBoxCacheService.StartCleanup(ctx)
	}
	launch(func() { runtime.statsCollector.Run(ctx) })

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		manager.ClosePerformanceSubscriptions()
		runtime.taskEventBus.Close()
		runtime.serviceStatusEventBus.Close()
		runtime.performerSubscriptionEventBus.Close()
		if logger := logging.Default(); logger != nil {
			stats := logger.EventStats()
			slog.Info("log events: shutdown summary", "published", stats.Published, "dropped", stats.Dropped, "sequence", stats.Sequence, "subscribers", stats.Subscribers)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logging.Warnf("shutdown: %v", err)
			_ = server.Close()
		}
	}()

	if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.Fatalf("serve: %v", err)
	}
	stop()
	<-shutdownDone
	requests.Wait()
	workers.Wait()
	if service, ok := runtime.subscriptionService.(interface{ WaitBackground() }); ok {
		service.WaitBackground()
	}
	if service, ok := runtime.taskRuntimeService.(interface{ WaitBackground() }); ok {
		service.WaitBackground()
	}

}

func newHTTPHandler(cfg *config.Config, version string) http.Handler {
	return newHTTPRuntime(cfg, version, nil).handler
}

type httpRuntime struct {
	downloader                    *downloader.Service
	handler                       http.Handler
	taskRuntimeService            graphqlapi.TaskRuntimeService
	stashService                  graphqlapi.StashService
	subscriptionService           graphqlapi.SubscriptionService
	statsCollector                *stats.Collector
	taskEventBus                  *taskruntime.TaskEventBus
	serviceStatusEventBus         *stats.ServiceStatusEventBus
	performerSubscriptionEventBus *subscription.PerformerSubscriptionEventBus
	stashBoxCacheService          *stashboxcache.Service
}

func newHTTPRuntime(cfg *config.Config, version string, configStore *config.Store, managers ...*storage.Manager) *httpRuntime {
	var manager *storage.Manager
	if len(managers) > 0 {
		manager = managers[0]
	}
	imageService, err := imagecache.New("cache/image", func() imagecache.Config {
		current := cfg.System.ImageCache.Normalize()
		if configStore != nil {
			current = configStore.Config().System.ImageCache.Normalize()
		}
		return imagecache.Config{Enabled: current.EffectiveEnabled(), MaxSizeMB: current.MaxSizeMB, RetentionDays: current.RetentionDays}
	})
	if err != nil {
		logging.Fatalf("configure image cache: %v", err)
	}
	imageService.StartCleanup(context.Background())
	cachePath := fmt.Sprintf("file:moji-stashbox-cache-%d?mode=memory&cache=shared", runtimeCacheSequence.Add(1))
	cacheProvider := func() stashboxcache.Config {
		current := cfg.System.StashBoxDataCache.Normalize()
		if configStore != nil {
			current = configStore.Config().System.StashBoxDataCache.Normalize()
		}
		return stashboxcache.Config{TTL: time.Duration(current.TTLHours) * time.Hour, StaleRetention: stashboxcache.DefaultStaleRetention, MaxBytes: int64(current.MaxSizeMB) << 20}
	}
	var stashBoxCacheService *stashboxcache.Service
	if manager != nil {
		stashBoxCacheService = stashboxcache.NewManaged(manager.Cache, cacheProvider)
	} else {
		stashBoxCacheService, err = stashboxcache.New(cachePath, cacheProvider)
	}
	if err != nil {
		logging.Fatalf("configure StashBox data cache: %v", err)
	}
	jackettTracker := tracker.NewJackettService(configureJackettConfigProvider(configStore, cfg))
	{
		current := storeJackett(cfg, configStore)
		logging.Infof("runtime: jackett tracker configured for %s", current.URL)
		if current.Password == "" {
			logging.Warn("runtime: jackett.password is empty; the home service card will report Jackett as 运行异常 because /api/v2.0/indexers requires a session cookie. Set jackett.password in config.yaml to enable it.")
		}
	}
	apiHandler := api.NewHandler(jackettTracker, api.WithLogFilePath(cfg.EffectiveLogFilePath()))
	qbittorrentClient, torrentClient := configureQBittorrent(cfg, configStore)
	stashClient := configureStashClient(cfg, configStore)
	taskEventBus := taskruntime.NewTaskEventBus(32)
	taskRuntimeService := configureTaskRuntime(cfg, configStore, jackettTracker, torrentClient, stashClient, taskEventBus, manager)
	taskFlowService := configureTaskFlow(taskRuntimeService)
	stashService := configureStashService(cfg, configStore, stashClient)
	metadataService := configureMetadata(stashClient)
	if metadataService != nil {
		metadataService.SetCache(stashBoxCacheService)
	}
	performerSubscriptionEventBus := subscription.NewPerformerSubscriptionEventBus(16)
	subscriptionService := configureSubscription(cfg, configStore, stashClient, metadataService, taskFlowService, imageService, manager)
	if subscriptionService != nil {
		subscriptionService.SetEventPublisher(performerSubscriptionEventBus)
	}
	applyAutomationSettings(cfg, subscriptionService, metadataService)
	if taskFlowService != nil && metadataService != nil {
		taskFlowService.SetDiscoveredSceneResolver(discovery.NewDiscoveredSceneResolver(metadataService.Registry()))
	}

	statsCollector := stats.NewCollector(
		stashClient,
		jackettClientOf(jackettTracker),
		qbittorrentClient,
		taskRuntimeService,
		logging.Default().Slog(),
	)
	serviceStatusEventBus := stats.NewServiceStatusEventBus(8)
	statsCollector.SetEventPublisher(serviceStatusEventBus)
	resolver := graphqlapi.NewResolver(jackettTracker, torrentClient, taskRuntimeService, stashService, version)
	if manager != nil {
		resolver.DatabasePerformanceSource = manager
	}
	if qbittorrentClient != nil {
		if manager != nil {
			resolver.Downloader, err = downloader.NewManaged(qbittorrentClient, taskRuntimeService, manager)
		} else {
			resolver.Downloader, err = downloader.New(qbittorrentClient, taskRuntimeService, ":memory:")
		}
		if err != nil {
			logging.Fatalf("configure downloader: %v", err)
		}
	}
	resolver.TaskFlow = taskFlowService
	if metadataService != nil {
		stashImage := func(ctx context.Context, raw string) string {
			if imageService == nil || strings.TrimSpace(raw) == "" {
				return raw
			}
			current := storeStash(cfg, configStore)
			value, err := imageService.Register(ctx, imagecache.Descriptor{Kind: imagecache.SourceStash, InstanceURL: current.URL, ImageURL: raw, APIKey: current.APIKey})
			if err != nil {
				return ""
			}
			return value
		}
		stashBoxImage := func(ctx context.Context, endpoint, raw string) string {
			if imageService == nil || strings.TrimSpace(raw) == "" {
				return raw
			}
			value, err := imageService.Register(ctx, imagecache.Descriptor{Kind: imagecache.SourceStashBox, InstanceURL: endpoint, ImageURL: raw, APIKey: metadataService.APIKey(endpoint)})
			if err != nil {
				logging.Warnf("discovery: register image proxy: %v", err)
				return ""
			}
			return value
		}
		performerService, err := performer.NewService(stashClient, metadataService, taskFlowService, taskRuntimeService, stashImage, stashBoxImage)
		if err != nil {
			logging.Fatalf("configure performer: %v", err)
		}
		resolver.Performer = performerService
		resolver.Discovery = discovery.NewService(metadataService, taskFlowService, stashBoxImage)
	}
	if subscriptionService != nil {
		resolver.PerformerSubscription = subscriptionService
	}
	resolver.TaskEventSource = taskEventBus
	resolver.ServiceStatusEventSource = serviceStatusEventBus
	resolver.PerformerSubscriptionEventSource = performerSubscriptionEventBus
	resolver.LogEventSource = logging.Default()
	resolver.StashBox = metadataService
	resolver.LogReader = logging.Default()
	resolver.RuntimeSettings = buildSettingsSnapshot(cfg, version)
	resolver.RuntimeStatus = buildSettingsStatusSnapshot(cfg, version, taskRuntimeService != nil, stashService != nil, stashClient, metadataService)
	resolver.Stats = statsCollector
	resolver.ImageCache = imageService
	resolver.StashBoxDataCache = stashBoxCacheService
	if configStore != nil {
		resolver.SettingsEditor = newRuntimeSettingsEditor(configStore, version, taskRuntimeService != nil, stashService != nil, stashClient, subscriptionService, metadataService)
	}
	graphqlHandler := graphqlapi.NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	apiMux := http.NewServeMux()
	apiHandler.Register(apiMux)
	imageService.RegisterHandler(apiMux)
	webDist := os.Getenv("MOJI_WEB_DIST")
	if webDist == "" {
		webDist = "web/dist"
	}
	webHandler := webui.NewHandler(webDist)

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/graphql" && (r.Method == http.MethodPost || (r.Method == http.MethodGet && websocket.IsWebSocketUpgrade(r))):
			graphqlHandler.ServeHTTP(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/playground":
			playground.Handler("Moji GraphQL Playground", "/graphql").ServeHTTP(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/graphql":
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		case r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/healthz":
			webHandler.ServeHTTP(w, r)
		default:
			apiMux.ServeHTTP(w, r)
		}
	})

	return &httpRuntime{
		downloader:                    resolver.Downloader,
		handler:                       router,
		taskRuntimeService:            taskRuntimeService,
		stashService:                  stashService,
		subscriptionService:           resolver.PerformerSubscription,
		statsCollector:                statsCollector,
		taskEventBus:                  taskEventBus,
		serviceStatusEventBus:         serviceStatusEventBus,
		performerSubscriptionEventBus: performerSubscriptionEventBus,
		stashBoxCacheService:          stashBoxCacheService,
	}
}

func configureTaskFlow(taskRuntimeService graphqlapi.TaskRuntimeService) *taskflow.Service {
	if taskRuntimeService == nil {
		logging.Infof("runtime: taskflow service disabled because task runtime is not available")
		return nil
	}
	return taskflow.NewService(taskRuntimeService)
}

// configureJackettConfigProvider returns the latest JackettConfig on every
// invocation so Web UI edits to jackett.url / api_key / password take
// effect on the next search/indexer refresh without restarting Moji.
func configureJackettConfigProvider(store *config.Store, cfg *config.Config) tracker.JackettConfigProvider {
	return func() tracker.JackettConfig {
		current := storeJackett(cfg, store)
		return tracker.JackettConfig{
			URL:      current.URL,
			APIKey:   current.APIKey,
			Password: current.Password,
		}
	}
}

// storeJackett returns the latest Jackett config block. When a Store is
// available it always reflects the most recent Web UI write.
func storeJackett(cfg *config.Config, store *config.Store) *config.JackettConfig {
	if store != nil {
		return &store.Config().Connection.Jackett
	}
	return &cfg.Connection.Jackett
}

func configureQBittorrent(cfg *config.Config, store *config.Store) (*qbittorrent.Client, graphqlapi.TorrentClient) {
	if cfg.Connection.QBittorrent.URL == "" {
		logging.Infof("runtime: qBittorrent client disabled because qbittorrent.url is empty")
		return nil, nil
	}
	if cfg.Connection.QBittorrent.Username == "" {
		logging.Warn("qBittorrent disabled: qbittorrent.username is required when qbittorrent.url is set")
		return nil, nil
	}
	if cfg.Connection.QBittorrent.Password == "" {
		logging.Warn("qBittorrent disabled: qbittorrent.password is required when qbittorrent.url is set")
		return nil, nil
	}

	client := qbittorrent.NewClient(configureQBittorrentConfigProvider(store, cfg))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Login(ctx, cfg.Connection.QBittorrent.Username, cfg.Connection.QBittorrent.Password); err != nil {
		logging.Warnf("initial qBittorrent login failed; background monitoring will reconnect: %v", err)
	}
	logging.Infof("runtime: qBittorrent client configured for %s as %s", cfg.Connection.QBittorrent.URL, cfg.Connection.QBittorrent.Username)

	defaultsProvider := func() taskruntime.TorrentDefaults {
		current := storeQBittorrent(cfg, store)
		return taskruntime.TorrentDefaults{
			SavePath: current.DefaultSavePath,
			Category: current.Category,
			Tags:     current.Tags,
		}
	}
	wrapped := taskruntime.NewDefaultingTorrentClient(client, defaultsProvider)
	return client, wrapped
}

// storeQBittorrent returns the latest qBittorrent config block. When a Store
// is available it always reflects the most recent Web UI write, so callers
// using it inside a provider see live edits without a restart.
func storeQBittorrent(cfg *config.Config, store *config.Store) *config.QBittorrentConfig {
	if store != nil {
		return &store.Config().Connection.QBittorrent
	}
	return &cfg.Connection.QBittorrent
}

// jackettClientOf returns the underlying jackett.Client for the stats
// collector. Returns nil if the tracker is not a JackettService (e.g. tests
// pass a stub).
func jackettClientOf(tr tracker.Tracker) *jackett.Client {
	if j, ok := tr.(*tracker.JackettService); ok {
		return j.Client()
	}
	return nil
}

func configureTaskRuntime(cfg *config.Config, configStore *config.Store, tr tracker.Tracker, torrent graphqlapi.TorrentClient, stashClient *stash.Client, taskEvents *taskruntime.TaskEventBus, managers ...*storage.Manager) graphqlapi.TaskRuntimeService {
	if torrent == nil {
		logging.Infof("runtime: task runtime disabled because qBittorrent client is not available")
		return nil
	}

	store, err := configureTaskStore(cfg, managers...)
	if err != nil {
		logging.Fatalf("configure task store: %v", err)
	}
	eventingStore, err := taskruntime.NewEventingTaskStore(store, taskEvents)
	if err != nil {
		logging.Fatalf("configure task event store: %v", err)
	}
	service, err := taskruntime.NewService(
		tr,
		torrent,
		eventingStore,
		taskruntime.WithSourcingRetryInterval(func() time.Duration {
			current := cfg
			if configStore != nil {
				current = configStore.Config()
			}
			return time.Duration(current.Automation.EffectiveSourcingRetryIntervalHours()) * time.Hour
		}),
		taskruntime.WithCandidateSelectionProvider(configureTorrentSelectionProvider(configStore, cfg)),
		taskruntime.WithTaskDeletePolicyProvider(configureTaskDeletePolicyProvider(configStore, cfg)),
		taskruntime.WithLibraryCodeChecker(stashLibraryCodeChecker{client: stashClient}),
		taskruntime.WithMediaPreparation(func() stashsync.IntegrationConfig {
			current := cfg
			if configStore != nil {
				current = configStore.Config()
			}
			return stashIntegrationConfig(current)
		}, nil),
	)
	if err != nil {
		logging.Fatalf("configure task runtime: %v", err)
	}
	logging.Infof("runtime: task runtime service initialized")
	return service
}

func configureTorrentSelectionProvider(store *config.Store, cfg *config.Config) func() config.TorrentSelectionConfig {
	return func() config.TorrentSelectionConfig {
		current := cfg
		if store != nil {
			current = store.Config()
		}
		return current.Automation.TorrentSelection.Effective()
	}
}

func configureTaskStore(cfg *config.Config, managers ...*storage.Manager) (taskruntime.TaskStore, error) {
	if len(managers) > 0 && managers[0] != nil {
		return taskruntime.NewRelationalTaskStore(managers[0].Core, managers[0].Observations), nil
	}
	return taskruntime.NewMemoryTaskStore(), nil
}

func configureProgressSyncInterval(cfg *config.Config) time.Duration {
	seconds := cfg.Automation.TaskProgressSyncIntervalSeconds
	if !cfg.Automation.EffectiveTaskProcessingEnabled() {
		return 0
	}
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

// configureProgressSyncIntervalProvider returns the current task sync
// interval, honoring the latest Web UI edit each time it is called. A return
// value <= 0 disables the worker.
func configureProgressSyncIntervalProvider(store *config.Store, cfg *config.Config) func() time.Duration {
	return func() time.Duration {
		return configureProgressSyncInterval(storeAutomation(cfg, store))
	}
}

func configureTaskDeletePolicyProvider(store *config.Store, cfg *config.Config) func() config.TaskDeletePolicy {
	return func() config.TaskDeletePolicy {
		if store != nil {
			return store.Config().System.EffectiveTaskDeletePolicy()
		}
		return cfg.System.EffectiveTaskDeletePolicy()
	}
}

type stashLibraryCodeChecker struct {
	client *stash.Client
}

func (c stashLibraryCodeChecker) HasCode(ctx context.Context, code string) (bool, error) {
	if c.client == nil {
		return false, nil
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	page := 1
	perPage := 1
	scenes, err := c.client.FindScenes(ctx, &stashgraphql.SceneFilterType{
		Code: &stashgraphql.StringCriterionInput{
			Value:    code,
			Modifier: stashgraphql.CriterionModifierEquals,
		},
	}, &stashgraphql.FindFilterType{
		Page:    &page,
		PerPage: &perPage,
	})
	if err != nil {
		return false, err
	}
	return len(scenes) > 0, nil
}

// configureQBittorrentConfigProvider returns the latest qBittorrent config
// block, honoring Web UI edits each time it is called. Used by the
// qbittorrent.Client to lazily rebuild its http client and base URL when the
// URL or credentials change.
func configureQBittorrentConfigProvider(store *config.Store, cfg *config.Config) qbittorrent.ConfigProvider {
	return func() qbittorrent.Config {
		current := storeQBittorrent(cfg, store)
		return qbittorrent.Config{
			URL:      current.URL,
			Username: current.Username,
			Password: current.Password,
		}
	}
}

func applyAutomationSettings(cfg *config.Config, subscriptions *subscription.Service, metadataService *metadata.Service) {
	if cfg == nil {
		return
	}
	if metadataService != nil {
		metadataService.SetEndpointOrder(cfg.Automation.StashBoxEndpoints)
	}
	if subscriptions != nil {
		subscriptions.SetReleasePolicy(cfg.Automation.SubscriptionReleasePolicy.Effective())
	}
}

func buildSettingsSnapshot(cfg *config.Config, version string) *graphqlapi.SettingsSnapshot {
	_ = version
	jackettConfigured := cfg.Connection.Jackett.URL != "" && cfg.Connection.Jackett.APIKey != ""
	stashConfigured := isStashConfigured(cfg)
	qbittorrentConfigured := cfg.Connection.QBittorrent.URL != "" && cfg.Connection.QBittorrent.Username != "" && cfg.Connection.QBittorrent.Password != ""

	return &graphqlapi.SettingsSnapshot{
		Stash: graphqlapi.StashSettingsSnapshot{
			Configured:       stashConfigured,
			URL:              cfg.Connection.Stash.URL,
			APIKeyConfigured: cfg.Connection.Stash.APIKey != "",
			APIKey:           cfg.Connection.Stash.APIKey,
		},
		Ingest: graphqlapi.IngestSettingsSnapshot{
			GenerateTranscodes: cfg.Ingest.GenerateTranscodes,
			KeepSubtitles:      cfg.Ingest.KeepSubtitles,
			Downloads: graphqlapi.DownloadsIngestSettingsSnapshot{
				QBRoot:   cfg.Ingest.Downloads.QBRoot,
				MojiRoot: cfg.Ingest.Downloads.MojiRoot,
			},
			Library: graphqlapi.LibraryIngestSettingsSnapshot{
				MojiRoot:  cfg.Ingest.Library.MojiRoot,
				StashRoot: cfg.Ingest.Library.StashRoot,
			},
			Transfer: graphqlapi.TransferIngestSettingsSnapshot{
				Action: cfg.Ingest.Transfer.Action,
			},
		},
		Jackett: graphqlapi.JackettSettingsSnapshot{
			Configured:         jackettConfigured,
			URL:                cfg.Connection.Jackett.URL,
			APIKeyConfigured:   cfg.Connection.Jackett.APIKey != "",
			APIKey:             cfg.Connection.Jackett.APIKey,
			PasswordConfigured: cfg.Connection.Jackett.Password != "",
			Password:           cfg.Connection.Jackett.Password,
		},
		QBittorrent: graphqlapi.QBittorrentSettingsSnapshot{
			Configured:         qbittorrentConfigured,
			URL:                cfg.Connection.QBittorrent.URL,
			Username:           cfg.Connection.QBittorrent.Username,
			UsernameConfigured: cfg.Connection.QBittorrent.Username != "",
			PasswordConfigured: cfg.Connection.QBittorrent.Password != "",
			Password:           cfg.Connection.QBittorrent.Password,
			DefaultSavePath:    cfg.Connection.QBittorrent.DefaultSavePath,
			Category:           cfg.Connection.QBittorrent.Category,
			Tags:               cfg.Connection.QBittorrent.Tags,
		},
		Automation: graphqlapi.AutomationSettingsSnapshot{
			TaskProcessingEnabled:           cfg.Automation.EffectiveTaskProcessingEnabled(),
			SourcingRetryIntervalHours:      cfg.Automation.EffectiveSourcingRetryIntervalHours(),
			TaskProgressSyncIntervalSeconds: effectiveTaskProgressSyncIntervalSeconds(cfg),
			SubscriptionPollIntervalHours:   effectiveSubscriptionPollIntervalHours(cfg),
			StashBoxEndpoints:               append([]string(nil), cfg.Automation.StashBoxEndpoints...),
			SubscriptionReleasePolicy: graphqlapi.SubscriptionReleasePolicySnapshot{
				SoloBehavior:           string(cfg.Automation.SubscriptionReleasePolicy.Effective().SoloBehavior),
				GroupBehavior:          string(cfg.Automation.SubscriptionReleasePolicy.Effective().GroupBehavior),
				CompilationBehavior:    string(cfg.Automation.SubscriptionReleasePolicy.Effective().CompilationBehavior),
				VRBehavior:             string(cfg.Automation.SubscriptionReleasePolicy.Effective().VRBehavior),
				MaxGroupPerformerCount: cfg.Automation.SubscriptionReleasePolicy.Effective().MaxGroupPerformerCount,
				ReleaseDateRange:       string(cfg.Automation.SubscriptionReleasePolicy.Effective().ReleaseDateRange),
			},
			TorrentSelection: torrentSelectionSnapshot(cfg.Automation.TorrentSelection.Effective()),
		},
		System: graphqlapi.SystemSettingsSnapshot{
			DeveloperMode:     cfg.System.DeveloperMode,
			TaskDeletePolicy:  string(cfg.System.EffectiveTaskDeletePolicy()),
			ImageCache:        graphqlapi.ImageCacheSettingsSnapshot{Enabled: cfg.System.ImageCache.EffectiveEnabled(), MaxSizeMB: cfg.System.ImageCache.Normalize().MaxSizeMB, RetentionDays: cfg.System.ImageCache.Normalize().RetentionDays},
			StashBoxDataCache: graphqlapi.StashBoxDataCacheSettingsSnapshot{TTLHours: cfg.System.StashBoxDataCache.Normalize().TTLHours},
		},
	}
}

func torrentSelectionSnapshot(cfg config.TorrentSelectionConfig) graphqlapi.TorrentSelectionSettingsSnapshot {
	cfg = cfg.Effective()
	orderedRules := cfg.OrderedRules()
	fastRuleTypes := make(map[config.TorrentSelectionRuleType]struct{}, len(cfg.FastRuleOrder))
	for _, ruleType := range cfg.FastRuleOrder {
		fastRuleTypes[ruleType] = struct{}{}
	}
	out := graphqlapi.TorrentSelectionSettingsSnapshot{
		Enabled:                  cfg.Enabled,
		InspectionCandidateLimit: cfg.InspectionCandidateLimit,
		FastRules:                make([]graphqlapi.TorrentSelectionRuleSnapshot, 0, len(cfg.FastRuleOrder)),
		TorrentRules:             make([]graphqlapi.TorrentSelectionRuleSnapshot, 0, len(orderedRules)),
	}
	for _, rule := range orderedRules {
		item := graphqlapi.TorrentSelectionRuleSnapshot{
			Type:    string(rule.Type),
			Enabled: rule.Enabled,
			IndexerPreference: graphqlapi.IndexerPreferenceRuleSnapshot{
				TrackerIDs: append([]string(nil), rule.IndexerPreference.TrackerIDs...),
			},
			PublishDate: graphqlapi.DirectionRuleSnapshot{
				Direction: string(rule.PublishDate.Direction),
			},
			Seeders: graphqlapi.DirectionRuleSnapshot{
				Direction: string(rule.Seeders.Direction),
			},
			Size: graphqlapi.DirectionRuleSnapshot{
				Direction: string(rule.Size.Direction),
			},
		}
		if len(rule.TitleMatch.Clauses) > 0 {
			item.TitleMatch.Clauses = make([]graphqlapi.TitleMatchClauseSnapshot, 0, len(rule.TitleMatch.Clauses))
			for _, clause := range rule.TitleMatch.Clauses {
				item.TitleMatch.Clauses = append(item.TitleMatch.Clauses, graphqlapi.TitleMatchClauseSnapshot{
					Pattern:     clause.Pattern,
					PatternMode: string(clause.PatternMode),
					Effect:      string(clause.Effect),
				})
			}
		}
		if len(rule.TorrentFileNameMatch.Clauses) > 0 {
			item.TorrentFileNameMatch.Clauses = make([]graphqlapi.TorrentFileNameMatchClauseSnapshot, 0, len(rule.TorrentFileNameMatch.Clauses))
			for _, clause := range rule.TorrentFileNameMatch.Clauses {
				item.TorrentFileNameMatch.Clauses = append(item.TorrentFileNameMatch.Clauses, graphqlapi.TorrentFileNameMatchClauseSnapshot{
					Pattern:     clause.Pattern,
					PatternMode: string(clause.PatternMode),
					Effect:      string(clause.Effect),
				})
			}
		}
		if _, ok := fastRuleTypes[rule.Type]; ok {
			out.FastRules = append(out.FastRules, item)
		} else {
			out.TorrentRules = append(out.TorrentRules, item)
		}
	}
	return out
}

func buildSettingsStatusSnapshot(cfg *config.Config, version string, downloaderEnabled bool, stashEnabled bool, stashClient *stash.Client, stashBoxService graphqlapi.StashBoxService) *graphqlapi.SettingsStatusSnapshot {
	_ = version
	jackettConfigured := cfg.Connection.Jackett.URL != "" && cfg.Connection.Jackett.APIKey != ""
	stashConfigured := isStashConfigured(cfg)
	ingestConfigured := isIngestConfigured(cfg)
	qbittorrentConfigured := cfg.Connection.QBittorrent.URL != "" && cfg.Connection.QBittorrent.Username != "" && cfg.Connection.QBittorrent.Password != ""

	stashBoxStatus := graphqlapi.StashBoxStatusSnapshot{
		StashBoxes:          []graphqlapi.StashBoxEndpointSnapshot{},
		StashBoxesLoaded:    false,
		StashBoxesLoadError: "",
	}
	if stashBoxService != nil {
		endpoints, state := stashBoxService.SnapshotState()
		stashBoxStatus.StashBoxesLoaded = state.Loaded
		stashBoxStatus.StashBoxesLoadError = state.ErrorMsg
		for _, box := range endpoints {
			stashBoxStatus.StashBoxes = append(stashBoxStatus.StashBoxes, graphqlapi.StashBoxEndpointSnapshot{
				Name:             box.Name,
				Endpoint:         box.Endpoint,
				APIKeyConfigured: box.APIKeyConfigured,
			})
		}
	}

	stashLibraries, stashLibrariesLoadError := loadStashLibraries(stashConfigured, stashClient)

	return &graphqlapi.SettingsStatusSnapshot{
		Stash: graphqlapi.ServiceStatusSnapshot{
			Configured: stashConfigured,
			Ready:      stashConfigured,
		},
		Ingest: graphqlapi.IngestStatusSnapshot{
			Configured: ingestConfigured,
		},
		Jackett: graphqlapi.ServiceStatusSnapshot{
			Configured: jackettConfigured,
			Ready:      jackettConfigured,
		},
		QBittorrent: graphqlapi.ServiceStatusSnapshot{
			Configured: qbittorrentConfigured,
			Ready:      qbittorrentConfigured,
		},
		Automation: graphqlapi.AutomationStatusSnapshot{
			TaskProgressSyncIntervalSeconds: effectiveTaskProgressSyncIntervalSeconds(cfg),
			TaskProgressSyncEnabled:         cfg.Automation.EffectiveTaskProcessingEnabled() && downloaderEnabled,
			SubscriptionPollIntervalHours:   effectiveSubscriptionPollIntervalHours(cfg),
			SubscriptionPollEnabled:         cfg.Automation.SubscriptionPollIntervalHours >= 0 && stashEnabled,
		},
		StashBox:                stashBoxStatus,
		StashLibraries:          stashLibraries,
		StashLibrariesLoadError: stashLibrariesLoadError,
	}
}

func loadStashLibraries(stashConfigured bool, stashClient *stash.Client) ([]graphqlapi.StashLibrarySnapshot, string) {
	if !stashConfigured || stashClient == nil {
		return []graphqlapi.StashLibrarySnapshot{}, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	libraries, err := stashClient.GetStashLibraries(ctx)
	if err != nil {
		logging.Warnf("runtime: failed to load stash libraries: %v", err)
		return []graphqlapi.StashLibrarySnapshot{}, err.Error()
	}

	out := make([]graphqlapi.StashLibrarySnapshot, 0, len(libraries))
	for _, library := range libraries {
		path := strings.TrimSpace(library.Path)
		if path == "" {
			continue
		}
		out = append(out, graphqlapi.StashLibrarySnapshot{Path: path})
	}
	return out, ""
}

func startTaskSyncWorker(ctx context.Context, service graphqlapi.TaskRuntimeService, stash graphqlapi.StashService, intervalProvider func() time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if service == nil || intervalProvider == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		// Keep observing configuration while paused so enabling never needs a restart.
		check := time.NewTicker(500 * time.Millisecond)
		defer check.Stop()
		timer := time.NewTimer(time.Hour)
		defer timer.Stop()
		var tick <-chan time.Time
		schedule := func(delay time.Duration, enabled bool) {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			tick = nil
			if enabled {
				timer.Reset(delay)
				tick = timer.C
			}
		}
		current := intervalProvider()
		schedule(current, current > 0)
		for {
			select {
			case <-ctx.Done():
				return
			case <-check.C:
				next := intervalProvider()
				if next != current {
					delay := next
					if current <= 0 && next > 0 {
						delay = 0
					}
					current = next
					schedule(delay, next > 0)
				}
			case <-tick:
				next := intervalProvider()
				if next != current {
					current = next
					schedule(next, next > 0)
					continue
				}
				syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				syncProgress := service.SyncProgress
				if bounded, ok := service.(interface {
					SyncActiveProgress(context.Context) error
				}); ok {
					syncProgress = func(ctx context.Context) ([]*taskruntime.Task, error) { return nil, bounded.SyncActiveProgress(ctx) }
				}
				if _, err := syncProgress(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
					logging.Errorf("sync download progress: %v", err)
				}
				if stash != nil {
					triggerScans := func() error { _, err := service.TriggerStashScans(syncCtx, stash); return err }
					if bounded, ok := service.(interface {
						TriggerActiveStashScans(context.Context, taskruntime.StashScanner) error
					}); ok {
						triggerScans = func() error { return bounded.TriggerActiveStashScans(syncCtx, stash) }
					}
					if err := triggerScans(); err != nil && !errors.Is(err, context.Canceled) {
						logging.Errorf("trigger stash scans: %v", err)
					}
				}
				cancel()
				current = intervalProvider()
				schedule(current, current > 0)
			}
		}
	}()
	return done
}

func configureStashClient(cfg *config.Config, store *config.Store) *stash.Client {
	current := storeStash(cfg, store)
	graphqlURL := current.GraphQLEndpoint()
	if graphqlURL == "" {
		logging.Infof("runtime: stash client disabled because stash.url is empty")
		return nil
	}
	logging.Infof("runtime: stash client configured for %s", graphqlURL)
	return stash.NewClient(configureStashConfigProvider(store, cfg))
}

// configureStashConfigProvider returns the latest Stash config block on
// every invocation so Web UI edits to stash.url / api_key take effect on
// the next GraphQL call without restarting Moji.
func configureStashConfigProvider(store *config.Store, cfg *config.Config) stash.ConfigProvider {
	return func() stash.Config {
		current := storeStash(cfg, store)
		return stash.Config{
			URL:    current.GraphQLEndpoint(),
			APIKey: current.APIKey,
		}
	}
}

// storeStash returns the latest Stash config block. When a Store is
// available it always reflects the most recent Web UI write.
func storeStash(cfg *config.Config, store *config.Store) *config.StashConfig {
	if store != nil {
		return &store.Config().Connection.Stash
	}
	return &cfg.Connection.Stash
}

func configureStashService(cfg *config.Config, store *config.Store, client *stash.Client) graphqlapi.StashService {
	if client == nil {
		logging.Infof("runtime: stash sync service disabled because stash client is not available")
		return nil
	}

	configProvider := func() stashsync.IntegrationConfig {
		current := cfg
		if store != nil {
			current = store.Config()
		}
		return stashIntegrationConfig(current)
	}
	service, err := stashsync.NewService(client, configProvider)
	if err != nil {
		logging.Fatalf("configure Stash: %v", err)
	}

	logging.Infof("runtime: stash sync service initialized")
	return service
}

func stashIntegrationConfig(cfg *config.Config) stashsync.IntegrationConfig {
	if cfg == nil {
		return stashsync.IntegrationConfig{}
	}
	return stashsync.IntegrationConfig{
		GenerateTranscodes: cfg.Ingest.GenerateTranscodes,
		KeepSubtitles:      cfg.Ingest.KeepSubtitles,
		Downloads: stashsync.DownloadsPathConfig{
			QBRoot:   strings.TrimSpace(cfg.Ingest.Downloads.QBRoot),
			MojiRoot: strings.TrimSpace(cfg.Ingest.Downloads.MojiRoot),
		},
		Library: stashsync.LibraryPathConfig{
			MojiRoot:  strings.TrimSpace(cfg.Ingest.Library.MojiRoot),
			StashRoot: strings.TrimSpace(cfg.Ingest.Library.StashRoot),
		},
		Transfer: stashsync.TransferConfig{
			Action: stashsync.TransferAction(strings.TrimSpace(cfg.Ingest.Transfer.Action)),
		},
	}
}

// isStashConfigured reports whether the Stash server has the minimum
// connection fields required by Moji. Stash can be deployed without an API
// key, so URL alone is enough to treat the service as configured.
func isStashConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.Connection.Stash.URL) != ""
}

// isIngestConfigured reports whether file preparation and delivery are fully configured.
func isIngestConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	action := strings.TrimSpace(cfg.Ingest.Transfer.Action)
	return (action == "COPY" || action == "MOVE") &&
		strings.TrimSpace(cfg.Ingest.Downloads.QBRoot) != "" &&
		strings.TrimSpace(cfg.Ingest.Downloads.MojiRoot) != "" &&
		strings.TrimSpace(cfg.Ingest.Library.MojiRoot) != "" &&
		strings.TrimSpace(cfg.Ingest.Library.StashRoot) != ""
}

func configureMetadata(stashClient *stash.Client) *metadata.Service {
	if stashClient == nil {
		logging.Infof("runtime: StashBox metadata service disabled because stash client is not available")
		return nil
	}

	registry := metadata.NewDefaultRegistry()
	service := metadata.NewService(stashClient, registry)
	refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.RefreshStashBoxes(refreshCtx); err != nil {
		logging.Warnf("runtime: failed to refresh stash-box endpoints at startup: %v", err)
	} else {
		logging.Infof("runtime: StashBox metadata service initialized with %d endpoint(s) from Stash", len(registry.Endpoints()))
	}
	return service
}

func configureSubscription(cfg *config.Config, configStore *config.Store, stashClient *stash.Client, metadataService *metadata.Service, taskFlowService *taskflow.Service, imageService *imagecache.Service, managers ...*storage.Manager) *subscription.Service {
	if stashClient == nil || metadataService == nil {
		logging.Infof("runtime: subscription service disabled because required metadata services are not available")
		return nil
	}

	store, err := configureSubscriptionStore(cfg, managers...)
	if err != nil {
		logging.Fatalf("configure subscription store: %v", err)
	}

	service, err := subscription.NewService(stashClient, metadataService, taskFlowService, store)
	if err != nil {
		logging.Fatalf("configure subscription: %v", err)
	}
	service.SetImageProxy(imageService, func() (string, string) { current := storeStash(cfg, configStore); return current.URL, current.APIKey })
	return service
}

func configureSubscriptionStore(cfg *config.Config, managers ...*storage.Manager) (subscription.Store, error) {
	if len(managers) > 0 && managers[0] != nil {
		return subscription.NewManagedStore(managers[0].Core), nil
	}
	_ = cfg
	return subscription.NewMemoryStore(), nil
}

func configureSubscriptionPollInterval(cfg *config.Config) time.Duration {
	hours := cfg.Automation.SubscriptionPollIntervalHours
	if hours < 0 {
		return 0
	}
	if hours == 0 {
		hours = 1
	}
	return time.Duration(hours) * time.Hour
}

// configureSubscriptionPollIntervalProvider returns the current subscription
// poll interval, honoring the latest Web UI edit each time it is called.
// A return value <= 0 disables the worker.
func configureSubscriptionPollIntervalProvider(store *config.Store, cfg *config.Config) func() time.Duration {
	return func() time.Duration {
		return configureSubscriptionPollInterval(storeAutomation(cfg, store))
	}
}

// storeAutomation returns the latest automation config block, falling back to
// the startup snapshot when no Store is wired up.
func storeAutomation(cfg *config.Config, store *config.Store) *config.Config {
	if store != nil {
		return store.Config()
	}
	return cfg
}

func startSubscriptionWorker(ctx context.Context, service graphqlapi.SubscriptionService, intervalProvider func() time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if service == nil || intervalProvider == nil {
		close(done)
		return done
	}
	tracked, hasTracking := service.(interface {
		ConfigureCheckRuntime(context.Context)
		SetCheckSchedule(bool, *time.Time)
		StartSubscriptionCheck(context.Context, []string, string) (*subscription.CheckRun, error)
	})
	if hasTracking {
		tracked.ConfigureCheckRuntime(ctx)
	}
	current := intervalProvider()
	var deadline *time.Time
	publishSchedule := func(interval time.Duration) {
		deadline = nil
		if interval > 0 {
			next := time.Now().Add(interval)
			deadline = &next
		}
		if hasTracking {
			tracked.SetCheckSchedule(interval > 0, deadline)
		}
	}
	publishSchedule(current)
	go func() {
		defer close(done)
		// Reconcile settings even while disabled, so enabling polling needs no restart.
		settingsTicker := time.NewTicker(5 * time.Second)
		defer settingsTicker.Stop()
		timer := time.NewTimer(time.Hour)
		if !timer.Stop() {
			<-timer.C
		}
		defer timer.Stop()
		arm := func() {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if deadline != nil {
				timer.Reset(time.Until(*deadline))
			}
		}
		arm()
		for {
			select {
			case <-ctx.Done():
				if hasTracking {
					tracked.SetCheckSchedule(false, nil)
				}
				return
			case <-settingsTicker.C:
				next := intervalProvider()
				if next != current {
					current = next
					publishSchedule(current)
					arm()
				}
			case <-timer.C:
				// Apply a just-disabled setting before dispatching another check.
				current = intervalProvider()
				delay := current
				if current > 0 {
					if hasTracking {
						if _, err := tracked.StartSubscriptionCheck(ctx, nil, "SCHEDULED"); err != nil && !errors.Is(err, context.Canceled) {
							var conflict *subscription.CheckConflictError
							if errors.As(err, &conflict) {
								// Retry soon after selected manual checks, instead of missing a whole polling interval.
								delay = min(current, 30*time.Second)
								logging.Infof("runtime: scheduled subscription check deferred by active run %s", conflict.RunID)
							} else {
								logging.Errorf("refresh subscription performers: %v", err)
							}
						}
					} else {
						syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
						if _, err := service.RefreshAll(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
							logging.Errorf("refresh subscription performers: %v", err)
						}
						cancel()
					}
				}
				publishSchedule(delay)
				arm()
			}
		}
	}()
	return done
}

func effectiveTaskProgressSyncIntervalSeconds(cfg *config.Config) int {
	seconds := cfg.Automation.TaskProgressSyncIntervalSeconds
	if seconds <= 0 {
		return 60
	}
	return seconds
}

func effectiveSubscriptionPollIntervalHours(cfg *config.Config) int {
	hours := cfg.Automation.SubscriptionPollIntervalHours
	if hours < 0 {
		return 0
	}
	if hours == 0 {
		return 1
	}
	return hours
}
