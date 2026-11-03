package indexerdefs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

type repository struct{ db *storage.Database }

func (r *repository) state(ctx context.Context) (State, error) {
	return readState(ctx, r.db.Reader)
}

type readDatabase interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	SelectContext(context.Context, any, string, ...any) error
}

func readState(ctx context.Context, reader readDatabase) (State, error) {
	var out State
	var active, previous, checked sql.NullString
	err := reader.QueryRowContext(ctx, `SELECT active_revision,previous_revision,revision,available_revision,etag,last_checked_at,check_error FROM indexer_definition_state WHERE singleton=1`).Scan(&active, &previous, &out.Revision, &out.Available, &out.ETag, &checked, &out.CheckError)
	if err != nil {
		return out, err
	}
	out.Active, out.Previous = active.String, previous.String
	if checked.Valid {
		instant, err := time.Parse(time.RFC3339Nano, checked.String)
		if err != nil {
			return out, err
		}
		out.LastCheckedAt = &instant
	}
	return out, nil
}

func (r *repository) recordCheck(ctx context.Context, revision, etag, errorCode string, at time.Time) error {
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if errorCode != "" {
			_, err := tx.ExecContext(ctx, `UPDATE indexer_definition_state SET last_checked_at=?,check_error=?,revision=revision+1 WHERE singleton=1`, stamp(at), errorCode)
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE indexer_definition_state SET available_revision=?,etag=?,last_checked_at=?,check_error='',revision=revision+1 WHERE singleton=1`, revision, etag, stamp(at))
		return err
	})
}

func readInstalled(ctx context.Context, reader readDatabase) ([]Manifest, error) {
	var records []string
	if err := reader.SelectContext(ctx, &records, `SELECT manifest_json FROM indexer_definition_bundles ORDER BY installed_at DESC,revision`); err != nil {
		return nil, err
	}
	manifests := make([]Manifest, 0, len(records))
	for _, record := range records {
		var manifest Manifest
		if err := json.Unmarshal([]byte(record), &manifest); err != nil {
			return nil, err
		}
		if manifest.Version != ManifestVersion || manifest.Schema != SchemaVersion || !ValidRevision(manifest.Revision) {
			return nil, ErrInvalidBundle
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func (r *repository) manifest(ctx context.Context, revision string) (Manifest, string, error) {
	var record, checksum string
	err := r.db.Reader.QueryRowContext(ctx, `SELECT manifest_json,manifest_sha256 FROM indexer_definition_bundles WHERE revision=?`, revision).Scan(&record, &checksum)
	if errors.Is(err, sql.ErrNoRows) {
		return Manifest{}, "", ErrNotInstalled
	}
	if err != nil {
		return Manifest{}, "", err
	}
	var manifest Manifest
	if err := json.Unmarshal([]byte(record), &manifest); err != nil {
		return Manifest{}, "", ErrInvalidBundle
	}
	if digest([]byte(record)) != checksum || manifest.Revision != revision {
		return Manifest{}, "", ErrInvalidBundle
	}
	return manifest, checksum, nil
}

const jobColumns = `id,kind,state,target_revision,expected_active,downloaded_bytes,error_code,correlation_id,started_at,finished_at,revision`

type scanner interface{ Scan(...any) error }

func readJob(row scanner) (*Job, error) {
	var job Job
	var started string
	var finished sql.NullString
	err := row.Scan(&job.ID, &job.Kind, &job.State, &job.Target, &job.ExpectedActive, &job.DownloadedBytes, &job.ErrorCode, &job.CorrelationID, &started, &finished, &job.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	job.StartedAt, err = time.Parse(time.RFC3339Nano, started)
	if err != nil {
		return nil, err
	}
	if finished.Valid {
		instant, err := time.Parse(time.RFC3339Nano, finished.String)
		if err != nil {
			return nil, err
		}
		job.FinishedAt = &instant
	}
	return &job, nil
}

func (r *repository) job(ctx context.Context, id string) (*Job, error) {
	return readJob(r.db.Reader.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM indexer_definition_jobs WHERE id=?`, id))
}
func (r *repository) currentJob(ctx context.Context) (*Job, error) {
	return readCurrentJob(ctx, r.db.Reader)
}

func readCurrentJob(ctx context.Context, reader readDatabase) (*Job, error) {
	return readJob(reader.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM indexer_definition_jobs ORDER BY (finished_at IS NULL) DESC,started_at DESC,id DESC LIMIT 1`))
}

func (r *repository) snapshot(ctx context.Context) (Status, error) {
	tx, err := r.db.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback()
	state, err := readState(ctx, tx)
	if err != nil {
		return Status{}, err
	}
	installed, err := readInstalled(ctx, tx)
	if err != nil {
		return Status{}, err
	}
	job, err := readCurrentJob(ctx, tx)
	if err != nil {
		return Status{}, err
	}
	if err := tx.Commit(); err != nil {
		return Status{}, err
	}
	return Status{State: state, Installed: installed, Job: job}, nil
}

func (r *repository) begin(ctx context.Context, job *Job) error {
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var active string
		if err := tx.GetContext(ctx, &active, `SELECT coalesce(active_revision,'') FROM indexer_definition_state WHERE singleton=1`); err != nil {
			return err
		}
		if active != job.ExpectedActive {
			return ErrConflict
		}
		var running int
		if err := tx.GetContext(ctx, &running, `SELECT count(*) FROM indexer_definition_jobs WHERE finished_at IS NULL`); err != nil {
			return err
		}
		if running > 0 {
			return ErrBusy
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO indexer_definition_jobs(id,kind,state,target_revision,expected_active,correlation_id,started_at) VALUES(?,?,?,?,?,?,?)`, job.ID, job.Kind, job.State, job.Target, job.ExpectedActive, job.CorrelationID, stamp(job.StartedAt))
		if err == nil {
			job.Revision = 1
		}
		return err
	})
}

func (r *repository) updateJob(ctx context.Context, job *Job) error {
	var nextRevision int64
	err := r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var finished any
		if job.FinishedAt != nil {
			finished = stamp(*job.FinishedAt)
		}
		result, err := tx.ExecContext(ctx, `UPDATE indexer_definition_jobs SET state=?,downloaded_bytes=?,error_code=?,finished_at=?,revision=revision+1 WHERE id=? AND revision=? AND finished_at IS NULL`, job.State, job.DownloadedBytes, job.ErrorCode, finished, job.ID, job.Revision)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrConflict
		}
		nextRevision = job.Revision + 1
		return nil
	})
	if err == nil {
		job.Revision = nextRevision
	}
	return err
}

// activate atomically records all required asset identities, the active pointer,
// job completion and the activation audit. The files already exist durably.
func (r *repository) activate(ctx context.Context, job *Job, bundle *Bundle, at time.Time) error {
	data, err := manifestData(bundle.Manifest)
	if err != nil {
		return err
	}
	if digest(data) != bundle.ManifestSHA256 {
		return ErrInvalidBundle
	}
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var active string
		if err := tx.GetContext(ctx, &active, `SELECT coalesce(active_revision,'') FROM indexer_definition_state WHERE singleton=1`); err != nil {
			return err
		}
		if active != job.ExpectedActive {
			return ErrConflict
		}
		var state string
		var revision int64
		if err := tx.QueryRowContext(ctx, `SELECT state,revision FROM indexer_definition_jobs WHERE id=?`, job.ID).Scan(&state, &revision); err != nil {
			return err
		}
		if state != string(Activating) || revision != job.Revision {
			return ErrConflict
		}
		var checksum string
		err := tx.GetContext(ctx, &checksum, `SELECT manifest_sha256 FROM indexer_definition_bundles WHERE revision=?`, bundle.Manifest.Revision)
		if err == nil {
			if checksum != bundle.ManifestSHA256 {
				return ErrInvalidBundle
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		} else {
			if _, err := tx.ExecContext(ctx, `INSERT INTO indexer_definition_bundles(revision,manifest_json,manifest_sha256,installed_at) VALUES(?,?,?,?)`, bundle.Manifest.Revision, string(data), bundle.ManifestSHA256, stamp(bundle.Manifest.InstalledAt)); err != nil {
				return err
			}
			for _, file := range bundle.Manifest.Files {
				if _, err := tx.ExecContext(ctx, `INSERT INTO indexer_definition_files(revision,name,sha256,bytes) VALUES(?,?,?,?)`, bundle.Manifest.Revision, file.Name, file.SHA256, file.Bytes); err != nil {
					return err
				}
			}
		}
		// Repairing the active bundle must preserve its previous rollback target.
		if active == bundle.Manifest.Revision {
			if _, err := tx.ExecContext(ctx, `UPDATE indexer_definition_state SET revision=revision+1 WHERE singleton=1`); err != nil {
				return err
			}
		} else {
			var previous any
			if active != "" {
				previous = active
			}
			if _, err := tx.ExecContext(ctx, `UPDATE indexer_definition_state SET active_revision=?,previous_revision=?,revision=revision+1 WHERE singleton=1`, bundle.Manifest.Revision, previous); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE indexer_definition_jobs SET state='COMPLETED',finished_at=?,error_code='',revision=revision+1 WHERE id=?`, stamp(at), job.ID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO indexer_definition_audit(job_id,previous_revision,active_revision,created_at) VALUES(?,?,?,?)`, job.ID, active, bundle.Manifest.Revision, stamp(at))
		return err
	})
}

func (r *repository) recoverInterrupted(ctx context.Context, at time.Time) error {
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE indexer_definition_jobs SET state='INTERRUPTED',finished_at=?,error_code='DEFINITION_UPDATE_INTERRUPTED',revision=revision+1 WHERE finished_at IS NULL`, stamp(at))
		return err
	})
}

func (r *repository) activationCommitted(ctx context.Context, id, revision string) (bool, error) {
	var state, active string
	err := r.db.Reader.QueryRowContext(ctx, `SELECT j.state,coalesce(s.active_revision,'') FROM indexer_definition_jobs j CROSS JOIN indexer_definition_state s WHERE j.id=? AND s.singleton=1`, id).Scan(&state, &active)
	return state == string(Completed) && active == revision, err
}

func (r *repository) cancel(ctx context.Context, id string, at time.Time) error {
	return r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		job, err := readJob(tx.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM indexer_definition_jobs WHERE id=?`, id))
		if err != nil {
			return err
		}
		if job == nil || job.State.Terminal() {
			return ErrJobTerminal
		}
		if job.State == Activating {
			return ErrCancelling
		}
		_, err = tx.ExecContext(ctx, `UPDATE indexer_definition_jobs SET state='CANCELLED',finished_at=?,revision=revision+1 WHERE id=?`, stamp(at), id)
		return err
	})
}

func stamp(at time.Time) string { return at.UTC().Format("2006-01-02T15:04:05.000000000Z") }
