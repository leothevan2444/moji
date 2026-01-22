package stash

import (
	"context"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestFindPerformerByID(t *testing.T) {
	client := NewClient(host, apiKey)
	performer, err := client.FindPerformerByID(context.Background(), "71")
	if err != nil {
		t.Errorf("failed to find performer: %v", err)
	}
	fmt.Printf("Found performer: %+v\n", performer)
}

func TestFindPerformers(t *testing.T) {
	client := NewClient(host, apiKey)
	performerFilter := graphql.PerformerFilterType{
		Age: &graphql.IntCriterionInput{
			Value:    25,
			Modifier: graphql.CriterionModifierEquals,
		},
	}
	performers, err := client.FindPerformers(context.Background(), &performerFilter, nil, nil, []string{"71", "75"})
	if err != nil {
		t.Errorf("failed to find performers: %v", err)
	}
	for _, performer := range performers {
		fmt.Printf("Performer: %+v\n", performer)
	}
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
