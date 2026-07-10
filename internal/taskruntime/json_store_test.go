package taskruntime

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestJSONTaskStorePersistsTasksAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	createdAt := time.Unix(100, 0).UTC()
	updatedAt := time.Unix(200, 0).UTC()

	store, err := NewJSONTaskStore(path)
	if err != nil {
		t.Fatalf("NewJSONTaskStore failed: %v", err)
	}
	task := &Task{
		ID:          "task-json",
		Code:        "ABCD-123",
		Stage:       TaskStageDownloading,
		StageStatus: TaskStageStatusRunning,
		TorrentURL:  "magnet:?xt=urn:btih:test",
		Candidate: Candidate{
			Title:   "ABCD-123",
			Tracker: "demo",
			Seeders: 10,
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	reopened, err := NewJSONTaskStore(path)
	if err != nil {
		t.Fatalf("reopen NewJSONTaskStore failed: %v", err)
	}
	stored, err := reopened.Find(context.Background(), "task-json")
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if stored.Code != task.Code || stored.Stage != TaskStageDownloading || stored.StageStatus != TaskStageStatusRunning || stored.Candidate.Title != task.Candidate.Title {
		t.Fatalf("unexpected restored task: %+v", stored)
	}
	if !stored.CreatedAt.Equal(createdAt) || !stored.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected restored times: created=%s updated=%s", stored.CreatedAt, stored.UpdatedAt)
	}
}

func TestJSONTaskStoreListSortsNewestFirst(t *testing.T) {
	store, err := NewJSONTaskStore(filepath.Join(t.TempDir(), "tasks.json"))
	if err != nil {
		t.Fatalf("NewJSONTaskStore failed: %v", err)
	}

	for _, task := range []*Task{
		{ID: "older", CreatedAt: time.Unix(100, 0).UTC()},
		{ID: "newer", CreatedAt: time.Unix(200, 0).UTC()},
	} {
		if err := store.Create(context.Background(), task); err != nil {
			t.Fatalf("Create %q failed: %v", task.ID, err)
		}
	}

	tasks, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 2 || tasks[0].ID != "newer" || tasks[1].ID != "older" {
		t.Fatalf("unexpected task order: %+v", tasks)
	}
}

func TestJSONTaskStoreDeleteRemovesPersistedTask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	store, err := NewJSONTaskStore(path)
	if err != nil {
		t.Fatalf("NewJSONTaskStore failed: %v", err)
	}

	task := &Task{
		ID:          "task-delete",
		Code:        "ABCD-123",
		Stage:       TaskStageCompleted,
		StageStatus: TaskStageStatusDone,
		CreatedAt:   time.Unix(100, 0).UTC(),
		UpdatedAt:   time.Unix(100, 0).UTC(),
	}
	if err := store.Create(context.Background(), task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	deleted, err := store.Delete(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if deleted.ID != task.ID {
		t.Fatalf("unexpected deleted task: %+v", deleted)
	}

	reopened, err := NewJSONTaskStore(path)
	if err != nil {
		t.Fatalf("reopen NewJSONTaskStore failed: %v", err)
	}
	tasks, err := reopened.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected deleted task to stay removed, got %+v", tasks)
	}
}
