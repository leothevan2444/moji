package subscription

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestHomeSummaryReadsPersistedHealthWithoutStash(t *testing.T) {
	ctx := context.Background()
	sqlite, err := NewSQLiteStore(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlite.db.Close()
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for name, store := range map[string]Store{"sqlite": sqlite, "memory": NewMemoryStore()} {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 8; i++ {
				state := &PerformerState{PerformerID: fmt.Sprintf("p-%02d", i)}
				if i < 5 {
					state.LastError = "upstream unavailable"
				}
				if i > 0 {
					checked := now.Add(-time.Duration(i) * time.Minute)
					state.LastCheckedAt = &checked
				}
				if err := store.Put(ctx, state); err != nil {
					t.Fatal(err)
				}
			}
			// No Stash client is installed: a homepage read must stay local.
			summary, err := (&Service{store: store}).HomeSummary(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if summary.Total != 8 || summary.Attention != 5 || summary.Unchecked != 1 || summary.LastCheckedAt == nil || !summary.LastCheckedAt.Equal(now.Add(-time.Minute)) {
				t.Fatalf("incorrect summary: %+v", summary)
			}
			if len(summary.AttentionPerformerIDs) != 3 || summary.AttentionPerformerIDs[0] != "p-00" {
				t.Fatalf("expected bounded issue links: %+v", summary)
			}
		})
	}
}
