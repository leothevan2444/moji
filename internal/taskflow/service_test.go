package taskflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type fakeRuntime struct {
	auto    taskruntime.DownloadRequest
	manual  taskruntime.AddTorrentRequest
	indexed taskruntime.QueueIndexedResourceRequest
	calls   int
}

func TestCatalogObservationIsReusedButClientConfirmationIsRevalidated(t *testing.T) {
	runtime := &fakeRuntime{}
	s := NewService(runtime)
	v := film.Snapshot{Endpoint: "https://example.test/graphql", SceneID: "scene", Code: "MMR-AA244", ConfirmedAt: time.Now()}
	s.SetDiscoveredSceneResolver(fakeSceneResolver{err: errors.New("must not refetch catalog observation")})
	if _, err := s.QueueSubscriptionRelease(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmFilm(context.Background(), ConfirmFilmInput{Endpoint: v.Endpoint, SceneID: v.SceneID, Digest: v.Digest()}); err == nil {
		t.Fatal("client bypassed revalidation")
	}
	if runtime.calls != 1 {
		t.Fatal(runtime.calls)
	}
}

func (f *fakeRuntime) QueueDownloadContext(_ context.Context, r taskruntime.DownloadRequest) (*taskruntime.Task, error) {
	f.auto = r
	f.calls++
	return &taskruntime.Task{ID: "auto", Film: r.Film, Code: r.Code}, nil
}
func (f *fakeRuntime) AddTorrentContext(_ context.Context, r taskruntime.AddTorrentRequest) (*taskruntime.Task, error) {
	f.manual = r
	f.calls++
	return &taskruntime.Task{ID: "manual", Film: r.Film}, nil
}
func (f *fakeRuntime) QueueIndexedResourceContext(_ context.Context, r taskruntime.QueueIndexedResourceRequest) (*taskruntime.Task, error) {
	f.indexed = r
	f.calls++
	return &taskruntime.Task{ID: "indexed", Film: r.Film}, nil
}

type fakeSceneResolver struct {
	value film.Snapshot
	err   error
}

func (f fakeSceneResolver) ResolveDiscoveredScene(_ context.Context, id, endpoint string) (ResolvedScene, error) {
	return f.value, f.err
}
func TestAllAdmissionsResolveFilmAndPreserveItsCompleteCode(t *testing.T) {
	for _, code := range []string{"300MIUM-1437", "T38-042", "081124_001-1pon", "34ID-003"} {
		runtime := &fakeRuntime{}
		service := NewService(runtime)
		v := film.Snapshot{Endpoint: "https://example.test/graphql", SceneID: "scene", Code: code, Date: "2020-01-01", Title: "Film"}
		service.SetDiscoveredSceneResolver(fakeSceneResolver{value: v})
		for _, resource := range []string{"", "reference", "magnet"} {
			input := ConfirmFilmInput{Endpoint: v.Endpoint, SceneID: v.SceneID, Digest: v.Digest()}
			if resource == "reference" {
				input.ResourceID = "server-reference"
			}
			if resource == "magnet" {
				input.URL = "magnet:?xt=urn:btih:hash"
			}
			task, err := service.ConfirmFilm(context.Background(), input)
			if err != nil || task.Film == nil || task.Film.Code != code {
				t.Fatalf("lost film identity: %+v %v", task, err)
			}
		}
		if runtime.calls != 3 {
			t.Fatal(runtime.calls)
		}
	}
}
func TestConfirmationRejectsChangedMissingAndFutureFilms(t *testing.T) {
	runtime := &fakeRuntime{}
	service := NewService(runtime)
	v := film.Snapshot{Endpoint: "https://example.test/graphql", SceneID: "scene", Code: "300MIUM-1437", Date: "2020-01-01"}
	service.SetDiscoveredSceneResolver(fakeSceneResolver{value: v})
	if _, err := service.ConfirmFilm(context.Background(), ConfirmFilmInput{Endpoint: v.Endpoint, SceneID: v.SceneID, Digest: "old"}); err == nil {
		t.Fatal("changed confirmation accepted")
	}
	v.Date = "2999-01-01"
	service.SetDiscoveredSceneResolver(fakeSceneResolver{value: v})
	if _, err := service.ConfirmFilm(context.Background(), ConfirmFilmInput{Endpoint: v.Endpoint, SceneID: v.SceneID, Digest: v.Digest()}); err == nil {
		t.Fatal("future film accepted")
	}
	service.SetDiscoveredSceneResolver(fakeSceneResolver{err: errors.New("offline")})
	if _, err := service.QueueDiscoveredScene(context.Background(), v.SceneID, v.Endpoint); err == nil {
		t.Fatal("unverified film admitted")
	}
	if runtime.calls != 0 {
		t.Fatal("invalid task persisted")
	}
}
func TestResourceSuggestionsUseLegacyParserOnlyForRecommendation(t *testing.T) {
	got := ResourceQueries("hdd800.com@SONE-522 title")
	if len(got) != 2 || got[0] != "SONE-522" {
		t.Fatal(got)
	}
	got = ResourceQueries("300MIUM-1437")
	if len(got) != 1 || got[0] != "300MIUM-1437" {
		t.Fatal(got)
	}
}
