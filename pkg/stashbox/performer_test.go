package stashbox

import (
	"context"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestFindPerformerByID(t *testing.T) {
	client := NewClient(apikey)
	performer, err := client.FindPerformerByID(context.Background(), "f8fe36db-b1a3-42af-803c-621d9638ff60")
	if err != nil {
		t.Errorf("failed to find performer: %v", err)
	}
	fmt.Printf("Found performer: %+v\n", performer)
}

func TestSearchPerformer(t *testing.T) {
	client := NewClient(apikey)
	performers, err := client.SearchPerformer(context.Background(), "鷲尾めい")
	if err != nil {
		t.Errorf("failed to find performers: %v", err)
	}
	for _, performer := range performers {
		fmt.Printf("Performer: %+v\n", performer)
	}
}

func TestQueryPerformers(t *testing.T) {
	client := NewClient(apikey)
	name := "鷲尾めい"
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
		t.Errorf("failed to get all performers: %v", err)
	}
	for _, performer := range performers {
		fmt.Printf("Performer: %+v\n", performer)
	}
}
