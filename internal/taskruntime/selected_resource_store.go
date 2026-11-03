package taskruntime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

// Admission needs a store that atomically writes the task, secret source record
// and rule pin. Legacy payload-only stores cannot acknowledge this guarantee.
func (s *RelationalTaskStore) CreateIndexedTask(ctx context.Context, task *Task) error {
	if task == nil || task.IndexerResource == nil {
		return fmt.Errorf("taskruntime: missing accepted resource")
	}
	return s.Create(ctx, task)
}
func validateSelectedResource(task *Task) error {
	if task.SelectedResourceID == "" {
		if task.IndexerResource != nil {
			return fmt.Errorf("taskruntime: resource snapshot without selection")
		}
		return nil
	}
	if task.Candidate.Source == nil || task.Candidate.Source.Version != 1 || task.Candidate.Source.ResourceID != task.SelectedResourceID {
		return fmt.Errorf("taskruntime: selected resource provenance mismatch")
	}
	if len(task.SelectedResourceID) > 128 {
		return fmt.Errorf("taskruntime: invalid selected resource")
	}
	if task.IndexerResource != nil {
		if task.IndexerResource.Validate() != nil || task.IndexerResource.Ref.ID != task.SelectedResourceID {
			return fmt.Errorf("taskruntime: invalid accepted resource")
		}
		return nil
	}
	if len(task.TorrentFileData) == 0 && normalizeMagnetURI(task.TorrentURL) == "" {
		return fmt.Errorf("taskruntime: accepted resource has no durable acquisition source")
	}
	return nil
}
func writeSelectedResource(ctx context.Context, tx *sqlx.Tx, task *Task) error {
	if err := validateSelectedResource(task); err != nil {
		return err
	}
	if err := indexerdefs.UnpinInTx(ctx, tx, "task-resource", task.ID); err != nil {
		return err
	}
	if task.SelectedResourceID == "" {
		_, err := tx.ExecContext(ctx, `DELETE FROM task_indexer_selections WHERE task_id=?`, task.ID)
		return err
	}
	var payload any
	if record := task.IndexerResource; record != nil {
		data, err := json.Marshal(record)
		if err != nil || len(data) > 256<<10 {
			return fmt.Errorf("taskruntime: invalid accepted resource payload")
		}
		payload = string(data)
		if record.Ref.BundleRevision != "" {
			if err := indexerdefs.PinInTx(ctx, tx, "task-resource", task.ID, record.Ref.BundleRevision); err != nil {
				return err
			}
		}
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO task_indexer_selections(task_id,payload_version,resource_id,record_json) VALUES(?,1,?,?) ON CONFLICT(task_id) DO UPDATE SET resource_id=excluded.resource_id,record_json=excluded.record_json`, task.ID, task.SelectedResourceID, payload)
	return err
}

func (s *RelationalTaskStore) UpdateIndexedTask(ctx context.Context, task *Task) error {
	if task == nil || task.IndexerResource == nil {
		return fmt.Errorf("taskruntime: missing accepted resource")
	}
	return s.update(ctx, task, true)
}
func guardIndexedChoice(ctx context.Context, tx *sqlx.Tx, previous *Task) error {
	if !eligibleIndexedChoice(previous) {
		return ErrSourcingUnavailable
	}
	var commands int
	err := tx.GetContext(ctx, &commands, `SELECT count(*) FROM external_commands WHERE task_id=? AND kind='TASK_ADD' AND (payload_version<>1 OR (json_type(payload_json,'$.generation') IS NOT NULL AND json_type(payload_json,'$.generation')<>'integer') OR (coalesce(json_extract(payload_json,'$.generation'),0)=? AND state<>'FAILED'))`, previous.ID, previous.Generation)
	if err != nil {
		return err
	}
	if commands > 0 {
		return ErrSubmissionUncertain
	}
	return nil
}
