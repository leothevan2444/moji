package graphqlapi

import (
	"context"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/logging"
)

type fakeLogEventSource struct {
	events chan *logging.LogEvent
}

func (f *fakeLogEventSource) Subscribe(context.Context) <-chan *logging.LogEvent {
	return f.events
}

func TestLogEventsResolverMapsEntries(t *testing.T) {
	observedAt := time.Date(2026, time.July, 13, 8, 30, 0, 0, time.UTC)
	source := &fakeLogEventSource{events: make(chan *logging.LogEvent, 1)}
	resolver := &Resolver{LogEventSource: source}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := resolver.Subscription().LogEvents(ctx)
	if err != nil {
		t.Fatalf("subscribe to log events: %v", err)
	}
	source.events <- &logging.LogEvent{
		Sequence: 7,
		Entry: logging.Entry{
			Sequence: 7,
			Time:     observedAt,
			Level:    "warning",
			Message:  "slow upstream",
		},
	}

	select {
	case event := <-events:
		if event.Sequence != 7 || event.Entry.Sequence != 7 {
			t.Fatalf("unexpected sequence: %+v", event)
		}
		if event.Entry.Level != model.LogLevelWarning || event.Entry.Message != "slow upstream" {
			t.Fatalf("unexpected mapped entry: %+v", event.Entry)
		}
		if event.Entry.Time != formatTime(observedAt) {
			t.Fatalf("unexpected event time: %q", event.Entry.Time)
		}
	case <-time.After(time.Second):
		t.Fatal("resolver did not forward log event")
	}
}

func TestLogEventsResolverStopsAfterContextCancellation(t *testing.T) {
	source := &fakeLogEventSource{events: make(chan *logging.LogEvent)}
	resolver := &Resolver{LogEventSource: source}
	ctx, cancel := context.WithCancel(context.Background())
	events, err := resolver.Subscription().LogEvents(ctx)
	if err != nil {
		t.Fatalf("subscribe to log events: %v", err)
	}
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected resolver output channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("resolver output did not close after cancellation")
	}
}
