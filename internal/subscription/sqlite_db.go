package subscription

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed sqlite_schema.sql
var sqliteSchema string

func ensureSubscriptionSQLiteSchema(db *sqlx.DB) error {
	if err := resetLegacySubscriptionSchemaIfNeeded(db); err != nil {
		return err
	}
	if _, err := db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("subscription: initialize sqlite schema: %w", err)
	}
	if err := ensureSubscriptionSQLiteRuntimeState(db); err != nil {
		return err
	}
	return nil
}

func resetLegacySubscriptionSchemaIfNeeded(db *sqlx.DB) error {
	needsReset, err := subscriptionSchemaNeedsReset(db)
	if err != nil {
		return err
	}
	if !needsReset {
		return nil
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("subscription: begin sqlite schema reset: %w", err)
	}
	defer tx.Rollback()

	statements := []string{
		`PRAGMA foreign_keys = OFF`,
		`DROP TABLE IF EXISTS subscription_performer_releases`,
		`DROP TABLE IF EXISTS subscription_release_entities`,
		`DROP TABLE IF EXISTS subscription_performer_state`,
		`PRAGMA foreign_keys = ON`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("subscription: reset sqlite schema with %q: %w", statement, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("subscription: commit sqlite schema reset: %w", err)
	}
	return nil
}

func subscriptionSchemaNeedsReset(db *sqlx.DB) (bool, error) {
	exists, err := sqliteTableExists(db, "subscription_release_entities")
	if err != nil {
		return false, err
	}
	if exists {
		for _, column := range []string{"query", "last_error"} {
			hasColumn, err := sqliteColumnExists(db, "subscription_release_entities", column)
			if err != nil {
				return false, err
			}
			if hasColumn {
				return true, nil
			}
		}
	}

	exists, err = sqliteTableExists(db, "subscription_performer_releases")
	if err != nil {
		return false, err
	}
	if exists {
		for _, column := range []string{"id", "created_at", "updated_at"} {
			hasColumn, err := sqliteColumnExists(db, "subscription_performer_releases", column)
			if err != nil {
				return false, err
			}
			if hasColumn {
				return true, nil
			}
		}
	}

	return false, nil
}

func ensureSubscriptionSQLiteRuntimeState(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("subscription: begin sqlite runtime state tx: %w", err)
	}
	defer tx.Rollback()

	if err := clearDanglingSubscriptionTaskReferences(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("subscription: commit sqlite runtime state tx: %w", err)
	}
	return nil
}

func clearDanglingSubscriptionTaskReferences(tx *sqlx.Tx) error {
	releaseTableExists, err := sqliteTableExistsTx(tx, "subscription_release_entities")
	if err != nil {
		return err
	}
	if !releaseTableExists {
		return nil
	}

	taskTableExists, err := sqliteTableExistsTx(tx, "tasks")
	if err != nil {
		return err
	}
	if !taskTableExists {
		return nil
	}

	if _, err := tx.Exec(`
UPDATE subscription_release_entities
SET task_id = NULL
WHERE task_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM tasks
    WHERE tasks.id = subscription_release_entities.task_id
  )`); err != nil {
		return fmt.Errorf("subscription: clear dangling subscription task references: %w", err)
	}
	return nil
}

func sqliteTableExists(db sqlx.ExtContext, table string) (bool, error) {
	var count int
	if err := sqlx.GetContext(context.Background(), db, &count, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table); err != nil {
		return false, fmt.Errorf("subscription: inspect sqlite table %s: %w", table, err)
	}
	return count > 0, nil
}

func sqliteTableExistsTx(tx *sqlx.Tx, table string) (bool, error) {
	var count int
	if err := tx.Get(&count, `SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table); err != nil {
		return false, fmt.Errorf("subscription: inspect sqlite table %s in tx: %w", table, err)
	}
	return count > 0, nil
}

func sqliteColumnExists(db *sqlx.DB, table string, column string) (bool, error) {
	rows, err := db.Queryx(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("subscription: inspect sqlite table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &pk); err != nil {
			return false, fmt.Errorf("subscription: scan sqlite table info for %s: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("subscription: iterate sqlite table info for %s: %w", table, err)
	}
	return false, nil
}

func sqliteTaskExists(ctx context.Context, tx *sqlx.Tx, taskID string) (bool, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false, nil
	}

	tableExists, err := sqliteTableExistsTx(tx, "tasks")
	if err != nil {
		return false, err
	}
	if !tableExists {
		return false, nil
	}

	var count int
	if err := tx.GetContext(ctx, &count, `SELECT COUNT(1) FROM tasks WHERE id = ?`, taskID); err != nil {
		return false, err
	}
	return count > 0, nil
}

func nullableStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
