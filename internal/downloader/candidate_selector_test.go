package downloader

import (
	"context"
	"strings"
	"testing"

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
	selector := defaultCandidateSelector{
		inspectTorrent: func(_ context.Context, torrentURL string) (torrentInspection, error) {
			inspected = append(inspected, torrentURL)
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
