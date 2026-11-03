package indexer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

type HealthKey struct{ InstanceID, ConfigRevision, BundleRevision string }
type HealthOperation string

const (
	HealthSearch HealthOperation = "SEARCH"
	HealthFetch  HealthOperation = "FETCH"
)

type Observation struct {
	StartedAt, FinishedAt    time.Time
	ErrorCode, CorrelationID string
	RetryAt                  *time.Time
}
type HealthState struct{ Search, Fetch *Observation }
type HealthStore struct{ db *storage.Database }

func NewHealthStore(manager *storage.Manager) (*HealthStore, error) {
	if manager == nil || manager.Runtime == nil {
		return nil, ErrNotConfigured
	}
	return &HealthStore{db: manager.Runtime}, nil
}
func HealthScope(instance InstanceSnapshot) HealthKey {
	return HealthKey{InstanceID: instance.ID, ConfigRevision: instance.ConfigRevision, BundleRevision: instance.BundleRevision}
}
func validHealthKey(key HealthKey) bool {
	return key.InstanceID != "" && key.ConfigRevision != "" && len(key.InstanceID) <= 256 && len(key.ConfigRevision) <= 128 && len(key.BundleRevision) <= 128
}

// Observe stores only stable codes and timing, never error messages, URLs or
// account settings. Caller cancellation is not evidence of a site failure.
// A late result from an older started request cannot replace a newer attempt.
func (h *HealthStore) Observe(ctx context.Context, instance InstanceSnapshot, op HealthOperation, started time.Time, result error, correlation string) (bool, error) {
	if errors.Is(result, context.Canceled) || ctx.Err() != nil {
		return false, nil
	}
	key := HealthScope(instance)
	if !validHealthKey(key) || (op != HealthSearch && op != HealthFetch) || started.IsZero() {
		return false, ErrInvalidSettings
	}
	finished := time.Now().UTC()
	if _, err := uuid.Parse(correlation); err != nil {
		return false, ErrInvalidSettings
	}
	if started.After(finished) {
		return false, ErrInvalidSettings
	}
	code := ""
	if result != nil {
		code = ErrorCode(result)
	}
	var retry any
	var cooling *CoolingDownError
	if errors.As(result, &cooling) {
		retry = instant(cooling.Until)
	}
	changed := false
	err := h.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO indexer_health(instance_id,config_revision,bundle_revision,operation,started_at,finished_at,error_code,correlation_id,retry_at) VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(instance_id,config_revision,bundle_revision,operation) DO UPDATE SET
started_at=excluded.started_at,finished_at=excluded.finished_at,error_code=excluded.error_code,correlation_id=excluded.correlation_id,retry_at=excluded.retry_at
WHERE excluded.started_at>indexer_health.started_at`, key.InstanceID, key.ConfigRevision, key.BundleRevision, string(op), instant(started), instant(finished), code, correlation, retry)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		changed = n > 0
		// Rebuildable observations have a hard row budget, including old accounts
		// and superseded rules. Eviction returns a site to unverified, not failed.
		_, err = tx.ExecContext(ctx, `DELETE FROM indexer_health WHERE rowid IN (SELECT rowid FROM indexer_health ORDER BY finished_at,instance_id,config_revision,bundle_revision,operation LIMIT MAX(0,(SELECT COUNT(*) FROM indexer_health)-8192))`)
		return err
	})
	return changed, err
}

// Snapshot reads only requested namespaces in bounded batches. The caller must
// supply the current configuration and active rule revision, not browser input.
func (h *HealthStore) Snapshot(ctx context.Context, keys []HealthKey) (map[HealthKey]HealthState, error) {
	if len(keys) > 2048 {
		return nil, ErrResponseBudget
	}
	result := make(map[HealthKey]HealthState)
	for offset := 0; offset < len(keys); offset += 100 {
		end := offset + 100
		if end > len(keys) {
			end = len(keys)
		}
		var conditions []string
		var args []any
		for _, key := range keys[offset:end] {
			if !validHealthKey(key) {
				return nil, ErrInvalidSettings
			}
			conditions = append(conditions, "(instance_id=? AND config_revision=? AND bundle_revision=?)")
			args = append(args, key.InstanceID, key.ConfigRevision, key.BundleRevision)
		}
		rows, err := h.db.Reader.QueryContext(ctx, `SELECT instance_id,config_revision,bundle_revision,operation,started_at,finished_at,error_code,correlation_id,retry_at FROM indexer_health WHERE `+strings.Join(conditions, " OR "), args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var key HealthKey
			var op, started, finished string
			var observation Observation
			var retry *string
			if err = rows.Scan(&key.InstanceID, &key.ConfigRevision, &key.BundleRevision, &op, &started, &finished, &observation.ErrorCode, &observation.CorrelationID, &retry); err != nil {
				break
			}
			observation.StartedAt, err = time.Parse(time.RFC3339Nano, started)
			if err != nil {
				break
			}
			observation.FinishedAt, err = time.Parse(time.RFC3339Nano, finished)
			if err != nil {
				break
			}
			if retry != nil {
				var at time.Time
				at, err = time.Parse(time.RFC3339Nano, *retry)
				if err != nil {
					break
				}
				observation.RetryAt = &at
			}
			state := result[key]
			if op == string(HealthSearch) {
				state.Search = &observation
			} else if op == string(HealthFetch) {
				state.Fetch = &observation
			} else {
				err = ErrInvalidSettings
				break
			}
			result[key] = state
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
