package indexerdefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestOfficialSourcePinsDownloadsAndUsesConditionalChecks(t *testing.T) {
	revision := strings.Repeat("a", 40)
	calls := 0
	source := NewGitHubSource(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
			t.Fatal("private credentials used for definitions")
		}
		response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("archive"))}
		switch calls {
		case 1:
			if request.URL.String() != sourceAPI || request.Header.Get("If-None-Match") != `"old"` {
				t.Fatal("wrong update request")
			}
			response.Header.Set("ETag", `"new"`)
			response.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`[{"sha":%q,"commit":{"committer":{"date":"2026-11-01T00:00:00Z"}}}]`, revision)))
		case 2:
			if request.URL.String() != sourceArchive+revision {
				t.Fatal("download does not pin discovered revision")
			}
		case 3:
			response.StatusCode = http.StatusNotModified
		}
		return response, nil
	})})
	got, unchanged, err := source.Latest(context.Background(), `"old"`)
	if err != nil || unchanged || got.Commit != revision || got.ETag != `"new"` {
		t.Fatalf("latest %+v %v %v", got, unchanged, err)
	}
	var content strings.Builder
	if err := source.Download(context.Background(), got.Commit, &content); err != nil || content.String() != "archive" {
		t.Fatal("download failed", err)
	}
	if _, unchanged, err := source.Latest(context.Background(), got.ETag); err != nil || !unchanged {
		t.Fatal("conditional check failed", err)
	}
	if err := source.Download(context.Background(), "main", io.Discard); !errors.Is(err, ErrInvalidBundle) {
		t.Fatal("mutable reference accepted", err)
	}
}

func TestOfficialSourceBoundsResponsesAndRejectsUnsafeRedirects(t *testing.T) {
	source := NewGitHubSource(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader("sensitive upstream body"))}, nil
	})})
	_, _, err := source.Latest(context.Background(), "")
	var upstream *SourceError
	if !errors.As(err, &upstream) || upstream.RetryAfter.Seconds() != 120 || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("rate limit metadata not safe", err)
	}
	for _, target := range []string{"http://api.github.com/", "https://example.invalid/", "https://api.github.com:8443/", "https://user:pass@api.github.com/"} {
		request, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := source.client.CheckRedirect(request, nil); err == nil {
			t.Fatalf("unsafe redirect accepted %s", target)
		}
	}
	source = NewGitHubSource(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, ContentLength: MaxArchiveBytes + 1, Body: io.NopCloser(strings.NewReader("unused"))}, nil
	})})
	if err := source.Download(context.Background(), strings.Repeat("a", 40), io.Discard); !errors.Is(err, ErrBudget) {
		t.Fatal("oversized archive accepted", err)
	}
}
