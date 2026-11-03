package graphqlapi

import (
	"context"
	"github.com/leothevan2444/moji/internal/indexer"
	"testing"
	"time"
)

type observedIndexerEvents struct {
	bus     *indexer.Events
	stopped chan struct{}
}

func (s observedIndexerEvents) Subscribe() (<-chan indexer.Event, func(), error) {
	ch, stop, err := s.bus.Subscribe()
	return ch, func() { stop(); close(s.stopped) }, err
}

func TestIndexerEventSubscriptionStopsWhenSourceClosesWithUnreadResults(t *testing.T) {
	bus := indexer.NewEvents()
	resolver := NewResolver(nil, nil, nil, "test")
	stopped := make(chan struct{})
	resolver.IndexerEventSource = observedIndexerEvents{bus: bus, stopped: stopped}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	results, err := resolver.indexerEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ready := <-results
	if ready.StreamID == "" || ready.Sequence != 0 {
		t.Fatal("missing initial boundary")
	}
	for i := 0; i < 1000; i++ {
		bus.Publish(indexer.EventSite)
	}
	filled := time.NewTicker(time.Millisecond)
	defer filled.Stop()
	fillDeadline := time.After(3 * time.Second)
	for len(results) == 0 {
		select {
		case <-filled.C:
		case <-fillDeadline:
			t.Fatal("API result queue did not fill")
		}
	}
	bus.Publish(indexer.EventSite)
	bus.Close()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("source shutdown waited for a stalled API consumer")
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-results:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("source shutdown blocked behind unread API results")
		}
	}
}
