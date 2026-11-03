package torrentsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexer"
)

func (s *Store) Get(ctx context.Context, id string, page Page, now time.Time) (Snapshot, error) {
	page, err := page.normalized()
	if err != nil || id == "" || len(id) > 128 {
		return Snapshot{}, ErrInvalid
	}
	// Snapshot + sites + candidates share one bounded read transaction, so a late
	// page cannot combine a newer count with an older lifecycle revision.
	tx, err := s.db.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Snapshot{}, err
	}
	defer tx.Rollback()
	var row struct {
		ID, State          string
		TaskID             string `db:"task_id"`
		Version            int    `db:"payload_version"`
		Query              string `db:"query_json"`
		Revision           int64
		CancelRequested    bool `db:"cancel_requested"`
		Partial, Truncated bool
		ErrorCode          string         `db:"error_code"`
		CorrelationID      string         `db:"correlation_id"`
		CreatedAt          string         `db:"created_at"`
		UpdatedAt          string         `db:"updated_at"`
		FinishedAt         sql.NullString `db:"finished_at"`
		ExpiresAt          string         `db:"expires_at"`
	}
	err = tx.GetContext(ctx, &row, `SELECT * FROM torrent_search_runs WHERE id=? AND expires_at>?`, id, instant(now))
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, ErrExpired
	}
	if err != nil {
		return Snapshot{}, err
	}
	if row.Version != 1 {
		return Snapshot{}, ErrInvalid
	}
	out := Snapshot{ID: id, State: row.State, TaskID: row.TaskID, Revision: row.Revision, CancelRequested: row.CancelRequested, Partial: row.Partial, Truncated: row.Truncated, ErrorCode: row.ErrorCode, CorrelationID: row.CorrelationID, Offset: page.Offset, Limit: page.Limit, Sites: []Site{}, Candidates: []Candidate{}}
	if err = json.Unmarshal([]byte(row.Query), &out.Query); err != nil {
		return Snapshot{}, ErrInvalid
	}
	for _, pair := range []struct {
		raw         string
		destination *time.Time
	}{{row.CreatedAt, &out.CreatedAt}, {row.UpdatedAt, &out.UpdatedAt}, {row.ExpiresAt, &out.ExpiresAt}} {
		value, err := time.Parse(time.RFC3339Nano, pair.raw)
		if err != nil {
			return Snapshot{}, ErrInvalid
		}
		*pair.destination = value
	}
	if row.FinishedAt.Valid {
		value, err := time.Parse(time.RFC3339Nano, row.FinishedAt.String)
		if err != nil {
			return Snapshot{}, ErrInvalid
		}
		out.FinishedAt = &value
	}
	var sites []struct {
		ID                string `db:"instance_id"`
		Name, Kind, State string
		Report            sql.NullString `db:"report_json"`
		Fetched           sql.NullString `db:"fetched_at"`
	}
	if err = tx.SelectContext(ctx, &sites, `SELECT instance_id,name,kind,state,report_json,fetched_at FROM torrent_search_sites WHERE search_id=? ORDER BY position,instance_id LIMIT 2048`, id); err != nil {
		return Snapshot{}, err
	}
	for _, row := range sites {
		site := Site{InstanceID: row.ID, Name: row.Name, Kind: indexer.Kind(row.Kind), State: row.State}
		if row.Report.Valid {
			if err = json.Unmarshal([]byte(row.Report.String), &site.Report); err != nil {
				return Snapshot{}, ErrInvalid
			}
		}
		if row.Fetched.Valid {
			value, err := time.Parse(time.RFC3339Nano, row.Fetched.String)
			if err != nil {
				return Snapshot{}, ErrInvalid
			}
			site.FetchedAt = &value
		}
		out.Sites = append(out.Sites, site)
	}
	if err = tx.GetContext(ctx, &out.Total, `SELECT count(*) FROM torrent_search_candidates WHERE search_id=?`, id); err != nil {
		return Snapshot{}, err
	}
	order := map[string]string{"RELEVANCE": "position", "SEEDERS_DESC": "seeders DESC NULLS LAST,position", "SIZE_DESC": "size DESC NULLS LAST,position", "DATE_DESC": "published_at DESC NULLS LAST,position"}[page.Sort]
	var candidates []struct {
		Version int    `db:"payload_version"`
		Payload string `db:"payload_json"`
	}
	if err = tx.SelectContext(ctx, &candidates, `SELECT payload_version,payload_json FROM torrent_search_candidates WHERE search_id=? ORDER BY `+order+` LIMIT ? OFFSET ?`, id, page.Limit, page.Offset); err != nil {
		return Snapshot{}, err
	}
	for _, row := range candidates {
		var candidate Candidate
		if row.Version != 1 || json.Unmarshal([]byte(row.Payload), &candidate) != nil {
			return Snapshot{}, ErrInvalid
		}
		out.Candidates = append(out.Candidates, candidate)
	}
	if err = tx.Commit(); err != nil {
		return Snapshot{}, err
	}
	return out, nil
}
func (s *Store) ResourceID(ctx context.Context, id, candidate string, now time.Time) (string, error) {
	if id == "" || candidate == "" || len(id) > 128 || len(candidate) > 128 {
		return "", ErrInvalid
	}
	var resource string
	err := s.db.Reader.GetContext(ctx, &resource, `SELECT c.resource_id FROM torrent_search_candidates c JOIN torrent_search_runs r ON r.id=c.search_id WHERE r.id=? AND c.id=? AND r.expires_at>? AND r.payload_version=1 AND c.payload_version=1`, id, candidate, instant(now))
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrExpired
	}
	return resource, err
}
func (s *Store) cancel(ctx context.Context, id string, now time.Time) error {
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE torrent_search_runs SET cancel_requested=1,state=CASE WHEN state='QUEUED' THEN 'CANCELLED' ELSE state END,finished_at=CASE WHEN state='QUEUED' THEN ? ELSE finished_at END,revision=revision+1,updated_at=? WHERE id=? AND state IN ('QUEUED','RUNNING') AND expires_at>?`, instant(now), instant(now), id, instant(now))
		if err != nil {
			return err
		}
		_, err = result.RowsAffected()
		return err
	})
}
func (s *Store) finish(ctx context.Context, id, state, code string, partial, truncated bool, now time.Time) error {
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {

		var cancelled bool
		if err := tx.GetContext(ctx, &cancelled, `SELECT cancel_requested FROM torrent_search_runs WHERE id=? AND state='RUNNING'`, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		siteState := "SKIPPED"
		if cancelled {
			siteState = "CANCELLED"
		} else if state == "INTERRUPTED" {
			siteState = "INTERRUPTED"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE torrent_search_sites SET state=CASE WHEN state='RUNNING' AND ? IN ('FAILED','PARTIAL') AND ?=0 THEN 'FAILED' ELSE ? END WHERE search_id=? AND state IN ('QUEUED','RUNNING')`, state, cancelled, siteState, id); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE torrent_search_runs SET state=CASE WHEN cancel_requested=1 THEN 'CANCELLED' ELSE ? END,error_code=?,partial=max(partial,?),truncated=max(truncated,?),finished_at=?,updated_at=?,revision=revision+1 WHERE id=? AND state='RUNNING'`, state, code, partial, truncated, instant(now), instant(now), id)
		return err
	})
}
