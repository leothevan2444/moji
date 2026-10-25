package subscription

import (
	"context"
	"encoding/json"
)

func (s *MemoryStore) LoadCheckRuns(context.Context) ([]CheckRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs := make([]CheckRun, 0, len(s.checkRuns))
	for _, run := range s.checkRuns {
		runs = append(runs, cloneCheckRun(run))
	}
	sortCheckRuns(runs)
	return runs, nil
}

func (s *MemoryStore) SaveCheckRun(_ context.Context, run CheckRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.checkRuns == nil {
		s.checkRuns = map[string]CheckRun{}
	}
	s.checkRuns[run.ID] = cloneCheckRun(run)
	if len(s.checkRuns) > checkHistoryLimit {
		runs := make([]CheckRun, 0, len(s.checkRuns))
		for _, saved := range s.checkRuns {
			runs = append(runs, saved)
		}
		sortCheckRuns(runs)
		scheduledKept := false
		for index, saved := range runs {
			keep := index < checkHistoryLimit || saved.Status == "RUNNING" || (saved.Trigger == "SCHEDULED" && !scheduledKept)
			if saved.Trigger == "SCHEDULED" {
				scheduledKept = true
			}
			if !keep {
				delete(s.checkRuns, saved.ID)
			}
		}
	}
	return nil
}

func (s *SQLiteStore) LoadCheckRuns(ctx context.Context) ([]CheckRun, error) {
	var records []string
	if err := s.db.SelectContext(ctx, &records, `SELECT payload FROM subscription_check_runs ORDER BY started_at DESC`); err != nil {
		return nil, err
	}
	runs := make([]CheckRun, 0, len(records))
	for _, record := range records {
		var run CheckRun
		if err := json.Unmarshal([]byte(record), &run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (s *SQLiteStore) SaveCheckRun(ctx context.Context, run CheckRun) error {
	payload, err := json.Marshal(run)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO subscription_check_runs (id, trigger, status, started_at, payload) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET status=excluded.status, payload=excluded.payload`, run.ID, run.Trigger, run.Status, run.StartedAt.Format("2006-01-02T15:04:05.000000000Z"), string(payload)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM subscription_check_runs WHERE status != 'RUNNING' AND id NOT IN (SELECT id FROM subscription_check_runs ORDER BY started_at DESC LIMIT ?) AND id NOT IN (SELECT id FROM subscription_check_runs WHERE trigger = 'SCHEDULED' ORDER BY started_at DESC LIMIT 1)`, checkHistoryLimit); err != nil {
		return err
	}
	return tx.Commit()
}
