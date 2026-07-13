package graphqlapi

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/stats"
	"github.com/leothevan2444/moji/internal/subscription"
)

func TestPerformerSubscriptionEventsResolverMapsStateAndDeletion(t *testing.T) {
	bus := subscription.NewPerformerSubscriptionEventBus(2)
	defer bus.Close()
	resolver := &subscriptionResolver{Resolver: &Resolver{PerformerSubscriptionEventSource: bus}}
	ctx, cancel := context.WithCancel(context.Background())
	channel, err := resolver.PerformerSubscriptionEvents(ctx)
	if err != nil {
		t.Fatalf("PerformerSubscriptionEvents: %v", err)
	}
	bus.Publish(&subscription.PerformerSubscriptionEvent{
		Type: subscription.PerformerSubscriptionEventUpdated, PerformerID: "p1",
		State: &subscription.SubscribedPerformer{Performer: performer.Performer{ID: "p1", Name: "Performer", Subscribed: true}},
	})
	event := <-channel
	if event.Sequence != 1 || event.PerformerID != "p1" || event.State == nil || event.State.Performer.ID != "p1" {
		t.Fatalf("unexpected mapped event: %+v", event)
	}
	cancel()
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected resolver channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("resolver channel did not close")
	}
}

func TestServiceStatusEventsOverGraphQLTransportWebSocket(t *testing.T) {
	bus := stats.NewServiceStatusEventBus(2)
	defer bus.Close()
	client := newGraphQLWebSocketTestClient(t, &Resolver{ServiceStatusEventSource: bus})
	client.subscribe(t, "service-events", `subscription { serviceStatusEvents { sequence services observedAt } }`)
	waitForSubscriber(t, bus.SubscriberCount)
	bus.Publish(&stats.ServiceStatusEvent{Services: []stats.ExternalService{stats.ExternalServiceStash}, ObservedAt: time.Now().UTC()})
	var payload struct {
		Data struct {
			Event struct {
				Sequence int      `json:"sequence"`
				Services []string `json:"services"`
			} `json:"serviceStatusEvents"`
		} `json:"data"`
	}
	client.readNext(t, "service-events", &payload)
	if payload.Data.Event.Sequence != 1 || len(payload.Data.Event.Services) != 1 || payload.Data.Event.Services[0] != "STASH" {
		t.Fatalf("unexpected service status payload: %+v", payload)
	}
}

func TestLogEventsOverGraphQLTransportWebSocket(t *testing.T) {
	logger, err := logging.New(logging.Options{ConsoleWriter: &bytes.Buffer{}, Level: "info"})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer logger.Close()
	client := newGraphQLWebSocketTestClient(t, &Resolver{LogEventSource: logger})
	client.subscribe(t, "log-events", `subscription { logEvents { sequence entry { sequence level message } } }`)
	waitForSubscriber(t, logger.SubscriberCount)
	logger.Slog().Info("streamed log")
	var payload struct {
		Data struct {
			Event struct {
				Sequence int `json:"sequence"`
				Entry    struct {
					Message string `json:"message"`
				} `json:"entry"`
			} `json:"logEvents"`
		} `json:"data"`
	}
	client.readNext(t, "log-events", &payload)
	if payload.Data.Event.Sequence != 1 || payload.Data.Event.Entry.Message != "streamed log" {
		t.Fatalf("unexpected log payload: %+v", payload)
	}
}

func TestPerformerSubscriptionEventsOverGraphQLTransportWebSocket(t *testing.T) {
	bus := subscription.NewPerformerSubscriptionEventBus(2)
	defer bus.Close()
	client := newGraphQLWebSocketTestClient(t, &Resolver{PerformerSubscriptionEventSource: bus})
	client.subscribe(t, "performer-events", `subscription { performerSubscriptionEvents { sequence type performerId state { performer { id name subscribed } pendingReleaseCount } } }`)
	waitForSubscriber(t, func() int { return bus.Stats().Subscribers })
	bus.Publish(&subscription.PerformerSubscriptionEvent{
		Type: subscription.PerformerSubscriptionEventCreated, PerformerID: "p1",
		State: &subscription.SubscribedPerformer{Performer: performer.Performer{ID: "p1", Name: "Performer", Subscribed: true}},
	})
	var payload struct {
		Data struct {
			Event struct {
				Sequence    int    `json:"sequence"`
				Type        string `json:"type"`
				PerformerID string `json:"performerId"`
			} `json:"performerSubscriptionEvents"`
		} `json:"data"`
	}
	client.readNext(t, "performer-events", &payload)
	if payload.Data.Event.Sequence != 1 || payload.Data.Event.Type != "CREATED" || payload.Data.Event.PerformerID != "p1" {
		t.Fatalf("unexpected performer subscription payload: %+v", payload)
	}
}
