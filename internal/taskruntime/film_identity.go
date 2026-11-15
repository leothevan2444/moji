package taskruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/leothevan2444/moji/internal/film"
)

// ValidateFilmTasks rejects an unmapped generation before any worker can run.
// The scan is bounded; it does not hydrate attachments or execution checkpoints.
func (s *RelationalTaskStore) ValidateFilmTasks(ctx context.Context) error {
	after := ""
	for {
		var rows []struct {
			ID       string         `db:"id"`
			Code     string         `db:"code"`
			Snapshot sql.NullString `db:"film_json"`
		}
		if err := s.core.Reader.SelectContext(ctx, &rows, `SELECT id,code,film_json FROM tasks WHERE id>? ORDER BY id LIMIT 256`, after); err != nil {
			return err
		}
		for _, row := range rows {
			var target film.Snapshot
			if !row.Snapshot.Valid || json.Unmarshal([]byte(row.Snapshot.String), &target) != nil || target.Validate() != nil || target.Endpoint != film.Endpoint(target.Endpoint) || film.Code(target.Code) != film.Code(row.Code) {
				return fmt.Errorf("task %s requires explicit StashBox film mapping before startup", row.ID)
			}
			after = row.ID
		}
		if len(rows) < 256 {
			return nil
		}
	}
}

func (s *RelationalTaskStore) FindByFilm(ctx context.Context, target film.Snapshot) (*Task, error) {
	t, err := s.read(ctx, s.core.Reader, " WHERE json_extract(film_json,'$.Endpoint')=? AND json_extract(film_json,'$.SceneID')=?", film.Endpoint(target.Endpoint), target.SceneID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}
func (s *EventingTaskStore) FindByFilm(ctx context.Context, target film.Snapshot) (*Task, error) {
	if store, ok := s.store.(interface {
		FindByFilm(context.Context, film.Snapshot) (*Task, error)
	}); ok {
		return store.FindByFilm(ctx, target)
	}
	t, err := s.store.FindByCode(ctx, film.Code(target.Code))
	if err != nil || t == nil {
		return t, err
	}
	if t.Film == nil || t.Film.Key() != target.Key() {
		return nil, nil
	}
	return t, nil
}
func (s *Service) ExistingFilmTask(ctx context.Context, target film.Snapshot) (*Task, error) {
	if store, ok := s.store.(interface {
		FindByFilm(context.Context, film.Snapshot) (*Task, error)
	}); ok {
		t, err := store.FindByFilm(ctx, target)
		if err != nil || t != nil {
			return t, err
		}
	}
	t, err := s.store.FindByCode(ctx, film.Code(target.Code))
	if err != nil || t == nil {
		return t, err
	}
	if t.Film == nil || t.Film.Key() != target.Key() {
		return nil, ErrDuplicateCodeTask
	}
	return t, nil
}
