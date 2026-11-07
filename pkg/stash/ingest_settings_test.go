package stash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestNativeIdentifySettingsKeepEmptyGendersAndSourceOverridesOnWire(t *testing.T) {
	transport := ingestRoundTripper(func(r *http.Request) (*http.Response, error) {
		var request struct {
			OperationName string                     `json:"operationName"`
			Variables     map[string]json.RawMessage `json:"variables"`
		}
		if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
			t.Fatal(e)
		}
		body := ""
		switch request.OperationName {
		case "IngestIdentifyCapabilities":
			body = `{"data":{"__type":{"fields":[{"name":"performerGenders"}]}}}`
		case "IngestDefaultsNative":
			body = `{"data":{"configuration":{"defaults":{"identify":{"options":{"performerGenders":[],"setCoverImage":true,"skipSingleNamePerformers":true},"sources":[{"source":{"stash_box_endpoint":"https://box.test/graphql"},"options":{"performerGenders":["MALE"],"setCoverImage":false,"fieldOptions":[{"field":"performers","strategy":"OVERWRITE","createMissing":false}]}}]}}}}}`
		case "MetadataIdentify":
			var input struct {
				Options struct {
					Genders []gql.GenderEnum `json:"performerGenders"`
				}
				Sources []struct {
					Options struct {
						Cover  *bool                           `json:"setCoverImage"`
						Fields []gql.IdentifyFieldOptionsInput `json:"fieldOptions"`
					}
				}
			}
			if e := json.Unmarshal(request.Variables["input"], &input); e != nil {
				t.Fatal(e)
			}
			if input.Options.Genders == nil || len(input.Options.Genders) != 0 || input.Sources[0].Options.Cover == nil || *input.Sources[0].Options.Cover || input.Sources[0].Options.Fields[0].CreateMissing == nil || *input.Sources[0].Options.Fields[0].CreateMissing {
				t.Fatal("native option null/empty/false semantics lost")
			}
			body = `{"data":{"metadataIdentify":"job"}}`
		default:
			t.Fatalf("unexpected API operation %s", request.OperationName)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client := NewClient(nil)
	client.graphql = gql.NewClient(&http.Client{Transport: transport}, "http://stash/graphql", nil).(*gql.Client)
	input, e := client.IngestIdentifyDefaults(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if input.Options.PerformerGenders == nil {
		t.Fatal("empty default gender list was discarded")
	}
	input.SceneIDs = []string{"one"}
	if _, e = client.MetadataIdentify(context.Background(), *input); e != nil {
		t.Fatal(e)
	}
}
func TestSceneSaveRetainsExplicitEmptyReplacementLists(t *testing.T) {
	in := ingestSceneInput{gql.SceneUpdateInput{ID: "one", PerformerIds: []string{}, TagIds: []string{}, StashIds: []*gql.StashIDInput{}}}
	raw, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	var values map[string]json.RawMessage
	_ = json.Unmarshal(raw, &values)
	for _, field := range []string{"performer_ids", "tag_ids", "stash_ids"} {
		if string(values[field]) != "[]" {
			t.Fatalf("%s lost explicit empty list", field)
		}
	}
	if _, exists := values["urls"]; exists {
		t.Fatal("omitted URL update became a destructive empty list")
	}
}
func TestCoverVerificationKeepsCredentialsAtConfiguredStashOnly(t *testing.T) {
	leaked := false
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer sink.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/base/scene/one/screenshot" || r.Header.Get("ApiKey") != "test-secret" {
			t.Error("wrong configured cover request")
		}
		http.Redirect(w, r, sink.URL, http.StatusFound)
	}))
	defer server.Close()
	client := NewClient(func() Config { return Config{URL: server.URL + "/base/graphql", APIKey: "test-secret"} })
	if _, e := client.IngestCoverDigest(context.Background(), client.IngestIdentity(), "one"); e == nil || leaked {
		t.Fatal("cover redirect leaked Stash credentials")
	}
}
