package graphqlapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"os"
	"path/filepath"
	"testing"
)

func TestDefinitionPolicyGraphQLReadsAndPartiallyUpdatesSavedConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerDefinitionPolicy = indexeradmin.NewService(store, nil, nil)
	type response struct {
		Errors []struct{ Extensions struct{ Code string } }
		Data   struct {
			Policy *model.IndexerDefinitionPolicy `json:"indexerDefinitionPolicy"`
			Update *model.IndexerDefinitionPolicy `json:"updateIndexerDefinitionPolicy"`
		}
	}
	var read response
	executeDefinitionGraphQLInto(t, r, `{indexerDefinitionPolicy{revision source schemaVersion checkIntervalHours autoUpdate}}`, &read)
	if len(read.Errors) != 0 || read.Data.Policy == nil || read.Data.Policy.Source != "prowlarr" || read.Data.Policy.SchemaVersion != 11 || read.Data.Policy.CheckIntervalHours != 24 || read.Data.Policy.AutoUpdate {
		t.Fatal("policy is not the actual default configuration", read)
	}
	initial := read.Data.Policy.Revision
	var updated response
	executeDefinitionGraphQLInto(t, r, fmt.Sprintf(`mutation{updateIndexerDefinitionPolicy(expectedRevision:%q,input:{checkIntervalHours:48,autoUpdate:true}){revision source schemaVersion checkIntervalHours autoUpdate}}`, initial), &updated)
	if len(updated.Errors) != 0 || updated.Data.Update == nil || updated.Data.Update.Revision != store.Revision() || !updated.Data.Update.AutoUpdate || updated.Data.Update.CheckIntervalHours != 48 {
		t.Fatal("update did not return exact acknowledged revision", updated)
	}
	var off response
	executeDefinitionGraphQLInto(t, r, fmt.Sprintf(`mutation{updateIndexerDefinitionPolicy(expectedRevision:%q,input:{autoUpdate:false}){revision checkIntervalHours autoUpdate}}`, store.Revision()), &off)
	if len(off.Errors) != 0 || off.Data.Update == nil || off.Data.Update.AutoUpdate || off.Data.Update.CheckIntervalHours != 48 {
		t.Fatal("explicit false or omission was lost", off)
	}
	for _, item := range []struct{ revision, input, code string }{{initial, `autoUpdate:true`, "CONFIG_REVISION_CONFLICT"}, {store.Revision(), `checkIntervalHours:169`, "SETTINGS_INVALID"}} {
		var rejected response
		executeDefinitionGraphQLInto(t, r, fmt.Sprintf(`mutation{updateIndexerDefinitionPolicy(expectedRevision:%q,input:{%s}){revision}}`, item.revision, item.input), &rejected)
		if len(rejected.Errors) != 1 || rejected.Errors[0].Extensions.Code != item.code || rejected.Data.Update != nil {
			t.Fatal("invalid policy edit was acknowledged", rejected)
		}
	}
}
func TestDefinitionPolicyWithoutConfiguredServiceDoesNotPretendToSave(t *testing.T) {
	r := NewResolver(nil, nil, nil, "test")
	if _, err := r.indexerDefinitionPolicy(context.Background()); !errors.Is(err, indexer.ErrNotConfigured) {
		t.Fatal(err)
	}
	if _, err := r.updateIndexerDefinitionPolicy(context.Background(), "revision", model.IndexerDefinitionPolicyPatchInput{}); !errors.Is(err, indexer.ErrNotConfigured) {
		t.Fatal(err)
	}
}
