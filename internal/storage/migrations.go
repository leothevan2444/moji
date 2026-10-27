package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Migration struct {
	Version   int
	Name, SQL string
}

var ErrUnrecognizedSchema = errors.New("storage: unversioned nonempty database requires explicit migration")
var ErrSchemaTooNew = errors.New("storage: database schema is newer than this program")
var ErrMigrationChanged = errors.New("storage: applied migration checksum changed")

func checksum(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

// Migrate validates all existing versions before applying any changes. Each
// migration and its ledger record commit together; there is no reset fallback.
func (d *Database) Migrate(ctx context.Context, migrations []Migration) error {
	for i, m := range migrations {
		if m.Version != i+1 || m.Name == "" || m.SQL == "" {
			return errors.New("storage: migrations must be named, nonempty and ordered from version one")
		}
	}
	type applied struct {
		Version  int    `db:"version"`
		Name     string `db:"name"`
		Checksum string `db:"checksum"`
	}
	var exists int
	if err := d.Reader.GetContext(ctx, &exists, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'"); err != nil {
		return err
	}
	if exists == 0 {
		var count int
		if err := d.Reader.GetContext(ctx, &count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"); err != nil {
			return err
		}
		if count != 0 {
			return ErrUnrecognizedSchema
		}
	}
	var rows []applied
	if exists != 0 {
		if err := d.Reader.SelectContext(ctx, &rows, "SELECT version,name,checksum FROM schema_migrations ORDER BY version"); err != nil {
			return err
		}
	}
	if exists != 0 && len(rows) == 0 {
		var count int
		if err := d.Reader.GetContext(ctx, &count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name<>'schema_migrations'"); err != nil {
			return err
		}
		if count != 0 {
			return ErrUnrecognizedSchema
		}
	}
	for i, row := range rows {
		if row.Version > len(migrations) {
			return fmt.Errorf("%w: version %d", ErrSchemaTooNew, row.Version)
		}
		if row.Version != i+1 || row.Checksum != checksum(migrations[i].SQL) || row.Name != migrations[i].Name {
			return fmt.Errorf("%w: version %d", ErrMigrationChanged, row.Version)
		}
	}
	for _, m := range migrations[len(rows):] {
		err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
			if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY,name TEXT NOT NULL,checksum TEXT NOT NULL,applied_at TEXT NOT NULL) STRICT`); err != nil {
				return err
			}
			// Another caller cannot apply the same migration behind our validation.
			var count int
			if err := tx.GetContext(ctx, &count, "SELECT count(*) FROM schema_migrations"); err != nil {
				return err
			}
			if count != m.Version-1 {
				return errors.New("storage: migration state changed; retry initialization")
			}
			if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
				return fmt.Errorf("migration %d %s: %w", m.Version, m.Name, err)
			}
			_, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)", m.Version, m.Name, checksum(m.SQL), time.Now().UTC().Format(time.RFC3339Nano))
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadVersion never mutates a database, including an unrecognized legacy file.
func ReadVersion(ctx context.Context, db *sqlx.DB) (int, error) {
	var version sql.NullInt64
	err := db.GetContext(ctx, &version, "SELECT MAX(version) FROM schema_migrations")
	if err != nil {
		return 0, err
	}
	return int(version.Int64), nil
}
