package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/controller/api"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/subscription"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/internal/webui"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"github.com/leothevan2444/moji/pkg/stash"
)

func main() {
	var (
		configPath = flag.String("config", "", "path to config yaml (or set MOJI_CONFIG)")
		addr       = flag.String("addr", ":10000", "http listen address")
	)
	flag.Parse()

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

	if cfg.Jackett.URL == "" {
		logging.Fatalf("invalid config: jackett.url is required")
	}
	if cfg.Jackett.APIKey == "" {
		logging.Fatalf("invalid config: jackett.api_key is required")
	}

	// Dependencies
	mux, downloaderService, stashService, subscriptionService := newHTTPRuntime(cfg, "dev", configStore)

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		logging.Fatalf("listen %s: %v", server.Addr, err)
	}

	logging.Infof("moji listening on %s", ln.Addr().String())

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startTaskSyncWorker(ctx, downloaderService, stashService, configureProgressSyncInterval(cfg))
	startSubscriptionWorker(ctx, subscriptionService, configureSubscriptionPollInterval(cfg))

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logging.Warnf("shutdown: %v", err)
		}
	}()

	if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.Fatalf("serve: %v", err)
	}
}

func newHTTPHandler(cfg *config.Config, version string) http.Handler {
	handler, _, _, _ := newHTTPRuntime(cfg, version, nil)
	return handler
}

func newHTTPRuntime(cfg *config.Config, version string, configStore *config.Store) (http.Handler, graphqlapi.DownloaderService, graphqlapi.StashService, graphqlapi.SubscriptionService) {
	jackettTracker := tracker.NewJackettService(cfg.Jackett.URL, cfg.Jackett.APIKey)
	logging.Infof("runtime: jackett tracker configured for %s", cfg.Jackett.URL)
	apiHandler := api.NewHandler(jackettTracker, api.WithLogFilePath(cfg.EffectiveLogFilePath()))
	torrentClient := configureQBittorrent(cfg)
	downloaderService := configureDownloader(cfg, jackettTracker, torrentClient)
	stashClient := configureStashClient(cfg)
	stashService := configureStashService(cfg, stashClient)
	subscriptionService := configureSubscription(cfg, stashClient, downloaderService)
	applySubscriptionOrder(cfg, subscriptionService)
	resolver := graphqlapi.NewResolver(jackettTracker, torrentClient, downloaderService, stashService, version)
	resolver.Subscription = subscriptionService
	resolver.LogReader = logging.Default()
	resolver.RuntimeSettings = buildSettingsSnapshot(cfg, version, torrentClient != nil, downloaderService != nil, stashService != nil, subscriptionService)
	if configStore != nil {
		resolver.SettingsEditor = newRuntimeSettingsEditor(configStore, version, torrentClient != nil, downloaderService != nil, stashService != nil, subscriptionService)
	}
	graphqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	apiMux := http.NewServeMux()
	apiHandler.Register(apiMux)
	webHandler := webui.NewHandler("web/dist")

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
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

	return router, downloaderService, stashService, subscriptionService
}

func configureQBittorrent(cfg *config.Config) graphqlapi.TorrentClient {
	if cfg.QBittorrent.URL == "" {
		logging.Infof("runtime: qBittorrent client disabled because qbittorrent.url is empty")
		return nil
	}
	if cfg.QBittorrent.Username == "" {
		logging.Warn("qBittorrent disabled: qbittorrent.username is required when qbittorrent.url is set")
		return nil
	}
	if cfg.QBittorrent.Password == "" {
		logging.Warn("qBittorrent disabled: qbittorrent.password is required when qbittorrent.url is set")
		return nil
	}

	client := qbittorrent.NewClient(cfg.QBittorrent.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Login(ctx, cfg.QBittorrent.Username, cfg.QBittorrent.Password); err != nil {
		logging.Fatalf("login qBittorrent: %v", err)
	}
	logging.Infof("runtime: qBittorrent client connected to %s as %s", cfg.QBittorrent.URL, cfg.QBittorrent.Username)

	return downloader.NewDefaultingTorrentClient(client, downloader.TorrentDefaults{
		SavePath: cfg.QBittorrent.DefaultSavePath,
		Category: cfg.QBittorrent.Category,
		Tags:     cfg.QBittorrent.Tags,
	})
}

func configureDownloader(cfg *config.Config, tr tracker.Tracker, torrent graphqlapi.TorrentClient) graphqlapi.DownloaderService {
	if torrent == nil {
		logging.Infof("runtime: downloader disabled because qBittorrent client is not available")
		return nil
	}

	store, err := configureTaskStore(cfg)
	if err != nil {
		logging.Fatalf("configure task store: %v", err)
	}
	service, err := downloader.NewService(tr, torrent, store)
	if err != nil {
		logging.Fatalf("configure downloader: %v", err)
	}
	logging.Infof("runtime: downloader service initialized")
	return service
}

func configureTaskStore(cfg *config.Config) (downloader.TaskStore, error) {
	switch cfg.Tasks.Store {
	case "", "sqlite":
		path := strings.TrimSpace(cfg.Tasks.DBPath)
		if path == "" {
			path = "moji.db"
		}
		return downloader.NewSQLiteTaskStore(path)
	case "memory":
		return downloader.NewMemoryTaskStore(), nil
	default:
		return nil, fmt.Errorf("unsupported tasks.store %q", cfg.Tasks.Store)
	}
}

func configureProgressSyncInterval(cfg *config.Config) time.Duration {
	seconds := cfg.Tasks.ProgressSyncIntervalSeconds
	if seconds < 0 {
		return 0
	}
	if seconds == 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func applySubscriptionOrder(cfg *config.Config, service graphqlapi.SubscriptionService) {
	if service == nil || cfg == nil {
		return
	}
	if concrete, ok := service.(*subscription.Service); ok {
		concrete.SetEndpointOrder(cfg.Subscription.StashBoxEndpoints)
	}
}

func buildSubscriptionSnapshot(cfg *config.Config, subscriptionService graphqlapi.SubscriptionService, store, dbPath string, pollIntervalSeconds int, pollEnabled bool) graphqlapi.SubscriptionSettingsSnapshot {
	out := graphqlapi.SubscriptionSettingsSnapshot{
		Store:               store,
		DBPath:              dbPath,
		PollIntervalSeconds: pollIntervalSeconds,
		PollEnabled:         pollEnabled,
		StashBoxes:          []graphqlapi.StashBoxEndpointSnapshot{},
		StashBoxEndpoints:   append([]string(nil), cfg.Subscription.StashBoxEndpoints...),
		StashBoxesLoaded:    false,
		StashBoxesLoadError: "",
	}
	if subscriptionService == nil {
		return out
	}
	endpoints, state := subscriptionService.SnapshotState()
	out.StashBoxesLoaded = state.Loaded
	out.StashBoxesLoadError = state.ErrorMsg
	if len(endpoints) == 0 {
		return out
	}
	for _, box := range endpoints {
		out.StashBoxes = append(out.StashBoxes, graphqlapi.StashBoxEndpointSnapshot{
			Name:             box.Name,
			Endpoint:         box.Endpoint,
			APIKeyConfigured: box.APIKeyConfigured,
		})
	}
	return out
}

func buildSettingsSnapshot(cfg *config.Config, version string, qbittorrentEnabled bool, downloaderEnabled bool, stashEnabled bool, subscriptionService graphqlapi.SubscriptionService) *graphqlapi.SettingsSnapshot {
	tasksStore := cfg.Tasks.Store
	if tasksStore == "" {
		tasksStore = "sqlite"
	}

	dbPath := cfg.Tasks.DBPath
	if dbPath == "" {
		dbPath = "moji.db"
	}

	progressSyncSeconds := cfg.Tasks.ProgressSyncIntervalSeconds
	progressSyncEnabled := progressSyncSeconds >= 0
	if progressSyncSeconds == 0 {
		progressSyncSeconds = 60
	}
	if progressSyncSeconds < 0 {
		progressSyncSeconds = 0
	}

	subscriptionStore := cfg.Subscription.Store
	if subscriptionStore == "" {
		subscriptionStore = "sqlite"
	}

	subscriptionDBPath := cfg.Subscription.DBPath
	if subscriptionDBPath == "" {
		dir := "."
		if cfg.Tasks.DBPath != "" {
			dir = filepath.Dir(cfg.Tasks.DBPath)
		}
		subscriptionDBPath = filepath.Join(dir, "moji.db")
	}

	subscriptionPollSeconds := cfg.Subscription.PollIntervalSeconds
	subscriptionPollEnabled := subscriptionPollSeconds >= 0
	if subscriptionPollSeconds == 0 {
		subscriptionPollSeconds = 3600
	}
	if subscriptionPollSeconds < 0 {
		subscriptionPollSeconds = 0
	}

	jackettConfigured := cfg.Jackett.URL != "" && cfg.Jackett.APIKey != ""
	stashConfigured := cfg.Stash.URL != "" && cfg.Stash.LibraryPath != ""
	qbittorrentConfigured := cfg.QBittorrent.URL != "" && cfg.QBittorrent.Username != "" && cfg.QBittorrent.Password != ""

	return &graphqlapi.SettingsSnapshot{
		Stash: graphqlapi.StashSettingsSnapshot{
			Configured:       stashConfigured,
			Enabled:          stashEnabled,
			URL:              cfg.Stash.URL,
			APIKeyConfigured: cfg.Stash.APIKey != "",
			APIKey:           cfg.Stash.APIKey,
			LibraryPath:      cfg.Stash.LibraryPath,
		},
		Jackett: graphqlapi.JackettSettingsSnapshot{
			Configured:       jackettConfigured,
			Enabled:          jackettConfigured,
			URL:              cfg.Jackett.URL,
			APIKeyConfigured: cfg.Jackett.APIKey != "",
			APIKey:           cfg.Jackett.APIKey,
		},
		QBittorrent: graphqlapi.QBittorrentSettingsSnapshot{
			Configured:         qbittorrentConfigured,
			Enabled:            qbittorrentEnabled,
			URL:                cfg.QBittorrent.URL,
			Username:           cfg.QBittorrent.Username,
			UsernameConfigured: cfg.QBittorrent.Username != "",
			PasswordConfigured: cfg.QBittorrent.Password != "",
			Password:           cfg.QBittorrent.Password,
			DefaultSavePath:    cfg.QBittorrent.DefaultSavePath,
			Category:           cfg.QBittorrent.Category,
			Tags:               cfg.QBittorrent.Tags,
		},
		Tasks: graphqlapi.TaskSettingsSnapshot{
			Store:                       tasksStore,
			DBPath:                      dbPath,
			ProgressSyncIntervalSeconds: progressSyncSeconds,
			ProgressSyncEnabled:         progressSyncEnabled && downloaderEnabled,
		},
		Subscription: buildSubscriptionSnapshot(cfg, subscriptionService, subscriptionStore, subscriptionDBPath, subscriptionPollSeconds, subscriptionPollEnabled && stashEnabled),
		Logging: graphqlapi.LoggingSettingsSnapshot{
			Level:            cfg.EffectiveLogLevel(),
			FilePath:         cfg.EffectiveLogFilePath(),
			MaxEntries:       cfg.EffectiveLogMaxEntries(),
			MaxFileSizeBytes: cfg.EffectiveLogMaxFileSizeBytes(),
			MaxFileBackups:   cfg.EffectiveLogMaxFileBackups(),
		},
		System: graphqlapi.SystemSettingsSnapshot{
			AppVersion: version,
		},
	}
}

func startTaskSyncWorker(ctx context.Context, service graphqlapi.DownloaderService, stash graphqlapi.StashService, interval time.Duration) {
	if service == nil || interval <= 0 {
		if service == nil {
			logging.Infof("runtime: task sync worker not started because downloader service is unavailable")
		} else {
			logging.Infof("runtime: task sync worker disabled by sync interval")
		}
		return
	}
	logging.Infof("runtime: starting task sync worker with interval %s", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				if _, err := service.SyncProgress(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
					logging.Errorf("sync download progress: %v", err)
				}
				if stash != nil {
					if _, err := service.TriggerStashScans(syncCtx, stash); err != nil && !errors.Is(err, context.Canceled) {
						logging.Errorf("trigger stash scans: %v", err)
					}
				}
				cancel()
			}
		}
	}()
}

func configureStashClient(cfg *config.Config) *stash.Client {
	graphqlURL := cfg.Stash.GraphQLEndpoint()
	if graphqlURL == "" {
		logging.Infof("runtime: stash client disabled because stash.url is empty")
		return nil
	}
	logging.Infof("runtime: stash client configured for %s", graphqlURL)
	return stash.NewClient(graphqlURL, cfg.Stash.APIKey)
}

func configureStashService(cfg *config.Config, client *stash.Client) graphqlapi.StashService {
	if client == nil {
		logging.Infof("runtime: stash sync service disabled because stash client is not available")
		return nil
	}

	service, err := stashsync.NewService(client, []string{cfg.Stash.LibraryPath})
	if err != nil {
		logging.Fatalf("configure Stash: %v", err)
	}
	logging.Infof("runtime: stash sync service initialized with library path %s", cfg.Stash.LibraryPath)

	return service
}

func configureSubscription(cfg *config.Config, stashClient *stash.Client, downloaderService graphqlapi.DownloaderService) graphqlapi.SubscriptionService {
	if stashClient == nil {
		logging.Infof("runtime: subscription service disabled because stash client is not available")
		return nil
	}

	store, err := configureSubscriptionStore(cfg)
	if err != nil {
		logging.Fatalf("configure subscription store: %v", err)
	}

	registry := subscription.NewDefaultStashboxRegistry()
	service, err := subscription.NewService(stashClient, registry, downloaderService, store)
	if err != nil {
		logging.Fatalf("configure subscription: %v", err)
	}

	refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := service.RefreshStashBoxes(refreshCtx); err != nil {
		logging.Warnf("runtime: failed to refresh stash-box endpoints at startup: %v", err)
	} else {
		logging.Infof("runtime: subscription service initialized with %d stash-box endpoint(s) from Stash", len(registry.Endpoints()))
	}
	return service
}

func configureSubscriptionStore(cfg *config.Config) (subscription.Store, error) {
	switch cfg.Subscription.Store {
	case "", "sqlite":
		path := strings.TrimSpace(cfg.Subscription.DBPath)
		if path == "" {
			dir := "."
			if cfg.Tasks.DBPath != "" {
				dir = filepath.Dir(cfg.Tasks.DBPath)
			}
			path = filepath.Join(dir, "moji.db")
		}
		return subscription.NewSQLiteStore(path)
	case "memory":
		return subscription.NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unsupported subscription.store %q", cfg.Subscription.Store)
	}
}

func configureSubscriptionPollInterval(cfg *config.Config) time.Duration {
	seconds := cfg.Subscription.PollIntervalSeconds
	if seconds < 0 {
		return 0
	}
	if seconds == 0 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

func startSubscriptionWorker(ctx context.Context, service graphqlapi.SubscriptionService, interval time.Duration) {
	if service == nil || interval <= 0 {
		if service == nil {
			logging.Infof("runtime: subscription worker not started because subscription service is unavailable")
		} else {
			logging.Infof("runtime: subscription worker disabled by poll interval")
		}
		return
	}
	logging.Infof("runtime: starting subscription worker with interval %s", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
				if _, err := service.RefreshAll(syncCtx); err != nil && !errors.Is(err, context.Canceled) {
					logging.Errorf("refresh subscription performers: %v", err)
				}
				cancel()
			}
		}
	}()
}
