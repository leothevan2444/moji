package stashsync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func codeBool(v bool) *bool { return &v }
func codeRule(field string, strategy gql.IdentifyFieldStrategy, create *bool) *gql.IdentifyFieldOptionsInput {
	return &gql.IdentifyFieldOptionsInput{Field: field, Strategy: strategy, CreateMissing: create}
}
func tinyCodeCover() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("test image payload"))
}

type creatingCodeClient struct {
	*codeClient
	entities    map[string]string
	studios     []gql.StudioCreateInput
	performers  []gql.PerformerCreateInput
	tags        []gql.TagCreateInput
	parents     []gql.StudioUpdateInput
	creationErr error
}

func (c *creatingCodeClient) FindIngestEntity(_ context.Context, kind, endpoint, id string) (string, error) {
	return c.entities[kind+":"+endpoint+":"+id], nil
}
func (c *creatingCodeClient) created(kind string, refs []*gql.StashIDInput) (string, error) {
	if c.creationErr != nil {
		return "", c.creationErr
	}
	id := fmt.Sprintf("new-%s-%d", kind, len(c.entities))
	if c.entities == nil {
		c.entities = map[string]string{}
	}
	if len(refs) > 0 {
		c.entities[kind+":"+refs[0].Endpoint+":"+refs[0].StashID] = id
	}
	return id, nil
}
func (c *creatingCodeClient) CreateIngestStudio(_ context.Context, _ string, in gql.StudioCreateInput) (string, error) {
	c.studios = append(c.studios, in)
	return c.created("studio", in.StashIds)
}
func (c *creatingCodeClient) CreateIngestPerformer(_ context.Context, _ string, in gql.PerformerCreateInput) (string, error) {
	c.performers = append(c.performers, in)
	return c.created("performer", in.StashIds)
}
func (c *creatingCodeClient) CreateIngestTag(_ context.Context, _ string, in gql.TagCreateInput) (string, error) {
	c.tags = append(c.tags, in)
	return c.created("tag", in.StashIds)
}
func (c *creatingCodeClient) UpdateIngestStudio(_ context.Context, _ string, in gql.StudioUpdateInput) error {
	c.parents = append(c.parents, in)
	return c.creationErr
}
func (c *creatingCodeClient) IngestStudioStashIDs(context.Context, string) ([]*gql.StashIDInput, error) {
	return []*gql.StashIDInput{{Endpoint: "other", StashID: "original"}}, nil
}

func TestCodeCoverUsesEffectiveSettingsAndVerifiesBytes(t *testing.T) {
	for _, tc := range []struct {
		name             string
		defaults, source *bool
		want             bool
	}{{"native default", nil, nil, true}, {"disabled", codeBool(false), nil, false}, {"source disables", codeBool(true), codeBool(false), false}, {"source enables", codeBool(false), codeBool(true), true}} {
		t.Run(tc.name, func(t *testing.T) {
			c := readyCodeClient()
			cover := tinyCodeCover()
			c.matches[0].Image = &cover
			c.defaults.Options = &gql.IdentifyMetadataOptionsInput{SetCoverImage: tc.defaults}
			c.defaults.Sources[0].Options = &gql.IdentifyMetadataOptionsInput{SetCoverImage: tc.source}
			s, _ := NewService(c, nil)
			plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
			if e != nil {
				t.Fatal(e)
			}
			if plan.Version != 2 {
				t.Fatal("cover-capable plan was not versioned")
			}
			if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil {
				t.Fatal(e)
			}
			if (c.savedInputs[0].CoverImage != nil) != tc.want {
				t.Fatal("cover setting was ignored")
			}
			if tc.want && c.cover == "" {
				t.Fatal("saved cover was not checked")
			}
			c.saveErr = nil
			if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil || c.saves != 1 {
				t.Fatal("resume repeated an applied metadata/cover save", e)
			}
		})
	}
}
func TestCodeSettingsSnapshotPreservesExplicitEmptyGenderList(t *testing.T) {
	options := effectiveCodeOptions(&gql.IdentifyMetadataOptionsInput{PerformerGenders: []gql.GenderEnum{gql.GenderEnumFemale}}, &gql.IdentifyMetadataOptionsInput{PerformerGenders: []gql.GenderEnum{}})
	raw, _ := json.Marshal(options)
	var loaded codeIdentifyOptions
	if e := json.Unmarshal(raw, &loaded); e != nil {
		t.Fatal(e)
	}
	if loaded.Genders == nil || loaded.acceptsGender(&gql.ScrapedPerformer{Gender: codeString("FEMALE")}) {
		t.Fatal("empty gender allowlist became all genders")
	}
	if !loaded.acceptsGender(&gql.ScrapedPerformer{}) {
		t.Fatal("native behavior retains performers with unknown gender")
	}
}
func TestCodeFieldsFollowNativeOverwriteAndIgnoreSemantics(t *testing.T) {
	scene := IngestScene{ID: "scene", Title: "Existing", Details: "Existing details", Organized: true, URLs: []string{"one", "two"}, PerformerIDs: []string{"old"}, TagIDs: []string{"old-tag"}, StashIDs: []*gql.StashIDInput{{Endpoint: "other", StashID: "old"}}}
	match := &gql.ScrapedScene{Title: codeString("New"), Details: codeString(""), Code: codeString("DEMO-123"), Date: codeString("not a date"), Urls: []string{"one"}, RemoteSiteID: codeString("remote"), Performers: []*gql.ScrapedPerformer{{Name: codeString("Unknown"), Gender: codeString("MALE")}}, Tags: []*gql.ScrapedTag{{Name: "Unknown tag"}}}
	defaults := &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("title", gql.IdentifyFieldStrategyOverwrite, nil), codeRule("details", gql.IdentifyFieldStrategyOverwrite, nil), codeRule("performers", gql.IdentifyFieldStrategyOverwrite, nil), codeRule("tags", gql.IdentifyFieldStrategyOverwrite, nil), codeRule("url", gql.IdentifyFieldStrategyOverwrite, nil), codeRule("stash_ids", gql.IdentifyFieldStrategyIgnore, nil)}, SetOrganized: codeBool(false), IncludeMalePerformers: codeBool(false)}
	source := &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("title", gql.IdentifyFieldStrategyIgnore, nil)}}
	input := codeSceneInput(scene, match, effectiveCodeOptions(defaults, source), "endpoint", time.Now(), false)
	if input.Title != nil || input.Details == nil || *input.Details != "" || input.Date != nil || input.Organized != nil || input.Urls != nil || input.StashIds != nil {
		t.Fatalf("native field behavior diverged: %+v", input)
	}
	if input.PerformerIds != nil || input.TagIds != nil {
		t.Fatal("native Identify preserves existing relationships when it resolves no IDs")
	}

}
func TestCodeCreationUsesSourceRulesProfilesAndSingleNameTags(t *testing.T) {
	c := &creatingCodeClient{codeClient: readyCodeClient()}
	c.matches[0].Studio = &gql.ScrapedStudio{Name: "New Studio", RemoteSiteID: codeString("studio-remote"), Aliases: codeString(" Alias , Second "), Parent: &gql.ScrapedStudio{Name: "Parent", StoredID: codeString("parent"), RemoteSiteID: codeString("parent-remote")}}
	c.matches[0].Performers = []*gql.ScrapedPerformer{{Name: codeString("First Last"), RemoteSiteID: codeString("performer-remote"), Gender: codeString("FEMALE"), Height: codeString("165"), CareerStart: codeString("2020"), Images: []string{"https://image.test/actor"}}, {Name: codeString("Single"), RemoteSiteID: codeString("skipped")}, {Name: codeString("Male Actor"), RemoteSiteID: codeString("male"), Gender: codeString("MALE")}}
	c.matches[0].Tags = []*gql.ScrapedTag{{Name: "New Tag", RemoteSiteID: codeString("tag-remote"), Description: codeString("Description"), AliasList: []string{"Alias"}}}
	c.defaults.Options = &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("studio", gql.IdentifyFieldStrategyMerge, codeBool(true)), codeRule("performers", gql.IdentifyFieldStrategyMerge, codeBool(true)), codeRule("tags", gql.IdentifyFieldStrategyMerge, codeBool(true))}, SkipSingleNamePerformers: codeBool(true), SkipSingleNamePerformerTag: codeString("skip-tag"), PerformerGenders: []gql.GenderEnum{gql.GenderEnumFemale}}
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil {
		t.Fatal(e)
	}
	if len(c.entities) != 0 {
		t.Fatal("preparation created entities")
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil {
		t.Fatal(e)
	}
	if len(c.studios) != 1 || len(c.parents) != 1 || len(c.performers) != 1 || len(c.tags) != 1 {
		t.Fatalf("configured creation/filtering diverged: studios=%d parents=%d performers=%d tags=%d", len(c.studios), len(c.parents), len(c.performers), len(c.tags))
	}
	if c.performers[0].HeightCm == nil || *c.performers[0].HeightCm != 165 || valueOrEmpty(c.performers[0].CareerStart) != "2020" || c.performers[0].Image == nil {
		t.Fatal("performer profile fields were discarded")
	}
	if len(c.savedInputs[0].TagIds) != 2 {
		t.Fatal("single-name skip marker was not applied")
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil || len(c.performers) != 1 || c.saves != 1 {
		t.Fatal("recovery duplicated related entities", e)
	}
}
func TestCodeSourceFieldRuleDoesNotInheritCreateMissing(t *testing.T) {
	options := effectiveCodeOptions(&gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("tags", gql.IdentifyFieldStrategyMerge, codeBool(true))}}, &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("tags", gql.IdentifyFieldStrategyOverwrite, nil)}})
	if options.rule("tags").CreateMissing {
		t.Fatal("source rule inherited default createMissing contrary to Stash")
	}
}
func TestCodeCreationConflictsDoNotGuessAssociations(t *testing.T) {
	c := &creatingCodeClient{codeClient: readyCodeClient(), creationErr: errors.New("studio with name 'Existing' already exists")}
	c.matches[0].Studio = &gql.ScrapedStudio{Name: "New", RemoteSiteID: codeString("remote")}
	c.defaults.Options = &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("studio", gql.IdentifyFieldStrategyMerge, codeBool(true))}}
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil {
		t.Fatal(e)
	}
	e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan)
	if !errors.Is(e, ErrStudioConflict) || c.saves != 0 {
		t.Fatal("studio conflict silently accepted", e)
	}
}

type orderedCodeClient struct {
	*codeClient
	keywords []string
}

func (c *orderedCodeClient) ScrapeIngestScene(_ context.Context, source gql.ScraperSourceInput, input gql.ScrapeSingleSceneInput, _ bool) ([]*gql.ScrapedScene, error) {
	if input.SceneID != nil {
		return nil, nil
	}
	c.keywords = append(c.keywords, *source.StashBoxEndpoint)
	return c.matches, nil
}
func TestCodeSourcesKeepConfiguredPriority(t *testing.T) {
	c := &orderedCodeClient{codeClient: readyCodeClient()}
	c.defaults.Sources = append(c.defaults.Sources, &gql.IdentifySourceInput{Source: &gql.ScraperSourceInput{StashBoxEndpoint: codeString("https://second.test/graphql")}})
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil || plan == nil || len(c.keywords) != 1 || plan.Endpoint != "https://box.test/graphql" {
		t.Fatal("source priority diverged", e)
	}
}
func TestCodeMultipleMatchesHonorSkipTagWithoutSelectingIdentity(t *testing.T) {
	c := readyCodeClient()
	copy := *c.matches[0]
	copy.RemoteSiteID = codeString("remote-two")
	c.matches = append(c.matches, &copy)
	c.defaults.Options = &gql.IdentifyMetadataOptionsInput{SkipMultipleMatches: codeBool(true), SkipMultipleMatchTag: codeString("skip-tag")}
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil {
		t.Fatal(e)
	}
	if plan.BlockReason != "multiple" || plan.Selected != nil || c.saves != 0 {
		t.Fatal("ambiguous selection was not preserved as a blocked intent")
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil {
		t.Fatal(e)
	}
	if c.savedInputs[0].Code != nil || len(c.savedInputs[0].StashIds) != 0 || len(c.savedInputs[0].TagIds) != 1 {
		t.Fatal("skip-tag operation selected a movie")
	}
}
func TestCodeCoverManualEditIsNotOverwritten(t *testing.T) {
	c := readyCodeClient()
	image := tinyCodeCover()
	c.matches[0].Image = &image
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil {
		t.Fatal(e)
	}
	c.cover = bytesDigest([]byte("manual cover"))
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); !errors.Is(e, ErrCodePlanChanged) || c.saves != 0 {
		t.Fatal("manual cover was overwritten", e)
	}
}

type lostCreationCodeClient struct {
	*creatingCodeClient
	lose bool
}

func (c *lostCreationCodeClient) CreateIngestPerformer(ctx context.Context, identity string, input gql.PerformerCreateInput) (string, error) {
	id, e := c.creatingCodeClient.CreateIngestPerformer(ctx, identity, input)
	if e != nil {
		return "", e
	}
	if c.lose {
		c.lose = false
		return "", errors.New("response lost after entity commit")
	}
	return id, nil
}
func TestCodeCreationLostResponseUsesExactRemoteIdentityOnRetry(t *testing.T) {
	c := &lostCreationCodeClient{creatingCodeClient: &creatingCodeClient{codeClient: readyCodeClient()}, lose: true}
	c.matches[0].Performers = []*gql.ScrapedPerformer{{Name: codeString("New Person"), RemoteSiteID: codeString("actor-remote")}}
	c.defaults.Options = &gql.IdentifyMetadataOptionsInput{FieldOptions: []*gql.IdentifyFieldOptionsInput{codeRule("performers", gql.IdentifyFieldStrategyMerge, codeBool(true))}}
	s, _ := NewService(c, nil)
	plan, e := s.PrepareCodeIdentify(context.Background(), "DEMO-123", "/library/DEMO-123", []string{"scene"})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e == nil {
		t.Fatal("lost response hidden")
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, plan); e != nil {
		t.Fatal(e)
	}
	if len(c.performers) != 1 || c.saves != 1 {
		t.Fatal("retry duplicated the created performer")
	}
}

type upgradingCodeClient struct {
	*codeClient
	lookups []string
}

func (c *upgradingCodeClient) ScrapeIngestScene(_ context.Context, _ gql.ScraperSourceInput, input gql.ScrapeSingleSceneInput, _ bool) ([]*gql.ScrapedScene, error) {
	if input.SceneID != nil {
		return nil, nil
	}
	c.lookups = append(c.lookups, valueOrEmpty(input.Query))
	if valueOrEmpty(input.Query) != "remote-one" {
		return nil, errors.New("upgrade must use the selected remote ID")
	}
	return c.matches, nil
}
func TestLegacyCodePlanUpgradeKeepsIdentityAndAddsMissingCover(t *testing.T) {
	c := &upgradingCodeClient{codeClient: readyCodeClient()}
	s, _ := NewService(c, nil)
	initial := gql.SceneUpdateInput{ID: "scene", Title: codeString("Movie"), Code: codeString("DEMO-123"), Date: codeString("2026-01-01"), StudioID: codeString("studio"), PerformerIds: []string{"performer"}, StashIds: []*gql.StashIDInput{{Endpoint: "https://box.test/graphql", StashID: "remote-one"}}}
	if e := c.SaveIngestScene(context.Background(), c.identity, initial); e != nil {
		t.Fatal(e)
	}
	image := tinyCodeCover()
	c.matches[0].Image = &image
	old := &CodeIdentifyPlan{Version: 1, Code: "DEMO-123", Identity: c.identity, Endpoint: "https://box.test/graphql", RemoteID: "remote-one", Scenes: []CodeSceneSave{{Input: gql.SceneUpdateInput{ID: "scene"}}}}
	plan, e := s.UpgradeCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, old)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(plan)
	var restored CodeIdentifyPlan
	if e = json.Unmarshal(raw, &restored); e != nil {
		t.Fatal(e)
	}
	if restored.Version != 2 || restored.RemoteID != "remote-one" || len(c.lookups) != 1 || c.lookups[0] != "remote-one" {
		t.Fatal("upgrade changed the chosen identity")
	}
	if e = s.ApplyCodeIdentify(context.Background(), "/library/DEMO-123", []string{"scene"}, &restored); e != nil || c.cover == "" || c.saves != 2 {
		t.Fatal("legacy metadata-only success prevented cover repair", e)
	}
}
