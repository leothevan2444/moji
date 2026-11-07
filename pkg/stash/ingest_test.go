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

func TestNativeCodeScrapeAndSaveUsesStashAndFencesConnectionChanges(t *testing.T) {
	cfg := Config{URL: "http://stash.test/graphql", APIKey: "private-key"}
	client := NewClient(func() Config { return cfg })
	identity := client.IngestIdentity()
	calls := 0
	transport := ingestRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			OperationName string `json:"operationName"`
			Variables     struct {
				Input  json.RawMessage            `json:"input"`
				Source graphql.ScraperSourceInput `json:"source"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		body := ""
		switch request.OperationName {
		case "IngestIdentifyCapabilities":
			body = `{"data":{"__type":{"fields":[{"name":"performerGenders"}]}}}`
		case "ScrapeIngestScene":
			var input graphql.ScrapeSingleSceneInput
			if json.Unmarshal(request.Variables.Input, &input) != nil || input.Query == nil || *input.Query != "DEMO-123" || input.SceneID != nil || request.Variables.Source.StashBoxEndpoint == nil {
				t.Fatal("keyword query was not sent through Stash")
			}
			body = `{"data":{"scrapeSingleScene":[{"code":"DEMO-123","remote_site_id":"remote-one"}]}}`
		case "SaveIngestScene":
			var input graphql.SceneUpdateInput
			if json.Unmarshal(request.Variables.Input, &input) != nil || input.ID != "scene-one" || input.Code == nil || *input.Code != "DEMO-123" {
				t.Fatal("saved the wrong scene or code")
			}
			body = `{"data":{"sceneUpdate":{"id":"scene-one"}}}`
		default:
			t.Fatalf("unexpected operation %s", request.OperationName)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	client.graphql = graphql.NewClient(&http.Client{Transport: transport}, cfg.URL, nil).(*graphql.Client)
	code, endpoint := "DEMO-123", "https://box.test/graphql"
	if results, err := client.ScrapeIngestScene(context.Background(), graphql.ScraperSourceInput{StashBoxEndpoint: &endpoint}, graphql.ScrapeSingleSceneInput{Query: &code}, true); err != nil || len(results) != 1 {
		t.Fatalf("scrape: %v", err)
	}
	input := graphql.SceneUpdateInput{ID: "scene-one", Code: &code}
	if err := client.SaveIngestScene(context.Background(), identity, input); err != nil {
		t.Fatal(err)
	}
	cfg.APIKey = "another-account"
	if err := client.SaveIngestScene(context.Background(), identity, input); err == nil || calls != 3 {
		t.Fatal("old plan was sent to another account")
	}
}

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
		case "IngestIdentifyCapabilities":
			body = `{"data":{"__type":{"fields":[]}}}`
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
	if len(calls) != 5 {
		t.Fatalf("unexpected API calls: %v", calls)
	}
}
