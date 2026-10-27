package storage

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReconcileTaskCommands uses a successful complete qB listing, never an outage,
// to confirm add/removal outcomes. It observes; it never retries a mutation.
func (m *Manager) ReconcileTaskCommands(ctx context.Context, present map[string]bool) error {
	after := ""
	for {
		var rows []struct {
			ID, Kind string
			Hash     string `db:"target_hash"`
		}
		if err := m.Core.Reader.SelectContext(ctx, &rows, `SELECT id,kind,target_hash FROM external_commands WHERE id>? AND state IN ('PENDING','SENT','UNCERTAIN') AND kind IN ('TASK_ADD','TASK_DELETE','TASK_RESTART_DELETE') AND target_hash<>'' ORDER BY id LIMIT 128`, after); err != nil {
			return err
		}
		var confirmed []string
		for _, row := range rows {
			exists := present[strings.ToLower(row.Hash)]
			if row.Kind == "TASK_ADD" && exists || row.Kind != "TASK_ADD" && !exists {
				confirmed = append(confirmed, row.ID)
			}
			after = row.ID
		}
		if len(confirmed) > 0 {
			if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
				for _, id := range confirmed {
					if _, err := tx.ExecContext(ctx, `UPDATE external_commands SET state='CONFIRMED',updated_at=? WHERE id=? AND state IN ('PENDING','SENT','UNCERTAIN')`, observationTime(time.Now()), id); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				return err
			}
		}
		if len(rows) < 128 {
			return nil
		}
	}
}
