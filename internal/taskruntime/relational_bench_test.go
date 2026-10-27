package taskruntime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

func benchmarkRelationalStore(b *testing.B) *RelationalTaskStore {
	b.Helper()
	ctx := context.Background()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(b.TempDir()))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { m.Close() })
	now := formatDurableTime(time.Now())
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for i := 0; i < 1000; i++ {
			id := fmt.Sprintf("task-%04d", i)
			stage, status := "COMPLETED", "DONE"
			if i < 50 {
				stage, status = "DOWNLOADING", "RUNNING"
			}
			if _, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES(?,'MANUAL','',?,?,?,?)`, id, stage, status, now, now); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_downloads(task_id,torrent_hash) VALUES(?,?)`, id, id); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_checkpoints(task_id) VALUES(?)`, id); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_ingest(task_id) VALUES(?)`, id); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_candidates(task_id) VALUES(?)`, id); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT INTO task_attachments(task_id,torrent_file_data) VALUES(?,zeroblob(8192))`, id); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Fatal(err)
	}
	return NewRelationalTaskStore(m.Core, m.Observations)
}
func BenchmarkRelationalEventStats1000Tasks(b *testing.B) {
	s := benchmarkRelationalStore(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.TaskStats(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkRelationalMonitorPage1000Tasks(b *testing.B) {
	s := benchmarkRelationalStore(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.TaskWorkPage(context.Background(), "MONITOR", "", 128); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkRelationalProgressOnly1000Tasks(b *testing.B) {
	s := benchmarkRelationalStore(b)
	ctx := context.Background()
	task, err := s.Find(ctx, "task-0000")
	if err != nil {
		b.Fatal(err)
	}
	writes := s.core.Metrics().Writes
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.Progress = float64(i%100) / 100
		task.UpdatedAt = time.Now().UTC()
		if err := s.Update(ctx, task); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if s.core.Metrics().Writes != writes {
		b.Fatal("progress wrote core")
	}
	b.ReportMetric(0, "core-writes/op")
}
