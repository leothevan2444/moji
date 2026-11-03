package torrentsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

type Store struct{ db *storage.Database }

func NewStore(manager *storage.Manager) (*Store, error) {
	if manager == nil || manager.Runtime == nil {
		return nil, indexer.ErrNotConfigured
	}
	return &Store{db: manager.Runtime}, nil
}
func (s *Store) interrupt(ctx context.Context, now time.Time) error {
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `UPDATE torrent_search_sites SET state='INTERRUPTED' WHERE state IN ('QUEUED','RUNNING') AND search_id IN (SELECT id FROM torrent_search_runs WHERE state IN ('QUEUED','RUNNING'))`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE torrent_search_runs SET state='INTERRUPTED',error_code='TORRENT_SEARCH_INTERRUPTED',finished_at=?,updated_at=?,revision=revision+1 WHERE state IN ('QUEUED','RUNNING')`, instant(now), instant(now))
		return err
	})
}
func (s *Store) create(ctx context.Context, input Input, now time.Time) (string, error) {
	encoded, err := json.Marshal(input.Query)
	if err != nil || len(encoded) > 262144 || len(input.TaskID) > 128 {
		return "", ErrInvalid
	}
	id := uuid.NewString()
	err = s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM torrent_search_runs WHERE expires_at<=?`, instant(now)); err != nil {
			return err
		}

		if input.TaskID != "" && input.Query.CachePolicy != indexer.CacheRefresh {
			var existing string
			err := tx.GetContext(ctx, &existing, `SELECT id FROM torrent_search_runs WHERE task_id=? AND json_remove(query_json,'$.CachePolicy')=json_remove(?,'$.CachePolicy') AND expires_at>? ORDER BY created_at DESC,id DESC LIMIT 1`, input.TaskID, string(encoded), instant(now))
			if err == nil {
				id = existing
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		var count int
		if err := tx.GetContext(ctx, &count, `SELECT count(*) FROM torrent_search_runs`); err != nil {
			return err
		}
		used, err := snapshotBytes(ctx, tx)
		if err != nil {
			return err
		}
		if used+int64(len(encoded))+256 > maxBytes {
			return ErrBusy
		}
		if count >= maxRuns {
			return ErrBusy
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO torrent_search_runs(id,payload_version,state,query_json,task_id,correlation_id,created_at,updated_at,expires_at) VALUES(?,1,'QUEUED',?,?,?,?,?,?)`, id, string(encoded), input.TaskID, uuid.NewString(), instant(now), instant(now), instant(now.Add(retention)))
		return err
	})
	return id, err
}
func (s *Store) queued(ctx context.Context, now time.Time) ([]string, error) {
	var ids []string
	err := s.db.Reader.SelectContext(ctx, &ids, `SELECT id FROM torrent_search_runs WHERE state='QUEUED' AND expires_at>? ORDER BY created_at,id LIMIT 50`, instant(now))
	return ids, err
}
func (s *Store) claim(ctx context.Context, id string, now time.Time) (bool, error) {
	claimed := false
	err := s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE torrent_search_runs SET state='RUNNING',revision=revision+1,updated_at=? WHERE id=? AND state='QUEUED' AND cancel_requested=0 AND expires_at>?`, instant(now), id, instant(now))
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		claimed = count == 1
		return err
	})
	return claimed, err
}
func safeDetails(raw string) string {
	target, err := url.Parse(raw)
	if err != nil || target.User != nil || target.Hostname() == "" || target.Scheme != "http" && target.Scheme != "https" {
		return ""
	}
	for key := range target.Query() {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "passkey") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") || lower == "key" || strings.Contains(lower, "auth") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "signature") {
			return ""
		}
	}
	target.Fragment = ""
	return target.String()
}
func candidateProjection(release indexer.Release) Candidate {
	return Candidate{ID: uuid.NewString(), InstanceID: release.InstanceID, InstanceName: release.InstanceName, Kind: release.Kind, Title: release.Title, DetailsURL: safeDetails(release.DetailsURL), InfoHash: release.InfoHash, Size: release.Size, Seeders: release.Seeders, Peers: release.Peers, PublishedAt: release.PublishedAt, Categories: append([]int(nil), release.Categories...)}
}
func (s *Store) appendBatch(ctx context.Context, id string, batch indexer.Batch, now time.Time) (bool, error) {
	truncated := false
	err := s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var state string
		var cancelled bool
		if err := tx.QueryRowContext(ctx, `SELECT state,cancel_requested FROM torrent_search_runs WHERE id=? AND expires_at>?`, id, instant(now)).Scan(&state, &cancelled); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrExpired
			}
			return err
		}
		if state != "RUNNING" || cancelled {
			return context.Canceled
		}
		var total, position int
		if err := tx.GetContext(ctx, &total, `SELECT count(*) FROM torrent_search_candidates`); err != nil {
			return err
		}
		if err := tx.GetContext(ctx, &position, `SELECT count(*) FROM torrent_search_candidates WHERE search_id=?`, id); err != nil {
			return err
		}
		used, err := snapshotBytes(ctx, tx)
		if err != nil {
			return err
		}
		for _, release := range batch.Releases {
			candidate := candidateProjection(release)
			payload, err := json.Marshal(candidate)
			if err != nil || len(payload) > 65536 || position >= 1000 || total >= maxCandidates || used+int64(len(payload))+128 > maxBytes {
				truncated = true
				break
			}
			var published any
			if candidate.PublishedAt != nil {
				published = instant(*candidate.PublishedAt)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO torrent_search_candidates(id,search_id,position,instance_id,resource_id,payload_version,payload_json,size,seeders,published_at) VALUES(?,?,?,?,?,1,?,?,?,?)`, candidate.ID, id, position, release.InstanceID, release.Resource.ID, string(payload), candidate.Size, candidate.Seeders, published); err != nil {
				return err
			}
			position++
			total++
			used += int64(len(payload)) + 128
		}
		_, err = tx.ExecContext(ctx, `UPDATE torrent_search_runs SET revision=revision+1,updated_at=?,truncated=max(truncated,?) WHERE id=?`, instant(now), truncated, id)
		return err
	})
	return truncated, err
}
func (s *Store) progress(ctx context.Context, id string, event indexer.SearchProgress, now time.Time) error {
	if len(event.Sources)+len(event.Diagnostics) > 2048 {
		return ErrInvalid
	}
	for _, source := range event.Sources {
		if source.ID == "" || len(source.ID) > 256 || len(source.Name) > 512 || (source.Kind != indexer.Builtin && source.Kind != indexer.Jackett) {
			return ErrInvalid
		}
	}
	for _, diagnostic := range event.Diagnostics {
		if len(diagnostic.InstanceID) > 256 || len(diagnostic.Code) > 256 || len(diagnostic.CorrelationID) > 256 {
			return ErrInvalid
		}
	}
	if len(event.InstanceID) > 256 || len(event.Report.Code) > 256 || len(event.Report.CorrelationID) > 256 {
		return ErrInvalid
	}
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var state string
		if err := tx.GetContext(ctx, &state, `SELECT state FROM torrent_search_runs WHERE id=? AND expires_at>?`, id, instant(now)); err != nil {
			return err
		}
		if state != "RUNNING" {
			return context.Canceled
		}
		switch event.Phase {
		case "READY":
			for position, source := range event.Sources {
				if _, err := tx.ExecContext(ctx, `INSERT INTO torrent_search_sites(search_id,instance_id,position,name,kind,state) VALUES(?,?,?,?,?,'QUEUED')`, id, source.ID, position, source.Name, string(source.Kind)); err != nil {
					return err
				}
			}
			for position, diagnostic := range event.Diagnostics {
				payload, _ := json.Marshal(diagnostic)
				if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO torrent_search_sites(search_id,instance_id,position,name,kind,state,report_json) VALUES(?,?,?,?,?,'FAILED',?)`, id, diagnostic.InstanceID, len(event.Sources)+position, diagnostic.InstanceID, "", string(payload)); err != nil {
					return err
				}
			}
		case "STARTED":
			if _, err := tx.ExecContext(ctx, `UPDATE torrent_search_sites SET state='RUNNING' WHERE search_id=? AND instance_id=?`, id, event.InstanceID); err != nil {
				return err
			}
		case "FINISHED":
			payload, err := json.Marshal(event.Report)
			if err != nil {
				return err
			}
			state := "COMPLETED"
			if event.Report.Code != "" {
				state = "FAILED"
			}
			var fetched any
			if !event.FetchedAt.IsZero() {
				fetched = instant(event.FetchedAt)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE torrent_search_sites SET state=?,report_json=?,fetched_at=? WHERE search_id=? AND instance_id=?`, state, string(payload), fetched, id, event.InstanceID); err != nil {
				return err
			}
		default:
			return ErrInvalid
		}
		used, err := snapshotBytes(ctx, tx)
		if err != nil {
			return err
		}
		if used > maxBytes {
			return ErrBusy
		}
		_, err = tx.ExecContext(ctx, `UPDATE torrent_search_runs SET revision=revision+1,updated_at=? WHERE id=?`, instant(now), id)
		return err
	})
}

func snapshotBytes(ctx context.Context, tx *sqlx.Tx) (int64, error) {
	var used int64
	err := tx.GetContext(ctx, &used, `SELECT (SELECT coalesce(sum(length(CAST(query_json AS BLOB))+256),0) FROM torrent_search_runs)+(SELECT coalesce(sum(length(CAST(payload_json AS BLOB))+128),0) FROM torrent_search_candidates)+(SELECT coalesce(sum(length(CAST(coalesce(report_json,'') AS BLOB))+length(CAST(name AS BLOB))+length(CAST(instance_id AS BLOB))+128),0) FROM torrent_search_sites)`)
	return used, err
}
