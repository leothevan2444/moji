package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/leothevan2444/moji/internal/stashsync"
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
	configStore, err = config.OpenStore(path)
	if err == nil {
		cfg = configStore.Config()
	}
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.Jackett.URL == "" {
		log.Fatalf("invalid config: jackett.url is required")
	}
	if cfg.Jackett.APIKey == "" {
		log.Fatalf("invalid config: jackett.api_key is required")
	}

	// Dependencies
	mux, downloaderService, stashService := newHTTPRuntime(cfg, "dev", configStore)

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Fatalf("listen %s: %v", server.Addr, err)
	}

	log.Printf("moji listening on %s", ln.Addr().String())

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startTaskSyncWorker(ctx, downloaderService, stashService, configureProgressSyncInterval(cfg))

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}

func newHTTPHandler(cfg *config.Config, version string) http.Handler {
	handler, _, _ := newHTTPRuntime(cfg, version, nil)
	return handler
}

func newHTTPRuntime(cfg *config.Config, version string, configStore *config.Store) (http.Handler, graphqlapi.DownloaderService, graphqlapi.StashService) {
	jackettTracker := tracker.NewJackettService(cfg.Jackett.URL, cfg.Jackett.APIKey)
	apiHandler := api.NewHandler(jackettTracker)
	torrentClient := configureQBittorrent(cfg)
	downloaderService := configureDownloader(cfg, jackettTracker, torrentClient)
	stashService := configureStash(cfg)
	resolver := graphqlapi.NewResolver(jackettTracker, torrentClient, downloaderService, stashService, version)
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

	return router, downloaderService, stashService
}

func configureQBittorrent(cfg *config.Config) graphqlapi.TorrentClient {
	if cfg.QBittorrent.URL == "" {
		return nil
	}
	if cfg.QBittorrent.Username == "" {
		log.Printf("qBittorrent disabled: qbittorrent.username is required when qbittorrent.url is set")
		return nil
	}
	if cfg.QBittorrent.Password == "" {
		log.Printf("qBittorrent disabled: qbittorrent.password is required when qbittorrent.url is set")
		return nil
	}

	client := qbittorrent.NewClient(cfg.QBittorrent.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Login(ctx, cfg.QBittorrent.Username, cfg.QBittorrent.Password); err != nil {
		log.Fatalf("login qBittorrent: %v", err)
	}

	return downloader.NewDefaultingTorrentClient(client, downloader.TorrentDefaults{
		SavePath: cfg.QBittorrent.DefaultSavePath,
		Category: cfg.QBittorrent.Category,
		Tags:     cfg.QBittorrent.Tags,
	})
}

func configureDownloader(cfg *config.Config, tr tracker.Tracker, torrent graphqlapi.TorrentClient) graphqlapi.DownloaderService {
	if torrent == nil {
		return nil
	}

	store, err := configureTaskStore(cfg)
	if err != nil {
		log.Fatalf("configure task store: %v", err)
	}
	service, err := downloader.NewService(tr, torrent, store)
	if err != nil {
		log.Fatalf("configure downloader: %v", err)
	}
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
		System: graphqlapi.SystemSettingsSnapshot{
			AppVersion: version,
		},
	}
}

func startTaskSyncWorker(ctx context.Context, service graphqlapi.DownloaderService, stash graphqlapi.StashService, interval time.Duration) {
	if service == nil || interval <= 0 {
		return
	}

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
					log.Printf("sync download progress: %v", err)
				}
				if stash != nil {
					if _, err := service.TriggerStashScans(syncCtx, stash); err != nil && !errors.Is(err, context.Canceled) {
						log.Printf("trigger stash scans: %v", err)
					}
				}
				cancel()
			}
		}
	}()
}

func configureStash(cfg *config.Config) graphqlapi.StashService {
	graphqlURL := cfg.Stash.GraphQLEndpoint()
	if graphqlURL == "" {
		return nil
	}

	client := stash.NewClient(graphqlURL, cfg.Stash.APIKey)
	service, err := stashsync.NewService(client, []string{cfg.Stash.LibraryPath})
	if err != nil {
		log.Fatalf("configure Stash: %v", err)
	}

	return service
}
