package stashsync

import (
	"context"
	"errors"
	"testing"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type codeClient struct {
	fakeIngestClient
	identity          string
	matches           []*gql.ScrapedScene
	fingerprints      []*gql.ScrapedScene
	queryErr, saveErr error
	queries, saves    int
	cover             string
	savedInputs       []gql.SceneUpdateInput
}

func codeString(s string) *string            { return &s }
func (c *codeClient) IngestIdentity() string { return c.identity }
func (c *codeClient) ScrapeIngestScene(_ context.Context, _ gql.ScraperSourceInput, input gql.ScrapeSingleSceneInput, images bool) ([]*gql.ScrapedScene, error) {
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	if input.SceneID != nil {
		return c.fingerprints, nil
	}
	c.queries++
	if input.Query == nil || *input.Query != "DEMO-123" {
		return nil, errors.New("unexpected search")
	}
	return c.matches, nil
}
func (c *codeClient) SaveIngestScene(_ context.Context, identity string, input gql.SceneUpdateInput) error {
	if identity != c.identity {
		return ErrCodePlanChanged
	}
	c.saves++
	c.savedInputs = append(c.savedInputs, input)
	for _, scene := range c.scenes {
		if scene.ID != input.ID {
			continue
		}
		if input.Title != nil {
			scene.Title = input.Title
		}
		if input.Code != nil {
			scene.Code = input.Code
		}
		if input.Date != nil {
			scene.Date = input.Date
		}
		if input.Details != nil {
			scene.Details = input.Details
		}
		if input.Director != nil {
			scene.Director = input.Director
		}
		if input.Organized != nil {
			scene.Organized = *input.Organized
		}
		if input.Urls != nil {
			scene.Urls = append([]string{}, input.Urls...)
		}
		if input.StudioID != nil {
			scene.Studio = &gql.IngestScenes_FindScenes_Scenes_Studio{ID: *input.StudioID}
		}
		if input.PerformerIds != nil {
			scene.Performers = []*gql.IngestScenes_FindScenes_Scenes_Performers{}
			for _, id := range input.PerformerIds {
				scene.Performers = append(scene.Performers, &gql.IngestScenes_FindScenes_Scenes_Performers{ID: id})
			}
		}
		if input.TagIds != nil {
			scene.Tags = []*gql.IngestScenes_FindScenes_Scenes_Tags{}
			for _, id := range input.TagIds {
				scene.Tags = append(scene.Tags, &gql.IngestScenes_FindScenes_Scenes_Tags{ID: id})
			}
		}
		if input.StashIds != nil {
			scene.StashIds = []*gql.IngestScenes_FindScenes_Scenes_StashIds{}
			for _, ref := range input.StashIds {
				scene.StashIds = append(scene.StashIds, &gql.IngestScenes_FindScenes_Scenes_StashIds{Endpoint: ref.Endpoint, StashID: ref.StashID})
			}
		}
		if input.CoverImage != nil {
			digest, err := coverDigest(input.CoverImage)
			if err != nil {
				return err
			}
			c.cover = digest
		}
	}

	return c.saveErr
}

func (c *codeClient) IngestCoverDigest(context.Context, string, string) (string, error) {
	return c.cover, nil
}

func readyCodeClient() *codeClient {
	c := &codeClient{identity: "instance-one"}
	c.defaults = &gql.IdentifyMetadataInput{Sources: []*gql.IdentifySourceInput{{Source: &gql.ScraperSourceInput{StashBoxEndpoint: codeString("https://box.test/graphql")}}}}
	c.scenes = []*gql.IngestScenes_FindScenes_Scenes{{ID: "scene", Files: []*gql.IngestScenes_FindScenes_Scenes_Files{{Path: "/library/DEMO-123/video.mp4"}}}}
	c.matches = []*gql.ScrapedScene{{Title: codeString("Movie"), Code: codeString("DEMO-123"), Date: codeString("2026-01-01"), RemoteSiteID: codeString("remote-one"), Studio: &gql.ScrapedStudio{StoredID: codeString("studio")}, Performers: []*gql.ScrapedPerformer{{StoredID: codeString("performer")}}}}
	return c
}

func TestCodeIdentifyOnlySavesUniqueExactResultAfterZeroFingerprintMatches(t *testing.T) {
	client := readyCodeClient()
	s, _ := NewService(client, nil)
	plan, err := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if err != nil || plan == nil || client.saves != 0 {
		t.Fatalf("preparation mutated Stash or failed: %v", err)
	}
	if err := s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); err != nil {
		t.Fatal(err)
	}
	if client.saves != 1 || *client.scenes[0].Code != "DEMO-123" || len(client.scenes[0].Performers) != 1 {
		t.Fatal("metadata was not saved through Stash")
	}
}

func TestCodeIdentifyRejectsUncertainOrConflictingCandidates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*codeClient)
		want   error
		noPlan bool
	}{
		{"existing fingerprint", func(c *codeClient) { c.fingerprints = c.matches }, nil, true},
		{"no exact match", func(c *codeClient) { c.matches[0].Code = codeString("OTHER-999") }, ErrCodeNoMatch, true},
		{"code embedded in title", func(c *codeClient) { c.matches[0].Code = codeString("Movie DEMO-123 complete") }, ErrCodeNoMatch, true},
		{"multiple exact matches", func(c *codeClient) {
			copy := *c.matches[0]
			copy.RemoteSiteID = codeString("remote-two")
			c.matches = append(c.matches, &copy)
		}, ErrCodeAmbiguous, true},
		{"upstream error", func(c *codeClient) { c.queryErr = errors.New("unavailable") }, nil, true},
		{"manual identity", func(c *codeClient) { c.scenes[0].Code = codeString("OTHER-999") }, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := readyCodeClient()
			tc.change(c)
			s, _ := NewService(c, nil)
			plan, err := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
			if plan != nil || c.saves != 0 {
				t.Fatal("unsafe result reached saving")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if c.queryErr != nil && !errors.Is(err, c.queryErr) {
				t.Fatal("upstream outage became no-match")
			}
		})
	}
}

func TestCodeIdentifyReconcilesLostResponseAndProtectsManualEdits(t *testing.T) {
	c := readyCodeClient()
	s, _ := NewService(c, nil)
	plan, err := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if err != nil {
		t.Fatal(err)
	}
	c.identity = "different-account"
	if err := s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); !errors.Is(err, ErrCodePlanChanged) || c.saves != 0 {
		t.Fatal("changed instance accepted")
	}
	c.identity = "instance-one"
	c.scenes[0].Title = codeString("Manual edit")
	if err := s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); !errors.Is(err, ErrCodePlanChanged) || c.saves != 0 {
		t.Fatal("manual edit overwritten")
	}
	c.scenes[0].Title = nil
	c.saveErr = errors.New("response lost")
	if err := s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); err == nil {
		t.Fatal("uncertain result hidden")
	}
	c.saveErr = nil
	if err := s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); err != nil {
		t.Fatal(err)
	}
	if c.saves != 1 {
		t.Fatal("recovery repeated a successful save")
	}
}
