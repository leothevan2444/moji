package indexercheck

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
)

type waitingConnection struct {
	fakeExecution
	id      string
	entered chan<- string
	release <-chan struct{}
}

func (e *waitingConnection) Connect(ctx context.Context) error {
	e.entered <- e.id
	select {
	case <-e.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func awaitActivity(t *testing.T, ch <-chan []Activity, predicate func([]Activity) bool) []Activity {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case snapshot, ok := <-ch:
			if !ok {
				t.Fatal("activity closed before expected state")
			}
			if predicate(snapshot) {
				return snapshot
			}
		case <-deadline:
			t.Fatal("activity transition did not arrive")
		}
	}
}
func activityState(snapshot []Activity, id string) string {
	for _, item := range snapshot {
		if item.RequestID == id {
			return item.State
		}
	}
	return ""
}
func TestConnectionQueueIsFIFOAndCancelsBeforeNetworkExecution(t *testing.T) {
	entered := make(chan string, 8)
	prepared := make(chan string, 8)
	releases := map[string]chan struct{}{}
	for _, id := range []string{"a", "b", "c", "d", "e", "f"} {
		releases[id] = make(chan struct{})
	}
	var version atomic.Int64
	version.Store(1)
	opened := make(chan indexer.InstanceSnapshot, 8)
	s, _ := New(context.Background(), sourceFunc(func(_ context.Context, input Input) (Prepared, error) {
		snapshot := indexer.InstanceSnapshot{ID: input.InstanceID, Kind: indexer.Builtin, ConfigRevision: fmt.Sprint(version.Load())}
		prepared <- input.InstanceID
		return Prepared{Instance: snapshot}, nil
	}), factoryFunc(func(_ context.Context, instance indexer.InstanceSnapshot) (Execution, error) {
		opened <- instance
		return &waitingConnection{id: instance.ID, entered: entered, release: releases[instance.ID]}, nil
	}))
	t.Cleanup(s.Close)
	activity, stop, err := s.SubscribeActivity("client")
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	done := make(chan error, 6)
	start := func(ctx context.Context, id string) {
		go func() {
			_, err := s.Check(ctx, Input{InstanceID: id, ClientID: "client", RequestID: id, ConnectionOnly: true})
			done <- err
		}()
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		start(context.Background(), id)
	}
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("initial probes did not start")
		}
	}
	start(context.Background(), "e")
	awaitActivity(t, activity, func(items []Activity) bool { return activityState(items, "e") == "QUEUED" })
	cancelled, cancel := context.WithCancel(context.Background())
	defer cancel()
	start(cancelled, "f")
	awaitActivity(t, activity, func(items []Activity) bool { return activityState(items, "f") == "QUEUED" })
	for i := 0; i < 4; i++ {
		select {
		case <-prepared:
		case <-time.After(time.Second):
			t.Fatal("initial configuration was not prepared")
		}
	}
	// Reconnecting observes the current queue, rather than relying on old events.
	replay, releaseReplay, _ := s.SubscribeActivity("client")
	if snapshot := <-replay; activityState(snapshot, "e") != "QUEUED" || activityState(snapshot, "f") != "QUEUED" {
		t.Fatal(snapshot)
	}
	releaseReplay()
	if _, err = s.Check(context.Background(), Input{InstanceID: "f", ConnectionOnly: true}); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("duplicate queued site admitted", err)
	}
	cancel()
	awaitActivity(t, activity, func(items []Activity) bool { return activityState(items, "f") == "" })
	version.Store(2)
	close(releases["a"])
	select {
	case id := <-entered:
		if id != "e" {
			t.Fatal("queue was not FIFO", id)
		}
	case <-time.After(time.Second):
		t.Fatal("queued probe never started")
	}
	awaitActivity(t, activity, func(items []Activity) bool { return activityState(items, "e") == "RUNNING" })
	found := false
	for i := 0; i < 5; i++ {
		instance := <-opened
		if instance.ID == "e" {
			found = true
			if instance.ConfigRevision != "2" {
				t.Fatal("queued probe did not validate the current account", instance)
			}
		}
	}
	if !found {
		t.Fatal("missing queued probe")
	}
	s.Close()
	for i := 0; i < 6; i++ {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("shutdown did not join queued and running requests")
		}
	}
}
func TestConnectionQueueCapacityAndSubscriberBounds(t *testing.T) {
	s, _, _ := checkFixture(t)
	entries := make([]*reservation, 0, maxRunningChecks+maxQueuedChecks)
	for i := 0; i < maxRunningChecks+maxQueuedChecks; i++ {
		entry, err := s.reserve(context.Background(), Input{InstanceID: fmt.Sprint(i), ConnectionOnly: true})
		if err != nil {
			t.Fatal(i, err)
		}
		entries = append(entries, entry)
	}
	if _, err := s.reserve(context.Background(), Input{InstanceID: "overflow", ConnectionOnly: true}); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal(err)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		s.release(entries[i])
	}
	stops := []func(){}
	for i := 0; i < 64; i++ {
		_, stop, err := s.SubscribeActivity(fmt.Sprint(i))
		if err != nil {
			t.Fatal("subscriber capacity", i, err)
		}
		stops = append(stops, stop)
	}
	if _, _, err := s.SubscribeActivity("overflow"); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("unbounded activity subscriptions", err)
	}
	for _, stop := range stops {
		stop()
	}
	if s.running != 0 || len(s.active) != 0 || len(s.queue) != 0 || len(s.requests) != 0 {
		t.Fatal("reservations leaked")
	}
}

func TestQueuedTimeoutDoesNotOpenAnExecutionAndReportsAStableFailure(t *testing.T) {
	s, execution, released := checkFixture(t)
	holders := []*reservation{}
	for i := 0; i < maxRunningChecks; i++ {
		entry, err := s.reserve(context.Background(), Input{InstanceID: fmt.Sprint(i), ConnectionOnly: true})
		if err != nil {
			t.Fatal(err)
		}
		holders = append(holders, entry)
	}
	defer func() {
		for _, entry := range holders {
			s.release(entry)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	report, err := s.Check(ctx, Input{InstanceID: "queued", ConnectionOnly: true})
	if err != nil || len(report.Stages) != 5 || report.Stages[4].ErrorCode != "INDEXER_TIMEOUT" {
		t.Fatal(report, err)
	}
	if execution.connect != 0 || execution.closed != 0 || released.Load() != 0 {
		t.Fatal("queued timeout prepared or opened a site execution")
	}
}
