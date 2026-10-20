package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestSourcingTorrentMultipartUpload(t *testing.T) {
	for _, data := range []string{"torrent file contents", ""} {
		runtime := &fakeTaskRuntime{resolveSourcingTask: &taskruntime.Task{ID: "waiting", Stage: taskruntime.TaskStageDownloading, StageStatus: taskruntime.TaskStageStatusRunning}}
		resolver := NewResolver(nil, nil, runtime, nil, "test")
		server := NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		operations := map[string]any{"query": `mutation($input: ResolveBlockedSourcingTaskInput!) { resolveBlockedSourcingTask(id:"waiting", input:$input) { id stage } }`, "variables": map[string]any{"input": map[string]any{"torrentFile": nil}}}
		encoded, _ := json.Marshal(operations)
		if err := writer.WriteField("operations", string(encoded)); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("map", `{"0":["variables.input.torrentFile"]}`); err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile("0", "movie.torrent")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, data); err != nil {
			t.Fatal(err)
		}
		writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/graphql", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, req)
		if data != "" {
			if !bytes.Equal(runtime.resolveSourcingReq.TorrentData, []byte(data)) || runtime.resolveSourcingReq.URL != "" || strings.Contains(recorder.Body.String(), `"errors"`) {
				t.Fatalf("multipart did not reach runtime intact: %s", recorder.Body.String())
			}
		} else if !strings.Contains(recorder.Body.String(), "INVALID_TORRENT_INPUT") || runtime.resolveSourcingID != "" {
			t.Fatalf("empty upload accepted: %s", recorder.Body.String())
		}
	}
}

func TestSourcingUploadLimitsDeclaredAndActualSize(t *testing.T) {
	for _, upload := range []*graphql.Upload{
		{File: strings.NewReader("x"), Size: taskruntime.MaxTorrentBytes + 1},
		{File: bytes.NewReader(make([]byte, taskruntime.MaxTorrentBytes+1)), Size: 1},
		{File: strings.NewReader("")},
	} {
		if _, err := readSourcingTorrentUpload(upload); err != taskruntime.ErrInvalidTorrentInput {
			t.Fatalf("invalid upload accepted: %v", err)
		}
	}
}

func TestWaitingResourceTaskCanSearchIndexerResults(t *testing.T) {
	next := time.Now().Add(time.Hour)
	runtime := &fakeTaskRuntime{findTask: &taskruntime.Task{ID: "waiting", Code: "OFJE-657", Stage: taskruntime.TaskStageSourcing, StageStatus: taskruntime.TaskStageStatusPending, SourcingNextCheckAt: &next}}
	tr := &fakeGraphQLTracker{results: []jackett.SearchResult{{Title: "OFJE657", MagnetURI: "magnet:?xt=urn:btih:match"}}}
	resolver := NewResolver(tr, nil, runtime, nil, "test")
	results, err := resolver.Query().BlockedTaskTorrentCandidates(context.Background(), "waiting", nil)
	if err != nil || len(results) != 1 || tr.query != "OFJE-657" {
		t.Fatalf("waiting task could not search: %+v %v", results, err)
	}
	runtime.findTask.Stage = taskruntime.TaskStageDownloading
	if _, err := resolver.Query().BlockedTaskTorrentCandidates(context.Background(), "waiting", nil); err != taskruntime.ErrSourcingUnavailable {
		t.Fatalf("active download accepted manual search: %v", err)
	}
}
