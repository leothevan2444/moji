package subscription

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

type releaseRetryStore interface {
	DueReleaseRetries(context.Context, time.Time, int) ([]ReleaseReview, error)
	GetReleaseReview(context.Context, string) (*ReleaseReview, error)
}

func (s *MemoryStore) GetReleaseReview(_ context.Context, key string) (*ReleaseReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.reviews[key]
	if !ok {
		return nil, nil
	}
	r = cloneReleaseReview(r)
	return &r, nil
}
func (s *MemoryStore) DueReleaseRetries(ctx context.Context, now time.Time, limit int) ([]ReleaseReview, error) {
	rows, err := s.ListReleaseReviews(ctx)
	if err != nil {
		return nil, err
	}
	out := []ReleaseReview{}
	for _, r := range rows {
		if r.Status == ReviewFailed && r.NextRetryAt != nil && !r.NextRetryAt.After(now) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NextRetryAt.Equal(*out[j].NextRetryAt) {
			return out[i].Key < out[j].Key
		}
		return out[i].NextRetryAt.Before(*out[j].NextRetryAt)
	})
	return out[:min(max(limit, 0), len(out))], nil
}

func (s *SQLiteStore) GetReleaseReview(ctx context.Context, key string) (*ReleaseReview, error) {
	var rows []string
	if err := s.db.SelectContext(ctx, &rows, `SELECT record_json FROM subscription_release_reviews WHERE release_key=?`, key); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	var r ReleaseReview
	if err := json.Unmarshal([]byte(rows[0]), &r); err != nil {
		return nil, err
	}
	return &r, nil
}
func (s *SQLiteStore) DueReleaseRetries(ctx context.Context, now time.Time, limit int) ([]ReleaseReview, error) {
	status := "json_extract(record_json,'$.Status')"
	if s.core != nil {
		status = "status"
	}
	var rows []string
	err := s.db.SelectContext(ctx, &rows, `SELECT record_json FROM subscription_release_reviews WHERE `+status+`='FAILED' AND julianday(json_extract(record_json,'$.NextRetryAt'))<=julianday(?) ORDER BY json_extract(record_json,'$.NextRetryAt'),release_key LIMIT ?`, now.UTC().Format(time.RFC3339Nano), min(max(limit, 0), 32))
	if err != nil {
		return nil, err
	}
	out := make([]ReleaseReview, 0, len(rows))
	for _, raw := range rows {
		var r ReleaseReview
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
