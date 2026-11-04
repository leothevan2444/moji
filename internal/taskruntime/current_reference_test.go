package taskruntime

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

func TestCurrentReferencePreviewChecksSourceBeforeInspectionCache(t *testing.T) {
	source := &selectedSource{record: selectionRecord()}
	service := &Service{resourceAccess: source, now: time.Now, inspectionCache: map[string]cachedTorrentInspection{}, candidateSelection: func() config.CandidateSelectionConfig {
		return candidateSelectionConfig(true, 0, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true}})
	}}
	handle := "indexer-resource:" + source.record.Ref.ID
	metadata, err := parseTorrentMetadata([]byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"})))
	if err != nil {
		t.Fatal(err)
	}
	service.storeCachedArtifact(handle, &metadata, "")
	request := PreviewTorrentSelectionRequest{Query: "ABCD-123", ApplyFileRules: true, Results: []indexer.SelectionCandidate{{Title: "ABCD-123", Link: handle}}}
	preview, err := service.PreviewTorrentSelectionContext(context.Background(), request)
	if err != nil || preview.Meta.InspectedCount != 1 || source.resolves != 1 {
		t.Fatal("valid preview did not validate its cached source", err)
	}
	source.expired = true
	preview, err = service.PreviewTorrentSelectionContext(context.Background(), request)
	if err != nil || preview.Meta.InspectedCount != 0 || source.resolves != 2 {
		t.Fatal("new preview reused cached files after its source became invalid", err)
	}
	if _, err := service.inspectSearchResultTorrent(context.Background(), handle); err != nil {
		t.Fatal("already admitted sourcing lost its frozen inspection", err)
	}
	if source.fetches != 0 || source.resolves != 2 {
		t.Fatal("cache validation consumed a new resource or changed accepted work")
	}
}

type expiringPreviewSource struct{ *selectedSource }

func (s *expiringPreviewSource) ResolveReference(ctx context.Context, id string) (*indexer.ResourceLease, error) {
	lease, err := s.selectedSource.ResolveReference(ctx, id)
	if err == nil {
		s.expired = true
	}
	return lease, err
}
func (s *expiringPreviewSource) FetchReference(ctx context.Context, id string) (indexer.Artifact, error) {
	if s.expired {
		return indexer.Artifact{}, indexer.ErrExpired
	}
	return s.FetchRecord(ctx, s.record)
}

func TestCurrentReferencePreviewKeepsAdmittedLeaseAfterExpiry(t *testing.T) {
	source := &expiringPreviewSource{selectedSource: &selectedSource{record: selectionRecord(), data: []byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"}))}}
	service, err := NewService(nil, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithIndexers(source), WithCandidateSelectionProvider(func() config.CandidateSelectionConfig {
		return candidateSelectionConfig(true, 0, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentSingleVideo, Enabled: true}})
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := PreviewTorrentSelectionRequest{Query: "ABCD-123", ApplyFileRules: true, Results: []indexer.SelectionCandidate{{Title: "ABCD-123", Link: "indexer-resource:" + source.record.Ref.ID}}}
	preview, err := service.PreviewTorrentSelectionContext(context.Background(), request)
	if err != nil || preview.Meta.InspectedCount != 1 || source.resolves != 1 || source.fetches != 1 {
		t.Fatal("admitted preview discarded its borrowed source after expiry", err)
	}
	preview, err = service.PreviewTorrentSelectionContext(context.Background(), request)
	if err != nil || preview.Meta.InspectedCount != 0 || source.fetches != 1 {
		t.Fatal("later preview reused an expired source", err)
	}
}
