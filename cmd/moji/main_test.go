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
)

func TestHTTPHandlerServesRootAndGraphQLPlayground(t *testing.T) {
	handler := newHTTPHandler(testConfig(), "test-version")

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	handler.ServeHTTP(rootRec, rootReq)
	if rootRec.Code != http.StatusOK {
		t.Fatalf("expected root status %d, got %d", http.StatusOK, rootRec.Code)
	}
	if body := rootRec.Body.String(); body != "moji is running\n" {
		t.Fatalf("unexpected root body: %q", body)
	}

	playgroundReq := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	playgroundRec := httptest.NewRecorder()
	handler.ServeHTTP(playgroundRec, playgroundReq)
	if playgroundRec.Code != http.StatusOK {
		t.Fatalf("expected playground status %d, got %d", http.StatusOK, playgroundRec.Code)
	}
	if body := playgroundRec.Body.String(); !strings.Contains(body, "Moji GraphQL Playground") {
		t.Fatalf("expected playground body to include title, got %q", body)
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

func TestStartProgressSyncWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := &fakeProgressSyncService{called: make(chan struct{}, 1)}
	startProgressSyncWorker(ctx, service, time.Millisecond)

	select {
	case <-service.called:
	case <-time.After(time.Second):
		t.Fatal("expected progress sync worker to call SyncProgress")
	}
}

type graphQLResponse struct {
	Data struct {
		Health struct {
			OK      bool   `json:"ok"`
			Message string `json:"message"`
		} `json:"health"`
		Version string `json:"version"`
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
	called chan struct{}
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
