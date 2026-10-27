package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
)

// ReadLegacyTasks decodes an explicitly supplied read-only legacy connection.
// It deliberately bypasses old constructors and their DDL/version-upgrade paths.
func ReadLegacyTasks(ctx context.Context, db *sqlx.DB) ([]*Task, error) {
	var rows []sqliteTaskRow
	if err := db.SelectContext(ctx, &rows, taskSelectSQL+" ORDER BY id"); err != nil {
		return nil, err
	}
	out := make([]*Task, 0, len(rows))
	for _, row := range rows {
		task, err := row.toTask()
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

// ImportLegacyTasks operates only on a fresh staging core. The caller publishes
// that generation after all domains and semantic comparisons have succeeded.
func (s *RelationalTaskStore) ImportLegacyTasks(ctx context.Context, tasks []*Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var count int
	if err := s.core.Reader.GetContext(ctx, &count, "SELECT count(*) FROM tasks"); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("taskruntime: legacy import requires an empty staging task table")
	}
	for offset := 0; offset < len(tasks); offset += 32 {
		end := offset + 32
		if end > len(tasks) {
			end = len(tasks)
		}
		if err := s.core.WithWrite(ctx, func(tx *sqlx.Tx) error {
			for _, source := range tasks[offset:end] {
				task := cloneTask(source)
				if task.Revision == 0 {
					task.Revision = 1
				}
				if err := writeRelationalTask(ctx, tx, task, false); err != nil {
					return err
				}
				if err := writeAudit(ctx, tx, task, "legacy-migrated", "", "legacy task preserved"); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		for _, source := range tasks[offset:end] {
			s.remember(source)
		}
	}
	return nil
}

// VerifyImportedTasks compares all domain fields, including choices and cold
// checkpoints; row counts alone are not migration acceptance evidence.
func (s *RelationalTaskStore) VerifyImportedTasks(ctx context.Context, tasks []*Task) error {
	for _, source := range tasks {
		expected := cloneTask(source)
		if expected.Revision == 0 {
			expected.Revision = 1
		}
		actual, err := s.Find(ctx, source.ID)
		if err != nil {
			return err
		}
		if expected.FilePlanJSON != "" {
			plan, err := strictFilePlan(expected.FilePlanJSON)
			if err != nil {
				return err
			}
			expected.setFilePlan(plan)
		}
		if actual.FilePlanJSON != "" {
			actual.setFilePlan(actual.FilePlan())
		}
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("taskruntime: imported task %q differs from source", source.ID)
		}
	}
	return nil
}
