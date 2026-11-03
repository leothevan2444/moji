package indexer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

const cooldownRows = 8192

type cooldownStore struct{ db *storage.Database }
type persistedCooldown struct {
	Failures int   `db:"failures"`
	Blocked  int64 `db:"blocked_until_ms"`
	Expires  int64 `db:"expires_at_ms"`
}

// NewPersistentNetwork keeps origin/egress throttling separate from account/rule
// health. A changed account or rule must not bypass an IP-level Retry-After.
// Configure this application owner before any requests are admitted.
func NewPersistentNetwork(parent context.Context, manager *storage.Manager, onError func(error)) (*Network, error) {
	if manager == nil || manager.Runtime == nil {
		return nil, ErrNotConfigured
	}
	n := NewNetwork(parent)
	n.cooldowns = &cooldownStore{db: manager.Runtime}
	n.cooldownError = onError
	return n, nil
}
func cooldownKey(raw string) string {
	sum := sha256.Sum256([]byte("indexer-origin-egress-v1\x00" + raw))
	return hex.EncodeToString(sum[:])
}
func (s *cooldownStore) load(ctx context.Context, key string) (persistedCooldown, error) {
	var value persistedCooldown
	err := s.db.Reader.GetContext(ctx, &value, `SELECT failures,blocked_until_ms,expires_at_ms FROM indexer_network_cooldowns WHERE scope_key=?`, key)
	if errors.Is(err, sql.ErrNoRows) {
		return persistedCooldown{}, nil
	}
	if err != nil {
		return persistedCooldown{}, err
	}
	if value.Failures < 1 || value.Failures > 7 || value.Blocked <= 0 || value.Expires < value.Blocked {
		return persistedCooldown{}, ErrStateUnavailable
	}
	if value.Expires <= time.Now().UnixMilli() {
		return persistedCooldown{}, nil
	}
	// A backwards wall-clock adjustment cannot produce unbounded admission loss.
	value.Blocked = min(value.Blocked, time.Now().Add(24*time.Hour).UnixMilli())
	return value, nil
}
func (s *cooldownStore) save(ctx context.Context, key string, gate *originGate) error {
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if gate.failures == 0 {
			_, err := tx.ExecContext(ctx, `DELETE FROM indexer_network_cooldowns WHERE scope_key=?`, key)
			return err
		}
		now := time.Now()
		if _, err := tx.ExecContext(ctx, `DELETE FROM indexer_network_cooldowns WHERE expires_at_ms<=?`, now.UnixMilli()); err != nil {
			return err
		}
		// Never evict an active cooldown to admit another source. On saturation the
		// caller retains the in-memory result and retries persistence before more IO.
		result, err := tx.ExecContext(ctx, `INSERT INTO indexer_network_cooldowns(scope_key,failures,blocked_until_ms,expires_at_ms)
SELECT ?,?,?,? WHERE EXISTS(SELECT 1 FROM indexer_network_cooldowns WHERE scope_key=?) OR (SELECT COUNT(*) FROM indexer_network_cooldowns)<?
ON CONFLICT(scope_key) DO UPDATE SET failures=excluded.failures,blocked_until_ms=excluded.blocked_until_ms,expires_at_ms=excluded.expires_at_ms`, key, min(gate.failures, 7), gate.blocked.UnixMilli(), max(now.Add(30*time.Minute).UnixMilli(), gate.blocked.Add(30*time.Minute).UnixMilli()), key, cooldownRows)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrBusy
		}
		return nil
	})
}
func (n *Network) reportCooldownError() {
	n.mu.Lock()
	emit := time.Since(n.cooldownWarningAt) >= time.Minute
	if emit {
		n.cooldownWarningAt = time.Now()
	}
	n.mu.Unlock()
	if emit && n.cooldownError != nil {
		n.cooldownError(ErrStateUnavailable)
	}
}
func (n *Network) loadCooldown(ctx context.Context, gate *originGate) error {
	if gate.loaded || n.cooldowns == nil {
		return nil
	}
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	value, err := n.cooldowns.load(readCtx, gate.key)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		n.reportCooldownError()
		return ErrStateUnavailable
	}
	gate.loaded = true
	gate.failures = value.Failures
	if value.Blocked > 0 {
		gate.blocked = time.UnixMilli(value.Blocked)
	}
	return nil
}
func (n *Network) persistCooldown(gate *originGate) error {
	if n.cooldowns == nil {
		gate.dirty = false
		return nil
	}
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		gate.dirty = true
		return context.Canceled
	}
	n.wg.Add(1)
	n.mu.Unlock()
	defer n.wg.Done()
	// A received upstream result still needs saving after caller cancellation.
	// Network.Close joins the admitted call, so this bounded write has an owner.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(n.ctx), 2*time.Second)
	defer cancel()
	err := n.cooldowns.save(ctx, gate.key, gate)
	gate.dirty = err != nil
	if err != nil {
		n.reportCooldownError()
		return ErrStateUnavailable
	}
	return nil
}
