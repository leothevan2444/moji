package stashsync

// Explicit integration test against TWO disposable Stash instances. Never pass
// production URLs: both must expose only the /moji-parity synthetic media root.
import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	stashclient "github.com/leothevan2444/moji/pkg/stash"
	gql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func parityPost(t *testing.T, endpoint, q string, vars map[string]any) map[string]json.RawMessage {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"query": q, "variables": vars})
	req, e := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Content-Type", "application/json")
	response, e := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	var out struct {
		Data   map[string]json.RawMessage
		Errors []struct{ Message string }
	}
	if e = json.NewDecoder(response.Body).Decode(&out); e != nil {
		t.Fatal(e)
	}
	if len(out.Errors) > 0 {
		t.Fatalf("isolated Stash API: %+v", out.Errors)
	}
	return out.Data
}
func parityImage(c color.NRGBA) []byte {
	im := image.NewNRGBA(image.Rect(0, 0, 16, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 16; x++ {
			im.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	return b.Bytes()
}
func waitParityJob(t *testing.T, c *stashclient.Client, id string) {
	t.Helper()
	end := time.Now().Add(30 * time.Second)
	for time.Now().Before(end) {
		job, e := c.FindJob(context.Background(), id)
		if e != nil {
			t.Fatal(e)
		}
		if job != nil {
			status := string(job.Status)
			if status == "FINISHED" {
				return
			}
			if status == "FAILED" || status == "CANCELLED" {
				t.Fatalf("job failed: %+v", job)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("isolated Stash job did not finish")
}
func TestOptInStashNativeIdentifyParity(t *testing.T) {
	endpoints := strings.Split(os.Getenv("MOJI_STASH_PARITY_URLS"), ",")
	if len(endpoints) != 2 {
		t.Skip("set MOJI_STASH_PARITY_URLS to two disposable local Stash endpoints")
	}
	for _, endpoint := range endpoints {
		u, e := url.Parse(endpoint)
		if e != nil || u.Hostname() != "127.0.0.1" || u.Path != "/graphql" {
			t.Fatal("parity accepts disposable loopback endpoints only")
		}
		port, _ := strconv.Atoi(u.Port())
		if port < 20000 {
			t.Fatal("parity requires temporary high ports")
		}
		roots := parityPost(t, endpoint, `query{configuration{general{stashes{path}}}}`, nil)
		var conf struct {
			General struct{ Stashes []struct{ Path string } }
		}
		if e = json.Unmarshal(roots["configuration"], &conf); e != nil || len(conf.General.Stashes) != 1 || conf.General.Stashes[0].Path != "/moji-parity" {
			t.Fatal("refusing instance without synthetic-only media root")
		}
	}
	runID := uint32(time.Now().UnixNano())
	runName := fmt.Sprintf("%08x", runID)
	cover := parityImage(color.NRGBA{R: 210, G: 40, B: 60, A: 255})
	before := parityImage(color.NRGBA{R: 20, G: 70, B: 210, A: 255})
	var mu sync.RWMutex
	var candidate map[string]any
	multiple := false
	var box *httptest.Server
	box = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cover.png" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(cover)
			return
		}
		var request struct {
			OperationName string                     `json:"operationName"`
			Variables     map[string]json.RawMessage `json:"variables"`
		}
		if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
			http.Error(w, "bad request", 400)
			return
		}
		mu.RLock()
		scene := candidate
		multi := multiple
		mu.RUnlock()
		filtered := map[string]any{}
		for k, v := range scene {
			filtered[k] = v
		}
		if studio, ok := scene["studio"].(map[string]any); ok {
			copy := map[string]any{}
			for k, v := range studio {
				copy[k] = v
			}
			if parent, ok := studio["parent"].(map[string]any); ok {
				copy["parent"] = map[string]any{"id": parent["id"], "name": parent["name"]}
			}
			filtered["studio"] = copy
		}
		results := []map[string]any{filtered}
		if multi {
			copy := map[string]any{}
			for k, v := range filtered {
				copy[k] = v
			}
			copy["id"] = "ffffffff-ffff-4fff-8fff-ffffffffffff"
			results = append(results, copy)
		}
		data := map[string]any{}
		switch request.OperationName {
		case "FindScenesBySceneFingerprints":
			var fps []json.RawMessage
			_ = json.Unmarshal(request.Variables["fingerprints"], &fps)
			batch := make([][]map[string]any, len(fps))
			for i := range batch {
				batch[i] = []map[string]any{}
				if r.Header.Get("ApiKey") == "NATIVE" {
					batch[i] = results
				}
			}
			data["findScenesBySceneFingerprints"] = batch
		case "SearchScene":
			data["searchScene"] = results
		case "FindScene":
			data["findScene"] = filtered
		case "FindStudio":
			data["findStudio"] = scene["studio"].(map[string]any)["parent"]
		default:
			http.Error(w, "unexpected fake StashBox operation: "+request.OperationName, 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer box.Close()
	clients := []*stashclient.Client{}
	services := []*Service{}
	sceneIDs := []string{}
	baselineStudios := []string{}
	baselineActors := []string{}
	baselineTags := []string{}
	markers := []string{}
	for i, endpoint := range endpoints {
		c := stashclient.NewClient(func() stashclient.Config { return stashclient.Config{URL: endpoint} })
		clients = append(clients, c)
		s, _ := NewService(c, nil)
		services = append(services, s)
		key := "NATIVE"
		if i == 1 {
			key = "FALLBACK"
		}
		parityPost(t, endpoint, `mutation($input:ConfigGeneralInput!){configureGeneral(input:$input){stashBoxes{endpoint}}}`, map[string]any{"input": map[string]any{"stashBoxes": []any{map[string]any{"endpoint": box.URL + "/graphql", "api_key": key, "name": "Parity source"}}}})
		job, e := c.MetadataScan(context.Background(), gql.ScanMetadataInput{Paths: []string{"/moji-parity"}, ScanGenerateCovers: codeBool(false), ScanGeneratePreviews: codeBool(false), ScanGenerateSprites: codeBool(false), ScanGeneratePhashes: codeBool(false)})
		if e != nil {
			t.Fatal(e)
		}
		waitParityJob(t, c, job)
		scenes, e := s.FindIngestScenes(context.Background(), "/moji-parity", nil)
		if e != nil || len(scenes) != 1 {
			t.Fatalf("synthetic scene missing: %v %+v", e, scenes)
		}
		sceneIDs = append(sceneIDs, scenes[0].ID)
		studio, e := c.CreateIngestStudio(context.Background(), c.IngestIdentity(), gql.StudioCreateInput{Name: "Baseline Studio " + runName})
		if e != nil {
			t.Fatal(e)
		}
		baselineStudios = append(baselineStudios, studio)
		actor, e := c.CreateIngestPerformer(context.Background(), c.IngestIdentity(), gql.PerformerCreateInput{Name: "Baseline Actor " + runName})
		if e != nil {
			t.Fatal(e)
		}
		baselineActors = append(baselineActors, actor)
		tag, e := c.CreateIngestTag(context.Background(), c.IngestIdentity(), gql.TagCreateInput{Name: "Baseline Tag " + runName})
		if e != nil {
			t.Fatal(e)
		}
		baselineTags = append(baselineTags, tag)
		marker, e := c.CreateIngestTag(context.Background(), c.IngestIdentity(), gql.TagCreateInput{Name: "Skipped by Identify " + runName})
		if e != nil {
			t.Fatal(e)
		}
		markers = append(markers, marker)
	}
	baseRules := func(create bool) []map[string]any {
		rules := []map[string]any{}
		for _, field := range []string{"title", "code", "date", "details", "director", "url", "studio", "performers", "tags", "stash_ids"} {
			rule := map[string]any{"field": field, "strategy": "OVERWRITE"}
			if field == "studio" || field == "performers" || field == "tags" {
				rule["createMissing"] = create
			}
			rules = append(rules, rule)
		}
		return rules
	}
	cases := []struct {
		name   string
		change func(map[string]any, map[string]any, map[string]any)
		create bool
		multi  bool
	}{
		{"all fields and entity creation", nil, true, false},
		{"source overrides and cover disabled", func(d, s, m map[string]any) {
			s["setCoverImage"] = false
			s["fieldOptions"] = []any{map[string]any{"field": "title", "strategy": "IGNORE"}, map[string]any{"field": "studio", "strategy": "IGNORE"}, map[string]any{"field": "performers", "strategy": "IGNORE"}, map[string]any{"field": "tags", "strategy": "IGNORE"}}
			s["setOrganized"] = false
		}, true, false},
		{"gender allowlist beats legacy flag", func(d, s, m map[string]any) {
			d["includeMalePerformers"] = false
			s["includeMalePerformers"] = true
			s["performerGenders"] = []string{"FEMALE"}
		}, true, false},
		{"empty gender allowlist", func(d, s, m map[string]any) {
			d["performerGenders"] = []string{"FEMALE"}
			s["performerGenders"] = []string{}
		}, true, false},
		{"create disabled preserves unresolved relationships", nil, false, false},
		{"single name skip marker with ignored tags", func(d, s, m map[string]any) {
			s["skipSingleNamePerformers"] = true
			s["fieldOptions"] = []any{map[string]any{"field": "tags", "strategy": "IGNORE"}}
		}, true, false},
		{"ignore remote identity field", func(d, s, m map[string]any) {
			s["fieldOptions"] = []any{map[string]any{"field": "stash_ids", "strategy": "IGNORE"}}
		}, true, false},
		{"empty scalars and invalid date", func(d, s, m map[string]any) {
			m["title"] = ""
			m["details"] = ""
			m["date"] = "invalid"
			s["setCoverImage"] = false
		}, true, false},
		{"legacy male filter", func(d, s, m map[string]any) { d["includeMalePerformers"] = false }, true, false},
		{"source field disables automatic creation", func(d, s, m map[string]any) {
			s["fieldOptions"] = []any{map[string]any{"field": "studio", "strategy": "OVERWRITE"}, map[string]any{"field": "performers", "strategy": "OVERWRITE"}, map[string]any{"field": "tags", "strategy": "OVERWRITE"}}
		}, true, false},
		{"URL overwrite subset preserves native URLs", func(d, s, m map[string]any) {
			m["urls"] = []any{map[string]any{"url": "https://example.test/old", "type": "STUDIO"}}
		}, true, false},
		{"new studio updates matching parent", func(d, s, m map[string]any) {
			m["studio"].(map[string]any)["parent"] = map[string]any{"id": "eeeeeeee-eeee-4eee-8eee-" + fmt.Sprintf("%012d", runID), "name": "Baseline Studio " + runName, "aliases": []string{"Parent Alias " + runName}, "urls": []any{}, "images": []any{}}
		}, true, false},

		{"multiple matches skip marker", nil, true, true},
	}
	for n, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actor := func(id, name string, gender any) map[string]any {
				return map[string]any{"id": id, "name": name, "gender": gender, "aliases": []string{"Alias " + name}, "urls": []any{map[string]any{"url": "https://example.test/actor", "type": "OTHER"}}, "images": []any{map[string]any{"id": "image", "url": box.URL + "/cover.png", "width": 16, "height": 12}}, "height": 165, "birth_date": "1990-01-01", "career_start_year": 2010, "country": "JP", "tattoos": []any{}, "piercings": []any{}, "merged_ids": []string{}}
			}
			remote := func(k int) string { return fmt.Sprintf("%08x-0000-4000-8000-%012d", runID, 100*n+k) }
			payload := map[string]any{"id": remote(1), "code": "DEMO-123", "title": "Scraped title", "details": "Scraped details", "director": "Director", "date": "2026-01-01", "duration": 3, "urls": []any{map[string]any{"url": "https://example.test/scene", "type": "STUDIO"}}, "images": []any{map[string]any{"id": "cover", "url": box.URL + "/cover.png", "width": 16, "height": 12}}, "studio": map[string]any{"id": remote(2), "name": fmt.Sprintf("Studio %s %d", runName, n), "aliases": []string{fmt.Sprintf("Studio Alias %s %d", runName, n)}, "urls": []any{}, "images": []any{}}, "tags": []any{map[string]any{"id": remote(3), "name": fmt.Sprintf("Tag %s %d", runName, n), "aliases": []string{}, "description": "Tag info"}}, "performers": []any{map[string]any{"performer": actor(remote(4), fmt.Sprintf("Female Actor %s %d", runName, n), "FEMALE")}, map[string]any{"performer": actor(remote(5), fmt.Sprintf("Male Actor %s %d", runName, n), "MALE")}, map[string]any{"performer": actor(remote(6), fmt.Sprintf("Single%s%d", runName, n), "FEMALE")}, map[string]any{"performer": actor(remote(7), fmt.Sprintf("Unknown Gender %s %d", runName, n), nil)}}, "fingerprints": []any{}}
			defaults := map[string]any{"fieldOptions": baseRules(tc.create), "setOrganized": true, "skipSingleNamePerformers": false, "skipMultipleMatches": true}
			source := map[string]any{}
			if tc.change != nil {
				tc.change(defaults, source, payload)
			}
			mu.Lock()
			candidate = payload
			multiple = tc.multi
			mu.Unlock()
			for i, endpoint := range endpoints {
				d := map[string]any{}
				for k, v := range defaults {
					d[k] = v
				}
				d["skipMultipleMatchTag"] = markers[i]
				d["skipSingleNamePerformerTag"] = markers[i]
				src := map[string]any{}
				for k, v := range source {
					src[k] = v
				}
				input := map[string]any{"identify": map[string]any{"sources": []any{map[string]any{"source": map[string]any{"stash_box_endpoint": box.URL + "/graphql"}, "options": src}}, "options": d}}
				parityPost(t, endpoint, `mutation($input:ConfigDefaultSettingsInput!){configureDefaults(input:$input){identify{options{setCoverImage}}}}`, map[string]any{"input": input})
				blank := ""
				title := "Before"
				date := "2001-01-01"
				coverData := "data:image/png;base64," + base64.StdEncoding.EncodeToString(before)
				reset := gql.SceneUpdateInput{ID: sceneIDs[i], Title: &title, Code: &blank, Date: &date, Details: &title, Director: &title, Organized: codeBool(true), Urls: []string{"https://example.test/old", "https://example.test/extra"}, StudioID: &baselineStudios[i], PerformerIds: []string{baselineActors[i]}, TagIds: []string{baselineTags[i]}, StashIds: []*gql.StashIDInput{}, CoverImage: &coverData}
				if e := clients[i].SaveIngestScene(context.Background(), clients[i].IngestIdentity(), reset); e != nil {
					t.Fatal(e)
				}
			}
			options, e := clients[0].IngestIdentifyDefaults(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			options.SceneIDs = []string{sceneIDs[0]}
			job, e := clients[0].MetadataIdentify(context.Background(), *options)
			if e != nil {
				t.Fatal(e)
			}
			waitParityJob(t, clients[0], job)
			plan, e := services[1].PrepareCodeIdentify(context.Background(), "DEMO-123", "/moji-parity", []string{sceneIDs[1]})
			if e != nil {
				t.Fatal(e)
			}
			if plan == nil {
				t.Fatal("fallback did not prepare")
			}
			encoded, e := json.Marshal(plan)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(t.TempDir()+"/intent.json", encoded, 0600); e != nil {
				t.Fatal(e)
			}
			if e = services[1].ApplyCodeIdentify(context.Background(), "/moji-parity", []string{sceneIDs[1]}, plan); e != nil {
				t.Fatal(e)
			}
			query := `query($id:ID!){findScene(id:$id){title code date details director urls organized studio{name parent_studio{name aliases stash_ids{endpoint stash_id}}} performers{name gender height_cm birthdate career_start career_end alias_list urls image_path} tags{name} stash_ids{endpoint stash_id}}}`
			results := []any{}
			for i, endpoint := range endpoints {
				data := parityPost(t, endpoint, query, map[string]any{"id": sceneIDs[i]})
				var value map[string]any
				if e = json.Unmarshal(data["findScene"], &value); e != nil {
					t.Fatal(e)
				}
				if performers, ok := value["performers"].([]any); ok {
					for _, performer := range performers {
						p := performer.(map[string]any)
						if imageURL, ok := p["image_path"].(string); ok && imageURL != "" {
							u, _ := url.Parse(imageURL)
							base, _ := url.Parse(endpoint)
							if u.Host != base.Host {
								t.Fatal("unexpected actor image origin")
							}
							req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, imageURL, nil)
							r, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
							if err != nil {
								t.Fatal(err)
							}
							raw, _ := io.ReadAll(io.LimitReader(r.Body, 8*1024*1024))
							r.Body.Close()
							if r.StatusCode == http.StatusOK {
								p["image_digest"] = bytesDigest(raw)
							}
						}
						delete(p, "image_path")
					}
				}
				results = append(results, value)
			}
			if !reflect.DeepEqual(results[0], results[1]) {
				native, _ := json.Marshal(results[0])
				fallback, _ := json.Marshal(results[1])
				t.Fatalf("native/fallback differ:\nnative %s\nfallback %s", native, fallback)
			}
			a, e := clients[0].IngestCoverDigest(context.Background(), clients[0].IngestIdentity(), sceneIDs[0])
			if e != nil {
				t.Fatal(e)
			}
			b, e := clients[1].IngestCoverDigest(context.Background(), clients[1].IngestIdentity(), sceneIDs[1])
			if e != nil || a != b {
				t.Fatal("cover behavior differs", e)
			}
		})
	}
}
