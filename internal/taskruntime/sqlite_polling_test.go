package taskruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSQLitePollingPersistsProgressWithoutGrowingHistory(t *testing.T) {
	store, err := NewSQLiteTaskStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	ctx := context.Background()
	at := time.Now().UTC()
	task := &Task{ID: "poll-task", Code: "ABCD-123", Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: at, UpdatedAt: at}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	task, err = store.Find(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	task.UpdatedAt = at.Add(time.Minute)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if !task.UpdatedAt.Equal(at) {
		t.Fatal("unchanged poll changed update timestamp")
	}
	task.Progress = .5
	task.UpdatedAt = at.Add(2 * time.Minute)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	got, err := store.Find(ctx, task.ID)
	if err != nil || got.Progress != .5 || got.LastProgressAt == nil || !got.LastProgressAt.Equal(task.UpdatedAt) {
		t.Fatalf("progress not persisted: %+v %v", got, err)
	}
	var count int
	if err := store.db.Get(&count, "SELECT COUNT(*) FROM task_events"); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("polling appended history: %d events", count)
	}
	task.StageStatus = TaskStageStatusBlocked
	task.StageErrorCode = "TORRENT_ERROR"
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := store.db.Get(&count, "SELECT COUNT(*) FROM task_events"); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("state transition history missing: %d events", count)
	}
}

// A deferred read-then-write transaction lets another connection commit between
// its SELECT and UPDATE, which makes the later upgrade fail with SQLITE_BUSY.
func TestSQLiteWriteTransactionsReserveWriterBeforeReading(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.db")
	store, err := NewSQLiteTaskStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	other, err := sql.Open("sqlite", path+"?_pragma=busy_timeout%280%29")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	ctx := context.Background()
	tx, err := store.db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var count int
	if err := tx.Get(&count, "SELECT COUNT(*) FROM tasks"); err != nil {
		t.Fatal(err)
	}
	_, err = other.Exec("INSERT INTO task_store_meta(key,value) VALUES('concurrent','value')")
	if err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("write reservation missing: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	readTx, err := store.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	if err := readTx.Get(&count, "SELECT COUNT(*) FROM tasks"); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Exec("INSERT INTO task_store_meta(key,value) VALUES('concurrent','value')"); err != nil {
		t.Fatalf("read-only overview blocked another writer: %v", err)
	}
}
