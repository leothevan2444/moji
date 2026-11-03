package indexer

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"strings"
	"sync"
	"testing"
	"time"
)

type countingRetainer struct {
	mu    sync.Mutex
	count int
}

func (r *countingRetainer) Retain(ctx context.Context, revision string) (func(), error) {
	r.mu.Lock()
	r.count++
	r.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { r.mu.Lock(); r.count--; r.mu.Unlock() }) }, ctx.Err()
}
func (r *countingRetainer) current() int { r.mu.Lock(); defer r.mu.Unlock(); return r.count }

func TestResourceReferencesFreezePersistAndProtectAcquisition(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	pins := &countingRetainer{}
	registry, err := NewResources(ctx, manager, pins)
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	instance := fixtureInstance()
	instance.BundleRevision = strings.Repeat("a", 40)
	ref := ResourceRef{Version: 1, ID: "opaque", Kind: Builtin, InstanceID: instance.ID, ConfigRevision: instance.ConfigRevision, BundleRevision: instance.BundleRevision, Locator: Locator{DownloadURL: "https://site.invalid/original"}}
	if err := registry.SaveBatch(ctx, instance, []ResourceRef{ref}); err != nil {
		t.Fatal(err)
	}
	*instance.Settings["password"].String = "changed"
	lease, err := registry.Resolve(ctx, ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	if *lease.Record.Instance.Settings["password"].String != "synthetic-password" || lease.Record.Ref.Locator.DownloadURL != ref.Locator.DownloadURL || pins.current() != 2 {
		t.Fatal("original context not frozen and retained")
	}
	if err := registry.SaveBatch(ctx, instance, []ResourceRef{ref}); err == nil {
		t.Fatal("immutable ID overwritten")
	}
	registry.Close()
	if pins.current() != 1 {
		t.Fatal("registry close released in-flight acquisition")
	}
	restored, err := NewResources(ctx, manager, pins)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if pins.current() != 2 {
		t.Fatal("persisted reference not retained after restart")
	}
	if err := manager.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `UPDATE indexer_resources SET expires_at=?`, instant(time.Now().Add(-time.Minute)))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := restored.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Resolve(ctx, ref.ID); !errors.Is(err, ErrExpired) {
		t.Fatal("expired reference resolved", err)
	}
	if pins.current() != 1 {
		t.Fatal("expiry removed in-flight reference")
	}
	lease.Release()
	if pins.current() != 0 {
		t.Fatal("lease leaked")
	}
}
