package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

// DiagnosticContext contains identifiers only, never credentials or resource URLs.
// Code is the validated task code, rather than arbitrary search input.
type DiagnosticContext struct {
	Trigger  string
	TaskID   string
	Code     string
	SearchID string
}
type diagnosticContextKey struct{}

func WithDiagnosticContext(ctx context.Context, details DiagnosticContext) context.Context {
	return context.WithValue(ctx, diagnosticContextKey{}, details)
}
func diagnosticContext(ctx context.Context) DiagnosticContext {
	details, _ := ctx.Value(diagnosticContextKey{}).(DiagnosticContext)
	return details
}

// Diagnostic preserves a site report and its execution context independently of
// task state and of the last health observation. No upstream error text is stored.
type Diagnostic struct {
	Executed     bool
	ElapsedKnown bool
	ID           string
	DiagnosticContext
	InstanceID, Name               string
	ConfigRevision, BundleRevision string
	Operation                      HealthOperation
	StartedAt, FinishedAt          time.Time
	Mode                           string
	Categories                     []int
	Limit, Offset                  int
	ResourceID                     string
	Report                         SiteReport
}

type DiagnosticRecorder interface {
	Record(context.Context, Diagnostic) error
}
type DiagnosticReader interface {
	Diagnostics(context.Context, DiagnosticFilter) (DiagnosticPage, error)
}
type DiagnosticStore struct{ db *storage.Database }

func NewDiagnosticStore(manager *storage.Manager) (*DiagnosticStore, error) {
	if manager == nil || manager.Runtime == nil {
		return nil, ErrNotConfigured
	}
	return &DiagnosticStore{db: manager.Runtime}, nil
}

// Records are the single source for history, recent outcomes and statistics.
func (s *DiagnosticStore) Record(ctx context.Context, record Diagnostic) error {
	if record.InstanceID == "" || len(record.InstanceID) > 256 || len(record.Name) > 512 || len(record.TaskID) > 128 || len(record.Code) > 128 || len(record.SearchID) > 128 || len(record.Trigger) > 32 || len(record.Categories) > 128 || record.StartedAt.IsZero() || record.FinishedAt.Before(record.StartedAt) {
		return ErrInvalidSettings
	}
	if record.Operation != HealthSearch && record.Operation != HealthFetch {
		return ErrInvalidSettings
	}
	if record.Report.CorrelationID == "" {
		record.Report.CorrelationID = uuid.NewString()
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	var payload map[string]json.RawMessage
	if err = json.Unmarshal(data, &payload); err != nil {
		return err
	}
	payload["StartedAt"], _ = json.Marshal(instant(record.StartedAt))
	payload["FinishedAt"], _ = json.Marshal(instant(record.FinishedAt))
	data, err = json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(data) > 16384 {
		return ErrResponseBudget
	}
	return s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexer_executions(message) VALUES(?) ON CONFLICT(instance_id,operation,correlation_id) DO NOTHING`, string(data))
		return err
	})
}

type DiagnosticFilter struct {
	InstanceID, Query, Operation, Outcome string
	Since                                 *time.Time
	Limit                                 int
	Before                                int64
}
type DiagnosticSite struct{ ID, Name string }
type DiagnosticPage struct {
	Items   []Diagnostic
	Sites   []DiagnosticSite
	Total   int
	HasMore bool
}

// All filters apply before keyset pagination. Search uses bound literal substrings,
// so percent and underscore never unexpectedly widen a user's query.
func (s *DiagnosticStore) Diagnostics(ctx context.Context, filter DiagnosticFilter) (DiagnosticPage, error) {
	page := DiagnosticPage{Items: []Diagnostic{}, Sites: []DiagnosticSite{}}
	if len(filter.InstanceID) > 256 || len(filter.Query) > 256 || filter.Limit < 1 || filter.Limit > 100 || filter.Before < 0 {
		return page, ErrInvalidQuery
	}
	if filter.Operation != "" && filter.Operation != "SEARCH" && filter.Operation != "FETCH" {
		return page, ErrInvalidQuery
	}
	if filter.Outcome != "" && filter.Outcome != "SUCCEEDED" && filter.Outcome != "FAILED" && filter.Outcome != "CANCELLED" {
		return page, ErrInvalidQuery
	}
	where := `1=1`
	args := []any{}
	if filter.Since != nil {
		where += ` AND finished_at>=?`
		args = append(args, instant(*filter.Since))
	}
	if filter.InstanceID != "" {
		where += ` AND instance_id=?`
		args = append(args, filter.InstanceID)
	}
	if filter.Operation != "" {
		where += ` AND operation=?`
		args = append(args, filter.Operation)
	}
	if filter.Outcome != "" {
		where += ` AND outcome=?`
		args = append(args, filter.Outcome)
	}
	if term := strings.TrimSpace(filter.Query); term != "" {
		where += ` AND (`
		fields := []string{"Name", "Code", "TaskID", "SearchID", "ResourceID", "Report.Code", "Report.CorrelationID"}
		for i, field := range fields {
			if i > 0 {
				where += ` OR `
			}
			where += `instr(lower(coalesce(json_extract(message,'$.` + field + `'),'')),lower(?))>0`
			args = append(args, term)
		}
		where += `)`
	}
	tx, err := s.db.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return page, err
	}
	defer tx.Rollback()
	if err = tx.GetContext(ctx, &page.Total, `SELECT count(*) FROM indexer_executions WHERE `+where, args...); err != nil {
		return page, err
	}
	pageArgs := append([]any{}, args...)
	if filter.Before > 0 {
		where += ` AND id<?`
		pageArgs = append(pageArgs, filter.Before)
	}
	pageArgs = append(pageArgs, filter.Limit+1)
	rows, err := tx.QueryContext(ctx, `SELECT id,message FROM indexer_executions WHERE `+where+` ORDER BY id DESC LIMIT ?`, pageArgs...)
	if err != nil {
		return page, err
	}
	for rows.Next() {
		var id int64
		var raw string
		var record Diagnostic
		if err = rows.Scan(&id, &raw); err != nil {
			break
		}
		if err = json.Unmarshal([]byte(raw), &record); err != nil {
			break
		}
		record.ID = strconv.FormatInt(id, 10)
		page.Items = append(page.Items, record)
	}
	rowErr := rows.Err()
	rows.Close()
	if err != nil {
		return page, err
	}
	if rowErr != nil {
		return page, rowErr
	}
	if len(page.Items) > filter.Limit {
		page.HasMore = true
		page.Items = page.Items[:filter.Limit]
	}
	rows, err = tx.QueryContext(ctx, `SELECT instance_id,json_extract(message,'$.Name') FROM indexer_executions WHERE id IN (SELECT max(id) FROM indexer_executions GROUP BY instance_id) ORDER BY 2,1 LIMIT 2048`)
	if err != nil {
		return page, err
	}
	for rows.Next() {
		var site DiagnosticSite
		if err = rows.Scan(&site.ID, &site.Name); err != nil {
			break
		}
		page.Sites = append(page.Sites, site)
	}
	rowErr = rows.Err()
	rows.Close()
	if err != nil {
		return page, err
	}
	if rowErr != nil {
		return page, rowErr
	}
	return page, tx.Commit()
}

func (s *Service) SetDiagnosticRecorder(recorder DiagnosticRecorder) { s.diagnostics = recorder }
func (s *Service) recordDiagnostic(ctx context.Context, record Diagnostic) error {
	if s.diagnostics == nil {
		return nil
	}
	// A site timeout still needs its report saved. The bounded write outlives only
	// this operation's cancellation; it never starts further upstream work.
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := s.diagnostics.Record(writeCtx, record); err != nil {
		if s.healthError != nil {
			s.healthError(err)
		}
		return ErrStateUnavailable
	}
	return nil
}
