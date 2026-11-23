package taskruntime

import (
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"testing"
)

func TestSelectionPreviewNeverResolvesOrFetchesIndexerArtifacts(t *testing.T) {
	source := &selectedSource{record: selectionRecord()}
	service, err := NewService(nil, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithIndexers(source), WithCandidateSelectionProvider(func() config.CandidateSelectionConfig { return config.DefaultCandidateSelectionConfig() }))
	if err != nil {
		t.Fatal(err)
	}
	source.expired = true // Source acquisition availability must not affect pure matching.
	out, err := service.PreviewTorrentSelectionContext(context.Background(), PreviewTorrentSelectionRequest{Query: "ABCD-123", Results: []indexer.SelectionCandidate{{Title: "ABCD-123", ResourceID: source.record.Ref.ID}}})
	if err != nil || out.Evaluations[0].Status != "ACCEPTABLE" || source.resolves != 0 || source.fetches != 0 {
		t.Fatalf("preview acquired a resource: %v resolves=%d fetches=%d", err, source.resolves, source.fetches)
	}
}
