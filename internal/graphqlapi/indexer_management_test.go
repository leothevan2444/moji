package graphqlapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type managementDefinitions struct{ bundle *indexerdefs.Bundle }

func (d managementDefinitions) Acquire(context.Context, string) (*indexerdefs.Lease, error) {
	return &indexerdefs.Lease{Bundle: d.bundle}, nil
}
func (managementDefinitions) RequireActive(context.Context, string) error { return nil }
func TestIndexerManagementGraphQLTypedChangesAndSecretProjection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	plan := &cardigann.Plan{Digest: "digest", Definition: cardigann.Definition{Login: &cardigann.Login{Method: "cookie"}, ID: "example", Name: "Example", Links: []string{"https://example.invalid/"}, Settings: []cardigann.Setting{{Name: "password", Type: "password"}, {Name: "flag", Type: "checkbox"}}}}
	admin := indexeradmin.NewService(store, managementDefinitions{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}, nil)
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.IndexerAdmin = admin
	type errorResponse struct {
		Message    string
		Extensions struct{ Code string }
	}
	var initial struct {
		Errors []errorResponse
		Data   struct {
			Indexers    model.IndexerInstances      `json:"indexerInstances"`
			Definitions model.IndexerDefinitionPage `json:"indexerDefinitions"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `{indexerInstances{revision items{id}} indexerDefinitions(input:{limit:1}){totalCount items{id loginSupported settings{name secret value{string}}}}}`, &initial)
	if len(initial.Errors) > 0 || initial.Data.Indexers.Revision != store.Revision() || initial.Data.Definitions.TotalCount != 1 || !initial.Data.Definitions.Items[0].LoginSupported {
		t.Fatal("management query contract", initial.Errors)
	}
	var added struct {
		Errors []errorResponse
		Data   struct {
			Result model.BuiltinIndexerResult `json:"addBuiltinIndexer"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {addBuiltinIndexer(expectedRevision:%q,input:{definitionId:"example",enabled:true,settings:[{name:"password",value:{string:"synthetic-secret"}},{name:"flag",value:{boolean:false}}]}){revision instance{id enabled loginSupported settings{name secret configured value{string boolean}}}}}`, store.Revision()), &added)
	if len(added.Errors) > 0 || added.Data.Result.Instance == nil || !added.Data.Result.Instance.Enabled || !added.Data.Result.Instance.LoginSupported {
		t.Fatal("add builtin", added.Errors)
	}
	body, _ := json.Marshal(added)
	if strings.Contains(string(body), "synthetic-secret") {
		t.Fatal("saved secret exposed through GraphQL")
	}
	sawFalse := false
	for _, setting := range added.Data.Result.Instance.Settings {
		if setting.Name == "password" && (!setting.Configured || !setting.Secret || setting.Value != nil) {
			t.Fatal("masked secret contract")
		}
		if setting.Name == "flag" {
			sawFalse = setting.Value != nil && setting.Value.Boolean != nil && !*setting.Value.Boolean
		}
	}
	if !sawFalse {
		t.Fatal("explicit false disappeared")
	}
	id := added.Data.Result.Instance.ID
	before, _ := os.ReadFile(path)
	var invalid struct{ Errors []errorResponse }
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {updateBuiltinIndexer(id:%q,expectedRevision:%q,input:{settings:[{name:"flag",value:{boolean:false,string:"ambiguous"}}]}){revision}}`, id, store.Revision()), &invalid)
	if len(invalid.Errors) != 1 || invalid.Errors[0].Extensions.Code != "INDEXER_SETTINGS_INVALID" {
		t.Fatal("ambiguous typed value accepted", invalid.Errors)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("rejected update changed disk")
	}
	var updated struct {
		Errors []errorResponse
		Data   struct {
			Result model.BuiltinIndexerResult `json:"updateBuiltinIndexer"`
		}
	}

	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {updateBuiltinIndexer(id:%q,expectedRevision:%q,input:{trustedOrigins:["https://download.example"]}){revision instance{id trustedOrigins}}}`, id, store.Revision()), &updated)
	if len(updated.Errors) > 0 || len(updated.Data.Result.Instance.TrustedOrigins) != 1 || updated.Data.Result.Instance.TrustedOrigins[0] != "https://download.example" {
		t.Fatal("trusted origin grant was not projected", updated.Errors)
	}
	protected, _ := os.ReadFile(path)
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {updateBuiltinIndexer(id:%q,expectedRevision:%q,input:{trustedOrigins:["https://download.example/private"]}){revision}}`, id, store.Revision()), &invalid)
	afterGrant, _ := os.ReadFile(path)
	if len(invalid.Errors) != 1 || invalid.Errors[0].Extensions.Code != "SETTINGS_INVALID" || string(protected) != string(afterGrant) {
		t.Fatal("invalid origin changed configuration", invalid.Errors)
	}
	invalid.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {updateBuiltinIndexer(id:%q,expectedRevision:%q,input:{name:"Renamed",trustedOrigins:[]}){revision instance{id name trustedOrigins}}}`, id, store.Revision()), &updated)
	if len(updated.Errors) > 0 || updated.Data.Result.Instance.Name != "Renamed" || len(updated.Data.Result.Instance.TrustedOrigins) != 0 || *store.Config().Indexers.Builtin.Instances[id].Settings["password"].String != "synthetic-secret" {
		t.Fatal("partial patch lost secret", updated.Errors)
	}
	invalid.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {setIndexerEnabled(id:%q,expectedRevision:%q,enabled:false){revision}}`, id, added.Data.Result.Revision), &invalid)
	if len(invalid.Errors) != 1 || invalid.Errors[0].Extensions.Code != "CONFIG_REVISION_CONFLICT" {
		t.Fatal("stale write accepted", invalid.Errors)
	}
	var cleared struct{ Errors []errorResponse }
	executeDefinitionGraphQLInto(t, resolver, fmt.Sprintf(`mutation {updateBuiltinIndexer(id:%q,expectedRevision:%q,input:{settings:[{name:"password",clear:true}]}){revision}}`, id, store.Revision()), &cleared)
	if len(cleared.Errors) > 0 {
		t.Fatal(cleared.Errors)
	}
	if _, exists := store.Config().Indexers.Builtin.Instances[id].Settings["password"]; exists {
		t.Fatal("explicit clear did not persist")
	}
}
