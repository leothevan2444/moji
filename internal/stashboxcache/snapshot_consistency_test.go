package stashboxcache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/storage"
)

func TestSnapshotReadIsConsistentDuringEvictionAndReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	service := NewManaged(manager.Cache, nil)
	defer service.Close()
	client := &fakeClient{scenes: testScenes(40)}
	if _, err := service.Load(ctx, client, "https://box", "p1", 40, CachePreferred); err != nil {
		t.Fatal(err)
	}
	key := PerformerKey{Endpoint: "https://box", PerformerID: "p1"}
	original, err := service.store.getSnapshot(ctx, key, true)
	if err != nil || original == nil {
		t.Fatal("fixture missing", err)
	}
	scenes := allSnapshotScenes(original, nil)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 50; i++ {
			if err := service.store.clear(ctx); err != nil {
				t.Error(err)
				return
			}
			if err := service.store.putSnapshot(ctx, original, scenes, true); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	for i := 0; i < 120; i++ {
		snapshot, err := service.store.getSnapshot(ctx, key, true)
		if err != nil {
			t.Error(err)
			break
		}
		if snapshot != nil && (snapshot.loadedCount() != 40 || !snapshot.Complete) {
			t.Errorf("observed partial committed snapshot: loaded=%d complete=%v", snapshot.loadedCount(), snapshot.Complete)
			break
		}
	}
	workers.Wait()
}
