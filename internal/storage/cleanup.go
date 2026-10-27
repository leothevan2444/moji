package storage

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// PruneRuntime bounds disposable diagnostics by age and by the most recent
// 100000 event IDs. Important audit/choices/commands are never TTL-deleted.
func (m *Manager) PruneRuntime(ctx context.Context, now time.Time) (int64, error) {
	var deleted int64
	err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM diagnostics WHERE id IN(SELECT id FROM diagnostics WHERE created_at<? OR id<=COALESCE((SELECT max(id)-100000 FROM diagnostics),0) ORDER BY id LIMIT 1000)`, observationTime(now.Add(-14*24*time.Hour)))
		if err != nil {
			return err
		}
		deleted, err = result.RowsAffected()
		return err
	})
	return deleted, err
}
