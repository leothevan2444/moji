package subscription

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/internal/performer"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestNormalizePerformerBatchIDs(t *testing.T) {
	ids, err := normalizePerformerBatchIDs([]string{" p1 ", "p1", "p2", ""})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(ids) != 2 || ids[0] != "p1" || ids[1] != "p2" {
		t.Fatalf("unexpected ids: %#v", ids)
	}
	if _, err := normalizePerformerBatchIDs(nil); !errors.Is(err, ErrPerformerBatchEmpty) {
		t.Fatalf("expected empty error, got %v", err)
	}
	tooMany := make([]string, MaxPerformerBatchSize+1)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("p-%d", index)
	}
	if _, err := normalizePerformerBatchIDs(tooMany); !errors.Is(err, ErrPerformerBatchTooLarge) {
		t.Fatalf("expected too large error, got %v", err)
	}
}

func TestPerformerBatchSubscribeAndUnsubscribeReturnPartialResults(t *testing.T) {
	stash := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "One", CustomFields: map[string]any{}},
		"p2": {ID: "p2", Name: "Two", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}
	service, err := newServiceForTest(stash, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	subscribed, err := service.SubscribePerformers(context.Background(), []string{"p1", "p2", "missing"})
	if err != nil {
		t.Fatalf("subscribe batch: %v", err)
	}
	if subscribed.Summary.SucceededCount != 1 || subscribed.Summary.SkippedCount != 2 || subscribed.Summary.FailedCount != 0 {
		t.Fatalf("unexpected subscribe summary: %#v", subscribed.Summary)
	}
	if subscribed.Results[0].ReasonCode != PerformerBatchReasonSubscribed || subscribed.Results[1].ReasonCode != PerformerBatchReasonAlreadySubscribed || subscribed.Results[2].ReasonCode != PerformerBatchReasonPerformerNotFound {
		t.Fatalf("unexpected subscribe results: %#v", subscribed.Results)
	}

	unsubscribed, err := service.UnsubscribePerformers(context.Background(), []string{"p1", "missing"})
	if err != nil {
		t.Fatalf("unsubscribe batch: %v", err)
	}
	if unsubscribed.Summary.SucceededCount != 1 || unsubscribed.Summary.SkippedCount != 1 {
		t.Fatalf("unexpected unsubscribe summary: %#v", unsubscribed.Summary)
	}
	if unsubscribed.Results[0].Performer == nil || unsubscribed.Results[0].Performer.Subscribed {
		t.Fatalf("expected unsubscribed performer snapshot: %#v", unsubscribed.Results[0])
	}
}

func TestBuildSubscribedPerformersUsesProvidedSnapshot(t *testing.T) {
	service, err := newServiceForTest(&fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{}}, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	items, err := service.BuildSubscribedPerformers(context.Background(), []performer.Performer{{ID: "p1", Name: "One", Subscribed: true}, {ID: "p2", Name: "Two"}})
	if err != nil {
		t.Fatalf("build subscribed performers: %v", err)
	}
	if len(items) != 1 || items[0].Performer.ID != "p1" {
		t.Fatalf("unexpected subscribed performers: %#v", items)
	}
}
