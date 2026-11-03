package config

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestCachePolicyPatchPreservesUnsubmittedFieldsAndRejectsStaleOrInvalidEdits(t *testing.T) {
	store := patchStore(t)
	ctx := context.Background()
	before := store.Config()
	revision := store.Revision()
	disabled := false
	saved, ack, err := store.UpdateIndexerSearchCache(ctx, revision, IndexerCachePatch{Enabled: &disabled})
	if err != nil || saved.Indexers.SearchCache.Enabled || ack != store.Revision() || ack == revision {
		t.Fatal("explicit false/revision acknowledgment lost", err)
	}
	if saved.Indexers.SearchCache.TTLSeconds != before.Indexers.SearchCache.TTLSeconds || saved.Connection.Jackett.APIKey != before.Connection.Jackett.APIKey {
		t.Fatal("cache update erased unsubmitted values")
	}
	protected, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	ttl := 600
	if _, _, err = store.UpdateIndexerSearchCache(ctx, revision, IndexerCachePatch{TTLSeconds: &ttl}); !errors.Is(err, ErrConfigRevision) {
		t.Fatal("stale edit accepted", err)
	}
	zero := 0
	if _, _, err = store.UpdateIndexerSearchCache(ctx, ack, IndexerCachePatch{TTLSeconds: &zero}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatal("invalid retention accepted", err)
	}
	after, err := os.ReadFile(store.Path())
	if err != nil || string(after) != string(protected) {
		t.Fatal("rejected policy edit altered durable file", err)
	}
	saved, ack, err = store.UpdateIndexerSearchCache(ctx, ack, IndexerCachePatch{TTLSeconds: &ttl})
	if err != nil || saved.Indexers.SearchCache.Enabled || saved.Indexers.SearchCache.TTLSeconds != 600 || saved.Indexers.SearchCache.EmptyTTLSeconds != before.Indexers.SearchCache.EmptyTTLSeconds {
		t.Fatal("partial policy update did not preserve explicit state", err)
	}
	reopened, err := OpenStore(store.Path())
	if err != nil || reopened.Revision() != ack || reopened.Config().Indexers.SearchCache.TTLSeconds != 600 {
		t.Fatal("acknowledged cache policy did not survive reopen", err)
	}
}
