package stash

import (
	"context"
	"fmt"
	"testing"
)

func TestFindPerformer(t *testing.T) {
	client := NewClient(host, apiKey)
	performer, err := client.FindPerformerByID(context.Background(), "71")
	if err != nil {
		t.Errorf("failed to find performer: %v", err)
	}
	fmt.Printf("Found performer: %+v\n", performer)
}

func TestAllPerformer(t *testing.T) {
	client := NewClient(host, apiKey)
	performers, err := client.AllPerformers(context.Background())
	if err != nil {
		t.Errorf("failed to get all performers: %v", err)
	}
	for _, performer := range performers {
		fmt.Printf("Performer: %+v\n", performer)
	}
}
