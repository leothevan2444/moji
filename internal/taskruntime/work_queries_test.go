package taskruntime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestBackgroundQueriesCoverPagesAndSkipColdPayloads(t *testing.T) {
	s, m := relationalStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	// Insert enough tasks for several batches with invalid attachments/checkpoints
	// off the worker's selected path. Header iteration must never decode them.
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for i := 0; i < 300; i++ {
			id := fmt.Sprintf("task-%04d", i)
			if _, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES(?,'MANUAL','','SOURCING','PENDING',?,?)`, id, formatDurableTime(now), formatDurableTime(now)); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_checkpoints(task_id,sourcing_request_json) VALUES(?,'{}')`, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	if err := walkTaskWork(ctx, s, "SOURCING", func(task *Task) error {
		if seen[task.ID] {
			t.Fatal("duplicate work item")
		}
		seen[task.ID] = true
		if task.Stage != TaskStageSourcing {
			t.Fatal("missing lifecycle header")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 300 {
		t.Fatalf("skipped page: visited %d", len(seen))
	}
	if err := walkTaskWork(ctx, s, "INGEST", func(*Task) error { t.Fatal("sourcing entered ingest queue"); return nil }); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := walkTaskWork(canceled, s, "SOURCING", func(*Task) error { return nil }); err == nil {
		t.Fatal("ignored cancellation")
	}
}
