package downloader

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestDefaultCandidateSelectorRejectsUndownloadableResults(t *testing.T) {
	selector := defaultCandidateSelector{}
	_, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{{Title: "nope"}}, config.DefaultCandidateSelectionConfig())
	if err == nil {
		t.Fatal("expected error when no downloadable candidate exists")
	}
}

func TestDefaultCandidateSelectorUsesIndexerPreference(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "beta", MagnetURI: "magnet:?xt=urn:btih:beta", TrackerID: "beta", Seeders: 100},
		{Title: "alpha", MagnetURI: "magnet:?xt=urn:btih:alpha", TrackerID: "alpha", Seeders: 1},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{
				ID:        "pref",
				Type:      config.CandidateSelectionRuleTypeIndexerPreference,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionAsc,
				IndexerPreference: config.IndexerPreferenceRuleConfig{
					TrackerIDs: []string{"alpha", "beta"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.TrackerID != "alpha" {
		t.Fatalf("expected alpha to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorUsesTitleMatchClauses(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "ABCD-123 无码", MagnetURI: "magnet:?xt=urn:btih:1"},
		{Title: "ABCD-123 SAMPLE", MagnetURI: "magnet:?xt=urn:btih:2"},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{
				ID:        "title",
				Type:      config.CandidateSelectionRuleTypeTitleMatch,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionDesc,
				TitleMatch: config.TitleMatchRuleConfig{
					Clauses: []config.TitleMatchClause{
						{Pattern: "无码", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectPrefer},
						{Pattern: "sample", PatternMode: config.TitleMatchPatternModeRegex, Effect: config.TitleMatchEffectAvoid},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.Title != "ABCD-123 无码" {
		t.Fatalf("expected preferred title to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorUsesPublishDateDesc(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "older", MagnetURI: "magnet:?xt=urn:btih:1", PublishDate: "2024-01-02"},
		{Title: "newer", MagnetURI: "magnet:?xt=urn:btih:2", PublishDate: "2025-01-02"},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{
				ID:        "date",
				Type:      config.CandidateSelectionRuleTypePublishDate,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionDesc,
			},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.Title != "newer" {
		t.Fatalf("expected newer title to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorUsesTitleSimilarity(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "random release", MagnetURI: "magnet:?xt=urn:btih:1"},
		{Title: "ABCD 123 uncensored", MagnetURI: "magnet:?xt=urn:btih:2"},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{
				ID:        "similarity",
				Type:      config.CandidateSelectionRuleTypeTitleSimilarity,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionDesc,
			},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.Title != "ABCD 123 uncensored" {
		t.Fatalf("expected most similar title to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorUsesTorrentSingleVideoInspection(t *testing.T) {
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			if strings.Contains(torrentURL, "single") {
				return torrentInspection{
					Paths:       []string{"ABCD-123.mp4"},
					VideoPaths:  []string{"ABCD-123.mp4"},
					SingleVideo: true,
				}, nil
			}
			return torrentInspection{
				Paths:      []string{"ABCD-123.mp4", "sample.jpg"},
				VideoPaths: []string{"ABCD-123.mp4"},
			}, nil
		},
	}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "multi", Link: "https://example.test/multi.torrent", Seeders: 10},
		{Title: "single", Link: "https://example.test/single.torrent", Seeders: 10},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{ID: "seeders", Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
			{ID: "single-video", Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.Title != "single" {
		t.Fatalf("expected single-video candidate to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorUsesTorrentFileNameLock(t *testing.T) {
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			if strings.Contains(torrentURL, "locked") {
				return torrentInspection{
					Paths: []string{"movie/hhd800.com-ABCD-123.mp4"},
				}, nil
			}
			return torrentInspection{
				Paths: []string{"movie/ABCD-123.mp4"},
			}, nil
		},
	}
	result, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "baseline", Link: "https://example.test/baseline.torrent", Seeders: 20},
		{Title: "locked", Link: "https://example.test/locked.torrent", Seeders: 1},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{ID: "seeders", Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
			{
				ID:        "file-name",
				Type:      config.CandidateSelectionRuleTypeTorrentFileNameMatch,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionDesc,
				TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{
					Clauses: []config.TorrentFileNameMatchClause{
						{Pattern: "hhd800.com", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TorrentFileMatchEffectLock},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if result.Title != "locked" {
		t.Fatalf("expected LOCK match to win, got %+v", result)
	}
}

func TestDefaultCandidateSelectorOnlyInspectsTopFiveTorrentCandidates(t *testing.T) {
	inspected := make([]string, 0)
	var inspectedMu sync.Mutex
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			inspectedMu.Lock()
			inspected = append(inspected, torrentURL)
			inspectedMu.Unlock()
			return torrentInspection{Paths: []string{"ABCD-123.mp4"}}, nil
		},
	}
	results := make([]jackett.SearchResult, 0, 6)
	for i := 0; i < 6; i++ {
		results = append(results, jackett.SearchResult{
			Title:   "candidate-" + string(rune('A'+i)),
			Link:    "https://example.test/" + string(rune('1'+i)) + ".torrent",
			Seeders: 100 - i,
		})
	}
	_, err := selector.Select(context.Background(), "ABCD-123", results, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{ID: "seeders", Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
			{ID: "single-video", Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
		},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if len(inspected) != 5 {
		t.Fatalf("expected 5 inspected candidates, got %d (%v)", len(inspected), inspected)
	}
}

func TestDefaultCandidateSelectorUsesConfiguredInspectionCandidateLimit(t *testing.T) {
	results := []jackett.SearchResult{
		{Title: "1", Link: "https://example.com/1.torrent", Seeders: 100},
		{Title: "2", Link: "https://example.com/2.torrent", Seeders: 90},
		{Title: "3", Link: "https://example.com/3.torrent", Seeders: 80},
		{Title: "4", Link: "https://example.com/4.torrent", Seeders: 70},
		{Title: "5", Link: "https://example.com/5.torrent", Seeders: 60},
		{Title: "6", Link: "https://example.com/6.torrent", Seeders: 50},
	}
	inspected := make([]string, 0, len(results))
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			inspected = append(inspected, torrentURL)
			if strings.Contains(torrentURL, "/6.torrent") {
				return torrentInspection{Paths: []string{"movie.mkv"}, VideoPaths: []string{"movie.mkv"}, SingleVideo: true}, nil
			}
			return torrentInspection{Paths: []string{"disc/file1.mkv", "disc/file2.srt"}, VideoPaths: []string{"disc/file1.mkv"}, SingleVideo: false}, nil
		},
	}

	got, err := selector.Select(context.Background(), "ABCD-123", results, config.CandidateSelectionConfig{
		Enabled:                  true,
		InspectionCandidateLimit: 6,
		Rules: []config.CandidateSelectionRule{
			{ID: "seeders", Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
			{ID: "single-video", Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
		},
	})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if got.Link != "https://example.com/6.torrent" {
		t.Fatalf("expected configured inspection limit to allow sixth candidate, got %+v", got)
	}
	if len(inspected) != 6 {
		t.Fatalf("expected 6 inspected candidates, got %d", len(inspected))
	}
}

func TestDefaultCandidateSelectorPreviewUsesFastRulesOnly(t *testing.T) {
	selector := defaultCandidateSelector{}
	preview, err := selector.Preview(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "beta", MagnetURI: "magnet:?xt=urn:btih:beta", TrackerID: "beta", Seeders: 100},
		{Title: "alpha", MagnetURI: "magnet:?xt=urn:btih:alpha", TrackerID: "alpha", Seeders: 1},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{
				ID:        "pref",
				Type:      config.CandidateSelectionRuleTypeIndexerPreference,
				Enabled:   true,
				Direction: config.CandidateSelectionDirectionAsc,
				IndexerPreference: config.IndexerPreferenceRuleConfig{
					TrackerIDs: []string{"alpha", "beta"},
				},
			},
		},
	}, true, false)
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if len(preview.Results) != 2 || preview.Results[0].TrackerID != "alpha" {
		t.Fatalf("unexpected preview order: %+v", preview.Results)
	}
	if !preview.Meta.AppliedFastRules || preview.Meta.AppliedFileRules {
		t.Fatalf("unexpected preview meta: %+v", preview.Meta)
	}
}

func TestDefaultCandidateSelectorPreviewUsesInputOrderForFileRules(t *testing.T) {
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			if strings.Contains(torrentURL, "second") {
				return torrentInspection{Paths: []string{"movie.mp4"}, VideoPaths: []string{"movie.mp4"}, SingleVideo: true}, nil
			}
			return torrentInspection{Paths: []string{"disc/movie.mp4", "sample.txt"}}, nil
		},
	}
	preview, err := selector.Preview(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "first", Link: "https://example.test/first.torrent", Seeders: 100},
		{Title: "second", Link: "https://example.test/second.torrent", Seeders: 1},
	}, config.CandidateSelectionConfig{
		Enabled: true,
		Rules: []config.CandidateSelectionRule{
			{ID: "single-video", Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
		},
	}, false, true)
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if preview.Results[0].Title != "second" {
		t.Fatalf("expected file-rule preview to reorder by input-order baseline, got %+v", preview.Results)
	}
	if preview.Meta.InspectableCount != 2 || preview.Meta.InspectedCount != 2 {
		t.Fatalf("unexpected inspection meta: %+v", preview.Meta)
	}
}

func TestPreviewJackettSelectionContextCachesTorrentInspection(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	requests := 0
	service, err := NewService(
		fakeTracker{},
		qbt,
		NewMemoryTaskStore(),
		WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				body := io.NopCloser(strings.NewReader(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"})))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       body,
					Header:     make(http.Header),
				}, nil
			}),
			Timeout: 2 * time.Second,
		}),
		WithCandidateSelectionProvider(func() config.CandidateSelectionConfig {
			return config.CandidateSelectionConfig{
				Enabled: true,
				Rules: []config.CandidateSelectionRule{
					{ID: "single-video", Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true, Direction: config.CandidateSelectionDirectionDesc},
				},
			}
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	req := PreviewJackettSelectionRequest{
		Query:          "ABCD-123",
		ApplyFileRules: true,
		Results: []jackett.SearchResult{
			{Title: "cached", Link: "https://example.test/cached.torrent"},
		},
	}
	if _, err := service.PreviewJackettSelectionContext(context.Background(), req); err != nil {
		t.Fatalf("first preview failed: %v", err)
	}
	if _, err := service.PreviewJackettSelectionContext(context.Background(), req); err != nil {
		t.Fatalf("second preview failed: %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected cached inspection to avoid duplicate HTTP fetches, got %d requests", requests)
	}
}

func TestDownloadMediaContextUsesConfiguredCandidateSelection(t *testing.T) {
	qbt := &fakeTorrentAdder{}
	service, err := NewService(
		fakeTracker{results: []jackett.SearchResult{
			{Title: "ABCD-123 larger", MagnetURI: "magnet:?xt=urn:btih:1", Seeders: 10, Size: 200},
			{Title: "ABCD-123 smaller", MagnetURI: "magnet:?xt=urn:btih:2", Seeders: 10, Size: 100},
		}},
		qbt,
		NewMemoryTaskStore(),
		WithCandidateSelectionProvider(func() config.CandidateSelectionConfig {
			return config.CandidateSelectionConfig{
				Enabled: true,
				Rules: []config.CandidateSelectionRule{
					{
						ID:        "size-asc",
						Type:      config.CandidateSelectionRuleTypeSize,
						Enabled:   true,
						Direction: config.CandidateSelectionDirectionAsc,
					},
				},
			}
		}),
	)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Query: "ABCD-123"})
	if err != nil {
		t.Fatalf("DownloadMediaContext failed: %v", err)
	}
	if task.Candidate.Title != "ABCD-123 smaller" {
		t.Fatalf("expected configured selector to choose smaller torrent, got %+v", task.Candidate)
	}
}
