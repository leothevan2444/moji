package stashbox

import (
	"context"
	"os"
	"testing"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestFindPerformerByID(t *testing.T) {
	client := requireStashboxClient(t)
	performerID := os.Getenv("MOJI_STASHBOX_TEST_PERFORMER_ID")
	if performerID == "" {
		t.Skip("set MOJI_STASHBOX_TEST_PERFORMER_ID to run this StashBox integration test")
	}

	performer, err := client.FindPerformerByID(context.Background(), performerID)
	if err != nil {
		t.Fatalf("failed to find performer: %v", err)
	}
	t.Logf("Found performer: %+v", performer)
}

func TestSearchPerformer(t *testing.T) {
	client := requireStashboxClient(t)
	performerName := os.Getenv("MOJI_STASHBOX_TEST_PERFORMER_NAME")
	if performerName == "" {
		t.Skip("set MOJI_STASHBOX_TEST_PERFORMER_NAME to run this StashBox integration test")
	}

	performers, err := client.SearchPerformer(context.Background(), performerName)
	if err != nil {
		t.Fatalf("failed to find performers: %v", err)
	}
	for _, performer := range performers {
		t.Logf("Performer: %+v", performer)
	}
}

func TestQueryPerformers(t *testing.T) {
	client := requireStashboxClient(t)
	name := os.Getenv("MOJI_STASHBOX_TEST_PERFORMER_NAME")
	if name == "" {
		t.Skip("set MOJI_STASHBOX_TEST_PERFORMER_NAME to run this StashBox integration test")
	}

	query := graphql.PerformerQueryInput{
		Names: &name,
		Age: &graphql.IntCriterionInput{
			Value:    25,
			Modifier: graphql.CriterionModifierEquals,
		},
		Direction: graphql.SortDirectionEnumAsc,
		Sort:      graphql.PerformerSortEnumBirthdate,
	}
	performers, err := client.QueryPerformers(context.Background(), query)
	if err != nil {
		t.Fatalf("failed to get all performers: %v", err)
	}
	for _, performer := range performers {
		t.Logf("Performer: %+v", performer)
	}
}
