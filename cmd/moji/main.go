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
		cfg *config.Config
		err error
	)
	if *configPath != "" {
		cfg, err = config.LoadFromPath(*configPath)
	} else {
		cfg, err = config.Load()
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
	mux, downloaderService := newHTTPRuntime(cfg, "dev")

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

	startProgressSyncWorker(ctx, downloaderService, configureProgressSyncInterval(cfg))

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
	handler, _ := newHTTPRuntime(cfg, version)
	return handler
}

func newHTTPRuntime(cfg *config.Config, version string) (http.Handler, graphqlapi.DownloaderService) {
	jackettTracker := tracker.NewJackettService(cfg.Jackett.URL, cfg.Jackett.APIKey)
	apiHandler := api.NewHandler(jackettTracker)
	torrentClient := configureQBittorrent(cfg)
	downloaderService := configureDownloader(cfg, jackettTracker, torrentClient)
	stashService := configureStash(cfg)
	resolver := graphqlapi.NewResolver(jackettTracker, torrentClient, downloaderService, stashService, version)
	graphqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	mux := http.NewServeMux()
	apiHandler.Register(mux)
	mux.Handle("GET /graphql", playground.Handler("Moji GraphQL Playground", "/graphql"))
	mux.Handle("POST /graphql", graphqlHandler)
	// Keep root simple until web UI lands in this repo.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("moji is running\n"))
	})

	return mux, downloaderService
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

func startProgressSyncWorker(ctx context.Context, service graphqlapi.DownloaderService, interval time.Duration) {
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
				cancel()
			}
		}
	}()
}

func configureStash(cfg *config.Config) graphqlapi.StashService {
	if cfg.Stash.GraphQLURL == "" {
		return nil
	}

	client := stash.NewClient(cfg.Stash.GraphQLURL, cfg.Stash.APIKey)
	service, err := stashsync.NewService(client, []string{cfg.Stash.LibraryPath})
	if err != nil {
		log.Fatalf("configure Stash: %v", err)
	}

	return service
}
