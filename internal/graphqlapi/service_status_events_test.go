package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/stats"
)

func TestServiceStatusEventsResolverMapsAndCloses(t *testing.T) {
	bus := stats.NewServiceStatusEventBus(2)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	resolver := &subscriptionResolver{Resolver: &Resolver{ServiceStatusEventSource: bus}}
	channel, err := resolver.ServiceStatusEvents(ctx)
	if err != nil {
		t.Fatalf("ServiceStatusEvents: %v", err)
	}
	observed := time.Date(2026, time.July, 13, 8, 0, 0, 0, time.UTC)
	bus.Publish(&stats.ServiceStatusEvent{
		Services:   []stats.ExternalService{stats.ExternalServiceStash, stats.ExternalServiceJackett},
		ObservedAt: observed,
	})
	select {
	case event := <-channel:
		if event.Sequence != 1 || len(event.Services) != 2 || event.ObservedAt != observed.Format(time.RFC3339) {
			t.Fatalf("unexpected mapped event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not forward event")
	}
	cancel()
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected resolver channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not stop after cancellation")
	}
}
