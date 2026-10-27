package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestTaskCommandIntentionPrecedesExternalActionAndRetainsAmbiguity(t *testing.T) {
	s, m := relationalStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	called := false
	actionErr := errors.New("response lost")
	err := s.RunTaskCommand(ctx, task, "TASK_ADD", "hash", nil, func() error {
		called = true
		var count int
		if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM external_commands WHERE state='SENT'"); err != nil || count != 1 {
			t.Fatal("network action ran before durable intention", err)
		}
		// A concurrent core write succeeds: no transaction spans the external call.
		if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec("UPDATE download_policies SET ratio=1"); return err }); err != nil {
			t.Fatal(err)
		}
		return actionErr
	})
	if !called || !errors.Is(err, actionErr) {
		t.Fatal("wrong action result", err)
	}
	var state string
	if err := m.Core.Reader.Get(&state, "SELECT state FROM external_commands"); err != nil || state != "UNCERTAIN" {
		t.Fatalf("lost ambiguous outcome: %s %v", state, err)
	}
	if err := m.ReconcileTaskCommands(ctx, map[string]bool{"hash": true}); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.Reader.Get(&state, "SELECT state FROM external_commands"); err != nil || state != "CONFIRMED" {
		t.Fatal("observed result not reconciled", err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER fail_intention BEFORE INSERT ON external_commands BEGIN SELECT RAISE(ABORT,'disk failure'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	called = false
	if err := s.RunTaskCommand(ctx, task, "TASK_DELETE", "hash", nil, func() error { called = true; return nil }); err == nil || called {
		t.Fatal("external action ran without a durable intention")
	}
}
func TestRelationalRestartPreservesRevisionAndAdvancesGeneration(t *testing.T) {
	_, _, q, old, _ := restartFixture(t)
	store, _ := relationalStore(t)
	ctx := context.Background()
	task := cloneTask(old)
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	stale := cloneTask(task)
	service, err := NewService(fakeTracker{}, q, store)
	if err != nil {
		t.Fatal(err)
	}
	next, err := service.RestartTaskSourcing(ctx, task.ID, &restartScanner{})
	if err != nil {
		t.Fatal(err)
	}
	if next.Generation != 1 || next.Revision <= task.Revision || next.Stage != TaskStageSourcing {
		t.Fatalf("wrong restart checkpoint: %+v", next)
	}
	if err := store.Update(ctx, stale); !errors.Is(err, ErrTaskRevisionConflict) {
		t.Fatal("stale pre-restart task accepted", err)
	}
	loaded, err := store.Find(ctx, task.ID)
	if err != nil || loaded.Generation != 1 || loaded.FilePlanJSON != "" {
		t.Fatal("restart generation not durable", err)
	}
}
