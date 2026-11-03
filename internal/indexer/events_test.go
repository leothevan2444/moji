package indexer

import (
	"sync"
	"testing"
)

func TestIndexerEventsKeepLatestSequenceAndReleaseSlowSubscribers(t *testing.T) {
	bus := NewEvents()
	ch, stop, err := bus.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	ready := <-ch
	if ready.Kind != EventReady || ready.StreamID == "" || ready.Sequence != 0 {
		t.Fatal("missing reconnect boundary")
	}
	for i := 0; i < 1000; i++ {
		bus.Publish(EventConfiguration)
	}
	if len(ch) != 1 {
		t.Fatal("notification queue unbounded")
	}
	last := <-ch
	if last.Sequence != 1000 || last.StreamID != ready.StreamID {
		t.Fatal("slow reader did not get latest sequence")
	}
	bus.Publish(EventSite)
	bus.Close()
	bus.Close()
	stop()
	stop()
	<-ch
	if _, ok := <-ch; ok {
		t.Fatal("shutdown did not close subscriber")
	}
	if _, _, err := bus.Subscribe(); err == nil {
		t.Fatal("closed source accepted subscriber")
	}
	other := NewEvents()
	defer other.Close()
	next, unsub, err := other.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	if (<-next).StreamID == ready.StreamID {
		t.Fatal("new process reused stream identity")
	}
}

func TestIndexerEventsConcurrentPublishUnsubscribeAndBudget(t *testing.T) {
	bus := NewEvents()
	var stops []func()
	for i := 0; i < 128; i++ {
		_, stop, err := bus.Subscribe()
		if err != nil {
			t.Fatal(err)
		}
		stops = append(stops, stop)
	}
	if _, _, err := bus.Subscribe(); err != ErrBusy {
		t.Fatal("subscriber limit missing", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			bus.Publish(EventCatalog)
		}
	}()
	for _, stop := range stops {
		wg.Add(1)
		go func() { defer wg.Done(); stop() }()
	}
	bus.Close()
	wg.Wait()
}
