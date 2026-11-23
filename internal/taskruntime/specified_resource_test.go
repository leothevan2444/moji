package taskruntime

import (
	"context"
	"testing"
)

func TestSpecifiedURLFailurePreservesChoiceAndRetrySkipsSearch(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTaskStore()
	adder := &fakeTorrentAdder{}
	s, err := NewService(fakeTracker{}, adder, store, torrentResponse("invalid torrent", 200))
	if err != nil {
		t.Fatal(err)
	}
	url := "https://download.example/opaque-resource"
	task, err := s.AddTorrentContext(ctx, AddTorrentRequest{Film: testFilm("SGKI-105B"), URL: url})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.processQueuedSourcing(ctx, task.ID)
	blocked, err := store.Find(ctx, task.ID)
	if err != nil || blocked.Stage != TaskStageDownloading || blocked.StageStatus != TaskStageStatusBlocked || blocked.TorrentURL != url || blocked.SourcingNextCheckAt != nil {
		t.Fatalf("explicit resource fell back to automatic sourcing: %+v %v", blocked, err)
	}
	torrentResponse(testTorrentFile("SGKI-105B", []string{"unknown.mp4"}), 200)(s)
	if _, err = s.RetryTask(ctx, task.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err = s.processQueuedSourcing(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Find(ctx, task.ID)
	if err != nil || loaded.Stage != TaskStageDownloading || loaded.StageStatus != TaskStageStatusRunning || loaded.TorrentURL != url || loaded.Film.SceneID != task.Film.SceneID {
		t.Fatalf("retry did not retain the confirmed choice: %+v %v", loaded, err)
	}
}
