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

func TestSelectionRequireRejectsUnmatchedAndInspectionFailure(t *testing.T) {
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
			if err == nil {
				t.Fatalf("REQUIRE must reject unverified or unmatched fallback: got %+v, err %v", got, err)
			}
		})
	}
}

func TestSelectionTitleHardConditionsOverridePreferences(t *testing.T) {
	rules := []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTitleMatch, Enabled: true, TitleMatch: config.TitleMatchRuleConfig{Clauses: []config.TitleMatchClause{
		{Pattern: "high", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectPrefer},
		{Pattern: "required", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TitleMatchEffectRequire},
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
	if _, err := selector.Select(context.Background(), "ABCD-123", results[:2], cfg); err == nil {
		t.Fatal("accepted candidates that fail hard conditions")
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
