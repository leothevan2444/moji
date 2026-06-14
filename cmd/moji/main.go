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
	"github.com/leothevan2444/moji/internal/following"
	"github.com/leothevan2444/moji/internal/graphqlapi"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/internal/webui"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"github.com/leothevan2444/moji/pkg/stash"
	"github.com/leothevan2444/moji/pkg/stashbox"
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
	mux, downloaderService, stashService, followingService := newHTTPRuntime(cfg, "dev", configStore)

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
	startFollowingWorker(ctx, followingService, configureFollowingPollInterval(cfg))

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

func newHTTPRuntime(cfg *config.Config, version string, configStore *config.Store) (http.Handler, graphqlapi.DownloaderService, graphqlapi.StashService, graphqlapi.FollowingService) {
	jackettTracker := tracker.NewJackettService(cfg.Jackett.URL, cfg.Jackett.APIKey)
	logging.Infof("runtime: jackett tracker configured for %s", cfg.Jackett.URL)
	apiHandler := api.NewHandler(jackettTracker, api.WithLogFilePath(cfg.EffectiveLogFilePath()))
	torrentClient := configureQBittorrent(cfg)
	downloaderService := configureDownloader(cfg, jackettTracker, torrentClient)
	stashClient := configureStashClient(cfg)
	stashService := configureStashService(cfg, stashClient)
	followingService := configureFollowing(cfg, stashClient, downloaderService)
	resolver := graphqlapi.NewResolver(jackettTracker, torrentClient, downloaderService, stashService, version)
	resolver.Following = followingService
	resolver.LogReader = logging.Default()
	resolver.RuntimeSettings = buildSettingsSnapshot(cfg, version, torrentClient != nil, downloaderService != nil, stashService != nil)
	if configStore != nil {
		resolver.SettingsEditor = newRuntimeSettingsEditor(configStore, version, torrentClient != nil, downloaderService != nil, stashService != nil)
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

	return router, downloaderService, stashService, followingService
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
	case "", "json":
		path := cfg.Tasks.JSONPath
		if path == "" {
			path = "moji-tasks.json"
		}
		return downloader.NewJSONTaskStore(path)
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

func buildSettingsSnapshot(cfg *config.Config, version string, qbittorrentEnabled bool, downloaderEnabled bool, stashEnabled bool) *graphqlapi.SettingsSnapshot {
	tasksStore := cfg.Tasks.Store
	if tasksStore == "" {
		tasksStore = "json"
	}

	jsonPath := cfg.Tasks.JSONPath
	if jsonPath == "" {
		jsonPath = "moji-tasks.json"
	}

	progressSyncSeconds := cfg.Tasks.ProgressSyncIntervalSeconds
	progressSyncEnabled := progressSyncSeconds >= 0
	if progressSyncSeconds == 0 {
		progressSyncSeconds = 60
	}
	if progressSyncSeconds < 0 {
		progressSyncSeconds = 0
	}

	followingStore := cfg.Following.Store
	if followingStore == "" {
		followingStore = "json"
	}

	followingJSONPath := cfg.Following.JSONPath
	if followingJSONPath == "" {
		dir := "."
		if cfg.Tasks.JSONPath != "" {
			dir = filepath.Dir(cfg.Tasks.JSONPath)
		}
		followingJSONPath = filepath.Join(dir, "moji-following.json")
	}

	followingPollSeconds := cfg.Following.PollIntervalSeconds
	followingPollEnabled := followingPollSeconds >= 0
	if followingPollSeconds == 0 {
		followingPollSeconds = 3600
	}
	if followingPollSeconds < 0 {
		followingPollSeconds = 0
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
			LibraryPath:      cfg.Stash.LibraryPath,
		},
		Jackett: graphqlapi.JackettSettingsSnapshot{
			Configured:       jackettConfigured,
			Enabled:          jackettConfigured,
			URL:              cfg.Jackett.URL,
			APIKeyConfigured: cfg.Jackett.APIKey != "",
		},
		QBittorrent: graphqlapi.QBittorrentSettingsSnapshot{
			Configured:         qbittorrentConfigured,
			Enabled:            qbittorrentEnabled,
			URL:                cfg.QBittorrent.URL,
			Username:           cfg.QBittorrent.Username,
			UsernameConfigured: cfg.QBittorrent.Username != "",
			PasswordConfigured: cfg.QBittorrent.Password != "",
			DefaultSavePath:    cfg.QBittorrent.DefaultSavePath,
			Category:           cfg.QBittorrent.Category,
			Tags:               cfg.QBittorrent.Tags,
		},
		Tasks: graphqlapi.TaskSettingsSnapshot{
			Store:                       tasksStore,
			JSONPath:                    jsonPath,
			ProgressSyncIntervalSeconds: progressSyncSeconds,
			ProgressSyncEnabled:         progressSyncEnabled && downloaderEnabled,
		},
		Following: graphqlapi.FollowingSettingsSnapshot{
			Store:                    followingStore,
			JSONPath:                 followingJSONPath,
			PollIntervalSeconds:      followingPollSeconds,
			PollEnabled:              followingPollEnabled && stashEnabled,
			JAVStashEnabled:          cfg.Following.JAVStashAPIKey != "",
			JAVStashAPIKeyConfigured: cfg.Following.JAVStashAPIKey != "",
		},
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

func configureFollowing(cfg *config.Config, stashClient *stash.Client, downloaderService graphqlapi.DownloaderService) graphqlapi.FollowingService {
	if stashClient == nil {
		logging.Infof("runtime: following service disabled because stash client is not available")
		return nil
	}

	store, err := configureFollowingStore(cfg)
	if err != nil {
		logging.Fatalf("configure following store: %v", err)
	}

	var javstashClient *stashbox.Client
	if strings.TrimSpace(cfg.Following.JAVStashAPIKey) != "" {
		javstashClient = stashbox.NewClient(cfg.Following.JAVStashAPIKey)
		logging.Infof("runtime: javstash client enabled for following checks")
	} else {
		logging.Infof("runtime: javstash client disabled because following.javstash_api_key is empty")
	}

	service, err := following.NewService(stashClient, javstashClient, downloaderService, store)
	if err != nil {
		logging.Fatalf("configure following: %v", err)
	}
	logging.Infof("runtime: following service initialized")
	return service
}

func configureFollowingStore(cfg *config.Config) (following.Store, error) {
	switch cfg.Following.Store {
	case "", "json":
		path := cfg.Following.JSONPath
		if path == "" {
			dir := "."
			if cfg.Tasks.JSONPath != "" {
				dir = filepath.Dir(cfg.Tasks.JSONPath)
			}
			path = filepath.Join(dir, "moji-following.json")
		}
		return following.NewJSONStore(path)
	case "memory":
		return following.NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unsupported following.store %q", cfg.Following.Store)
	}
}

func configureFollowingPollInterval(cfg *config.Config) time.Duration {
	seconds := cfg.Following.PollIntervalSeconds
	if seconds < 0 {
		return 0
	}
	if seconds == 0 {
		seconds = 3600
	}
	return time.Duration(seconds) * time.Second
}

func startFollowingWorker(ctx context.Context, service graphqlapi.FollowingService, interval time.Duration) {
	if service == nil || interval <= 0 {
		if service == nil {
			logging.Infof("runtime: following worker not started because following service is unavailable")
		} else {
			logging.Infof("runtime: following worker disabled by poll interval")
		}
		return
	}
	logging.Infof("runtime: starting following worker with interval %s", interval)

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
					logging.Errorf("refresh following performers: %v", err)
				}
				cancel()
			}
		}
	}()
}
