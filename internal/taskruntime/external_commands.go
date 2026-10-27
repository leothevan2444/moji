package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type taskCommandJournal interface {
	RunTaskCommand(context.Context, *Task, string, string, map[string]any, func() error) error
}

func (s *Service) runTaskCommand(ctx context.Context, task *Task, kind, hash string, payload map[string]any, action func() error) error {
	if journal, ok := s.store.(taskCommandJournal); ok {
		return journal.RunTaskCommand(ctx, task, kind, hash, payload, action)
	}
	return action()
}
func (s *EventingTaskStore) RunTaskCommand(ctx context.Context, task *Task, kind, hash string, payload map[string]any, action func() error) error {
	if journal, ok := s.store.(taskCommandJournal); ok {
		return journal.RunTaskCommand(ctx, task, kind, hash, payload, action)
	}
	return action()
}
func (s *RelationalTaskStore) RunTaskCommand(ctx context.Context, task *Task, kind, hash string, payload map[string]any, action func() error) error {
	if task == nil {
		return errors.New("taskruntime: command task required")
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["task_id"], payload["generation"] = task.ID, task.Generation
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	id := uuid.NewString()
	now := formatDurableTime(time.Now())
	if err := s.core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var revision, generation int64
		var checkpointVersion, filePlanVersion int
		if err := tx.QueryRowContext(ctx, "SELECT t.revision,t.generation,COALESCE(k.payload_version,1),COALESCE(p.payload_version,1) FROM tasks t LEFT JOIN task_checkpoints k ON k.task_id=t.id LEFT JOIN task_file_plans p ON p.task_id=t.id WHERE t.id=?", task.ID).Scan(&revision, &generation, &checkpointVersion, &filePlanVersion); err != nil {
			return err
		}
		if checkpointVersion > 1 || filePlanVersion > 1 {
			return errors.New("taskruntime: unsupported task checkpoint version")
		}
		if revision != task.Revision || generation != task.Generation {
			return ErrTaskRevisionConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO external_commands(id,task_id,target_hash,kind,state,payload_json,created_at,updated_at) VALUES(?,?,?,?,'SENT',?,?,?)`, id, task.ID, hash, kind, string(raw), now, now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(operation_id,task_id,subject_id,event_type,level,message,payload_json,created_at) VALUES(?,?,?,'external-command','info',?,?,?)`, id, task.ID, task.ID, kind, string(raw), now)
		return err
	}); err != nil {
		return err
	}
	// Network work is deliberately outside the core writer transaction.
	actionErr := action()
	if actionErr == nil {
		return nil
	}
	resultCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	saveErr := s.core.WithWrite(resultCtx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(resultCtx, `UPDATE external_commands SET state='UNCERTAIN',last_error=?,updated_at=? WHERE id=?`, actionErr.Error(), formatDurableTime(time.Now()), id)
		return err
	})
	return errors.Join(actionErr, saveErr)
}
