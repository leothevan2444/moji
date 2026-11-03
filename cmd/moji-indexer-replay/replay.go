package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexercategory"
)

type scenario struct {
	Definition          string                      `json:"definition"`
	Query               cardigann.Query             `json:"query"`
	Settings            map[string]cardigann.Scalar `json:"settings"`
	Responses           []responseFixture           `json:"responses"`
	Authenticate        bool                        `json:"authenticate"`
	ChallengeAnswer     *string                     `json:"challengeAnswer"`
	Fetch               bool                        `json:"fetch"`
	Now                 time.Time                   `json:"now"`
	RelativeDateFilters []relativeDateFilter        `json:"relativeDateFilters,omitempty"`
}
type responseFixture struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Status  int         `json:"status"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body"`
}
type requestRecord struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Body    []byte            `json:"body"`
	Headers map[string]string `json:"headers"`
}
type resultRecord struct {
	Title       string     `json:"title"`
	DownloadURL string     `json:"downloadUrl"`
	MagnetURI   string     `json:"magnetUri"`
	InfoHash    string     `json:"infoHash"`
	DetailsURL  string     `json:"detailsUrl"`
	Description string     `json:"description"`
	PosterURL   string     `json:"posterUrl"`
	Size        int64      `json:"size"`
	Seeders     *int       `json:"seeders"`
	Leechers    *int       `json:"leechers"`
	Categories  []int      `json:"categories"`
	PublishedAt *time.Time `json:"publishedAt"`
}
type artifactRecord struct {
	SHA256 string `json:"sha256"`
	Magnet string `json:"magnet"`
}
type replayReport struct {
	Engine        string               `json:"engine"`
	Requests      []requestRecord      `json:"requests"`
	Results       []resultRecord       `json:"results"`
	Artifact      *artifactRecord      `json:"artifact,omitempty"`
	Challenge     *challengeRecord     `json:"challenge,omitempty"`
	Error         string               `json:"error,omitempty"`
	Mode          string               `json:"mode,omitempty"`
	RelativeDates []relativeDateResult `json:"relativeDates,omitempty"`
}

type relativeDateFilter struct {
	Name          string `json:"name"`
	Input         string `json:"input"`
	ExpectInvalid bool   `json:"expectInvalid"`
}
type relativeDateResult struct {
	Name    string `json:"name"`
	Input   string `json:"input"`
	Value   string `json:"value"`
	Invalid bool   `json:"invalid"`
}

type challengeRecord struct {
	Type        string `json:"type"`
	Prompt      string `json:"prompt"`
	ImageSHA256 string `json:"imageSha256"`
}

func replay(ctx context.Context, input scenario) replayReport {
	report := replayReport{Engine: "moji", Requests: []requestRecord{}, Results: []resultRecord{}}
	if err := execute(ctx, input, &report); err != nil {
		report.Error = err.Error()
	}
	return report
}

func execute(ctx context.Context, input scenario, report *replayReport) error {
	if input.RelativeDateFilters != nil {
		return replayRelativeDates(ctx, input, report)
	}
	plan, err := cardigann.Load([]byte(input.Definition))
	if err != nil {
		return err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	session := &scriptedSession{input.Responses, report, jar}
	now := input.Now
	if now.IsZero() {
		now = time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	}
	execution, err := plan.NewExecution(session, "", input.Settings, now)
	if err != nil {
		return err
	}
	if input.Authenticate {
		challenge, err := execution.Authenticate(ctx)
		if err != nil {
			return err
		}
		if challenge != nil {
			report.Challenge = &challengeRecord{Type: challenge.Type, Prompt: challenge.Prompt}
			if len(challenge.Image) > 0 {
				sum := sha256.Sum256(challenge.Image)
				report.Challenge.ImageSHA256 = hex.EncodeToString(sum[:])
			}
			if input.ChallengeAnswer == nil {
				return fmt.Errorf("interactive challenge requires a scripted answer")
			}
			if err := execution.AnswerChallenge(ctx, *input.ChallengeAnswer); err != nil {
				return err
			}
		} else if input.ChallengeAnswer != nil {
			return fmt.Errorf("scripted answer requires a login challenge")
		}
	} else if input.ChallengeAnswer != nil {
		return fmt.Errorf("scripted answer requires authentication")
	}
	result, err := execution.Search(ctx, input.Query)
	if err != nil {
		return err
	}
	for _, row := range result.Results {
		categories := []int{}
		seen := map[int]bool{}
		for _, name := range row.Categories {
			if id, ok := indexercategory.ID(name); ok && !seen[id] {
				categories = append(categories, id)
				seen[id] = true
			}
		}
		sort.Ints(categories)
		report.Results = append(report.Results, resultRecord{row.Title, row.DownloadURL, row.MagnetURI, row.InfoHash,
			row.DetailsURL, row.Description, row.PosterURL, row.Size, row.Seeders, row.Leechers, categories, row.PublishedAt})
	}
	if input.Fetch && len(result.Results) > 0 {
		artifact, err := execution.Fetch(ctx, result.Results[0])
		if err != nil {
			return err
		}
		report.Artifact = &artifactRecord{Magnet: artifact.Magnet}
		if len(artifact.TorrentBytes) > 0 {
			sum := sha256.Sum256(artifact.TorrentBytes)
			report.Artifact.SHA256 = hex.EncodeToString(sum[:])
		}
	}
	return nil
}

// This helper mode compares Go filters with the real upstream date helper's
// explicit relativeFrom argument. It does not claim to run a search scenario.
func replayRelativeDates(ctx context.Context, input scenario, report *replayReport) error {
	if input.Now.IsZero() || input.Definition != "" || len(input.Responses) != 0 || len(input.Settings) != 0 || input.ChallengeAnswer != nil || input.Fetch || input.Authenticate || len(input.RelativeDateFilters) == 0 || len(input.RelativeDateFilters) > 100 {
		return fmt.Errorf("relative-date helper requires a fixed clock and 1..100 isolated inputs")
	}
	report.Mode = "relative-date-helper"
	for _, test := range input.RelativeDateFilters {
		if (test.Name != "timeago" && test.Name != "reltime") || len(test.Input) > 4096 {
			return fmt.Errorf("invalid relative-date helper input")
		}
		value, err := cardigann.ApplyFilters(ctx, test.Input, []cardigann.Filter{{Name: test.Name}}, cardigann.FilterContext{Now: input.Now.UTC()})
		if (err != nil) != test.ExpectInvalid {
			return fmt.Errorf("relative-date outcome differs from declared expectation for %q", test.Input)
		}
		report.RelativeDates = append(report.RelativeDates, relativeDateResult{Name: test.Name, Input: test.Input, Value: value, Invalid: err != nil})
	}
	return nil
}

// The only transport is this fixture dispatcher. Unscripted requests fail;
// neither DNS resolution nor an actual socket is possible from the session.
type scriptedSession struct {
	responses []responseFixture
	report    *replayReport
	jar       http.CookieJar
}

func (s *scriptedSession) SetCookieHeader(origin, header string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return err
	}
	request := &http.Request{Header: http.Header{"Cookie": {header}}}
	s.jar.SetCookies(u, request.Cookies())
	return nil
}

func (s *scriptedSession) Do(ctx context.Context, request cardigann.Request) (cardigann.HTTPResponse, error) {
	r, err := http.NewRequestWithContext(ctx, request.Method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return cardigann.HTTPResponse{}, err
	}
	r.Header = request.Headers.Clone()
	if r.Header == nil {
		r.Header = http.Header{}
	}
	client := http.Client{Transport: s, Jar: s.jar}
	if !request.FollowRedirect {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	response, err := client.Do(r)
	if err != nil {
		return cardigann.HTTPResponse{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, cardigann.MaxPageBytes+1))
	if err != nil || len(body) > cardigann.MaxPageBytes {
		return cardigann.HTTPResponse{}, fmt.Errorf("fixture response exceeds byte budget")
	}
	return cardigann.HTTPResponse{Status: response.StatusCode, Headers: response.Header, URL: response.Request.URL.String(), Body: body}, nil
}

func (s *scriptedSession) RoundTrip(request *http.Request) (*http.Response, error) {
	if len(s.report.Requests) >= 100 {
		return nil, fmt.Errorf("request budget exceeded")
	}
	body := []byte{}
	if request.Body != nil {
		var err error
		body, err = io.ReadAll(io.LimitReader(request.Body, (16<<20)+1))
		if err != nil {
			return nil, err
		}
		if len(body) > 16<<20 {
			return nil, fmt.Errorf("fixture request exceeds byte budget")
		}
	}
	headers := map[string]string{}
	for name, values := range request.Header {
		headers[name] = strings.Join(values, ",")
	}
	s.report.Requests = append(s.report.Requests, requestRecord{request.Method, request.URL.String(), body, headers})
	for _, fixture := range s.responses {
		if fixture.Method == request.Method && fixture.Path == request.URL.EscapedPath() {
			return &http.Response{StatusCode: fixture.Status, Header: fixture.Headers.Clone(), Request: request,
				Body: io.NopCloser(bytes.NewReader(fixture.Body)), ContentLength: int64(len(fixture.Body))}, nil
		}
	}
	return nil, fmt.Errorf("unscripted request %s %s", request.Method, request.URL.EscapedPath())
}
