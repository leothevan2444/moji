package qbittorrent

import (
	"context"
	"testing"
)

func TestStartSearch(t *testing.T) {
	id, err := client.StartSearch(context.Background(), "superman", "all", "all")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}
	t.Logf("Search started with ID: %d", id)
}

func TestStopSearch(t *testing.T) {
	id, err := client.StartSearch(context.Background(), "superman", "all", "all")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}

	err = client.StopSearch(context.Background(), id)
	if err != nil {
		t.Fatalf("Failed to stop search: %v", err)
	}
	t.Logf("Search with ID %d stopped successfully", id)
}

func TestGetSearchStatus(t *testing.T) {
	id, err := client.StartSearch(context.Background(), "superman", "all", "all")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}

	status, err := client.GetSearchStatus(context.Background(), &id)
	if err != nil {
		t.Fatalf("Failed to get search status: %v", err)
	}
	for _, status := range status {
		t.Logf("Search status for ID %d: %s", id, status.Status)
	}
}

func TestDeleteSearch(t *testing.T) {
	id, err := client.StartSearch(context.Background(), "superman", "all", "all")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}

	err = client.DeleteSearch(context.Background(), id)
	if err != nil {
		t.Fatalf("Failed to delete search: %v", err)
	}
	t.Logf("Search with ID %d deleted successfully", id)
}

func TestGetSearchResults(t *testing.T) {
	id, err := client.StartSearch(context.Background(), "superman", "all", "all")
	if err != nil {
		t.Fatalf("Failed to start search: %v", err)
	}

	status, results, err := client.GetSearchResults(context.Background(), id)
	if err != nil {
		t.Fatalf("Failed to get search results: %v", err)
	}

	t.Logf("Search status: %s", status)
	for _, result := range results {
		t.Logf("Result: %+v", result)
	}
}
