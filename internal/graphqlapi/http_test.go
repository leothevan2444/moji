package graphqlapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/leothevan2444/moji/internal/graphqlapi/generated"
)

func TestGraphQLHealthVersionSmoke(t *testing.T) {
	resolver := NewResolver(nil, nil, nil, nil, "test-version")
	graphqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	mux := http.NewServeMux()
	mux.Handle("POST /graphql", graphqlHandler)

	body := bytes.NewBufferString(`{"query":"{ health { ok message } version }"}`)
	req := httptest.NewRequest(http.MethodPost, "/graphql", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data struct {
			Health struct {
				OK      bool   `json:"ok"`
				Message string `json:"message"`
			} `json:"health"`
			Version string `json:"version"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("expected no GraphQL errors, got %+v", resp.Errors)
	}
	if !resp.Data.Health.OK || resp.Data.Health.Message != "ok" {
		t.Fatalf("unexpected health response: %+v", resp.Data.Health)
	}
	if resp.Data.Version != "test-version" {
		t.Fatalf("expected version %q, got %q", "test-version", resp.Data.Version)
	}
}
