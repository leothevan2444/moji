package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexeradmin"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type homeIndexerSummarySource struct{ snapshot indexeradmin.Summary }

func (s *homeIndexerSummarySource) Summary(context.Context) (indexeradmin.Summary, error) {
	return s.snapshot, nil
}

func TestHomeIndexerSummaryGraphQLPreservesUnknownAndConfirmedZero(t *testing.T) {
	source := &homeIndexerSummarySource{snapshot: indexeradmin.Summary{Revision: "config-revision", BuiltinEnabled: 1, BuiltinUnverified: 1, Issues: []indexeradmin.SummaryIssue{}, Jackett: &indexeradmin.JackettSummary{}}}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.HomeIndexers = source
	const query = `{homeIndexerSummary{revision total builtinEnabled builtinUnverified attention issues{id name errorCode} jackett{sites stale updatedAt errorCode}}}`
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Summary model.HomeIndexerSummary `json:"homeIndexerSummary"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	if len(response.Errors) > 0 || response.Data.Summary.Total != nil || response.Data.Summary.Jackett == nil || response.Data.Summary.Jackett.Sites != nil || response.Data.Summary.BuiltinUnverified != 1 {
		t.Fatal("unknown counts converted to zero", response.Errors)
	}
	zero := 0
	confirmed := time.Date(2026, 11, 2, 1, 2, 3, 0, time.UTC)
	source.snapshot.Total = &zero
	source.snapshot.Jackett = &indexeradmin.JackettSummary{Sites: &zero, Stale: true, UpdatedAt: &confirmed, ErrorCode: indexer.ErrTransport.Error()}
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	got := response.Data.Summary
	if len(response.Errors) > 0 || got.Total == nil || *got.Total != 0 || got.Jackett.Sites == nil || *got.Jackett.Sites != 0 || !got.Jackett.Stale || got.Jackett.UpdatedAt == nil || *got.Jackett.UpdatedAt != "2026-11-02T01:02:03Z" || got.Jackett.ErrorCode != indexer.ErrTransport.Error() {
		t.Fatal("confirmed empty/stale summary lost", response.Errors)
	}
}

func TestHomeDefinitionFailureUsesExistingJobContractAndLocalErrorProjection(t *testing.T) {
	at := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	source := &homeIndexerSummarySource{snapshot: indexeradmin.Summary{DefinitionJob: &indexerdefs.Job{ID: "job", Kind: "UPDATE", State: indexerdefs.Failed, ErrorCode: indexerdefs.ErrSource.Error(), CorrelationID: "synthetic-correlation", StartedAt: at, FinishedAt: &at, Revision: 3}}}
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.HomeIndexers = source
	var response struct {
		Errors []struct{ Message string }
		Data   struct {
			Summary model.HomeIndexerSummary `json:"homeIndexerSummary"`
		}
	}
	query := `{homeIndexerSummary{definitionJob{id state errorCode correlationId finishedAt} definitionErrorCode}}`
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	job := response.Data.Summary.DefinitionJob
	if len(response.Errors) > 0 || job == nil || job.ID != "job" || job.State != model.IndexerDefinitionJobStateFailed || job.FinishedAt == nil {
		t.Fatal("persisted job omitted from Home", response.Errors)
	}
	source.snapshot.DefinitionJob = nil
	source.snapshot.DefinitionErrorCode = indexerdefs.ErrUnavailable.Error()
	executeDefinitionGraphQLInto(t, resolver, query, &response)
	if len(response.Errors) > 0 || response.Data.Summary.DefinitionJob != nil || response.Data.Summary.DefinitionErrorCode != indexerdefs.ErrUnavailable.Error() {
		t.Fatal("job read error became false empty success", response.Errors)
	}
}
