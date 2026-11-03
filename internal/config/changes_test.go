package config

import (
	"sync"
	"testing"
)

func TestConfigChangesFollowDurableUpdatesAndCoalesce(t *testing.T) {
	s := patchStore(t)
	changes, stop, err := s.SubscribeChanges()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	<-changes
	if _, err = s.ApplyPatch(map[string]any{"indexers": map[string]any{"jackett": map[string]any{"enabled": false}}}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changes:
	default:
		t.Fatal("patch did not notify")
	}
	if s.Config().Indexers.Jackett.Enabled {
		t.Fatal("notification preceded committed config")
	}
	if _, err = s.UpdateJackett("https://example.invalid", "synthetic-new-key", ""); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changes:
	default:
		t.Fatal("legacy update did not notify")
	}
	if _, err = s.ApplyPatch(map[string]any{"system": map[string]any{"image_cache": map[string]any{"max_size_mb": -1}}}); err == nil {
		t.Fatal("invalid configuration accepted")
	}
	select {
	case <-changes:
		t.Fatal("rejected validation published a change")
	default:
	}
	for i := 0; i < 12; i++ {
		if _, err = s.ApplyPatch(map[string]any{"system": map[string]any{"developer_mode": i%2 == 0}}); err != nil {
			t.Fatal(err)
		}
	}
	if len(changes) != 1 {
		t.Fatal("slow subscriber accumulated unbounded changes")
	}
	stop()
	stop()
}

func TestConfigChangeSubscriberLimitAndConcurrentRelease(t *testing.T) {
	s := patchStore(t)
	stops := make([]func(), 0, 128)
	for i := 0; i < 128; i++ {
		_, stop, err := s.SubscribeChanges()
		if err != nil {
			t.Fatal(err)
		}
		stops = append(stops, stop)
	}
	if _, _, err := s.SubscribeChanges(); err != ErrChangeSubscribersFull {
		t.Fatal("subscriber budget not enforced", err)
	}
	var wg sync.WaitGroup
	for _, stop := range stops {
		wg.Add(1)
		go func() { defer wg.Done(); stop(); stop() }()
	}
	if _, err := s.ApplyPatch(map[string]any{"system": map[string]any{"developer_mode": false}}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	_, stop, err := s.SubscribeChanges()
	if err != nil {
		t.Fatal("subscriber capacity not released", err)
	}
	stop()
}
