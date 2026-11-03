package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

func executeDefinitionGraphQLInto(t *testing.T, resolver *Resolver, query string, target any) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatal(err)
	}
	server := NewGraphQLServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GraphQL response %d", response.Code)
	}
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatal(err)
	}
}

type apiDefinitionSource struct {
	checks    atomic.Int32
	started   chan struct{}
	cancelled chan struct{}
}

func (s *apiDefinitionSource) Latest(context.Context, string) (indexerdefs.Revision, bool, error) {
	s.checks.Add(1)
	return indexerdefs.Revision{Commit: strings.Repeat("a", 40)}, false, nil
}
func (s *apiDefinitionSource) Download(ctx context.Context, _ string, _ io.Writer) error {
	close(s.started)
	<-ctx.Done()
	close(s.cancelled)
	return ctx.Err()
}

type apiDefinitionValidator struct{}
type apiDefinitionGuard struct{}

func (apiDefinitionValidator) Validate(context.Context, *indexerdefs.Bundle) (indexerdefs.ActivationGuard, error) {
	return apiDefinitionGuard{}, nil
}
func (apiDefinitionGuard) WithCurrent(ctx context.Context, commit func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return commit()
}

func TestDefinitionGraphQLPersistsJobAcrossRequestEndAndSharesCheckBudget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	source := &apiDefinitionSource{started: make(chan struct{}), cancelled: make(chan struct{})}
	service, err := indexerdefs.NewService(manager, source, apiDefinitionValidator{})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	resolver := NewResolver(nil, nil, nil, "test")
	resolver.DefinitionLibrary = service
	var initial struct {
		Errors []any
		Data   struct {
			Status struct{ ActiveRevision, ErrorCode string } `json:"indexerDefinitionStatus"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `{indexerDefinitionStatus{activeRevision errorCode installed{revision}}}`, &initial)
	if len(initial.Errors) != 0 || initial.Data.Status.ActiveRevision != "" || initial.Data.Status.ErrorCode != indexerdefs.ErrNotInstalled.Error() || source.checks.Load() != 0 {
		t.Fatal("status query fetched rules or lost uninstalled state")
	}
	var check struct{ Errors []any }
	for i := 0; i < 2; i++ {
		executeDefinitionGraphQLInto(t, resolver, `mutation {checkIndexerDefinitionUpdate{availableRevision}}`, &check)
		if len(check.Errors) > 0 {
			t.Fatal(check.Errors)
		}
	}
	if source.checks.Load() != 1 {
		t.Fatal("interactive checks bypassed throttling")
	}
	var begun struct {
		Errors []any
		Data   struct {
			Job struct{ ID, State string } `json:"installIndexerDefinitions"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation {installIndexerDefinitions(input:{expectedActiveRevision:""}){id state targetRevision correlationId}}`, &begun)
	if len(begun.Errors) > 0 || begun.Data.Job.ID == "" {
		t.Fatal("install was not accepted", begun.Errors)
	}
	select {
	case <-source.started:
	case <-time.After(time.Second):
		t.Fatal("accepted job did not run")
	}
	select {
	case <-source.cancelled:
		t.Fatal("completed HTTP request cancelled accepted download")
	default:
	}
	job, err := service.Job(ctx, begun.Data.Job.ID)
	if err != nil || job.State != indexerdefs.Downloading {
		t.Fatal("accepted job not persisted", err)
	}
	stream, err := resolver.Subscription().IndexerDefinitionEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case snapshot := <-stream:
		if snapshot.Job == nil || snapshot.Job.ID != job.ID {
			t.Fatal("subscription did not recover durable job")
		}
	case <-time.After(time.Second):
		t.Fatal("missing initial persisted snapshot")
	}
	var cancelled struct {
		Errors []any
		Data   struct {
			Job struct{ State string } `json:"cancelIndexerDefinitionUpdate"`
		}
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation {cancelIndexerDefinitionUpdate(id:"`+job.ID+`"){id state}}`, &cancelled)
	if len(cancelled.Errors) > 0 || cancelled.Data.Job.State != "CANCELLED" {
		t.Fatal("cancellation not durable", cancelled.Errors)
	}
	executeDefinitionGraphQLInto(t, resolver, `mutation {cancelIndexerDefinitionUpdate(id:"`+job.ID+`"){id state}}`, &cancelled)
	if len(cancelled.Errors) > 0 || cancelled.Data.Job.State != "CANCELLED" {
		t.Fatal("repeated cancellation lost durable terminal outcome")
	}
	service.CloseSubscriptions()
	cancel()
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-stream:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("subscription did not close")
		}
	}
}

func TestDefinitionGraphQLUnconfiguredErrorHasStableCode(t *testing.T) {
	var response struct {
		Errors []struct {
			Message    string
			Extensions struct{ Code, CorrelationID string }
		}
	}
	executeDefinitionGraphQLInto(t, NewResolver(nil, nil, nil, "test"), `{indexerDefinitionStatus{activeRevision}}`, &response)
	if len(response.Errors) != 1 || response.Errors[0].Message != "request failed" || response.Errors[0].Extensions.Code != indexerdefs.ErrUnavailable.Error() || response.Errors[0].Extensions.CorrelationID == "" {
		t.Fatal("definition errors did not use the public error contract")
	}
}
