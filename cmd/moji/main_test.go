package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestHTTPHandlerServesSettingsSnapshot(t *testing.T) {
	handler := newHTTPHandler(testConfig(), "test-version")

	resp := postGraphQL(t, handler, `{
		settings {
			jackett { configured enabled url apiKeyConfigured }
			qbittorrent { configured enabled url usernameConfigured passwordConfigured defaultSavePath }
			stash { configured enabled graphqlUrl apiKeyConfigured libraryPath }
			tasks { store jsonPath progressSyncIntervalSeconds progressSyncEnabled }
			system { appVersion }
		}
	}`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no GraphQL errors, got %+v", resp.Errors)
	}
	if !resp.Data.Settings.Jackett.Configured || !resp.Data.Settings.Jackett.Enabled {
		t.Fatalf("expected jackett settings to be enabled, got %+v", resp.Data.Settings.Jackett)
	}
	if resp.Data.Settings.Tasks.Store != "json" || resp.Data.Settings.Tasks.ProgressSyncIntervalSeconds != 60 {
		t.Fatalf("unexpected task settings: %+v", resp.Data.Settings.Tasks)
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
	handler := newHTTPHandler(testConfig(), "test-version")

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

	snapshot := buildSettingsSnapshot(cfg, "test-version", false, false, false)
	if snapshot.Jackett.URL != "http://jackett.invalid" || !snapshot.Jackett.APIKeyConfigured {
		t.Fatalf("unexpected jackett snapshot: %+v", snapshot.Jackett)
	}
	if snapshot.Tasks.Store != "json" || snapshot.Tasks.JSONPath != "moji-tasks.json" {
		t.Fatalf("unexpected task store snapshot: %+v", snapshot.Tasks)
	}
	if snapshot.Tasks.ProgressSyncIntervalSeconds != 60 || snapshot.Tasks.ProgressSyncEnabled {
		t.Fatalf("unexpected task sync snapshot: %+v", snapshot.Tasks)
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
		Version string `json:"version"`
		Settings struct {
			Stash struct {
				Configured       bool   `json:"configured"`
				Enabled          bool   `json:"enabled"`
				GraphqlURL       string `json:"graphqlUrl"`
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
				JSONPath                    string `json:"jsonPath"`
				ProgressSyncIntervalSeconds int    `json:"progressSyncIntervalSeconds"`
				ProgressSyncEnabled         bool   `json:"progressSyncEnabled"`
			} `json:"tasks"`
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
