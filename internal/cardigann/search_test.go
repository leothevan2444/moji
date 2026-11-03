package cardigann

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func queryInteger(value int) *int { return &value }

type fakeSession struct {
	requests     []Request
	respond      func(Request) (HTTPResponse, error)
	cookieOrigin string
	cookieHeader string
}

func TestStructuredModeUsesCapabilityNameAndTorznabTemplateType(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "modes: {search: [q]}", "modes: {search: [q], movie-search: [q, imdbid]}", 1)
	definition = strings.Replace(definition, "  rows: {selector: article}", `  inputs:
    t: '{{ .Query.Type }}'
    q: '{{ .Query.Keywords }}'
    limit: '{{ .Query.Limit }}'
    offset: '{{ .Query.Offset }}'
    imdb: '{{ .Query.IMDBIDShort }}'
  rows: {selector: article}`, 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(request Request) (HTTPResponse, error) {
		target, _ := url.Parse(request.URL)
		query := target.Query()
		if query.Get("t") != "movie" || query.Get("q") != "Example 2026" || query.Get("limit") != "50" || query.Get("offset") != "25" || query.Get("imdb") != "1234567" {
			t.Errorf("structured query lost semantics: %v", query)
		}
		return HTTPResponse{Status: 200, Body: []byte(`<article><h2>Example</h2><a href="/dl/1">get</a></article>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "https://example.invalid/", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execution.Search(context.Background(), Query{Mode: "movie", Keywords: "Example", Parameters: map[string]string{"Year": "2026", "IMDBID": "tt1234567"}, Limit: queryInteger(50), Offset: queryInteger(25)}); err != nil {
		t.Fatal(err)
	}
}

func (s *fakeSession) Do(ctx context.Context, r Request) (HTTPResponse, error) {
	if err := ctx.Err(); err != nil {
		return HTTPResponse{}, err
	}
	s.requests = append(s.requests, r)
	return s.respond(r)
}
func (s *fakeSession) SetCookieHeader(origin, header string) error {
	s.cookieOrigin, s.cookieHeader = origin, header
	return nil
}

func TestSearchExecutesAllPathsAndEscapesOnlyTemplateValues(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", `paths:
    - {path: 'search/{{ .Keywords }}'}
    - {path: 'search/{{ .Keywords }}?page=2'}`, 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		if r.Method != http.MethodGet {
			return HTTPResponse{}, fmt.Errorf("wrong method")
		}
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>ABCD-123</h2><a href="/dl/123">download</a></article>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "https://example.invalid/", nil, time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{Keywords: "a/b & c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(session.requests) != 2 || session.requests[0].URL != "https://example.invalid/search/a%2Fb%20%26%20c" || !strings.HasSuffix(session.requests[1].URL, "?page=2") {
		t.Fatalf("bad request sequence: %#v", session.requests)
	}
	if len(result.Results) != 1 || len(result.Issues) != 0 {
		t.Fatalf("deduplication or parsing failed: %#v", result)
	}
	release := result.Results[0]
	if release.Title != "ABCD-123" || release.DownloadURL != "https://example.invalid/dl/123" || release.Size != 1<<30 || release.Seeders == nil || *release.Seeders != 0 {
		t.Fatalf("incorrect result: %#v", release)
	}
}

func TestSearchJSONRetainsLargeIDsAndResultDependencies(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", "paths: [{path: /api, response: {type: json}}]", 1)
	definition = strings.Replace(definition, "rows: {selector: article}", "rows: {selector: data}", 1)
	definition = strings.Replace(definition, "title: {selector: h2}", "title: {selector: name}", 1)
	definition = strings.Replace(definition, "download: {selector: a, attribute: href}", "_id: {selector: id}\n    download: {text: '/dl/{{ .Result._id }}'}\n    date: {selector: date, filters: [{name: dateparse, args: 'yyyy-MM-dd HH:mm:ss zzz'}]}", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`{"data":[{"id":9007199254740993,"name":"ABCD-123","date":"2026-10-30 10:11:12 +08:00"}]}`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{Keywords: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 || result.Results[0].DownloadURL != "https://example.invalid/dl/9007199254740993" || result.Results[0].PublishedAt == nil {
		t.Fatalf("incorrect JSON projection: %#v", result)
	}
}

func TestSearchXMLAndParsingFailuresAreDistinguishedFromEmpty(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", "paths: [{path: /api, response: {type: xml}}]", 1)
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	body := `<feed><article><h2>ABCD-123</h2><a href="/download?a=1&amp;b=2"/></article></feed>`
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) { return HTTPResponse{Status: 200, Body: []byte(body)}, nil }}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result, err := execution.Search(context.Background(), Query{})
	if err != nil || len(result.Results) != 1 || !strings.HasSuffix(result.Results[0].DownloadURL, "a=1&b=2") {
		t.Fatalf("XML: %#v %v", result, err)
	}
	body = `<feed><article><h2>broken`
	if _, err = execution.Search(context.Background(), Query{}); err == nil {
		t.Fatal("malformed XML reported as empty")
	}
	body = `<feed/>`
	result, err = execution.Search(context.Background(), Query{})
	if err != nil || len(result.Results) != 0 || len(result.Issues) != 0 {
		t.Fatalf("normal empty: %#v %v", result, err)
	}
}

func TestSearchBudgetAndUnknownMode(t *testing.T) {
	plan, err := Load([]byte(minimalDefinition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) { return HTTPResponse{Status: 200}, nil }}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = execution.Search(context.Background(), Query{Mode: "execute"}); err == nil || len(session.requests) != 0 {
		t.Fatal("unsupported mode contacted a site")
	}
	execution.maxRequests = 0
	if _, err = execution.Search(context.Background(), Query{}); err == nil {
		t.Fatal("request budget ignored")
	}
}

func TestSearchInheritedInputsPreserveBothValuesAndSkipWhitespace(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", `paths:
    - path: /search
      method: `+method+`
      inputs: {q: local, localOnly: path}
    - path: /isolated
      method: `+method+`
      inheritinputs: false
      inputs: {q: isolated}
  inputs: {q: inherited, inheritedOnly: global, blank: '   '}`, 1)
			plan, e := Load([]byte(definition))
			if e != nil {
				t.Fatal(e)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				target, _ := url.Parse(r.URL)
				values := target.Query()
				if r.Method == http.MethodPost {
					values, _ = url.ParseQuery(string(r.Body))
				}
				if target.Path == "/search" {
					if strings.Join(values["q"], ",") != "inherited,local" || values.Get("inheritedOnly") != "global" || values.Get("localOnly") != "path" {
						t.Errorf("inherited values were replaced: %v", values)
					}
				} else if values.Get("q") != "isolated" || values.Has("inheritedOnly") {
					t.Errorf("disabled inheritance leaked shared inputs: %v", values)
				}
				if values.Has("blank") {
					t.Error("whitespace input was sent despite allowEmptyInputs=false")
				}
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href="/dl/1">get</a></article>`)}, nil
			}}
			execution, e := plan.NewExecution(session, "https://example.invalid/", nil, time.Now())
			if e != nil {
				t.Fatal(e)
			}
			if _, e = execution.Search(context.Background(), Query{Keywords: "Example", Limit: queryInteger(20)}); e != nil {
				t.Fatal(e)
			}
			if len(session.requests) != 2 {
				t.Fatal("declared input scopes not both executed")
			}
		})
	}
}
func TestSearchDeduplicatesGetButExecutesEachDeclaredPostPath(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", `paths:
    - {path: /search, method: `+method+`}
    - {path: /search, method: `+method+`}`, 1)
			plan, e := Load([]byte(definition))
			if e != nil {
				t.Fatal(e)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href="/dl/1">get</a></article>`)}, nil
			}}
			execution, e := plan.NewExecution(session, "https://example.invalid/", nil, time.Now())
			if e != nil {
				t.Fatal(e)
			}
			if _, e = execution.Search(context.Background(), Query{Keywords: "Example", Limit: queryInteger(20)}); e != nil {
				t.Fatal(e)
			}
			want := 1
			if method == "post" {
				want = 2
			}
			if len(session.requests) != want {
				t.Fatalf("request count=%d want %d", len(session.requests), want)
			}
		})
	}
}

func TestSearchRawInputsFollowTwoStageEncodingAndKeepRepeatedValues(t *testing.T) {
	definition := strings.Replace(minimalDefinition, "paths: [{path: /}]", `paths:
    - path: /search
      inputs: {$raw: '&flag&cat=local&=ignored'}
  inputs:
    $raw: 'q={{ .Keywords }}&cat=shared&equals=a=b'`, 1)
	plan, e := Load([]byte(definition))
	if e != nil {
		t.Fatal(e)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		target, _ := url.Parse(r.URL)
		values := target.Query()
		if values.Get("q") != url.QueryEscape("A/B & C") || strings.Join(values["cat"], ",") != "shared,local" || values.Get("equals") != "a=b" || !values.Has("flag") || values.Has("") {
			t.Errorf("raw search pairs lost upstream encoding/scope semantics: %v", values)
		}
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href="/dl/1">get</a></article>`)}, nil
	}}
	execution, e := plan.NewExecution(session, "https://example.invalid/", nil, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = execution.Search(context.Background(), Query{Keywords: "A/B & C", Limit: queryInteger(20)}); e != nil {
		t.Fatal(e)
	}
}
func TestSearchUsesDeclaredLegacyEncodingForQueryAndFormValues(t *testing.T) {
	for _, sample := range []struct{ name, keywords, expected string }{
		{"representable", "Тест", "q=%D2%E5%F1%F2"},
		{"best-fit-and-replacement", "ē 汉😀", "q=e+%3F%3F%3F"},
	} {
		for _, method := range []string{"get", "post"} {
			t.Run(sample.name+"/"+method, func(t *testing.T) {
				definition := strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: windows-1251", 1)
				definition = strings.Replace(definition, "paths: [{path: /}]", `paths: [{path: /search, method: `+method+`}]
  inputs: {q: '{{ .Keywords }}'}`, 1)
				plan, e := Load([]byte(definition))
				if e != nil {
					t.Fatal(e)
				}
				session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
					target, _ := url.Parse(r.URL)
					encoded := target.RawQuery
					if r.Method == http.MethodPost {
						encoded = string(r.Body)
					}
					if encoded != sample.expected {
						t.Errorf("declared Windows-1251 was ignored: %q", encoded)
					}
					return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<article><h2>Example</h2><a href="/dl/1">get</a></article>`)}, nil
				}}
				execution, e := plan.NewExecution(session, "https://example.invalid/", nil, time.Now())
				if e != nil {
					t.Fatal(e)
				}
				if _, e = execution.Search(context.Background(), Query{Keywords: sample.keywords, Limit: queryInteger(20)}); e != nil {
					t.Fatal(e)
				}
			})
		}
	}
}
