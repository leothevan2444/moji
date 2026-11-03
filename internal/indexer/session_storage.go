package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

type sessionPayload struct {
	Version       int            `json:"version"`
	Cookies       []cookieRecord `json:"cookies"`
	Authenticated bool           `json:"authenticated"`
	UserAgent     string         `json:"userAgent"`
}
type sessionRepository struct{ db *storage.Database }

func (r sessionRepository) load(ctx context.Context, key string) (*sessionPayload, error) {
	var data string
	err := r.db.Reader.QueryRowContext(ctx, `SELECT payload_json FROM indexer_sessions WHERE session_key=? AND expires_at>?`, key, instant(time.Now())).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 256<<10 {
		return nil, ErrResponseBudget
	}
	var payload sessionPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil || payload.Version != 1 || len(payload.Cookies) > 512 || len(payload.UserAgent) > 4096 {
		return nil, ErrInvalidSettings
	}
	return &payload, nil
}

func (r sessionRepository) save(ctx context.Context, key, instance string, payload sessionPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(data) > 256<<10 {
		return ErrResponseBudget
	}
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM indexer_sessions WHERE session_key IN (SELECT session_key FROM indexer_sessions WHERE expires_at<=? ORDER BY expires_at,session_key LIMIT 128)`, instant(time.Now())); err != nil {
			return err
		}
		var count int
		if err := tx.GetContext(ctx, &count, `SELECT count(*) FROM indexer_sessions`); err != nil {
			return err
		}
		if count >= 512 {
			var exists int
			if err := tx.GetContext(ctx, &exists, `SELECT count(*) FROM indexer_sessions WHERE session_key=?`, key); err != nil {
				return err
			}
			if exists == 0 {
				return ErrBusy
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO indexer_sessions(session_key,instance_id,payload_version,payload_json,expires_at,updated_at) VALUES(?,?,1,?,?,?) ON CONFLICT(session_key) DO UPDATE SET payload_json=excluded.payload_json,expires_at=excluded.expires_at,updated_at=excluded.updated_at`, key, instance, string(data), instant(time.Now().Add(24*time.Hour)), instant(time.Now()))
		return err
	})
}

func instant(now time.Time) string { return now.UTC().Format("2006-01-02T15:04:05.000000000Z") }
