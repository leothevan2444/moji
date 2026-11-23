package taskruntime

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/resourceselection"
	"testing"
)

func TestDefaultCandidateSelectorHasNoMediaPreferences(t *testing.T) {
	rows := []indexer.SelectionCandidate{{Title: "ABCD-123 无码破解", Link: "https://example.test/modified", Seeders: 100}, {Title: "ABCD-123", Link: "https://example.test/plain", Seeders: 1}}
	result, err := (defaultCandidateSelector{}).Preview(context.Background(), "ABCD-123", rows, config.DefaultCandidateSelectionConfig())
	if err != nil || result.Results[0].Link != rows[0].Link {
		t.Fatalf("wrong resource: %+v %v", result, err)
	}
}
func TestSelectionUnknownIndexerExclusionDoesNotPass(t *testing.T) {
	p := config.DefaultCandidateSelectionConfig()
	p.Rules = []resourceselection.Rule{{ID: "file", Name: "file exclusion", Conditions: []resourceselection.Condition{{Scope: "SEEDERS", Minimum: "1"}}}}
	p.ActiveGroup = "main"
	p.Groups = []resourceselection.Group{{Fallback: true, ID: "main", Name: "Main", Exclude: []string{"file"}, Tiers: []resourceselection.Tier{}}}
	preview, err := (defaultCandidateSelector{}).Preview(context.Background(), "ABCD-123", []indexer.SelectionCandidate{{Title: "ABCD-123", Link: "https://example.test/file", Seeders: -1}}, p)
	if err != nil || len(preview.Evaluations) != 1 || preview.Evaluations[0].Status != "INSUFFICIENT_DATA" {
		t.Fatalf("missing source field silently passed: %+v %v", preview, err)
	}
}
