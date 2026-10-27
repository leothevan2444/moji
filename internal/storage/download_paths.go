package storage

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// recoverDownloadPaths is an idempotent bridge for the pre-release v1 layout.
// New v2 repositories persist this input directly. A runtime observation can
// fill an absent path; it never overwrites an existing durable resolution.
func (m *Manager) recoverDownloadPaths(ctx context.Context) error {
	var ids []string
	if err := m.Core.Reader.SelectContext(ctx, &ids, "SELECT task_id FROM task_downloads WHERE content_path='' ORDER BY task_id"); err != nil {
		return err
	}
	type value struct{ ID, Path string }
	var pending []value
	for _, id := range ids {
		if observed, ok := m.Observations.Get(id); ok && observed.ContentPath != "" {
			pending = append(pending, value{id, observed.ContentPath})
		}
	}
	for start := 0; start < len(pending); start += 128 {
		end := start + 128
		if end > len(pending) {
			end = len(pending)
		}
		if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
			for _, item := range pending[start:end] {
				if _, err := tx.ExecContext(ctx, "UPDATE task_downloads SET content_path=? WHERE task_id=? AND content_path=''", item.Path, item.ID); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
