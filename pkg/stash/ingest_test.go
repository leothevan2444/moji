package stash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	graphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type ingestRoundTripper func(*http.Request) (*http.Response, error)

func (f ingestRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRemoveIngestSceneRetainsMediaAndRemovesGeneratedAssets(t *testing.T) {
	for _, success := range []bool{true, false} {
		transport := ingestRoundTripper(func(r *http.Request) (*http.Response, error) {
			var request struct {
				OperationName string
				Variables     struct{ Input graphql.SceneDestroyInput }
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			input := request.Variables.Input
			if request.OperationName != "RemoveIngestScene" || input.ID != "4525" || input.DeleteFile == nil || *input.DeleteFile || input.DeleteGenerated == nil || !*input.DeleteGenerated {
				t.Fatalf("unsafe removal request: %+v", request)
			}
			body := "{\"data\":{\"sceneDestroy\":false}}"
			if success {
				body = "{\"data\":{\"sceneDestroy\":true}}"
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		client := NewClient(nil)
		client.graphql = graphql.NewClient(&http.Client{Transport: transport}, "http://stash/graphql", nil).(*graphql.Client)
		if err := client.RemoveIngestScene(context.Background(), "4525"); (err == nil) != success {
			t.Fatalf("removal result ignored: %v", err)
		}
	}
}

func TestStashIngestGraphQLRequests(t *testing.T) {
	calls := []string{}
	transport := ingestRoundTripper(func(r *http.Request) (*http.Response, error) {
		var request struct {
			OperationName string                     `json:"operationName"`
			Variables     map[string]json.RawMessage `json:"variables"`
			Query         string                     `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, request.OperationName)
		body := ""
		switch request.OperationName {
		case "IngestDefaults":
			body = `{"data":{"configuration":{"defaults":{"identify":{"sources":[{"source":{"scraper_id":"jav"},"options":{"setCoverImage":true,"fieldOptions":[{"field":"title","strategy":"OVERWRITE","createMissing":true}]}}],"options":{"skipMultipleMatches":true,"setOrganized":true}}}}}}`
		case "MetadataIdentify":
			var input graphql.IdentifyMetadataInput
			if err := json.Unmarshal(request.Variables["input"], &input); err != nil {
				t.Fatal(err)
			}
			if len(input.Sources) != 1 || *input.Sources[0].Source.ScraperID != "jav" || input.Sources[0].Options.FieldOptions[0].Strategy != graphql.IdentifyFieldStrategyOverwrite || !*input.Options.SkipMultipleMatches || input.SceneIDs[0] != "1" {
				t.Fatalf("Identify settings not preserved: %+v", input)
			}
			body = `{"data":{"metadataIdentify":"identify-job"}}`
		case "MetadataGenerate":
			body = `{"data":{"metadataGenerate":"generate-job"}}`
		case "IngestScenes":
			if !strings.Contains(string(request.Variables["filter"]), `"per_page":-1`) {
				t.Fatalf("scene lookup must not truncate at default page size: %s", request.Variables["filter"])
			}
			body = `{"data":{"findScenes":{"scenes":[{"id":"1","title":"Film","code":"ABCD-123","date":"2026-01-01","studio":{"id":"2"},"performers":[{"id":"3"}],"files":[{"path":"/library/film.mp4"}]}]}}}`
		default:
			t.Fatalf("unexpected Stash API operation: %s", request.OperationName)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client := NewClient(nil)
	client.graphql = graphql.NewClient(&http.Client{Transport: transport}, "http://stash/graphql", nil).(*graphql.Client)
	input, err := client.IngestIdentifyDefaults(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	input.SceneIDs = []string{"1"}
	if id, err := client.MetadataIdentify(context.Background(), *input); err != nil || id != "identify-job" {
		t.Fatalf("identify: %s %v", id, err)
	}
	if id, err := client.MetadataGenerate(context.Background(), graphql.GenerateMetadataInput{SceneIDs: []string{"1"}}); err != nil || id != "generate-job" {
		t.Fatalf("generate: %s %v", id, err)
	}
	scenes, err := client.FindIngestScenes(context.Background(), "/library", nil)
	if err != nil || len(scenes) != 1 {
		t.Fatalf("lookup: %+v %v", scenes, err)
	}
	if len(calls) != 4 {
		t.Fatalf("unexpected API calls: %v", calls)
	}
}
