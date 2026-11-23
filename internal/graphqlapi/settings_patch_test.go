package graphqlapi

import (
	"github.com/leothevan2444/moji/internal/config"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

func TestSettingsPatchPreservesEmptyListsFalseAndUntouchedCredentials(t *testing.T) {
	disabled := false
	empty := ""
	patch := settingsPatchValue(reflect.ValueOf(model.SettingsPatchInput{
		Indexers: &model.IndexersPatchInput{QueryOrder: []string{}, Jackett: &model.JackettPatchInput{Password: &empty}},
		System:   &model.SystemPatchInput{DeveloperMode: &disabled},
	})).(map[string]any)
	expected := map[string]any{"indexers": map[string]any{"query_order": []any{}, "jackett": map[string]any{"password": ""}}, "system": map[string]any{"developer_mode": false}}
	if !reflect.DeepEqual(patch, expected) {
		t.Fatalf("unexpected patch: %#v", patch)
	}
}

type patchRecordingEditor struct {
	*fakeSettingsEditor
	patch map[string]any
}

func (f *patchRecordingEditor) PatchSettings(patch map[string]any) (*SettingsSnapshot, error) {
	f.patch = patch
	return &SettingsSnapshot{}, nil
}
func TestPatchSettingsGraphQLDistinguishesOmissionFalseEmptyAndNull(t *testing.T) {
	editor := &patchRecordingEditor{fakeSettingsEditor: &fakeSettingsEditor{}}
	var response struct{ Errors []struct{ Message string } }
	executeGraphQLInto(t, &Resolver{SettingsEditor: editor}, `mutation {patchSettings(input:{indexers:{query_order:[],jackett:{password:"",api_key:null}},system:{developer_mode:false}}){system{developerMode}}}`, &response)
	if len(response.Errors) > 0 {
		t.Fatalf("unexpected API error: %+v", response.Errors)
	}
	expected := map[string]any{"indexers": map[string]any{"query_order": []any{}, "jackett": map[string]any{"password": ""}}, "system": map[string]any{"developer_mode": false}}
	if !reflect.DeepEqual(editor.patch, expected) {
		t.Fatalf("unexpected mutation patch: %#v", editor.patch)
	}
	executeGraphQLInto(t, &Resolver{SettingsEditor: editor}, `mutation{patchSettings(input:{system:{language:null}}){system{developerMode}}}`, &response)
	if len(response.Errors) == 0 {
		t.Fatal("empty patch accepted")
	}
}

func TestSettingsPatchTimezone(t *testing.T) {
	manual := "America/New_York"
	value := settingsPatchValue(reflect.ValueOf(model.SettingsPatchInput{System: &model.SystemPatchInput{Timezone: &manual}}))
	patch := value.(map[string]any)
	if patch["system"].(map[string]any)["timezone"] != manual {
		t.Fatalf("timezone lost: %+v", patch)
	}
}

func TestResourceGroupPatchPreservesReferenceNamesAndFalse(t *testing.T) {
	p := settingsPatchValue(reflect.ValueOf(model.ResourceRulesPatchInput{Groups: []*model.ResourceRuleGroupInput{{ID: "main", Name: "Main", Require: []string{}, Exclude: []string{}, Tiers: []*model.ResourceRuleTierInput{{ID: "one", Name: "One", Rules: []string{"shared"}}}}}})).(map[string]any)
	g := p["groups"].([]any)[0].(map[string]any)
	tier := g["tiers"].([]any)[0].(map[string]any)
	ref := tier["rules"].([]any)[0]
	if g["fallback"] != false || ref != "shared" {
		t.Fatalf("invalid YAML patch keys: %#v", p)
	}
}

func TestGlobalRuleReferencesAreScalarIDsInPatch(t *testing.T) {
	input := model.ResourceRuleGroupInput{ID: "main", Name: "Main", Require: []string{"wanted"}, Exclude: []string{"unwanted"}, Tiers: []*model.ResourceRuleTierInput{}}
	value := settingsPatchValue(reflect.ValueOf(input)).(map[string]any)
	if !reflect.DeepEqual(value["require"], []any{"wanted"}) || !reflect.DeepEqual(value["exclude"], []any{"unwanted"}) {
		t.Fatalf("global references still carry relations: %#v", value)
	}
}

func TestAtomicResourcePatchSurvivesConfigReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	version, active := 7, "main"
	input := model.ResourceRulesPatchInput{
		Version: &version, ActiveGroup: &active,
		Rules: []*model.ResourceRuleInput{
			{ID: "text", Name: "Literal label", Conditions: []*model.ResourceConditionInput{{Scope: "TITLE", Value: "a.b"}}},
			{ID: "size", Name: "Size", Conditions: []*model.ResourceConditionInput{{Scope: "SIZE", Minimum: "1073741824", Maximum: "21474836480"}}},
		},
		Groups: []*model.ResourceRuleGroupInput{{ID: "main", Name: "Main", Require: []string{"text", "size"}, Exclude: []string{}, Tiers: []*model.ResourceRuleTierInput{{ID: "first", Name: "First", Rules: []string{"text"}}}}},
	}
	patch := settingsPatchValue(reflect.ValueOf(input))
	if _, err := store.ApplyPatch(map[string]any{"tasks": map[string]any{"torrent_selection": patch}}); err != nil {
		t.Fatal(err)
	}
	next, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	p := next.Config().Automation.TorrentSelection
	g := p.Groups[0]
	if p.Version != 7 || p.Rules[0].Conditions[0].Value != "a.b" || p.Rules[1].Conditions[0].Minimum != "1073741824" || p.Rules[1].Conditions[0].Maximum != "21474836480" || g.Fallback || len(g.Exclude) != 0 || g.Tiers[0].Rules[0] != "text" {
		t.Fatalf("resource settings lost on reload: %#v", p)
	}
}
