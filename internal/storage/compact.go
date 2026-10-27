package storage

import (
	"context"
	"errors"
	"os"
)

type CompactionResult struct{ BeforeBytes, AfterBytes int64 }

// Compact is an explicit offline maintenance operation. The CLI holds the
// instance directory lock and creates a protected backup before calling it.
// VACUUM uses SQLite's transaction/rollback mechanism; no .db/WAL is unlinked.
func (d *Database) Compact(ctx context.Context) (result CompactionResult, err error) {
	if d.closed.Load() {
		return result, ErrClosed
	}
	select {
	case d.writerGate <- struct{}{}:
	case <-ctx.Done():
		return result, ctx.Err()
	}
	defer func() { <-d.writerGate }()
	if d.closed.Load() {
		return result, ErrClosed
	}
	checkpoint := func() error {
		var c Checkpoint
		if err := d.writer.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&c.Busy, &c.LogFrames, &c.CheckpointedFrames); err != nil {
			return err
		}
		if c.Busy != 0 {
			return errors.New("storage: compaction refused while a reader holds a snapshot")
		}
		return nil
	}
	if err := checkpoint(); err != nil {
		return result, err
	}
	info, err := os.Stat(d.Path)
	if err != nil {
		return result, err
	}
	result.BeforeBytes = info.Size()
	if _, err := d.writer.ExecContext(ctx, "VACUUM"); err != nil {
		return result, err
	}
	if err := checkpoint(); err != nil {
		return result, err
	}
	info, err = os.Stat(d.Path)
	if err != nil {
		return result, err
	}
	result.AfterBytes = info.Size()
	return result, nil
}
