package subscription

import (
	"context"
	"testing"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestPerformerSubscriptionEventBusPublishesToAllSubscribersInSequence(t *testing.T) {
	bus := NewPerformerSubscriptionEventBus(2)
	defer bus.Close()
	first := bus.Subscribe(context.Background())
	second := bus.Subscribe(context.Background())
	state := &SubscribedPerformer{PendingReleaseCount: 1}
	bus.Publish(&PerformerSubscriptionEvent{Type: PerformerSubscriptionEventCreated, PerformerID: "p1", State: state})
	state.PendingReleaseCount = 99

	for index, events := range []<-chan *PerformerSubscriptionEvent{first, second} {
		event := <-events
		if event.Sequence != 1 || event.PerformerID != "p1" || event.State == nil || event.State.PendingReleaseCount != 1 {
			t.Fatalf("subscriber %d received unexpected event: %+v", index, event)
		}
	}
	stats := bus.Stats()
	if stats.Subscribers != 2 || stats.Published != 1 || stats.Sequence != 1 || stats.Dropped != 0 {
		t.Fatalf("unexpected bus stats: %+v", stats)
	}
}

func TestPerformerSubscriptionEventBusCancellationAndSlowSubscriber(t *testing.T) {
	bus := NewPerformerSubscriptionEventBus(1)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	events := bus.Subscribe(ctx)
	bus.Publish(&PerformerSubscriptionEvent{Type: PerformerSubscriptionEventUpdated, PerformerID: "p1"})

	done := make(chan struct{})
	go func() {
		bus.Publish(&PerformerSubscriptionEvent{Type: PerformerSubscriptionEventUpdated, PerformerID: "p1"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on a slow subscriber")
	}
	if bus.Stats().Dropped != 1 {
		t.Fatalf("expected one dropped delivery, got %+v", bus.Stats())
	}

	<-events
	cancel()
	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected subscriber channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber channel did not close")
	}
}

func TestPerformerSubscriptionEventBusCloseIsIdempotent(t *testing.T) {
	bus := NewPerformerSubscriptionEventBus(1)
	events := bus.Subscribe(context.Background())
	bus.Close()
	bus.Close()
	if _, ok := <-events; ok {
		t.Fatal("expected subscriber channel to close")
	}
	bus.Publish(&PerformerSubscriptionEvent{Type: PerformerSubscriptionEventDeleted, PerformerID: "p1"})
}

type collectingPerformerSubscriptionPublisher struct {
	events []*PerformerSubscriptionEvent
}

func (p *collectingPerformerSubscriptionPublisher) Publish(event *PerformerSubscriptionEvent) {
	p.events = append(p.events, clonePerformerSubscriptionEvent(event))
}

type failingPutStore struct {
	*MemoryStore
}

type failingDeleteStore struct {
	*MemoryStore
}

func (s failingDeleteStore) Delete(context.Context, string) error {
	return context.Canceled
}

func (s failingPutStore) Put(context.Context, *PerformerState) error {
	return context.Canceled
}

func TestServicePublishesCreatedUpdatedAndDeletedPerformerEvents(t *testing.T) {
	stashClient := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "Performer"},
	}}
	service, err := newServiceForTest(stashClient, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	publisher := &collectingPerformerSubscriptionPublisher{}
	service.SetEventPublisher(publisher)

	if _, err := service.SubscribePerformer(context.Background(), "p1"); err != nil {
		t.Fatalf("SubscribePerformer: %v", err)
	}
	if _, err := service.RefreshSubscribedPerformer(context.Background(), "p1"); err == nil {
		t.Fatal("expected refresh without metadata endpoints to return an error")
	}
	if err := service.UnsubscribePerformer(context.Background(), "p1"); err != nil {
		t.Fatalf("UnsubscribePerformer: %v", err)
	}

	if len(publisher.events) != 3 {
		t.Fatalf("expected three events, got %#v", publisher.events)
	}
	if publisher.events[0].Type != PerformerSubscriptionEventCreated || publisher.events[0].State == nil {
		t.Fatalf("unexpected created event: %+v", publisher.events[0])
	}
	if publisher.events[1].Type != PerformerSubscriptionEventUpdated || publisher.events[1].State == nil || publisher.events[1].State.LastError == "" {
		t.Fatalf("expected persisted refresh error event, got %+v", publisher.events[1])
	}
	if publisher.events[2].Type != PerformerSubscriptionEventDeleted || publisher.events[2].State != nil {
		t.Fatalf("unexpected deleted event: %+v", publisher.events[2])
	}
}

func TestServiceDoesNotPublishRefreshWhenPersistenceFails(t *testing.T) {
	stashClient := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "Performer", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}
	service, err := newServiceForTest(stashClient, nil, nil, failingPutStore{MemoryStore: NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	publisher := &collectingPerformerSubscriptionPublisher{}
	service.SetEventPublisher(publisher)
	_, _ = service.RefreshSubscribedPerformer(context.Background(), "p1")
	if len(publisher.events) != 0 {
		t.Fatalf("persistence failure must not publish, got %#v", publisher.events)
	}
}

func TestServiceDoesNotPublishDeletionWhenCleanupFails(t *testing.T) {
	stashClient := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "Performer", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}
	service, err := newServiceForTest(stashClient, nil, nil, failingDeleteStore{MemoryStore: NewMemoryStore()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	publisher := &collectingPerformerSubscriptionPublisher{}
	service.SetEventPublisher(publisher)
	if err := service.UnsubscribePerformer(context.Background(), "p1"); err == nil {
		t.Fatal("expected cleanup failure")
	}
	if len(publisher.events) != 0 {
		t.Fatalf("failed cleanup must not publish, got %#v", publisher.events)
	}
}

func TestRefreshAllPublishesOnlyPerformerEvents(t *testing.T) {
	stashClient := &fakeStashClient{performers: map[string]*stashgraphql.PerformerFragment{
		"p1": {ID: "p1", Name: "One", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
		"p2": {ID: "p2", Name: "Two", CustomFields: map[string]any{DefaultCustomFieldKey: true}},
	}}
	service, err := newServiceForTest(stashClient, nil, nil, NewMemoryStore())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	publisher := &collectingPerformerSubscriptionPublisher{}
	service.SetEventPublisher(publisher)
	if _, err := service.RefreshAll(context.Background()); err != nil {
		t.Fatalf("RefreshAll: %v", err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("expected one event per performer without an aggregate event, got %#v", publisher.events)
	}
	for _, event := range publisher.events {
		if event.Type != PerformerSubscriptionEventUpdated {
			t.Fatalf("unexpected refresh-all event: %+v", event)
		}
	}
}
