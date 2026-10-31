package graphqlapi

import (
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

func TestSettingsPatchReleaseTimezone(t *testing.T) {
	manual := "America/New_York"
	value := settingsPatchValue(reflect.ValueOf(model.SettingsPatchInput{System: &model.SystemPatchInput{ReleaseTimezone: &manual}}))
	patch := value.(map[string]any)
	if patch["system"].(map[string]any)["release_timezone"] != manual {
		t.Fatalf("timezone lost: %+v", patch)
	}
}
