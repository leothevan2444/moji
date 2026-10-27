package downloader

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func TestManagedRemovalIntentSurvivesRuntimeLoss(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	q := &fakeClient{torrents: []qbittorrent.Torrent{{Hash: testHash, Name: "unlinked torrent", Progress: 1, State: qbittorrent.TorrentStateStalledUP}}, errorDelete: errors.New("response lost")}
	s, err := NewManaged(q, &fakeTasks{}, m)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Action(ctx, testHash, "REMOVE"); err == nil {
		t.Fatal("expected ambiguous response")
	}
	var state string
	if err := m.Core.Reader.Get(&state, "SELECT state FROM external_commands WHERE target_hash=?", testHash); err != nil || state != "UNCERTAIN" {
		t.Fatalf("state=%s err=%v", state, err)
	}
	policy := Policy{Enabled: true, Ratio: 2, DeleteFiles: true}
	if err := s.SetPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	runtimePath := m.Runtime.Path
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(runtimePath); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	q.torrents = nil // qB executed the command; only the response was lost.
	s, err = NewManaged(q, &fakeTasks{}, m)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := s.Policy(ctx)
	if err != nil || restored != policy {
		t.Fatal("policy lost with runtime cache", restored, err)
	}
	snap, _ := s.Snapshot(ctx)
	if len(snap.Torrents) != 1 || snap.Torrents[0].RemovalReason != "MANUAL_PENDING" {
		t.Fatalf("intent not restored: %+v", snap)
	}
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.Reader.Get(&state, "SELECT state FROM external_commands WHERE target_hash=?", testHash); err != nil || state != "CONFIRMED" {
		t.Fatalf("state=%s err=%v", state, err)
	}
	snap, _ = s.Snapshot(ctx)
	if len(snap.Torrents) != 1 || snap.Torrents[0].State != "removed" || q.deleted != 1 {
		t.Fatal("removal was replayed or provenance lost")
	}
}
func TestManagedRemovalNeverCallsQBWithoutDurableIntent(t *testing.T) {
	ctx := context.Background()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	q := &fakeClient{torrents: []qbittorrent.Torrent{{Hash: testHash, Progress: 1, State: qbittorrent.TorrentStateStalledUP}}}
	s, err := NewManaged(q, &fakeTasks{}, m)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER fail_command BEFORE INSERT ON external_commands BEGIN SELECT RAISE(ABORT,'simulated disk failure'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Action(ctx, testHash, "REMOVE"); err == nil {
		t.Fatal("ignored intent failure")
	}
	if q.deleted != 0 {
		t.Fatal("qB mutated despite failed durable intent")
	}
	var count int
	if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM audit_events"); err != nil || count != 0 {
		t.Fatal("partial audit committed", err)
	}
}
