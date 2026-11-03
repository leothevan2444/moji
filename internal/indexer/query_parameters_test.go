package indexer

import (
	"context"
	"errors"
	"testing"
)

func TestInvalidQueryParametersFailBeforeSourceSnapshot(t *testing.T) {
	calls := 0
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) {
		calls++
		return Registry{}, nil
	}), testResources{}, fakeAdapter{}, fakeAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, parameters := range []map[string]string{
		{"Q": "replacement"}, {"Type": "movie"}, {"Limit": "100000"}, {"Offset": "99"},
		{"IsSearch": "True"}, {"Categories": "6000"}, {"APIKey": "synthetic"},
		{"IMDBIDShort": "forged"}, {"unknown": "value"}, {"Year": "2026", "year": "2025"},
	} {
		_, err := service.Search(context.Background(), Query{Keywords: "ABCD-123", Parameters: parameters})
		if !errors.Is(err, ErrInvalidQuery) || calls != 0 {
			t.Fatalf("invalid query reached source snapshot: %v calls=%d", err, calls)
		}
	}
}

func TestValidateQueryCanonicalizesNamesWithoutAliasingCaller(t *testing.T) {
	parameters := map[string]string{"year": "002026", "imdbid": "tt1234567", "ep": "9"}
	query, err := ValidateQuery(Query{Parameters: parameters})
	if err != nil || query.Parameters["Year"] != "002026" || query.Parameters["IMDBID"] != "tt1234567" || query.Parameters["Ep"] != "9" {
		t.Fatal("query field casing changed or lost values", query, err)
	}
	parameters["year"] = "2025"
	if query.Parameters["Year"] != "002026" {
		t.Fatal("frozen query aliases caller parameters")
	}
}
