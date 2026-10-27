package storage

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type TaskObservation struct {
	TaskID           string  `db:"task_id"`
	TorrentName      string  `db:"torrent_name"`
	Progress         float64 `db:"progress"`
	QBittorrentState string  `db:"qbittorrent_state"`
	ContentPath      string  `db:"content_path"`
	LastProgressAt   *string `db:"last_progress_at"`
	ObservedAt       string  `db:"observed_at"`
}
type observationEntry struct {
	value    TaskObservation
	sequence uint64
	deleted  bool
}
type ObservationStore struct {
	db       *Database
	mu       sync.RWMutex
	flushMu  sync.Mutex
	values   map[string]TaskObservation
	dirty    map[string]observationEntry
	sequence uint64
}

func NewObservationStore(ctx context.Context, db *Database) (*ObservationStore, error) {
	var rows []TaskObservation
	if err := db.Reader.SelectContext(ctx, &rows, "SELECT task_id,torrent_name,progress,qbittorrent_state,content_path,last_progress_at,observed_at FROM task_observations"); err != nil {
		return nil, err
	}
	s := &ObservationStore{db: db, values: map[string]TaskObservation{}, dirty: map[string]observationEntry{}}
	for _, value := range rows {
		s.values[value.TaskID] = value
	}
	return s, nil
}
func cloneObservation(v TaskObservation) TaskObservation {
	if v.LastProgressAt != nil {
		value := *v.LastProgressAt
		v.LastProgressAt = &value
	}
	return v
}
func (s *ObservationStore) Get(id string) (TaskObservation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[id]
	return cloneObservation(v), ok
}
func (s *ObservationStore) Put(value TaskObservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.values[value.TaskID]
	left, right := previous, value
	left.ObservedAt, right.ObservedAt = "", ""
	if exists && reflect.DeepEqual(left, right) {
		return
	}
	s.sequence++
	value = cloneObservation(value)
	s.values[value.TaskID] = value
	s.dirty[value.TaskID] = observationEntry{value: value, sequence: s.sequence}
}
func (s *ObservationStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	delete(s.values, id)
	s.dirty[id] = observationEntry{sequence: s.sequence, deleted: true}
}
func (s *ObservationStore) Pending() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.dirty) }

// Flush coalesces updates per task and commits at most 256 tasks per transaction.
// Failed writes retain the dirty entries; a newer observation is never cleared
// by an earlier flush which finishes after the replacement was queued.
func (s *ObservationStore) Flush(ctx context.Context) error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	s.mu.RLock()
	batch := make(map[string]observationEntry)
	for key, value := range s.dirty {
		batch[key] = value
		if len(batch) == 256 {
			break
		}
	}
	s.mu.RUnlock()
	if len(batch) == 0 {
		return nil
	}
	if err := s.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for id, entry := range batch {
			if entry.deleted {
				if _, err := tx.ExecContext(ctx, "DELETE FROM task_observations WHERE task_id=?", id); err != nil {
					return err
				}
				continue
			}
			if _, err := tx.NamedExecContext(ctx, `INSERT INTO task_observations(task_id,torrent_name,progress,qbittorrent_state,content_path,last_progress_at,observed_at) VALUES(:task_id,:torrent_name,:progress,:qbittorrent_state,:content_path,:last_progress_at,:observed_at)
    ON CONFLICT(task_id) DO UPDATE SET torrent_name=excluded.torrent_name,progress=excluded.progress,qbittorrent_state=excluded.qbittorrent_state,content_path=excluded.content_path,last_progress_at=excluded.last_progress_at,observed_at=excluded.observed_at`, entry.value); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, entry := range batch {
		if current, ok := s.dirty[id]; ok && current.sequence == entry.sequence {
			delete(s.dirty, id)
		}
	}
	return nil
}
func (s *ObservationStore) FlushAll(ctx context.Context) error {
	for s.Pending() > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}
func observationTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z") }
