package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type UpcomingRelease struct {
	Release           Release
	OriginalDate      string
	Endpoint          string
	State             string
	FirstSeenAt       time.Time
	MetadataCheckedAt time.Time
	NextCheckAt       time.Time
	LastErrorCode     string
	Revision          int
	Performers        []ReviewPerformer
}
type UpcomingQuery struct {
	Search, PerformerID, Before string
	Page, PageSize, Days        int
}
type UpcomingPage struct {
	Items                                  []UpcomingRelease
	Performers                             []ReviewPerformer
	TotalCount, Page, PageSize, TotalPages int
	SubscribedCount                        int
	Automatic                              bool
	Timezone                               string
	Today                                  string
}
type UpcomingStore interface {
	UpcomingReleaseStates(context.Context, []string) (map[string]string, error)
	GetUpcoming(context.Context, string) (*UpcomingRelease, error)
	PutUpcoming(context.Context, *UpcomingRelease) error
	PutUpcomingReview(context.Context, *UpcomingRelease, ReleaseReview) error
	QueryUpcoming(context.Context, UpcomingQuery, []string) (UpcomingPage, error)
	DueUpcoming(context.Context, time.Time, int) ([]UpcomingRelease, error)
}

var errUpcomingRevision = errors.New("subscription: upcoming release revision changed")

func normalizeUpcomingQuery(q UpcomingQuery) UpcomingQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 {
		q.PageSize = 24
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
	return q
}
func cloneUpcoming(r UpcomingRelease) UpcomingRelease {
	r.Performers = append([]ReviewPerformer(nil), r.Performers...)
	r.Release.PerformerNames = append([]string(nil), r.Release.PerformerNames...)
	return r
}
func (s *MemoryStore) GetUpcoming(_ context.Context, key string) (*UpcomingRelease, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.upcoming[key]
	if !ok {
		return nil, nil
	}
	r = cloneUpcoming(r)
	return &r, nil
}
func (s *MemoryStore) PutUpcoming(_ context.Context, r *UpcomingRelease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.upcoming == nil {
		s.upcoming = map[string]UpcomingRelease{}
	}
	old := s.upcoming[r.Release.Key]
	if old.Revision != r.Revision {
		return errUpcomingRevision
	}
	r.Revision++
	s.upcoming[r.Release.Key] = cloneUpcoming(*r)
	return nil
}
func (s *MemoryStore) QueryUpcoming(_ context.Context, q UpcomingQuery, ids []string) (UpcomingPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = normalizeUpcomingQuery(q)
	page := UpcomingPage{Items: []UpcomingRelease{}, Performers: []ReviewPerformer{}, Page: q.Page, PageSize: q.PageSize}
	active := map[string]bool{}
	for _, id := range ids {
		active[id] = true
	}
	actors := map[string]ReviewPerformer{}
	records := []UpcomingRelease{}
	for _, r := range s.upcoming {
		if r.State == "HANDED_OFF" {
			continue
		}
		match := false
		for _, actor := range r.Performers {
			if active[actor.ID] {
				actors[actor.ID] = actor
				if q.PerformerID == "" || q.PerformerID == actor.ID {
					match = true
				}
			}
		}
		if !match || q.Before != "" && r.Release.Date > q.Before {
			continue
		}
		if q.Search != "" && !strings.Contains(strings.ToLower(r.Release.Code+" "+r.Release.Title), strings.ToLower(q.Search)) {
			continue
		}
		records = append(records, cloneUpcoming(r))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Release.Date != records[j].Release.Date {
			return records[i].Release.Date < records[j].Release.Date
		}
		return records[i].Release.Key < records[j].Release.Key
	})
	page.TotalCount = len(records)
	page.TotalPages = (len(records) + q.PageSize - 1) / q.PageSize
	start := min((q.Page-1)*q.PageSize, len(records))
	page.Items = records[start:min(start+q.PageSize, len(records))]
	for _, a := range actors {
		page.Performers = append(page.Performers, a)
	}
	sort.Slice(page.Performers, func(i, j int) bool { return page.Performers[i].Name < page.Performers[j].Name })
	return page, nil
}
func (s *MemoryStore) DueUpcoming(_ context.Context, now time.Time, limit int) ([]UpcomingRelease, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []UpcomingRelease{}
	for _, r := range s.upcoming {
		if r.State != "HANDED_OFF" && !r.NextCheckAt.After(now) {
			out = append(out, cloneUpcoming(r))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextCheckAt.Before(out[j].NextCheckAt) })
	return out[:min(limit, len(out))], nil
}

type upcomingRow struct {
	Raw      string `db:"record_json"`
	Revision int    `db:"revision"`
}

func decodeUpcoming(row upcomingRow) (UpcomingRelease, error) {
	var r UpcomingRelease
	err := json.Unmarshal([]byte(row.Raw), &r)
	r.Revision = row.Revision
	return r, err
}
func (s *SQLiteStore) GetUpcoming(ctx context.Context, key string) (*UpcomingRelease, error) {
	var row upcomingRow
	err := s.db.GetContext(ctx, &row, "SELECT record_json,revision FROM subscription_upcoming_releases WHERE release_key=?", key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r, err := decodeUpcoming(row)
	return &r, err
}
func (s *SQLiteStore) PutUpcoming(ctx context.Context, r *UpcomingRelease) error {
	next := cloneUpcoming(*r)
	next.Revision++
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	err = s.withWrite(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `INSERT INTO subscription_upcoming_releases(release_key,release_date,state,next_check_at,code,title,revision,record_json)
   VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(release_key) DO UPDATE SET release_date=excluded.release_date,state=excluded.state,next_check_at=excluded.next_check_at,code=excluded.code,title=excluded.title,revision=excluded.revision,record_json=excluded.record_json WHERE subscription_upcoming_releases.revision=?`, next.Release.Key, next.Release.Date, next.State, formatTimestamp(next.NextCheckAt), next.Release.Code, next.Release.Title, next.Revision, string(raw), r.Revision)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return errUpcomingRevision
		}
		for _, actor := range next.Performers {
			if _, err = tx.ExecContext(ctx, `INSERT INTO subscription_upcoming_performers(release_key,performer_id,name) VALUES(?,?,?) ON CONFLICT(release_key,performer_id) DO UPDATE SET name=excluded.name`, next.Release.Key, actor.ID, actor.Name); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		*r = next
	}
	return err
}
func (s *SQLiteStore) QueryUpcoming(ctx context.Context, q UpcomingQuery, ids []string) (UpcomingPage, error) {
	q = normalizeUpcomingQuery(q)
	page := UpcomingPage{Items: []UpcomingRelease{}, Performers: []ReviewPerformer{}, Page: q.Page, PageSize: q.PageSize}
	if len(ids) == 0 {
		return page, nil
	}
	// All projections use one snapshot; pagination never decodes the full catalog.
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer tx.Rollback()
	actorsSQL, args, err := sqlx.In(`SELECT DISTINCT p.performer_id AS id,p.name FROM subscription_upcoming_performers p JOIN subscription_upcoming_releases u USING(release_key) WHERE u.state!='HANDED_OFF' AND p.performer_id IN (?) ORDER BY p.name,p.performer_id`, ids)
	if err != nil {
		return page, err
	}
	rows, err := tx.QueryxContext(ctx, actorsSQL, args...)
	if err != nil {
		return page, err
	}
	for rows.Next() {
		var a ReviewPerformer
		if err = rows.Scan(&a.ID, &a.Name); err != nil {
			rows.Close()
			return page, err
		}
		page.Performers = append(page.Performers, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return page, err
	}
	where := `u.state!='HANDED_OFF' AND EXISTS(SELECT 1 FROM subscription_upcoming_performers p WHERE p.release_key=u.release_key AND p.performer_id IN (?))`
	values := []any{ids}
	if q.PerformerID != "" {
		where += ` AND EXISTS(SELECT 1 FROM subscription_upcoming_performers p WHERE p.release_key=u.release_key AND p.performer_id=?)`
		values = append(values, q.PerformerID)
		found := false
		for _, id := range ids {
			if id == q.PerformerID {
				found = true
			}
		}
		if !found {
			return page, nil
		}
	}
	if q.Before != "" {
		where += ` AND u.release_date<=?`
		values = append(values, q.Before)
	}
	if q.Search != "" {
		where += ` AND instr(lower(u.code||' '||u.title),lower(?))>0`
		values = append(values, q.Search)
	}
	query, args, err := sqlx.In("SELECT count(*) FROM subscription_upcoming_releases u WHERE "+where, values...)
	if err != nil {
		return page, err
	}
	if err = tx.GetContext(ctx, &page.TotalCount, query, args...); err != nil {
		return page, err
	}
	page.TotalPages = (page.TotalCount + q.PageSize - 1) / q.PageSize
	query, args, err = sqlx.In("SELECT record_json,revision FROM subscription_upcoming_releases u WHERE "+where+" ORDER BY release_date,release_key LIMIT ? OFFSET ?", append(values, q.PageSize, (q.Page-1)*q.PageSize)...)
	if err != nil {
		return page, err
	}
	var stored []upcomingRow
	if err = tx.SelectContext(ctx, &stored, query, args...); err != nil {
		return page, err
	}
	for _, row := range stored {
		r, err := decodeUpcoming(row)
		if err != nil {
			return page, fmt.Errorf("upcoming: %w", err)
		}
		page.Items = append(page.Items, r)
	}
	return page, tx.Commit()
}
func (s *SQLiteStore) DueUpcoming(ctx context.Context, now time.Time, limit int) ([]UpcomingRelease, error) {
	var rows []upcomingRow
	err := s.db.SelectContext(ctx, &rows, `SELECT record_json,revision FROM subscription_upcoming_releases WHERE state!='HANDED_OFF' AND next_check_at<=? ORDER BY next_check_at,release_key LIMIT ?`, formatTimestamp(now), limit)
	if err != nil {
		return nil, err
	}
	out := []UpcomingRelease{}
	for _, row := range rows {
		r, err := decodeUpcoming(row)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// PutUpcomingReview commits release decisions and handoff progress in one core transaction.
func (s *SQLiteStore) PutUpcomingReview(ctx context.Context, r *UpcomingRelease, review ReleaseReview) error {
	next := cloneUpcoming(*r)
	next.Revision++
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	err = s.withWrite(ctx, func(tx *sqlx.Tx) error {
		if err := s.putTypedReviewsTx(ctx, tx, []ReleaseReview{review}); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE subscription_upcoming_releases SET state=?,next_check_at=?,revision=?,record_json=? WHERE release_key=? AND revision=?`, next.State, formatTimestamp(next.NextCheckAt), next.Revision, string(raw), next.Release.Key, r.Revision)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return errUpcomingRevision
		}
		return nil
	})
	if err == nil {
		*r = next
	}
	return err
}
func (s *MemoryStore) PutUpcomingReview(_ context.Context, r *UpcomingRelease, review ReleaseReview) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.upcoming[r.Release.Key].Revision != r.Revision {
		return errUpcomingRevision
	}
	r.Revision++
	s.upcoming[r.Release.Key] = cloneUpcoming(*r)
	if s.reviews == nil {
		s.reviews = map[string]ReleaseReview{}
	}
	s.reviews[review.Key] = cloneReleaseReview(review)
	return nil
}

// UpcomingReleaseStates reads only the requested identities, including paused records.
// Terminal records are history and must not keep a scene marked as upcoming.
func (s *MemoryStore) UpcomingReleaseStates(_ context.Context, keys []string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	states := make(map[string]string, len(keys))
	for _, key := range keys {
		if record, ok := s.upcoming[key]; ok && record.State != "HANDED_OFF" {
			states[key] = record.State
		}
	}
	return states, nil
}
func (s *SQLiteStore) UpcomingReleaseStates(ctx context.Context, keys []string) (map[string]string, error) {
	states := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return states, nil
	}
	query, args, err := sqlx.In("SELECT release_key,state FROM subscription_upcoming_releases WHERE state!='HANDED_OFF' AND release_key IN (?)", keys)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Key   string `db:"release_key"`
		State string `db:"state"`
	}
	if err = s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	for _, row := range rows {
		states[row.Key] = row.State
	}
	return states, nil
}
