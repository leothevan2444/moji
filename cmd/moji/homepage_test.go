package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestHomepageWidgetReadsManagedTasksWithoutUpstreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	store := taskruntime.NewRelationalTaskStore(manager.Core, manager.Observations)
	now := time.Now().Add(-time.Hour)
	if err := store.Create(ctx, &taskruntime.Task{ID: "completed", Code: "TEST-1", Source: taskruntime.TaskSourceManual, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone, CreatedAt: now, UpdatedAt: now, IngestCompletedAt: &now}); err != nil {
		t.Fatal(err)
	}
	runtime := newHTTPRuntimeWithContext(ctx, &config.Config{}, "widget-test", nil, manager)
	defer runtime.indexers.Close()
	defer runtime.stashBoxCacheService.Close()
	w := httptest.NewRecorder()
	runtime.handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/widget/homepage", nil))
	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal("widget returned non-JSON", w.Body.String(), err)
	}
	if w.Code != 200 || len(got) != 4 || got["completedLast24Hours"] != 1 {
		t.Fatalf("widget did not read managed tasks: %d %s", w.Code, w.Body.String())
	}
	if runtime.taskRuntimeService != nil {
		t.Fatal("fixture unexpectedly enabled task execution")
	}
}
