package stashsync

import (
	"context"
	"reflect"
	"testing"

	graphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type fakeIngestClient struct {
	fakeClient
	generate graphql.GenerateMetadataInput
	identify graphql.IdentifyMetadataInput
	defaults *graphql.IdentifyMetadataInput
	scenes   []*graphql.IngestScenes_FindScenes_Scenes
	calls    []string
}

func (f *fakeIngestClient) MetadataGenerate(_ context.Context, input graphql.GenerateMetadataInput) (string, error) {
	f.calls = append(f.calls, "generate")
	f.generate = input
	return "generate-job", nil
}
func (f *fakeIngestClient) MetadataIdentify(_ context.Context, input graphql.IdentifyMetadataInput) (string, error) {
	f.calls = append(f.calls, "identify")
	f.identify = input
	return "identify-job", nil
}
func (f *fakeIngestClient) IngestIdentifyDefaults(context.Context) (*graphql.IdentifyMetadataInput, error) {
	return f.defaults, nil
}
func (f *fakeIngestClient) FindIngestScenes(context.Context, string, []string) ([]*graphql.IngestScenes_FindScenes_Scenes, error) {
	return f.scenes, nil
}

func TestIngestJobsAreScopedAndUseStashIdentifySettings(t *testing.T) {
	source := "jav-scraper"
	merge := graphql.IdentifyFieldStrategyMerge
	client := &fakeIngestClient{defaults: &graphql.IdentifyMetadataInput{
		Sources: []*graphql.IdentifySourceInput{{Source: &graphql.ScraperSourceInput{ScraperID: &source}, Options: &graphql.IdentifyMetadataOptionsInput{FieldOptions: []*graphql.IdentifyFieldOptionsInput{{Field: "title", Strategy: merge}}}}},
	}}
	service, _ := NewService(client, nil)
	ids := []string{"1", "2"}
	if _, err := service.MetadataGenerate(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(client.generate.SceneIDs, ids) || client.generate.Phashes == nil || !*client.generate.Phashes || client.generate.Transcodes == nil || *client.generate.Transcodes {
		t.Fatalf("missing scope or playback/fingerprint generation: %+v", client.generate)
	}
	if client.generate.Overwrite != nil || client.generate.ForceTranscodes != nil {
		t.Fatal("generation should not overwrite assets or force transcodes")
	}
	if _, err := service.MetadataIdentify(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(client.identify.SceneIDs, ids) || len(client.identify.Paths) != 0 || client.identify.Sources[0].Source.ScraperID != &source || client.identify.Sources[0].Options.FieldOptions[0].Strategy != merge {
		t.Fatalf("Identify settings or scope lost: %+v", client.identify)
	}
}

func TestIngestTranscodesFollowCurrentSettings(t *testing.T) {
	client := &fakeIngestClient{}
	cfg := IntegrationConfig{}
	service, _ := NewService(client, func() IntegrationConfig { return cfg })
	for _, enabled := range []bool{false, true, false} {
		cfg.GenerateTranscodes = enabled
		if _, err := service.MetadataGenerate(context.Background(), []string{"1"}); err != nil {
			t.Fatal(err)
		}
		input := client.generate
		if input.Transcodes == nil || *input.Transcodes != enabled {
			t.Fatalf("transcodes should follow current setting %t: %+v", enabled, input)
		}
		for name, flag := range map[string]*bool{"covers": input.Covers, "previews": input.Previews, "sprites": input.Sprites, "phashes": input.Phashes} {
			if flag == nil || !*flag {
				t.Fatalf("%s must remain enabled", name)
			}
		}
		if input.Overwrite != nil || input.ForceTranscodes != nil {
			t.Fatal("generation must not overwrite or force transcodes")
		}
	}
}

func TestIngestRejectsEmptyScopeAndMissingIdentifySources(t *testing.T) {
	client := &fakeIngestClient{defaults: &graphql.IdentifyMetadataInput{}}
	service, _ := NewService(client, nil)
	if _, err := service.MetadataGenerate(context.Background(), nil); err == nil {
		t.Fatal("empty scope would process entire library")
	}
	if _, err := service.MetadataIdentify(context.Background(), nil); err == nil {
		t.Fatal("empty scope would process entire library")
	}
	if _, err := service.MetadataIdentify(context.Background(), []string{"1"}); err == nil {
		t.Fatal("missing sources must block")
	}
	if len(client.calls) != 0 {
		t.Fatal("invalid requests reached Stash")
	}
}

func TestIngestSceneLookupExcludesSimilarPaths(t *testing.T) {
	client := &fakeIngestClient{scenes: []*graphql.IngestScenes_FindScenes_Scenes{
		{ID: "1", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/ABCD-123/movie.mp4"}}},
		{ID: "2", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/ABCD-1234/movie.mp4"}}},
		{ID: "3", Files: []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/other/ABCD-123/movie.mp4"}}},
	}}
	service, _ := NewService(client, nil)
	scenes, err := service.FindIngestScenes(context.Background(), "/library/ABCD-123", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 || scenes[0].ID != "1" {
		t.Fatalf("incorrect path scope: %+v", scenes)
	}
}

func TestIngestSceneIncludesMovieMetadata(t *testing.T) {
	title, code, date, image := "Movie", "ABCD-123", "2026-10-18", "/scene/1/screenshot"
	client := &fakeIngestClient{scenes: []*graphql.IngestScenes_FindScenes_Scenes{{
		ID: "1", Title: &title, Code: &code, Date: &date,
		Studio:     &graphql.IngestScenes_FindScenes_Scenes_Studio{ID: "studio", Name: "Studio"},
		Performers: []*graphql.IngestScenes_FindScenes_Scenes_Performers{{ID: "actor", Name: "Actor"}, nil},
		Paths:      graphql.IngestScenes_FindScenes_Scenes_Paths{Screenshot: &image},
		Files:      []*graphql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/movie.mp4"}},
	}}}
	service, _ := NewService(client, nil)
	scenes, err := service.FindIngestScenes(context.Background(), "/library/movie.mp4", []string{"1"})
	if err != nil || len(scenes) != 1 {
		t.Fatalf("unexpected scenes: %+v %v", scenes, err)
	}
	scene := scenes[0]
	if scene.Title != title || scene.Code != code || scene.Date != date || scene.ImageURL != image || scene.StudioName != "Studio" || !reflect.DeepEqual(scene.PerformerNames, []string{"Actor"}) {
		t.Fatalf("missing movie metadata: %+v", scene)
	}
}
