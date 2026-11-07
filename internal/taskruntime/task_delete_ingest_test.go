package taskruntime

import (
	"context"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/internal/stashsync"
)

type failedIdentifyDeleter struct {
	path string
	ids  []string
}

func (*failedIdentifyDeleter) DeletionIdentity() string { return "test-stash" }
func (f *failedIdentifyDeleter) DeleteTaskScenes(_ context.Context, _ string, path string, ids []string) error {
	f.path = path
	f.ids = append([]string{}, ids...)
	return nil
}

func TestIdentifyFailureKeepsSceneLinkAndAllowsDeletion(t *testing.T) {
	store := NewMemoryTaskStore()
	createIngestTask(t, store)
	scanner := readyIngestScanner()
	scanner.identifyErr = stashsync.ErrIdentifySourcesMissing
	service, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, store, WithFileOperator(&fakeFileOperator{}))
	advanceIngest(t, service, scanner)
	advanceIngest(t, service, scanner)
	if _, err := service.TriggerStashScans(context.Background(), scanner); err == nil {
		t.Fatal("expected identify failure")
	}
	task, err := store.Find(context.Background(), "ingest")
	if err != nil || task.StageStatus != TaskStageStatusBlocked || !reflect.DeepEqual(task.StashSceneIDs(), []string{"scene-1"}) {
		t.Fatal("scan identity lost after identify failure", err)
	}
	deleter := &failedIdentifyDeleter{}
	if _, err := service.DeleteTaskWithOptions(context.Background(), task.ID, DeleteTaskOptions{DeleteStashScene: true}, deleter); err != nil {
		t.Fatal(err)
	}
	if deleter.path != task.StashScanPath || !reflect.DeepEqual(deleter.ids, []string{"scene-1"}) {
		t.Fatal("deletion did not use the persisted scan identity")
	}
	if _, err := store.Find(context.Background(), task.ID); err == nil {
		t.Fatal("task was not deleted")
	}
}
