package taskruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/pkg/jackett"
)

// Regression cases found during the U-002/R-002 baseline audit.
func TestSelectionDisabledPreservesInputOrder(t *testing.T) {
	selector := defaultCandidateSelector{}
	got, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "ABCD-123", Link: "https://example.test/first", Seeders: 1},
		{Title: "ABCD-123", Link: "https://example.test/second", Seeders: 20},
	}, candidateSelectionConfig(false, 1, []config.CandidateSelectionRule{
		{Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Seeders: config.SeedersRuleConfig{Direction: config.CandidateSelectionDirectionAsc}},
	}))
	if err != nil || got.Seeders != 1 {
		t.Fatalf("disabled chain must preserve upstream order: got %+v, err %v", got, err)
	}
}

func TestSelectionFileAvoidDemotes(t *testing.T) {
	selector := defaultCandidateSelector{inspectTorrent: func(_ context.Context, url string) (torrentInspection, error) {
		if url == "https://example.test/avoided" {
			return torrentInspection{Paths: []string{"sample.mp4"}}, nil
		}
		return torrentInspection{Paths: []string{"movie.mp4"}}, nil
	}}
	got, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "ABCD-123", Link: "https://example.test/avoided"},
		{Title: "ABCD-123", Link: "https://example.test/plain"},
	}, candidateSelectionConfig(true, 2, []config.CandidateSelectionRule{
		{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{
			Clauses: []config.TorrentFileNameMatchClause{{Pattern: "sample", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TorrentFileMatchEffectAvoid}},
		}},
	}))
	if err != nil || got.Link != "https://example.test/plain" {
		t.Fatalf("legacy file AVOID must demote: got %+v, err %v", got, err)
	}
}

func TestSelectionLockFallsBackOnUnmatchedAndInspectionFailure(t *testing.T) {
	for _, inspectionFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "lock_unmatched", true: "inspection_failed"}[inspectionFails], func(t *testing.T) {
			selector := defaultCandidateSelector{inspectTorrent: func(context.Context, string) (torrentInspection, error) {
				if inspectionFails {
					return torrentInspection{}, errors.New("fake inspection failure")
				}
				return torrentInspection{Paths: []string{"movie.mp4"}}, nil
			}}
			got, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
				{Title: "ABCD-123", Link: "https://example.test/plain"},
			}, candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{
				{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{
					Clauses: []config.TorrentFileNameMatchClause{{Pattern: "required", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TorrentFileMatchEffectLock}},
				}},
			}))
			if err != nil || got.Link != "https://example.test/plain" {
				t.Fatalf("unmatched or unverified lock must leave fallback eligible: got %+v, err %v", got, err)
			}
		})
	}
}

func TestSelectionTitleHardConditionsOverridePreferences(t *testing.T) {
	rules := []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTitleMatch, Enabled: true, TitleMatch: config.TitleMatchRuleConfig{Clauses: []config.TitleMatchClause{
		{Pattern: "high", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectPrefer},
		{Pattern: "required", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectLock},
		{Pattern: "excluded", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectExclude},
	}}}}
	selector := defaultCandidateSelector{}
	results := []jackett.SearchResult{
		{Title: "ABCD-123 high required excluded", Link: "https://example.test/excluded"},
		{Title: "ABCD-123 high", Link: "https://example.test/unmatched"},
		{Title: "ABCD-123 required", Link: "https://example.test/eligible"},
	}
	cfg := candidateSelectionConfig(true, 1, rules)
	got, err := selector.Select(context.Background(), "ABCD-123", results, cfg)
	if err != nil || got.Link != "https://example.test/eligible" {
		t.Fatalf("hard conditions bypassed: %+v, %v", got, err)
	}
	if got, err := selector.Select(context.Background(), "ABCD-123", results[:2], cfg); err != nil || got.Link != "https://example.test/unmatched" {
		t.Fatalf("excluded lock must not prevent ordinary fallback: got %+v, err %v", got, err)
	}
}

func TestSelectionFileHardConditionsDoNotEscapeInspectionBudget(t *testing.T) {
	calls := 0
	selector := defaultCandidateSelector{inspectTorrent: func(context.Context, string) (torrentInspection, error) {
		calls++
		return torrentInspection{Paths: []string{"movie.mp4"}}, nil
	}}
	cfg := candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{Clauses: []config.TorrentFileNameMatchClause{
		{Pattern: "movie", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TorrentFileMatchEffectExclude},
	}}}})
	_, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "ABCD-123", Link: "https://example.test/first"},
		{Title: "ABCD-123", Link: "https://example.test/uninspected"},
		{Title: "ABCD-123", MagnetURI: "magnet:?xt=urn:btih:123"},
	}, cfg)
	if err == nil || calls != 1 {
		t.Fatalf("unchecked candidate escaped hard exclusion or budget: calls=%d err=%v", calls, err)
	}
}

func TestSelectionLocksAnyTitleOrFileMatchAfterExclusions(t *testing.T) {
	results := []jackett.SearchResult{
		{Title: "ABCD-123 plain", Link: "https://example.test/plain", Seeders: 100},
		{Title: "ABCD-123 title-lock", Link: "https://example.test/title", Seeders: 10},
		{Title: "ABCD-123 other", Link: "https://example.test/file", Seeders: 20},
	}
	rules := []config.CandidateSelectionRule{
		{Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Seeders: config.SeedersRuleConfig{Direction: config.CandidateSelectionDirectionDesc}},
		{Type: config.CandidateSelectionRuleTypeTitleMatch, Enabled: true, TitleMatch: config.TitleMatchRuleConfig{Clauses: []config.TitleMatchClause{
			{Pattern: "title-lock", Effect: config.TitleMatchEffectLock},
			{Pattern: "never", Effect: config.TitleMatchEffectLock},
		}}},
		{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{Clauses: []config.TorrentFileNameMatchClause{
			{Pattern: "file-lock", Effect: config.TorrentFileMatchEffectLock},
			{Pattern: "never", Effect: config.TorrentFileMatchEffectLock},
			{Pattern: "excluded", Effect: config.TorrentFileMatchEffectExclude},
		}}},
	}
	for _, tc := range []struct {
		name            string
		excludeFileLock bool
		fast, file      bool
		want            string
	}{
		{"cross_stage_or_and_existing_ranking", false, true, true, "file"},
		{"file_exclusion_overrides_lock", true, true, true, "title"},
		{"title_stage_only", false, true, false, "title"},
		{"file_stage_only", false, false, true, "file"},
		{"no_stages", false, false, false, "plain"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selector := defaultCandidateSelector{inspectTorrent: func(_ context.Context, url string) (torrentInspection, error) {
				path := "movie.mp4"
				if url == "https://example.test/file" {
					path = "file-lock/movie.mp4"
					if tc.excludeFileLock {
						path = "excluded/" + path
					}
				}
				return torrentInspection{Paths: []string{path}}, nil
			}}
			preview, err := selector.Preview(context.Background(), "ABCD-123", results, candidateSelectionConfig(true, 3, rules), tc.fast, tc.file)
			if err != nil || len(preview.Results) == 0 || preview.Results[0].Link != "https://example.test/"+tc.want {
				t.Fatalf("got %+v, err %v", preview.Results, err)
			}
		})
	}
}

func TestSelectionTitleLocksUseORAndKeepNormalTieBreaking(t *testing.T) {
	rules := []config.CandidateSelectionRule{
		{Type: config.CandidateSelectionRuleTypeTitleMatch, Enabled: true, TitleMatch: config.TitleMatchRuleConfig{Clauses: []config.TitleMatchClause{
			{Pattern: "first", Effect: config.TitleMatchEffectLock},
			{Pattern: "second", Effect: config.TitleMatchEffectLock},
			{Pattern: "plain", Effect: config.TitleMatchEffectPrefer},
		}}},
		{Type: config.CandidateSelectionRuleTypeSeeders, Enabled: true, Seeders: config.SeedersRuleConfig{Direction: config.CandidateSelectionDirectionDesc}},
	}
	results := []jackett.SearchResult{
		{Title: "plain", Link: "https://example.test/plain", Seeders: 100},
		{Title: "first", Link: "https://example.test/first", Seeders: 5},
		{Title: "second", Link: "https://example.test/second", Seeders: 10},
	}
	selector := defaultCandidateSelector{}
	for _, enabled := range []bool{true, false} {
		got, err := selector.Select(context.Background(), "ABCD-123", results, candidateSelectionConfig(enabled, 1, rules))
		want := "second"
		if !enabled {
			want = "plain"
		}
		if err != nil || got.Title != want {
			t.Fatalf("enabled=%v got %+v, err %v", enabled, got, err)
		}
	}
}

func TestSelectionFileLockWithoutInspectionKeepsFallback(t *testing.T) {
	cfg := candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{Clauses: []config.TorrentFileNameMatchClause{
		{Pattern: "lock", Effect: config.TorrentFileMatchEffectLock},
	}}}})
	for _, selector := range []defaultCandidateSelector{
		{},
		{inspectTorrent: func(context.Context, string) (torrentInspection, error) {
			t.Fatal("magnet must not be inspected")
			return torrentInspection{}, nil
		}},
	} {
		got, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{{Title: "plain", MagnetURI: "magnet:?xt=urn:btih:123"}}, cfg)
		if err != nil || got.Title != "plain" {
			t.Fatalf("got %+v, err %v", got, err)
		}
	}
}

func TestSelectionFileLockRespectsInspectionBudget(t *testing.T) {
	selector := defaultCandidateSelector{inspectTorrent: func(_ context.Context, url string) (torrentInspection, error) {
		if url != "https://example.test/first" {
			t.Fatalf("inspected candidate beyond budget: %s", url)
		}
		return torrentInspection{Paths: []string{"ordinary.mp4"}}, nil
	}}
	cfg := candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{Clauses: []config.TorrentFileNameMatchClause{
		{Pattern: "lock", Effect: config.TorrentFileMatchEffectLock},
	}}}})
	got, err := selector.Select(context.Background(), "ABCD-123", []jackett.SearchResult{
		{Title: "first", Link: "https://example.test/first"},
		{Title: "second", Link: "https://example.test/second"},
	}, cfg)
	if err != nil || got.Title != "first" {
		t.Fatalf("expected normal fallback within inspection budget: got %+v, err %v", got, err)
	}
}
