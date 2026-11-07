package graphqlapi

import (
	"context"
	"encoding/json"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalNetworkGraphQLProtectsPasswordAndScopedWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.IndexerNetwork = indexeradmin.NewService(store, nil, nil)
	var response struct {
		Errors []struct {
			Message    string
			Extensions struct{ Code string }
		}
		Data json.RawMessage
	}
	run := func(input string) {
		t.Helper()
		executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetwork(expectedRevision:"`+store.Revision()+`",input:{`+input+`}){flareSolverrMaxTimeoutMs revision proxyEnabled proxyUrl proxyUsername proxyPasswordConfigured flareSolverrEnabled flareSolverrUrl}}`, &response)
		if len(response.Errors) > 0 {
			t.Fatal(response.Errors)
		}
	}
	run(`proxyEnabled:true,proxyUrl:"http://proxy.example:8080",proxyUsername:"alice",proxyPassword:"synthetic-secret",flareSolverrMaxTimeoutMs:45000,flareSolverrEnabled:true,flareSolverrUrl:"http://solver.example:8191"`)
	if !strings.Contains(string(response.Data), `"flareSolverrMaxTimeoutMs":45000`) || store.Config().Indexers.Builtin.Network.FlareSolverr.MaxTimeoutMS != 45000 || strings.Contains(string(response.Data), "synthetic-secret") || !strings.Contains(string(response.Data), `"proxyPasswordConfigured":true`) {
		t.Fatal("secret leaked or not acknowledged")
	}
	run(`proxyEnabled:false,proxyPassword:""`)
	if store.Config().Indexers.Builtin.Network.Proxy.Password != "synthetic-secret" || store.Config().Indexers.Builtin.Network.Proxy.Enabled {
		t.Fatal("blank password or disable erased credentials")
	}
	run(`clearProxyPassword:true`)
	if store.Config().Indexers.Builtin.Network.Proxy.Password != "" {
		t.Fatal("clear password failed")
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetwork(expectedRevision:"stale",input:{proxyEnabled:true}){revision}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != "CONFIG_REVISION_CONFLICT" {
		t.Fatal("stale network write accepted", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetwork(expectedRevision:"`+store.Revision()+`",input:{siteTimeoutSeconds:150}){siteTimeoutSeconds}}`, &response)
	if len(response.Errors) > 0 || store.Config().Indexers.Builtin.SiteTimeoutSeconds != 150 || store.Config().Indexers.Builtin.Network.FlareSolverr.URL != "http://solver.example:8191" {
		t.Fatal("timeout overwrote shared network")
	}
	definition := "example"
	if _, _, err = store.UpdateBuiltinInstance(context.Background(), "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition}); err != nil {
		t.Fatal(err)
	}
	executeDefinitionGraphQLInto(t, resolver, `{__type(name:"Mutation"){fields{name}}}`, &response)
	if len(response.Errors) > 0 || strings.Contains(string(response.Data), "IndexerNetworkProfile") {
		t.Fatal("retired mutation survived", response.Errors)
	}
}
