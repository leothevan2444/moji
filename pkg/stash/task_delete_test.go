package stash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestTaskDeletionSendsNativeFileAndGeneratedFlagsAndFencesConnection(t *testing.T) {
	cfg := Config{URL: "http://stash.test/graphql", APIKey: "fake-key"}
	c := NewClient(func() Config { return cfg })
	identity := c.IngestIdentity()
	calls := 0
	transport := ingestRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			OperationName string
			Variables     struct {
				Input       gql.SceneDestroyInput
				SceneFilter gql.SceneFilterType `json:"scene_filter"`
				Filter      gql.FindFilterType
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		body := `{"data":{"sceneDestroy":true}}`
		if request.OperationName == "RemoveIngestScene" {
			in := request.Variables.Input
			if in.ID != "12" || in.DeleteFile == nil || !*in.DeleteFile || in.DeleteGenerated == nil || !*in.DeleteGenerated {
				t.Fatalf("wrong native deletion flags: %+v", in)
			}
		} else {
			if request.OperationName != "IngestScenes" || request.Variables.SceneFilter.ID == nil || request.Variables.SceneFilter.ID.Value != 12 {
				t.Fatal("unscoped lookup")
			}
			body = `{"data":{"findScenes":{"scenes":[]}}}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	c.graphql = gql.NewClient(&http.Client{Transport: transport}, cfg.URL, nil).(*gql.Client)
	if scene, err := c.FindSceneForDeletion(context.Background(), identity, "12"); err != nil || scene != nil {
		t.Fatal("missing scene", err)
	}
	if err := c.DeleteSceneWithFiles(context.Background(), identity, "12"); err != nil {
		t.Fatal(err)
	}
	cfg.APIKey = "changed"
	if err := c.DeleteSceneWithFiles(context.Background(), identity, "12"); err == nil || calls != 2 {
		t.Fatal("deletion retargeted another connection")
	}
}
