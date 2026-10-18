package graphqlapi

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/internal/imagecache"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type taskScenesStash struct {
	fakeStashService
	scenes []stashsync.IngestScene
	path   string
	ids    []string
	calls  int
	err    error
}

func (s *taskScenesStash) FindIngestScenes(_ context.Context, path string, ids []string) ([]stashsync.IngestScene, error) {
	s.path, s.ids = path, ids
	s.calls++
	return s.scenes, s.err
}

type taskScenesImageCache struct{ descriptor imagecache.Descriptor }

func (c *taskScenesImageCache) Register(_ context.Context, d imagecache.Descriptor) (string, error) {
	c.descriptor = d
	return "/api/images/proxied", nil
}
func (*taskScenesImageCache) Status(context.Context) (imagecache.Status, error) {
	return imagecache.Status{}, nil
}
func (*taskScenesImageCache) Clear(context.Context) (imagecache.Status, error) {
	return imagecache.Status{}, nil
}
func (*taskScenesImageCache) Cleanup(context.Context) error { return nil }

func TestTaskStashScenesReturnsOnlyLinkedMoviesWithProxiedImages(t *testing.T) {
	stash := &taskScenesStash{scenes: []stashsync.IngestScene{
		{ID: "1", Title: "First", Code: "ABCD-123", StudioName: "Studio", Date: "2026-10-18", PerformerNames: []string{"Actor"}, ImageURL: "http://internal/cover?apikey=private"},
		{ID: "2", Title: "Second"}, {ID: "unrelated", Title: "Unrelated"},
	}}
	runtime := &fakeTaskRuntime{findTask: &taskruntime.Task{ID: "task", StashScanPath: "/library/movie", StashSceneIDsJSON: `["2","1","missing","1"]`}}
	resolver := NewResolver(nil, nil, runtime, stash, "test")
	resolver.RuntimeSettings = &SettingsSnapshot{Stash: StashSettingsSnapshot{URL: "https://user:pass@stash.test/base?apikey=private", APIKey: "secret"}}
	cache := &taskScenesImageCache{}
	resolver.ImageCache = cache
	var response struct {
		Data struct {
			Scenes []struct {
				ID, Title, Code, Studio, Date string
				Performers                    []string
				ImageURL, URL                 *string
			} `json:"taskStashScenes"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	executeGraphQLInto(t, resolver, `{ taskStashScenes(id:"task") { id title code studio date performers imageUrl url } }`, &response)
	if len(response.Errors) > 0 || len(response.Data.Scenes) != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
	scenes := response.Data.Scenes
	if scenes[0].ID != "2" || scenes[1].ID != "1" || scenes[1].Studio != "Studio" || !reflect.DeepEqual(scenes[1].Performers, []string{"Actor"}) {
		t.Fatalf("incorrect movie data: %+v", scenes)
	}
	if scenes[1].ImageURL == nil || *scenes[1].ImageURL != "/api/images/proxied" || scenes[1].URL == nil || *scenes[1].URL != "https://stash.test/base/scenes/1" {
		t.Fatalf("unsafe or missing URLs: %+v", scenes[1])
	}
	if cache.descriptor.APIKey != "secret" || cache.descriptor.Kind != imagecache.SourceStash {
		t.Fatal("image authentication was not passed to proxy")
	}
	if stash.path != "/library/movie" || !reflect.DeepEqual(stash.ids, []string{"2", "1", "missing", "1"}) {
		t.Fatal("scene lookup was not scoped to task")
	}
}

func TestTaskStashScenesEmptyAndUnavailable(t *testing.T) {
	stash := &taskScenesStash{err: errors.New("unavailable")}
	runtime := &fakeTaskRuntime{findTask: &taskruntime.Task{ID: "pending"}}
	resolver := NewResolver(nil, nil, runtime, stash, "test")
	scenes, err := resolver.loadTaskStashScenes(context.Background(), "pending")
	if err != nil || len(scenes) != 0 || stash.calls != 0 {
		t.Fatalf("empty task contacted Stash: %+v %v", scenes, err)
	}
	runtime.findTask.StashSceneIDsJSON = `["1"]`
	if _, err := resolver.loadTaskStashScenes(context.Background(), "pending"); err == nil {
		t.Fatal("Stash error was hidden")
	}
	runtime.findTask = nil
	if scenes, err := resolver.loadTaskStashScenes(context.Background(), "deleted"); err != nil || len(scenes) != 0 {
		t.Fatal("deleted task must be empty")
	}
	for _, base := range []string{"javascript:alert(1)", "/relative", "https://"} {
		if stashTaskSceneURL(base, "1") != nil {
			t.Fatal("invalid Stash URL accepted")
		}
	}
}
