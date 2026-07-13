package performer

import (
	"context"
	"testing"

	"github.com/leothevan2444/moji/internal/metadata"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type testStashClient struct {
	performers []*stashgraphql.PerformerFragment
}

func (s testStashClient) AllPerformers(context.Context) ([]*stashgraphql.PerformerFragment, error) {
	return s.performers, nil
}

func (testStashClient) FindPerformerByID(context.Context, string) (*stashgraphql.PerformerFragment, error) {
	return nil, nil
}

func (testStashClient) FindScenes(context.Context, *stashgraphql.SceneFilterType, *stashgraphql.FindFilterType) ([]*stashgraphql.SceneFragment, error) {
	return nil, nil
}

func TestServiceListsPerformersWithOnlyPerformerDependencies(t *testing.T) {
	service, err := NewService(
		testStashClient{performers: []*stashgraphql.PerformerFragment{
			{ID: "2", Name: "Beth"},
			{ID: "1", Name: "Alice", AliasList: []string{"A"}, CustomFields: map[string]any{DefaultCustomFieldKey: true}},
		}},
		metadata.NewService(nil, metadata.NewRegistry(nil)),
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	items, err := service.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 || items[0].ID != "1" || !items[0].Subscribed || items[1].ID != "2" {
		t.Fatalf("unexpected performers: %+v", items)
	}
}
