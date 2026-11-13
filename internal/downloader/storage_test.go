package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	s, err := NewManaged(q, &fakeTasks{}, m, newTestSettings(t))
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
	policy := Policy{Enabled: true, AfterIngest: true, DeleteFiles: true, MinFreeGB: 50, CapacityOrder: "DISK_USAGE"}
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
	if _, err := storage.RepairDerived(ctx, root, "runtime"); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	q.torrents = nil // qB executed the command; only the response was lost.
	s, err = NewManaged(q, &fakeTasks{}, m, s.settings)
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
	s, err := NewManaged(q, &fakeTasks{}, m, newTestSettings(t))
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

func TestManagedCapacityNoProgressGuardSurvivesRuntimeLoss(t *testing.T) {
	ctx := context.Background()
	standalone, q, tasks := capacityFixture(t)
	policy, err := standalone.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewManaged(q, tasks, m, newTestSettings(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	q.released = 0
	clock := time.Now()
	s.now = func() time.Time { return clock }
	for i := 0; i < cleanupNoGainLimit; i++ {
		if err := s.Cycle(ctx); err != nil {
			t.Fatal(err)
		}
		clock = clock.Add(cleanupSettleTime)
	}
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	runtimePath := m.Runtime.Path
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(runtimePath); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.RepairDerived(ctx, root, "runtime"); err != nil {
		t.Fatal(err)
	}
	m, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s, err = NewManaged(q, tasks, m, s.settings)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return clock }
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(ctx)
	if q.deleted != cleanupNoGainLimit || snap.Cleanup.Phase != "WARNING" || snap.Cleanup.Reason != "NO_SPACE_GAIN" {
		t.Fatal("runtime loss discarded capacity deletion guard", q.deleted, snap.Cleanup.Reason)
	}
	*q.space = 60e9
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	var control cleanupControl
	if err := s.read(ctx, "cleanup", &control); err != nil || control.Step != "" || control.NoGain != 0 {
		t.Fatal(control, err)
	}
}

func TestManagedIntentFailureIsAnErrorWithoutPhantomPendingDelete(t *testing.T) {
	ctx := context.Background()
	standalone, q, tasks := capacityFixture(t)
	policy, err := standalone.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	service, err := NewManaged(q, tasks, manager, newTestSettings(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SetPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if err := manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TRIGGER reject_cleanup_intent BEFORE INSERT ON external_commands BEGIN SELECT RAISE(ABORT,'test write failure'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Cycle(ctx); !errors.Is(err, ErrCleanupPersistence) {
		t.Fatal(err)
	}
	snapshot, _ := service.Snapshot(ctx)
	if q.deleted != 0 || snapshot.Cleanup.Phase != "ERROR" || snapshot.Cleanup.Reason != "STATE_SAVE_FAILED" {
		t.Fatal(q.deleted, snapshot.Cleanup)
	}
	var control cleanupControl
	if err := service.read(ctx, "cleanup", &control); err != nil || control.Step != "FAILED" {
		t.Fatal(control, err)
	}
	no := false
	if _, err := service.PatchPolicy(ctx, &no, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := service.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = service.Snapshot(ctx)
	if snapshot.Cleanup.Phase != "IDLE" || q.deleted != 0 {
		t.Fatal("preparation failure became a ghost operation", snapshot.Cleanup, q.deleted)
	}
}

func TestCleanupPolicyTableRemovedWithoutImportingConfiguration(t *testing.T) {
	ctx := context.Background()

	database, err := storage.Open(ctx, storage.Options{Path: filepath.Join(t.TempDir(), "moji.db"), Durability: storage.Durable})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	migrations := storage.CoreMigrations()
	if err := database.Migrate(ctx, migrations[:8]); err != nil {
		t.Fatal(err)
	}
	if err := database.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE download_policies SET enabled=1,after_ingest=0,capacity_enabled=1,min_free_gb=99`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, migrations); err != nil {
		t.Fatal(err)
	}
	var tables int
	if err := database.Reader.Get(&tables, `SELECT count(*) FROM sqlite_master WHERE name='download_policies'`); err != nil || tables != 0 {
		t.Fatal("obsolete policy table retained", tables, err)
	}
	settings := newTestSettings(t)
	service := &Service{settings: settings}
	p, err := service.Policy(ctx)
	if err != nil || p != DefaultPolicy() {
		t.Fatal("database policy imported unexpectedly", p, err)
	}
	// The running service observes a saved YAML setting without recreation.
	c := settings.DownloaderCleanup()
	c.MinFreeGB = 77
	if err := settings.UpdateDownloaderCleanup(c); err != nil {
		t.Fatal(err)
	}
	p, err = service.Policy(ctx)
	if err != nil || p.MinFreeGB != 77 {
		t.Fatal("saved YAML setting not observed", p, err)
	}
}
