package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestObservationCoalescingAndRetry(t *testing.T) {
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := m.Observations
	for i := 0; i < 1000; i++ {
		s.Put(TaskObservation{TaskID: "one", Progress: float64(i) / 1000, ObservedAt: "2026-10-27T01:00:00Z"})
	}
	if s.Pending() != 1 {
		t.Fatalf("dirty=%d", s.Pending())
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`CREATE TRIGGER disk_failure BEFORE INSERT ON task_observations BEGIN SELECT RAISE(ABORT,'simulated write failure'); END`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(ctx); err == nil {
		t.Fatal("expected write failure")
	}
	if s.Pending() != 1 {
		t.Fatal("failed flush discarded progress")
	}
	if err := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { _, e := tx.Exec(`DROP TRIGGER disk_failure`); return e }); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	if s.Pending() != 0 {
		t.Fatal("dirty entries not acknowledged")
	}
	var progress float64
	if err := m.Runtime.Reader.Get(&progress, "SELECT progress FROM task_observations WHERE task_id='one'"); err != nil || progress != .999 {
		t.Fatalf("progress=%f err=%v", progress, err)
	}
	s.Delete("one")
	if err := s.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewObservationStore(ctx, m.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Get("one"); ok {
		t.Fatal("deleted observation returned after restart")
	}
}
func TestObservationConcurrentFlushDoesNotLoseNewerValues(t *testing.T) {
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := m.Observations
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				s.Put(TaskObservation{TaskID: fmt.Sprintf("task-%d", worker), Progress: float64(i) / 500, ObservedAt: "2026-10-27T01:00:00Z"})
				if i%20 == 0 {
					if err := s.Flush(ctx); err != nil {
						t.Error(err)
						return
					}
				}
			}
		}(worker)
	}
	wg.Wait()
	if err := s.FlushAll(ctx); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewObservationStore(ctx, m.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		v, ok := loaded.Get(fmt.Sprintf("task-%d", i))
		if !ok || v.Progress != .998 {
			t.Fatalf("lost newest value: %+v", v)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	s.Put(TaskObservation{TaskID: "canceled"})
	if err := s.FlushAll(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled flush: %v", err)
	}
	if s.Pending() != 1 {
		t.Fatal("canceled flush discarded observation")
	}
}
