package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
)

func TestDownloadMediaCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		downloadTask: &downloader.Task{
			ID:         "task-1",
			Query:      "ABCD-123",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:test",
			Candidate: downloader.Candidate{
				Title:   "ABCD-123",
				Tracker: "demo",
				Seeders: 5,
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { query: "ABCD-123", limit: 1 }) {
			id
			query
			status
			torrentUrl
			candidate { title tracker seeders }
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	task := resp.Data.DownloadMedia
	if task.ID != "task-1" || task.Status != "added" || task.Candidate.Title != "ABCD-123" {
		t.Fatalf("unexpected download task response: %+v", task)
	}
	if downloader.downloadRequest.Query != "ABCD-123" || downloader.downloadRequest.Limit != 1 {
		t.Fatalf("unexpected download request: %+v", downloader.downloadRequest)
	}
}

func TestAddTorrentCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		addTask: &downloader.Task{
			ID:         "task-manual",
			Query:      "magnet:?xt=urn:btih:manual",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:manual",
			Candidate: downloader.Candidate{
				Title:     "magnet:?xt=urn:btih:manual",
				MagnetURI: "magnet:?xt=urn:btih:manual",
			},
			CreatedAt: time.Unix(100, 0).UTC(),
			UpdatedAt: time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		addTorrent(input: { url: "magnet:?xt=urn:btih:manual", category: "moji" }) {
			id
			status
			torrentUrl
		}
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if resp.Data.AddTorrent.ID != "task-manual" || resp.Data.AddTorrent.Status != "added" {
		t.Fatalf("unexpected add torrent response: %+v", resp.Data.AddTorrent)
	}
	if downloader.addRequest.URL != "magnet:?xt=urn:btih:manual" || downloader.addRequest.Category != "moji" {
		t.Fatalf("unexpected add torrent request: %+v", downloader.addRequest)
	}
}

func TestDeprecatedQBittorrentAddCreatesTask(t *testing.T) {
	downloader := &fakeDownloader{
		addTask: &downloader.Task{
			ID:         "task-manual",
			Status:     downloader.TaskStatusAdded,
			TorrentURL: "magnet:?xt=urn:btih:manual",
			CreatedAt:  time.Unix(100, 0).UTC(),
			UpdatedAt:  time.Unix(200, 0).UTC(),
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		qbittorrentAdd(input: { url: "magnet:?xt=urn:btih:manual" })
	}`)

	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if !resp.Data.QbittorrentAdd {
		t.Fatal("expected qbittorrentAdd to return true")
	}
	if downloader.addRequest.URL != "magnet:?xt=urn:btih:manual" {
		t.Fatalf("unexpected add torrent request: %+v", downloader.addRequest)
	}
}

func TestTasksQueryListsTasks(t *testing.T) {
	downloader := &fakeDownloader{
		listTasks: []*downloader.Task{
			{ID: "task-2", Query: "BBBB-222", Status: downloader.TaskStatusAdded, CreatedAt: time.Unix(200, 0).UTC(), UpdatedAt: time.Unix(200, 0).UTC()},
			{ID: "task-1", Query: "AAAA-111", Status: downloader.TaskStatusFailed, CreatedAt: time.Unix(100, 0).UTC(), UpdatedAt: time.Unix(100, 0).UTC()},
		},
	}
	resolver := NewResolver(nil, nil, downloader, nil, "test-version")

	resp := executeGraphQL(t, resolver, `{ tasks { id query status } }`)
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if len(resp.Data.Tasks) != 2 || resp.Data.Tasks[0].ID != "task-2" || resp.Data.Tasks[1].ID != "task-1" {
		t.Fatalf("unexpected tasks response: %+v", resp.Data.Tasks)
	}
}

func TestDownloadMediaRequiresDownloader(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")

	resp := executeGraphQL(t, resolver, `mutation {
		downloadMedia(input: { query: "ABCD-123" }) { id }
	}`)
	if len(resp.Errors) == 0 {
		t.Fatal("expected downloader configuration error")
	}
	if got := resp.Errors[0].Message; got != "downloader is not configured" {
		t.Fatalf("unexpected error: %q", got)
	}
}

type fakeDownloader struct {
	addRequest      downloader.AddTorrentRequest
	downloadRequest downloader.DownloadRequest
	addTask         *downloader.Task
	downloadTask    *downloader.Task
	findTask        *downloader.Task
	listTasks       []*downloader.Task
}

func (f *fakeDownloader) AddTorrentContext(_ context.Context, req downloader.AddTorrentRequest) (*downloader.Task, error) {
	f.addRequest = req
	return f.addTask, nil
}

func (f *fakeDownloader) DownloadMediaContext(_ context.Context, req downloader.DownloadRequest) (*downloader.Task, error) {
	f.downloadRequest = req
	return f.downloadTask, nil
}

func (f *fakeDownloader) FindTask(_ context.Context, _ string) (*downloader.Task, error) {
	return f.findTask, nil
}

func (f *fakeDownloader) ListTasks(_ context.Context) ([]*downloader.Task, error) {
	return f.listTasks, nil
}

type graphQLTaskResponse struct {
	Data struct {
		AddTorrent struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			TorrentURL string `json:"torrentUrl"`
		} `json:"addTorrent"`
		DownloadMedia struct {
			ID         string `json:"id"`
			Query      string `json:"query"`
			Status     string `json:"status"`
			TorrentURL string `json:"torrentUrl"`
			Candidate  struct {
				Title   string `json:"title"`
				Tracker string `json:"tracker"`
				Seeders int    `json:"seeders"`
			} `json:"candidate"`
		} `json:"downloadMedia"`
		Tasks []struct {
			ID     string `json:"id"`
			Query  string `json:"query"`
			Status string `json:"status"`
		} `json:"tasks"`
		QbittorrentAdd bool `json:"qbittorrentAdd"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func executeGraphQL(t *testing.T, resolver *Resolver, query string) graphQLTaskResponse {
	t.Helper()

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp graphQLTaskResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}
