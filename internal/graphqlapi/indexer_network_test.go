package graphqlapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexeradmin"
)

func TestNetworkGraphQLMasksCredentialsAndPreservesScopedWriteSemantics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\n"), 0600); err != nil {
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
	executeDefinitionGraphQLInto(t, resolver, `mutation{addIndexerNetworkProfile(id:"private",expectedRevision:"`+store.Revision()+`",input:{proxyUrl:"http://synthetic-user:synthetic-password@proxy.example:8080"}){revision profiles{id proxyConfigured usedBy}}}`, &response)
	if len(response.Errors) > 0 || !strings.Contains(string(response.Data), `"proxyConfigured":true`) || strings.Contains(string(response.Data), "synthetic") || strings.Contains(string(response.Data), "proxy.example") {
		t.Fatal("network projection exposed endpoint or credentials", response.Errors)
	}
	revision := store.Revision()
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkProfile(id:"private",expectedRevision:"`+revision+`",input:{proxyUrl:""}){revision}}`, &response)
	if len(response.Errors) > 0 || !strings.Contains(store.Config().Indexers.Builtin.NetworkProfiles["private"].ProxyURL, "synthetic-password") {
		t.Fatal("blank edit erased configured proxy", response.Errors)
	}
	definition, profile := "example", "private"
	if _, _, err = store.UpdateBuiltinInstance(context.Background(), "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition, NetworkProfile: &profile}); err != nil {
		t.Fatal(err)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{removeIndexerNetworkProfile(id:"private",expectedRevision:"`+store.Revision()+`"){revision}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != "INDEXER_NETWORK_PROFILE_IN_USE" {
		t.Fatal("referenced removal did not return actionable code", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkTimeouts(expectedRevision:"stale",input:{requestTimeoutSeconds:40}){revision}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Extensions.Code != "CONFIG_REVISION_CONFLICT" {
		t.Fatal("stale timeout update accepted", response.Errors)
	}
	response.Errors = nil
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkProfile(id:"private",expectedRevision:"`+store.Revision()+`",input:{clearProxy:true}){profiles{id proxyConfigured usedBy}}}`, &response)
	if len(response.Errors) > 0 || store.Config().Indexers.Builtin.NetworkProfiles["private"].ProxyURL != "" {
		t.Fatal("explicit clear failed", response.Errors)
	}
}

func TestNetworkGraphQLSolverOnlyAndTimeoutNeverReturnPrivateEndpoints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("config_version: 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.IndexerNetwork = indexeradmin.NewService(store, nil, nil)
	var response struct {
		Errors []struct{ Message string }
		Data   json.RawMessage
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{addIndexerNetworkProfile(id:"browser",expectedRevision:"`+store.Revision()+`",input:{flareSolverrUrl:"http://solver.example:8191/private-endpoint"}){revision challengeTimeoutSeconds profiles{id proxyConfigured flareSolverrConfigured}}}`, &response)
	if len(response.Errors) > 0 || !strings.Contains(string(response.Data), `"flareSolverrConfigured":true`) || strings.Contains(string(response.Data), "solver.example") || strings.Contains(string(response.Data), "private-endpoint") {
		t.Fatal("solver-only profile failed or leaked endpoint", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkTimeouts(expectedRevision:"`+store.Revision()+`",input:{challengeTimeoutSeconds:150}){challengeTimeoutSeconds requestTimeoutSeconds}}`, &response)
	if len(response.Errors) > 0 || !strings.Contains(string(response.Data), `"challengeTimeoutSeconds":150`) || store.Config().Indexers.Builtin.ChallengeTimeoutSeconds != 150 {
		t.Fatal("challenge timeout not saved", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkProfile(id:"browser",expectedRevision:"`+store.Revision()+`",input:{flareSolverrUrl:""}){profiles{id flareSolverrConfigured}}}`, &response)
	if len(response.Errors) > 0 || store.Config().Indexers.Builtin.NetworkProfiles["browser"].FlareSolverrURL == "" {
		t.Fatal("blank solver edit cleared saved URL", response.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation{updateIndexerNetworkProfile(id:"browser",expectedRevision:"`+store.Revision()+`",input:{clearFlareSolverr:true}){profiles{id flareSolverrConfigured}}}`, &response)
	if len(response.Errors) > 0 || store.Config().Indexers.Builtin.NetworkProfiles["browser"].FlareSolverrURL != "" {
		t.Fatal("explicit solver clear failed", response.Errors)
	}
}
