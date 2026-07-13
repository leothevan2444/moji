package logging

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func newEventTestLogger(t *testing.T) *Logger {
	t.Helper()
	logger, err := New(Options{ConsoleWriter: &bytes.Buffer{}, Level: "debug", MaxEntries: 500})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	return logger
}

func TestLogEventsReachMultipleSubscribersInSequence(t *testing.T) {
	logger := newEventTestLogger(t)
	first := logger.Subscribe(context.Background())
	second := logger.Subscribe(context.Background())

	logger.logf(parseLevel("info"), "first")
	logger.logf(parseLevel("warn"), "second")

	for name, events := range map[string]<-chan *LogEvent{"first": first, "second": second} {
		one := <-events
		two := <-events
		if one.Sequence != 1 || one.Entry.Sequence != 1 || one.Entry.Message != "first" {
			t.Fatalf("%s subscriber received unexpected first event: %+v", name, one)
		}
		if two.Sequence != 2 || two.Entry.Sequence != 2 || two.Entry.Message != "second" {
			t.Fatalf("%s subscriber received unexpected second event: %+v", name, two)
		}
	}
	if stats := logger.EventStats(); stats.Subscribers != 2 || stats.Published != 2 || stats.Dropped != 0 || stats.Sequence != 2 {
		t.Fatalf("unexpected event stats: %+v", stats)
	}
}

func TestLogEventSubscriberIsRemovedAfterContextCancellation(t *testing.T) {
	logger := newEventTestLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	events := logger.Subscribe(ctx)
	if logger.SubscriberCount() != 1 {
		t.Fatalf("expected one subscriber, got %d", logger.SubscriberCount())
	}
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected subscriber channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber channel did not close after cancellation")
	}
	if logger.SubscriberCount() != 0 {
		t.Fatalf("expected no subscribers, got %d", logger.SubscriberCount())
	}
}

func TestSlowLogSubscriberDoesNotBlockOrRecursivelyLogDrops(t *testing.T) {
	logger := newEventTestLogger(t)
	_ = logger.Subscribe(context.Background())

	done := make(chan struct{})
	go func() {
		for index := 0; index < defaultLogEventBufferSize+20; index++ {
			logger.logf(parseLevel("info"), "entry %d", index)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("logging blocked on a slow subscriber")
	}
	if logger.DroppedEventCount() == 0 {
		t.Fatal("expected events to be dropped for the full subscriber buffer")
	}
	entries := logger.Entries(0, "debug")
	if len(entries) != defaultLogEventBufferSize+20 {
		t.Fatalf("drop handling generated recursive log entries: got %d", len(entries))
	}
}

func TestLoggerCloseClosesSubscribersAndIsIdempotent(t *testing.T) {
	logger := newEventTestLogger(t)
	events := logger.Subscribe(context.Background())
	if err := logger.Close(); err != nil {
		t.Fatalf("close logger: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("close logger again: %v", err)
	}
	if _, ok := <-events; ok {
		t.Fatal("expected subscriber channel to close")
	}
}
