package downloader

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

// NewManaged separates durable policy/commands from replaceable observations.
func NewManaged(client Client, tasks Tasks, m *storage.Manager) (*Service, error) {
	s := &Service{manager: m, client: client, tasks: tasks, now: time.Now, observations: make(chan []qbittorrent.Torrent, 1)}
	if err := s.readManaged(context.Background(), "snapshot", &s.snapshot); err != nil {
		return nil, err
	}
	// The core remains authoritative when runtime.db has been discarded.
	if err := s.restoreRemovalHistory(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) readManaged(ctx context.Context, key string, out any) error {
	if key == "policy" {
		p, ok := out.(*Policy)
		if !ok {
			return errors.New("downloader: invalid policy destination")
		}
		return s.manager.Core.Reader.QueryRowContext(ctx, `SELECT after_ingest,enabled,ratio,seed_hours,match_all,delete_files FROM download_policies WHERE id=1`).Scan(&p.AfterIngest, &p.Enabled, &p.Ratio, &p.SeedHours, &p.MatchAll, &p.DeleteFiles)
	}
	if key != "snapshot" {
		return errors.New("downloader: unknown runtime key")
	}
	var raw string
	err := s.manager.Runtime.Reader.GetContext(ctx, &raw, "SELECT value FROM downloader_state WHERE key='snapshot'")
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}
func (s *Service) writeManaged(ctx context.Context, key string, value any) error {
	if key == "policy" {
		p, ok := value.(Policy)
		if !ok {
			return errors.New("downloader: invalid policy value")
		}
		return s.manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
			_, err := tx.ExecContext(ctx, `UPDATE download_policies SET after_ingest=?,enabled=?,ratio=?,seed_hours=?,match_all=?,delete_files=?,revision=revision+1 WHERE id=1 AND (after_ingest IS NOT ? OR enabled IS NOT ? OR ratio IS NOT ? OR seed_hours IS NOT ? OR match_all IS NOT ? OR delete_files IS NOT ?)`, p.AfterIngest, p.Enabled, p.Ratio, p.SeedHours, p.MatchAll, p.DeleteFiles, p.AfterIngest, p.Enabled, p.Ratio, p.SeedHours, p.MatchAll, p.DeleteFiles)
			return err
		})
	}
	if key != "snapshot" {
		return errors.New("downloader: unknown runtime key")
	}
	snapshot, ok := value.(Snapshot)
	if !ok {
		return errors.New("downloader: invalid snapshot")
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	// A successful absence observation confirms the already durable intention.
	if err := s.confirmRemovals(ctx, snapshot); err != nil {
		return err
	}
	err = s.manager.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloader_state(key,value,observed_at) VALUES('snapshot',?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,observed_at=excluded.observed_at WHERE value IS NOT excluded.value`, string(raw), s.now().UTC().Format(time.RFC3339Nano))
		return err
	})
	if err == nil {
		s.cacheSnapshot(snapshot)
	}
	return err
}

// removeTorrent persists the intention before invoking qB. An error can mean
// the response was lost after execution, so it remains UNCERTAIN until observed.
func (s *Service) removeTorrent(ctx context.Context, hash string, deleteFiles bool, reason string) error {
	if s.manager == nil {
		return s.client.DeleteTorrents(ctx, []string{hash}, deleteFiles)
	}
	id := uuid.NewString()
	now := s.now().UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(map[string]any{"reason": reason, "delete_files": deleteFiles})
	if err != nil {
		return err
	}
	if err := s.manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO external_commands(id,target_hash,kind,state,payload_json,created_at,updated_at) VALUES(?,?,'REMOVE','SENT',?,?,?)`, id, hash, string(payload), now, now)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO audit_events(operation_id,subject_id,event_type,level,message,payload_json,created_at) VALUES(?,?,'torrent-remove-intent','info','torrent removal requested',?,?)`, id, hash, string(payload), now)
		return err
	}); err != nil {
		return fmt.Errorf("persist removal intent: %w", err)
	}
	mutationErr := s.client.DeleteTorrents(ctx, []string{hash}, deleteFiles)
	if mutationErr != nil {
		// Do not use a possibly canceled request context for the durable result.
		resultCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		saveErr := s.manager.Core.WithWrite(resultCtx, func(tx *sqlx.Tx) error {
			_, err := tx.ExecContext(resultCtx, `UPDATE external_commands SET state='UNCERTAIN',last_error=?,updated_at=? WHERE id=?`, mutationErr.Error(), s.now().UTC().Format(time.RFC3339Nano), id)
			return err
		})
		return errors.Join(mutationErr, saveErr)
	}
	return nil
}
func (s *Service) confirmRemovals(ctx context.Context, snapshot Snapshot) error {
	var removed []string
	for _, t := range snapshot.Torrents {
		if !t.Present && (t.RemovalReason == "MANUAL" || t.RemovalReason == "POLICY") {
			removed = append(removed, t.Hash)
		}
	}
	if len(removed) == 0 {
		return nil
	}
	// Check first through the read pool; repeated snapshots never reserve core.
	var pending int
	if err := s.manager.Core.Reader.GetContext(ctx, &pending, `SELECT count(*) FROM external_commands WHERE kind='REMOVE' AND state IN ('PENDING','SENT','UNCERTAIN')`); err != nil {
		return err
	}
	if pending == 0 {
		return nil
	}
	return s.manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for _, hash := range removed {
			if _, err := tx.ExecContext(ctx, `UPDATE external_commands SET state='CONFIRMED',updated_at=? WHERE target_hash=? AND kind='REMOVE' AND state IN ('PENDING','SENT','UNCERTAIN')`, s.now().UTC().Format(time.RFC3339Nano), hash); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Service) restoreRemovalHistory(ctx context.Context) error {
	type command struct {
		Hash      string `db:"target_hash"`
		State     string `db:"state"`
		Payload   string `db:"payload_json"`
		CreatedAt string `db:"created_at"`
	}
	var commands []command
	// Latest command per torrent, including confirmed provenance, rebuilds UI state.
	if err := s.manager.Core.Reader.SelectContext(ctx, &commands, `SELECT c.target_hash,c.state,c.payload_json,c.created_at FROM external_commands c WHERE c.kind='REMOVE' AND c.id=(SELECT n.id FROM external_commands n WHERE n.target_hash=c.target_hash AND n.kind='REMOVE' ORDER BY n.created_at DESC,n.id DESC LIMIT 1)`); err != nil {
		return err
	}
	for _, cmd := range commands {
		var payload struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(cmd.Payload), &payload); err != nil {
			return err
		}
		reason := strings.ToUpper(payload.Reason)
		if reason != "MANUAL" && reason != "POLICY" {
			return errors.New("downloader: unknown durable removal reason")
		}
		if cmd.State != "CONFIRMED" {
			reason += "_PENDING"
		}
		found := false
		for i := range s.snapshot.Torrents {
			if s.snapshot.Torrents[i].Hash == cmd.Hash {
				s.snapshot.Torrents[i].RemovalReason = reason
				found = true
				break
			}
		}
		if !found {
			s.snapshot.Torrents = append(s.snapshot.Torrents, Torrent{Hash: cmd.Hash, RemovalReason: reason})
		}
	}
	return nil
}

// ImportLegacyState transfers policy and provenance into a fresh generation.
// It never constructs a qB client or replays an external operation.
func ImportLegacyState(ctx context.Context, m *storage.Manager, records map[string]string) error {
	for key := range records {
		if key != "policy" && key != "snapshot" {
			return fmt.Errorf("downloader: unsupported legacy state key %q", key)
		}
	}
	s := &Service{manager: m, now: time.Now}
	policy := DefaultPolicy()
	if raw := records["policy"]; raw != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&policy); err != nil {
			return err
		}
	}
	if err := s.writeManaged(ctx, "policy", policy); err != nil {
		return err
	}
	var snapshot Snapshot
	if raw := records["snapshot"]; raw != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&snapshot); err != nil {
			return err
		}
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for _, torrent := range snapshot.Torrents {
			if torrent.RemovalReason == "" {
				continue
			}
			reason := strings.TrimSuffix(torrent.RemovalReason, "_PENDING")
			if reason != "MANUAL" && reason != "POLICY" {
				return fmt.Errorf("downloader: unsupported legacy removal reason %q", torrent.RemovalReason)
			}
			state := "CONFIRMED"
			if strings.HasSuffix(torrent.RemovalReason, "_PENDING") {
				state = "UNCERTAIN"
			}
			payload, err := json.Marshal(map[string]any{"reason": reason, "delete_files_known": false, "legacy_snapshot": torrent})
			if err != nil {
				return err
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			if _, err := tx.ExecContext(ctx, `INSERT INTO external_commands(id,target_hash,kind,state,payload_json,created_at,updated_at) VALUES(?,?,'REMOVE',?,?,?,?)`, uuid.NewString(), torrent.Hash, state, string(payload), now, now); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return s.writeManaged(ctx, "snapshot", snapshot)
}
