package subscription

import (
	"context"
	"encoding/json"
	"fmt"
)

// Review history is independent of performer snapshots and their display limits.
type ReleaseReviewStore interface {
	ListReleaseReviews(context.Context) ([]ReleaseReview, error)
	PutReleaseReviews(context.Context, []ReleaseReview) error
}

func (s *MemoryStore) ListReleaseReviews(context.Context) ([]ReleaseReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ReleaseReview, 0, len(s.reviews))
	for _, record := range s.reviews {
		out = append(out, cloneReleaseReview(record))
	}
	return out, nil
}

func (s *MemoryStore) PutReleaseReviews(_ context.Context, records []ReleaseReview) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reviews == nil {
		s.reviews = make(map[string]ReleaseReview)
	}
	for _, record := range records {
		s.reviews[record.Key] = cloneReleaseReview(record)
	}
	return nil
}

func cloneReleaseReview(record ReleaseReview) ReleaseReview {
	record.Performers = append([]ReviewPerformer(nil), record.Performers...)
	record.PerformerNames = append([]string(nil), record.PerformerNames...)
	if record.ReviewedAt != nil {
		v := *record.ReviewedAt
		record.ReviewedAt = &v
	}
	return record
}

func (s *SQLiteStore) ListReleaseReviews(ctx context.Context) ([]ReleaseReview, error) {
	if s.core != nil {
		return s.listTypedReviews(ctx)
	}
	var rows []string
	if err := s.db.SelectContext(ctx, &rows, `SELECT record_json FROM subscription_release_reviews ORDER BY release_key`); err != nil {
		return nil, fmt.Errorf("subscription: list release reviews: %w", err)
	}
	out := make([]ReleaseReview, 0, len(rows))
	for _, raw := range rows {
		var record ReleaseReview
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			return nil, fmt.Errorf("subscription: decode release review: %w", err)
		}
		out = append(out, record)
	}
	return out, nil
}

func (s *SQLiteStore) PutReleaseReviews(ctx context.Context, records []ReleaseReview) error {
	if s.core != nil {
		return s.putTypedReviews(ctx, records)
	}
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO subscription_release_reviews (release_key, record_json) VALUES (?, ?)
ON CONFLICT(release_key) DO UPDATE SET record_json = excluded.record_json`, record.Key, string(raw)); err != nil {
			return fmt.Errorf("subscription: save release review: %w", err)
		}
	}
	return tx.Commit()
}
