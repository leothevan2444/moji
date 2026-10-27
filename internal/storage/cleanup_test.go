package storage

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestDiagnosticsCleanupIsBoundedAndLeavesDurableAudits(t *testing.T) {
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	now := time.Now().UTC()
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		for i := 0; i < 1500; i++ {
			if _, err := tx.Exec(`INSERT INTO diagnostics(component,level,message,created_at) VALUES('test','debug','old',?)`, observationTime(now.Add(-30*24*time.Hour))); err != nil {
				return err
			}
		}
		_, err := tx.Exec(`INSERT INTO diagnostics(component,level,message,created_at) VALUES('test','error','new',?)`, observationTime(now))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO audit_events(subject_id,event_type,level,message,created_at) VALUES('choice','manual-decision','info','retain forever','2000-01-01')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	removed, err := m.PruneRuntime(ctx, now)
	if err != nil || removed != 1000 {
		t.Fatalf("cleanup not bounded: %d %v", removed, err)
	}
	removed, err = m.PruneRuntime(ctx, now)
	if err != nil || removed != 500 {
		t.Fatalf("remaining expired data: %d %v", removed, err)
	}
	var count int
	if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM audit_events"); err != nil || count != 1 {
		t.Fatal("durable decision pruned", err)
	}
	if err := m.Runtime.Reader.Get(&count, "SELECT count(*) FROM diagnostics"); err != nil || count != 1 {
		t.Fatal("fresh diagnostic pruned", err)
	}
}
