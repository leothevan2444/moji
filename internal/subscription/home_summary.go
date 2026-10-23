package subscription

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"
)

type HomeSummary struct {
	Total, Attention, Unchecked int
	LastCheckedAt               *time.Time
	AttentionPerformerIDs       []string
}

// HomeSummary reads persisted checks only; it never fetches Stash's catalogue.
func (s *Service) HomeSummary(ctx context.Context) (HomeSummary, error) {
	if store, ok := s.store.(interface {
		HomeSummary(context.Context) (HomeSummary, error)
	}); ok {
		return store.HomeSummary(ctx)
	}
	states, err := s.store.List(ctx)
	if err != nil {
		return HomeSummary{}, err
	}
	result := HomeSummary{}
	for _, state := range states {
		if state == nil {
			continue
		}
		result.Total++
		if strings.TrimSpace(state.LastError) != "" {
			result.Attention++
			result.AttentionPerformerIDs = append(result.AttentionPerformerIDs, state.PerformerID)
		}
		if state.LastCheckedAt == nil {
			result.Unchecked++
		} else if result.LastCheckedAt == nil || state.LastCheckedAt.After(*result.LastCheckedAt) {
			result.LastCheckedAt = state.LastCheckedAt
		}
	}
	sort.Strings(result.AttentionPerformerIDs)
	result.AttentionPerformerIDs = result.AttentionPerformerIDs[:min(3, len(result.AttentionPerformerIDs))]
	return result, nil
}

func (s *SQLiteStore) HomeSummary(ctx context.Context) (HomeSummary, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return HomeSummary{}, err
	}
	defer tx.Rollback()
	var row struct {
		Total       int            `db:"total"`
		Attention   int            `db:"attention"`
		Unchecked   int            `db:"unchecked"`
		LastChecked sql.NullString `db:"last_checked"`
	}
	if err := tx.GetContext(ctx, &row, `SELECT COUNT(*) total, COUNT(CASE WHEN TRIM(COALESCE(last_error,'')) != '' THEN 1 END) attention, COUNT(CASE WHEN last_checked_at IS NULL THEN 1 END) unchecked, MAX(last_checked_at) last_checked FROM subscription_performer_state`); err != nil {
		return HomeSummary{}, err
	}
	result := HomeSummary{Total: row.Total, Attention: row.Attention, Unchecked: row.Unchecked}
	if row.LastChecked.Valid {
		checked, err := time.Parse(time.RFC3339Nano, row.LastChecked.String)
		if err != nil {
			return HomeSummary{}, err
		}
		result.LastCheckedAt = &checked
	}
	if err := tx.SelectContext(ctx, &result.AttentionPerformerIDs, `SELECT performer_id FROM subscription_performer_state WHERE TRIM(COALESCE(last_error,'')) != '' ORDER BY performer_id ASC LIMIT 3`); err != nil {
		return HomeSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return HomeSummary{}, err
	}
	return result, nil
}
