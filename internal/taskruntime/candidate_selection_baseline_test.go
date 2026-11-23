package taskruntime

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"testing"
)

func TestNeutralSelectionStillUsesAvailabilityAndDeterministicTies(t *testing.T) {
	p := config.DefaultCandidateSelectionConfig()
	rows := []indexer.SelectionCandidate{{Title: "ABCD-123", Link: "https://example.test/unknown", Seeders: -1}, {Title: "ABCD-123", Link: "https://example.test/zero", Seeders: 0}, {Title: "ABCD-123 无码破解", Link: "https://example.test/seeded", Seeders: 3}}
	got, err := (defaultCandidateSelector{}).Preview(context.Background(), "ABCD-123", rows, p)
	if err != nil || got.Results[0].Link != rows[2].Link {
		t.Fatalf("neutral policy lost availability fallback: %+v %v", got, err)
	}
	got, err = (defaultCandidateSelector{}).Preview(context.Background(), "ABCD-123", rows[:2], p)
	if err != nil || got.Results[0].Link != rows[0].Link {
		t.Fatalf("unknown availability should precede zero: %+v %v", got, err)
	}
}
