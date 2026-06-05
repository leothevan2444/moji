package stash

import (
	"context"
	"os"
	"testing"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestFindPerformerByID(t *testing.T) {
	client := requireStashClient(t)
	performerID := os.Getenv("MOJI_STASH_TEST_PERFORMER_ID")
	if performerID == "" {
		t.Skip("set MOJI_STASH_TEST_PERFORMER_ID to run this Stash integration test")
	}

	performer, err := client.FindPerformerByID(context.Background(), performerID)
	if err != nil {
		t.Fatalf("failed to find performer: %v", err)
	}
	t.Logf("Found performer: %+v", performer)
}

func TestFindPerformers(t *testing.T) {
	client := requireStashClient(t)
	performerID := os.Getenv("MOJI_STASH_TEST_PERFORMER_ID")
	if performerID == "" {
		t.Skip("set MOJI_STASH_TEST_PERFORMER_ID to run this Stash integration test")
	}

	performerFilter := graphql.PerformerFilterType{
		Age: &graphql.IntCriterionInput{
			Value:    25,
			Modifier: graphql.CriterionModifierEquals,
		},
	}
	performers, err := client.FindPerformers(context.Background(), &performerFilter, nil, nil, []string{performerID})
	if err != nil {
		t.Fatalf("failed to find performers: %v", err)
	}
	for _, performer := range performers {
		t.Logf("Performer: %+v", performer)
	}
}

func TestAllPerformer(t *testing.T) {
	client := requireStashClient(t)
	performers, err := client.AllPerformers(context.Background())
	if err != nil {
		t.Fatalf("failed to get all performers: %v", err)
	}
	for _, performer := range performers {
		t.Logf("Performer: %+v", performer)
	}
}
