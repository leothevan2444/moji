package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/stashsync"
)

func TestHTTPHandlerServesRootAndGraphQLPlayground(t *testing.T) {
	handler := newHTTPHandler(testConfig(), "test-version")

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	handler.ServeHTTP(rootRec, rootReq)
	if rootRec.Code != http.StatusOK {
		t.Fatalf("expected root status %d, got %d", http.StatusOK, rootRec.Code)
	}
	if body := rootRec.Body.String(); body != "moji web ui is not built; run `make web-build` or `make web-dev`\n" {
		t.Fatalf("unexpected root body: %q", body)
	}

	playgroundReq := httptest.NewRequest(http.MethodGet, "/playground", nil)
	playgroundRec := httptest.NewRecorder()
	handler.ServeHTTP(playgroundRec, playgroundReq)
	if playgroundRec.Code != http.StatusOK {
		t.Fatalf("expected playground status %d, got %d", http.StatusOK, playgroundRec.Code)
	}
	if body := playgroundRec.Body.String(); !strings.Contains(body, "Moji GraphQL Playground") {
		t.Fatalf("expected playground body to include title, got %q", body)
	}

	graphqlGetReq := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	graphqlGetRec := httptest.NewRecorder()
	handler.ServeHTTP(graphqlGetRec, graphqlGetReq)
	if graphqlGetRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected graphql GET status %d, got %d", http.StatusMethodNotAllowed, graphqlGetRec.Code)
	}
}

func TestHTTPHandlerServesGraphQLHealth(t *testing.T) {
	handler := newHTTPHandler(testConfig(), "test-version")

	resp := postGraphQL(t, handler, `{ health { ok message } version }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no GraphQL errors, got %+v", resp.Errors)
	}
	if !resp.Data.Health.OK || resp.Data.Health.Message != "ok" {
		t.Fatalf("unexpected health response: %+v", resp.Data.Health)
	}
	if resp.Data.Version != "test-version" {
		t.Fatalf("expected version %q, got %q", "test-version", resp.Data.Version)
	}
}

func TestHTTPHandlerServesCurrentLogFile(t *testing.T) {
	cfg := testConfig()
	path := "moji.log"
	if err := os.WriteFile(path, []byte("test log\n"), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	handler := newHTTPHandler(cfg, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/api/logs/current", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "test log\n" {
		t.Fatalf("unexpected log body: %q", body)
	}
}

func TestHTTPHandlerServesSettingsSnapshot(t *testing.T) {
	handler := newHTTPHandler(testConfig(), "test-version")

	resp := postGraphQL(t, handler, `{
		settings {
			jackett { configured enabled url apiKeyConfigured }
			qbittorrent { configured enabled url usernameConfigured passwordConfigured defaultSavePath }
			stash { configured enabled url apiKeyConfigured libraryPath }
			tasks { store dbPath progressSyncIntervalSeconds progressSyncEnabled }
			subscription { store dbPath pollIntervalSeconds pollEnabled stashBoxes { name endpoint apiKeyConfigured } stashBoxEndpoints stashBoxesLoaded }
			logging { level filePath maxEntries maxFileSizeBytes maxFileBackups }
			system { appVersion }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no GraphQL errors, got %+v", resp.Errors)
	}
	if !resp.Data.Settings.Jackett.Configured || !resp.Data.Settings.Jackett.Enabled {
		t.Fatalf("expected jackett settings to be enabled, got %+v", resp.Data.Settings.Jackett)
	}
	if resp.Data.Settings.Tasks.Store != "sqlite" || resp.Data.Settings.Tasks.ProgressSyncIntervalSeconds != 60 {
		t.Fatalf("unexpected task settings: %+v", resp.Data.Settings.Tasks)
	}
	if resp.Data.Settings.Subscription.Store != "sqlite" || !resp.Data.Settings.Subscription.PollEnabled {
		t.Fatalf("unexpected subscription settings: %+v", resp.Data.Settings.Subscription)
	}
	if len(resp.Data.Settings.Subscription.StashBoxes) != 0 || len(resp.Data.Settings.Subscription.StashBoxEndpoints) != 0 {
		t.Fatalf("expected empty stash box selection in snapshot, got %+v", resp.Data.Settings.Subscription)
	}
	if resp.Data.Settings.Logging.Level != "info" || resp.Data.Settings.Logging.MaxEntries != 500 {
		t.Fatalf("unexpected logging settings: %+v", resp.Data.Settings.Logging)
	}
	if resp.Data.Settings.System.AppVersion != "test-version" {
		t.Fatalf("expected app version %q, got %q", "test-version", resp.Data.Settings.System.AppVersion)
	}
}

func TestIncompleteQBittorrentConfigDisablesResolver(t *testing.T) {
	cfg := testConfig()
	cfg.QBittorrent.URL = "http://qbittorrent.invalid"

	handler := newHTTPHandler(cfg, "test-version")

	resp := postGraphQL(t, handler, `{ qbittorrentTorrents { hash } }`)
	if len(resp.Errors) == 0 {
		t.Fatalf("expected GraphQL error when qBittorrent is disabled")
	}
	if got := resp.Errors[0].Message; got != "qBittorrent client is not configured" {
		t.Fatalf("unexpected GraphQL error: %q", got)
	}
}

func TestMissingStashConfigDisablesStashResolvers(t *testing.T) {
	cfg := testConfig()
	cfg.Stash.URL = ""
	handler := newHTTPHandler(cfg, "test-version")

	jobResp := postGraphQL(t, handler, `{ stashJob(id: "job-1") { id } }`)
	if len(jobResp.Errors) == 0 {
		t.Fatalf("expected GraphQL error when Stash is disabled")
	}
	if got := jobResp.Errors[0].Message; got != "stash client is not configured" {
		t.Fatalf("unexpected stashJob GraphQL error: %q", got)
	}

	scanResp := postGraphQL(t, handler, `mutation {
		stashMetadataScan(input: { paths: ["/library"] })
	}`)
	if len(scanResp.Errors) == 0 {
		t.Fatalf("expected GraphQL error when Stash is disabled")
	}
	if got := scanResp.Errors[0].Message; got != "stash client is not configured" {
		t.Fatalf("unexpected stashMetadataScan GraphQL error: %q", got)
	}
}

func TestConfigureProgressSyncInterval(t *testing.T) {
	cfg := testConfig()
	if got := configureProgressSyncInterval(cfg); got != time.Minute {
		t.Fatalf("expected default interval %s, got %s", time.Minute, got)
	}

	cfg.Tasks.ProgressSyncIntervalSeconds = 5
	if got := configureProgressSyncInterval(cfg); got != 5*time.Second {
		t.Fatalf("expected configured interval %s, got %s", 5*time.Second, got)
	}

	cfg.Tasks.ProgressSyncIntervalSeconds = -1
	if got := configureProgressSyncInterval(cfg); got != 0 {
		t.Fatalf("expected disabled interval, got %s", got)
	}
}

func TestBuildSettingsSnapshotNormalizesDefaults(t *testing.T) {
	cfg := testConfig()
	cfg.Tasks.DBPath = ""
	cfg.Subscription.DBPath = ""

	snapshot := buildSettingsSnapshot(cfg, "test-version", false, false, false, nil)
	if snapshot.Jackett.URL != "http://jackett.invalid" || !snapshot.Jackett.APIKeyConfigured {
		t.Fatalf("unexpected jackett snapshot: %+v", snapshot.Jackett)
	}
	if snapshot.Tasks.Store != "sqlite" || snapshot.Tasks.DBPath != "moji.db" {
		t.Fatalf("unexpected task store snapshot: %+v", snapshot.Tasks)
	}
	if snapshot.Tasks.ProgressSyncIntervalSeconds != 60 || snapshot.Tasks.ProgressSyncEnabled {
		t.Fatalf("unexpected task sync snapshot: %+v", snapshot.Tasks)
	}
	if snapshot.Subscription.Store != "sqlite" || snapshot.Subscription.DBPath != "moji.db" || snapshot.Subscription.PollIntervalSeconds != 3600 || snapshot.Subscription.PollEnabled {
		t.Fatalf("unexpected subscription snapshot: %+v", snapshot.Subscription)
	}
	if len(snapshot.Subscription.StashBoxes) != 0 || len(snapshot.Subscription.StashBoxEndpoints) != 0 {
		t.Fatalf("expected empty stash box selection in default snapshot, got %+v", snapshot.Subscription)
	}
	if snapshot.Logging.Level != "info" || snapshot.Logging.MaxEntries != 500 || snapshot.Logging.MaxFileBackups != 5 {
		t.Fatalf("unexpected logging snapshot: %+v", snapshot.Logging)
	}
}

func TestStartProgressSyncWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := &fakeProgressSyncService{called: make(chan struct{}, 1)}
	startTaskSyncWorker(ctx, service, nil, time.Millisecond)

	select {
	case <-service.called:
	case <-time.After(time.Second):
		t.Fatal("expected progress sync worker to call SyncProgress")
	}
}

func TestStartTaskSyncWorkerTriggersStashScans(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := &fakeProgressSyncService{
		called:      make(chan struct{}, 1),
		stashCalled: make(chan struct{}, 1),
	}
	startTaskSyncWorker(ctx, service, &fakeConfiguredStashService{}, time.Millisecond)

	select {
	case <-service.stashCalled:
	case <-time.After(time.Second):
		t.Fatal("expected task sync worker to call TriggerStashScans")
	}
}

type graphQLResponse struct {
	Data struct {
		Health struct {
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"health"`
		Version  string `json:"version"`
		Settings struct {
			Stash struct {
				Configured       bool   `json:"configured"`
				Enabled          bool   `json:"enabled"`
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
				LibraryPath      string `json:"libraryPath"`
			} `json:"stash"`
			Jackett struct {
				Configured       bool   `json:"configured"`
				Enabled          bool   `json:"enabled"`
				URL              string `json:"url"`
				APIKeyConfigured bool   `json:"apiKeyConfigured"`
			} `json:"jackett"`
			Qbittorrent struct {
				Configured         bool   `json:"configured"`
				Enabled            bool   `json:"enabled"`
				URL                string `json:"url"`
				UsernameConfigured bool   `json:"usernameConfigured"`
				PasswordConfigured bool   `json:"passwordConfigured"`
				DefaultSavePath    string `json:"defaultSavePath"`
			} `json:"qbittorrent"`
			Tasks struct {
				Store                       string `json:"store"`
				DBPath                      string `json:"dbPath"`
				ProgressSyncIntervalSeconds int    `json:"progressSyncIntervalSeconds"`
				ProgressSyncEnabled         bool   `json:"progressSyncEnabled"`
			} `json:"tasks"`
			Subscription struct {
				Store               string `json:"store"`
				DBPath              string `json:"dbPath"`
				PollIntervalSeconds int    `json:"pollIntervalSeconds"`
				PollEnabled         bool   `json:"pollEnabled"`
				StashBoxes          []struct {
					Name             string `json:"name"`
					Endpoint         string `json:"endpoint"`
					APIKeyConfigured bool   `json:"apiKeyConfigured"`
				} `json:"stashBoxes"`
				StashBoxEndpoints []string `json:"stashBoxEndpoints"`
				StashBoxesLoaded  bool     `json:"stashBoxesLoaded"`
			} `json:"subscription"`
			Logging struct {
				Level            string `json:"level"`
				FilePath         string `json:"filePath"`
				MaxEntries       int    `json:"maxEntries"`
				MaxFileSizeBytes int    `json:"maxFileSizeBytes"`
				MaxFileBackups   int    `json:"maxFileBackups"`
			} `json:"logging"`
			System struct {
				AppVersion string `json:"appVersion"`
			} `json:"system"`
		} `json:"settings"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func postGraphQL(t *testing.T, handler http.Handler, query string) graphQLResponse {
	t.Helper()

	body := bytes.NewBufferString(`{"query":` + strconvQuote(query) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected GraphQL status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp graphQLResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode GraphQL response: %v", err)
	}
	return resp
}

func testConfig() *config.Config {
	var cfg config.Config
	cfg.Jackett.URL = "http://jackett.invalid"
	cfg.Jackett.APIKey = "test-api-key"
	cfg.Stash.URL = "http://stash.invalid"
	cfg.Tasks.DBPath = filepath.Join(os.TempDir(), "moji-test.db")
	cfg.Subscription.DBPath = filepath.Join(os.TempDir(), "moji-test.db")
	return &cfg
}

func strconvQuote(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

type fakeProgressSyncService struct {
	called      chan struct{}
	stashCalled chan struct{}
}

func (f *fakeProgressSyncService) AddTorrentContext(context.Context, downloader.AddTorrentRequest) (*downloader.Task, error) {
	return nil, nil
}

func (f *fakeProgressSyncService) DownloadMediaContext(context.Context, downloader.DownloadRequest) (*downloader.Task, error) {
	return nil, nil
}

func (f *fakeProgressSyncService) FindTask(context.Context, string) (*downloader.Task, error) {
	return nil, nil
}

func (f *fakeProgressSyncService) ListTasks(context.Context) ([]*downloader.Task, error) {
	return nil, nil
}

func (f *fakeProgressSyncService) SyncProgress(context.Context) ([]*downloader.Task, error) {
	select {
	case f.called <- struct{}{}:
	default:
	}
	return nil, nil
}

func (f *fakeProgressSyncService) TriggerTaskStashScan(context.Context, string, downloader.StashScanner) (*downloader.Task, error) {
	return nil, nil
}

func (f *fakeProgressSyncService) TriggerStashScans(context.Context, downloader.StashScanner) ([]*downloader.Task, error) {
	select {
	case f.stashCalled <- struct{}{}:
	default:
	}
	return nil, nil
}

type fakeConfiguredStashService struct{}

func (fakeConfiguredStashService) MetadataScan(context.Context, stashsync.ScanRequest) (string, error) {
	return "job-test", nil
}

func (fakeConfiguredStashService) FindJob(context.Context, string) (*stashsync.Job, error) {
	return nil, nil
}
