package taskruntime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestTaskPageUsesHeadersAndLiveObservations(t *testing.T) {
	store, m := relationalStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for n := 0; n < 35; n++ {
		task := &Task{ID: fmt.Sprintf("page-%02d", n), Code: fmt.Sprintf("PAGE-%02d", n), Source: TaskSourceManual, Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, CreatedAt: now.Add(time.Duration(n) * time.Minute), UpdatedAt: now}
		if n < 4 {
			task.StageStatus = TaskStageStatusBlocked
		}
		if n == 4 {
			task.Stage = TaskStageSourcing
			task.StageStatus = TaskStageStatusPending
			task.SourcingNextCheckAt = &now
		}
		if n == 5 {
			task.Stage = TaskStageCompleted
			task.StageStatus = TaskStageStatusDone
			task.IngestCompletedAt = &now
		}
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	// Cold payloads must not participate in list/home/queue reads.
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE task_checkpoints SET payload_version=999`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	m.Observations.Put(storage.TaskObservation{TaskID: "page-00", TorrentName: "Live torrent title", Progress: .99, ObservedAt: now.Add(10 * time.Hour).Format(time.RFC3339Nano)})
	page, err := store.TaskPage(ctx, TaskPageInput{Page: 2, PageSize: 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 12 || page.TotalCount != 35 || page.Items[0].ID != "page-22" || page.TotalPages != 3 {
		t.Fatalf("wrong page: %+v", page)
	}
	if page.Stats.Attention != 4 || page.Stats.Waiting != 1 || page.Stats.Completed != 1 {
		t.Fatalf("nonexclusive counts: %+v", page.Stats)
	}
	progress, err := store.TaskPage(ctx, TaskPageInput{Sort: "PROGRESS", PageSize: 12})
	if err != nil || progress.Items[0].ID != "page-00" || progress.Items[0].Progress != .99 {
		t.Fatalf("stale progress ordering: %+v %v", progress, err)
	}
	updated, err := store.TaskPage(ctx, TaskPageInput{Sort: "UPDATED"})
	if err != nil || updated.Items[0].ID != "page-00" {
		t.Fatalf("stale updated ordering: %+v %v", updated, err)
	}
	found, err := store.TaskPage(ctx, TaskPageInput{Search: "torrent title"})
	if err != nil || found.TotalCount != 1 {
		t.Fatalf("live name search: %+v %v", found, err)
	}
	waiting, err := store.TaskPage(ctx, TaskPageInput{Status: "WAITING"})
	if err != nil || waiting.TotalCount != 1 || waiting.Items[0].ID != "page-04" {
		t.Fatalf("waiting filter: %+v %v", waiting, err)
	}
	recent, err := store.TaskPage(ctx, TaskPageInput{CompletedWithin24Hours: true})
	if err != nil || recent.TotalCount != 1 {
		t.Fatalf("recent filter: %+v %v", recent, err)
	}
	selected, err := store.TaskPage(ctx, TaskPageInput{IDs: []string{"page-00", "page-34"}})
	if err != nil || len(selected.Items) != 2 {
		t.Fatalf("cross-page IDs: %+v %v", selected, err)
	}
	clamped, err := store.TaskPage(ctx, TaskPageInput{Page: 999999999, PageSize: 24})
	if err != nil || clamped.Page != 2 || len(clamped.Items) != 11 {
		t.Fatalf("page clamp: %+v %v", clamped, err)
	}
	if _, err := store.TaskPage(ctx, TaskPageInput{PageSize: 101}); err == nil {
		t.Fatal("unbounded page accepted")
	}
	home, err := readHomeTaskSnapshot(ctx, store)
	if err != nil || len(home.AttentionTasks) != 3 || len(home.ActiveTasks) != 5 || home.CompletedLast24Hours != 1 {
		t.Fatalf("home must use bounded headers: %+v %v", home, err)
	}
}

func TestWorkspaceSnapshotBoundaryAndIndependentQueue(t *testing.T) {
	base, _ := relationalStore(t)
	bus := NewTaskEventBus(2)
	defer bus.Close()
	store, err := NewEventingTaskStore(base, bus)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := bus.SubscribeWithReady(ctx)
	ready := <-events
	if ready.Type != TaskEventReady || ready.Sequence != 0 {
		t.Fatal("missing initial boundary")
	}
	now := time.Now().UTC()
	for n := 0; n < 4; n++ {
		task := &Task{ID: fmt.Sprintf("queue-%d", n), Source: TaskSourceManual, Stage: TaskStageSourcing, StageStatus: TaskStageStatusBlocked, CreatedAt: now.Add(time.Duration(n) * time.Second), UpdatedAt: now}
		if err := store.Create(ctx, task); err != nil {
			t.Fatal(err)
		}
	}
	page, err := store.TaskPage(ctx, TaskPageInput{Status: "COMPLETED", PageSize: 1})
	if err != nil || page.TotalCount != 0 || page.Sequence != 4 || page.StreamID != ready.StreamID {
		t.Fatalf("snapshot boundary: %+v %v", page, err)
	}
	queue, err := store.AttentionQueue(ctx)
	if err != nil || len(queue.Items) != 4 || queue.Items[0].ID != "queue-0" || queue.Sequence != 4 {
		t.Fatalf("queue incorrectly scoped to list: %+v %v", queue, err)
	}
	// Even if the final event overflows a slow subscriber, a READY boundary forces
	// recovery without waiting for a later event to expose a sequence gap.
	foundResync := false
	for len(events) > 0 {
		e := <-events
		if e.Type == TaskEventReady && e.Sequence >= 3 {
			foundResync = true
		}
	}
	if !foundResync {
		t.Fatal("overflow did not signal recovery")
	}
}

func TestFirstProgressEventAfterRestartDoesNotChangeMembership(t *testing.T) {
	base, _ := relationalStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	task := &Task{ID: "existing", Source: TaskSourceManual, Stage: TaskStageDownloading, StageStatus: TaskStageStatusRunning, Progress: .2, CreatedAt: now, UpdatedAt: now}
	if err := base.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	bus := NewTaskEventBus(8)
	defer bus.Close()
	store, _ := NewEventingTaskStore(base, bus)
	stream, cancel := context.WithCancel(ctx)
	defer cancel()
	events := bus.Subscribe(stream)
	task.Progress = .5
	task.UpdatedAt = now.Add(time.Second)
	if err := store.Update(ctx, task); err != nil {
		t.Fatal(err)
	}
	if event := <-events; event.ListChanged {
		t.Fatal("a restarted publisher treated an ordinary progress update as membership change")
	}
}
