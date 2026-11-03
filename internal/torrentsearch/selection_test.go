package torrentsearch

import (
	"context"
	"errors"
	"github.com/leothevan2444/moji/internal/indexer"
	"testing"
	"time"
)

func TestSelectionReadsServerMembershipAndProjectsBeforePaging(t *testing.T) {
	service, _ := fixture(t, completedSource(0))
	ctx := context.Background()
	now := time.Now()
	id, err := service.store.create(ctx, Input{Query: indexer.Query{Keywords: "ABCD-123"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Selection(ctx, id, nil); !errors.Is(err, ErrBusy) {
		t.Fatal("running intent was eligible for file preview", err)
	}
	if _, err = service.store.claim(ctx, id, now); err != nil {
		t.Fatal(err)
	}
	zero, big := int64(0), int64(200)
	batch := indexer.Batch{Releases: []indexer.Release{{Title: "first", Kind: indexer.Builtin, Size: &zero, Resource: indexer.ResourceRef{ID: "opaque-first"}}, {Title: "second", Kind: indexer.Builtin, Size: nil, Resource: indexer.ResourceRef{ID: "opaque-second"}}, {Title: "lock on another page", Kind: indexer.Builtin, Size: &big, Resource: indexer.ResourceRef{ID: "opaque-third"}}}}
	if _, err = service.store.appendBatch(ctx, id, batch, now); err != nil {
		t.Fatal(err)
	}
	if err = service.store.finish(ctx, id, "COMPLETED", "", false, false, now); err != nil {
		t.Fatal(err)
	}
	selection, err := service.Selection(ctx, id, nil)
	if err != nil || len(selection.Candidates) != 3 {
		t.Fatal("server selection missing candidates", err)
	}
	ranked := selection.RankingCandidates()
	if ranked[0].ResourceID != "opaque-first" || ranked[0].GUID != selection.Candidates[0].Candidate.ID || ranked[1].Size != -1 {
		t.Fatal("ranking lost opaque identity/unknown metadata")
	}
	page, total, err := selection.Project(ranked[2:], Page{Limit: 1})
	if err != nil || total != 1 || page[0].Title != "lock on another page" {
		t.Fatal("filtered ranking was paged before global LOCK result", err)
	}
	page, total, err = selection.Project(ranked, Page{Limit: 2, Sort: "SIZE_DESC"})
	if err != nil || total != 3 || *page[0].Size != 200 || *page[1].Size != 0 {
		t.Fatal("preview sort confused zero and unknown size", err)
	}
	empty, err := service.Selection(ctx, id, []string{})
	if err != nil || len(empty.Candidates) != 0 {
		t.Fatal("explicit empty selection became all candidates", err)
	}
	subset, err := service.Selection(ctx, id, []string{selection.Candidates[1].Candidate.ID})
	if err != nil || len(subset.Candidates) != 1 || subset.Candidates[0].ResourceID != "opaque-second" {
		t.Fatal("subset membership lost", err)
	}
	if _, err = service.Selection(ctx, id, []string{"foreign"}); !errors.Is(err, ErrExpired) {
		t.Fatal("foreign candidate accepted", err)
	}
	if _, err = service.Selection(ctx, id, []string{ranked[0].GUID, ranked[0].GUID}); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate candidate accepted", err)
	}
	ranked[0].ResourceID = "replacement"
	if _, _, err = selection.Project(ranked, Page{}); !errors.Is(err, indexer.ErrReference) {
		t.Fatal("ranking could replace server resource metadata", err)
	}
}
