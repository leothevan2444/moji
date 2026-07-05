package downloader

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/pkg/jackett"
)

func TestDefaultCandidateSelectorRejectsUndownloadableResults(t *testing.T) {
	selector := defaultCandidateSelector{}
	_, err := selector.Select("ABCD-123", []jackett.SearchResult{{Title: "nope"}}, config.DefaultCandidateSelectionConfig())
	if err == nil {
		t.Fatal("expected error when no downloadable candidate exists")
	}
}

func TestDefaultCandidateSelectorUsesIndexerPreference(t *testing.T) {
	selector := defaultCandidateSelector{}
	result, err := selector.Select("ABCD-123", []jackett.SearchResult{
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
	result, err := selector.Select("ABCD-123", []jackett.SearchResult{
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
	result, err := selector.Select("ABCD-123", []jackett.SearchResult{
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
	result, err := selector.Select("ABCD-123", []jackett.SearchResult{
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
