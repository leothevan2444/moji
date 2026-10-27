package taskruntime

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestUnknownCheckpointVersionPreventsReadsAndExternalActions(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	task := richRelationalTask()
	if err := s.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec("UPDATE task_checkpoints SET payload_version=2 WHERE task_id=?", task.ID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Find(ctx, task.ID); err == nil {
		t.Fatal("unknown checkpoint decoded")
	}
	called := false
	if err := s.RunTaskCommand(ctx, task, "TASK_DELETE", "hash", nil, func() error { called = true; return nil }); err == nil || called {
		t.Fatal("future checkpoint mutated external state")
	}
}
func TestMalformedOrFutureFilePlanIsNeverSilentlyReduced(t *testing.T) {
	for _, plan := range []string{`[]`, `null`, `{"files":[],"future_policy":"keep"}`} {
		t.Run(plan, func(t *testing.T) {
			s, _ := relationalStore(t)
			task := richRelationalTask()
			task.FilePlanJSON = plan
			if err := s.Create(context.Background(), task); err == nil {
				t.Fatal("unrecognized file plan accepted")
			}
			if _, err := s.Find(context.Background(), task.ID); err == nil {
				t.Fatal("failed creation left a partial task")
			}
		})
	}
}
