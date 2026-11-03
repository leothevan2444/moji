package taskruntime

import (
	"bytes"
	"context"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"testing"
)

type indexedFixture struct {
	data    []byte
	fetches int
}

func (f *indexedFixture) SearchSequential(ctx context.Context, q indexer.Query, accept func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
	_, err := accept(ctx, indexer.Batch{InstanceID: "builtin-one", Releases: []indexer.Release{{Title: "ABCD-123", InstanceID: "builtin-one", InstanceName: "Builtin", Kind: indexer.Builtin, Resource: indexer.ResourceRef{Version: 1, ID: "opaque-resource", InstanceID: "builtin-one", Kind: indexer.Builtin}}}})
	return indexer.SearchReport{}, err
}
func (f *indexedFixture) FetchReference(ctx context.Context, id string) (indexer.Artifact, error) {
	f.fetches++
	return indexer.Artifact{TorrentBytes: append([]byte(nil), f.data...)}, nil
}

type persistedTorrentAdder struct {
	fakeTorrentAdder
	store    TaskStore
	expected []byte
	t        *testing.T
}

func (a *persistedTorrentAdder) AddNewTorrent(ctx context.Context, options qbittorrent.AddTorrentOptions) error {
	task, err := a.store.Find(ctx, "indexed-task")
	if err != nil || !bytes.Equal(task.TorrentFileData, a.expected) {
		a.t.Error("raw resource was not durable before downloader effect")
	}
	return a.fakeTorrentAdder.AddNewTorrent(ctx, options)
}

func TestIndexedSourcingInspectsAndSubmitsIdenticalPersistedBytes(t *testing.T) {
	data := []byte(testTorrentFile("ABCD-123", []string{"ABCD-123.mp4"}))
	source := &indexedFixture{data: data}
	store, _ := relationalStore(t)
	adder := &persistedTorrentAdder{store: store, expected: data, t: t}
	rules := candidateSelectionConfig(true, 1, []config.CandidateSelectionRule{{Type: config.CandidateSelectionRuleTypeTorrentFileNameMatch, Enabled: true, TorrentFileNameMatch: config.TorrentFileNameMatchRuleConfig{Clauses: []config.TorrentFileNameMatchClause{{Pattern: "sample", PatternMode: config.TitleMatchPatternModePlain, Effect: config.TorrentFileMatchEffectExclude}}}}})
	service, err := NewService(nil, adder, store, WithIndexers(source), WithIDGenerator(func() string { return "indexed-task" }), WithCandidateSelectionProvider(func() config.CandidateSelectionConfig { return rules }))
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.DownloadMediaContext(context.Background(), DownloadRequest{Code: "ABCD-123"})
	if err != nil {
		t.Fatal(err)
	}
	if source.fetches != 1 {
		t.Fatalf("inspection and preparation grabbed %d times", source.fetches)
	}
	if len(adder.options.Torrents) != 1 || !bytes.Equal(adder.options.Torrents[0].Data, data) || !bytes.Equal(task.TorrentFileData, data) {
		t.Fatal("download does not use inspected and persisted bytes")
	}
	if task.Candidate.Link != "indexer-resource:opaque-resource" {
		t.Fatal("stable source reference missing")
	}
	stored, err := store.Find(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Candidate.Source == nil || stored.Candidate.Source.InstanceID != "builtin-one" || stored.Candidate.Source.ResourceID != "opaque-resource" {
		t.Fatal("source provenance did not survive core read")
	}
	restarted, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, store)
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := parseTorrentMetadata(stored.TorrentFileData)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := restarted.prepareTorrentWithMetadata(context.Background(), stored.Code, stored.Candidate, stored.TorrentURL, &metadata)
	if err != nil || len(recovered.options.Torrents) != 1 || !bytes.Equal(recovered.options.Torrents[0].Data, data) {
		t.Fatal("recovery depended on the expired runtime resource", err)
	}
}
