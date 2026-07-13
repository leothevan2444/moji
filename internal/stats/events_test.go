package stats

import (
	"context"
	"testing"
	"time"
)

func TestServiceStatusEventBusPublishesToAllSubscribers(t *testing.T) {
	bus := NewServiceStatusEventBus(2)
	defer bus.Close()
	first := bus.Subscribe(context.Background())
	second := bus.Subscribe(context.Background())
	bus.Publish(&ServiceStatusEvent{Services: []ExternalService{ExternalServiceStash}})

	for index, channel := range []<-chan *ServiceStatusEvent{first, second} {
		select {
		case event := <-channel:
			if event.Sequence != 1 || len(event.Services) != 1 || event.Services[0] != ExternalServiceStash || event.ObservedAt.IsZero() {
				t.Fatalf("subscriber %d received unexpected event: %#v", index, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d did not receive event", index)
		}
	}
}

func TestServiceStatusEventBusCancellationAndSlowSubscriber(t *testing.T) {
	bus := NewServiceStatusEventBus(1)
	defer bus.Close()
	ctx, cancel := context.WithCancel(context.Background())
	channel := bus.Subscribe(ctx)
	bus.Publish(&ServiceStatusEvent{Services: []ExternalService{ExternalServiceQBittorrent}})

	done := make(chan struct{})
	go func() {
		bus.Publish(&ServiceStatusEvent{Services: []ExternalService{ExternalServiceQBittorrent}})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Publish blocked on a slow subscriber")
	}
	first := <-channel
	bus.Publish(&ServiceStatusEvent{Services: []ExternalService{ExternalServiceQBittorrent}})
	third := <-channel
	if first.Sequence != 1 || third.Sequence != 3 {
		t.Fatalf("expected dropped event to produce a sequence gap, got %d then %d", first.Sequence, third.Sequence)
	}

	cancel()
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("expected canceled subscriber channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not close after cancellation")
	}
}

type collectingServiceStatusPublisher struct {
	events []*ServiceStatusEvent
}

func (p *collectingServiceStatusPublisher) Publish(event *ServiceStatusEvent) {
	p.events = append(p.events, cloneServiceStatusEvent(event))
}

func TestCollectorPublishesOnlyChangedCandidateServices(t *testing.T) {
	publisher := &collectingServiceStatusPublisher{}
	collector := NewCollector(nil, nil, nil, nil, nil)
	collector.SetEventPublisher(publisher)
	before := Snapshot{}
	after := Snapshot{}
	after.QBitt.DownloadSpeed = 1024
	after.Jackett.LastError = "unavailable"
	collector.publishSnapshotChanges(before, after, []ExternalService{
		ExternalServiceStash,
		ExternalServiceJackett,
		ExternalServiceQBittorrent,
	})
	if len(publisher.events) != 1 {
		t.Fatalf("expected one combined event, got %d", len(publisher.events))
	}
	services := publisher.events[0].Services
	if len(services) != 2 || services[0] != ExternalServiceJackett || services[1] != ExternalServiceQBittorrent {
		t.Fatalf("unexpected changed services: %#v", services)
	}

	collector.publishSnapshotChanges(after, after, []ExternalService{ExternalServiceQBittorrent})
	if len(publisher.events) != 1 {
		t.Fatalf("unchanged snapshot published an event, got %d", len(publisher.events))
	}
}

func TestSnapshotComparisonUsesSceneCountValues(t *testing.T) {
	leftCount, rightCount := 12, 12
	left := StashStats{SceneCount: &leftCount}
	right := StashStats{SceneCount: &rightCount}
	if !stashStatsEqual(left, right) {
		t.Fatal("equal scene-count values should not be treated as a change")
	}
	rightCount = 13
	if stashStatsEqual(left, right) {
		t.Fatal("different scene-count values should be treated as a change")
	}
}
