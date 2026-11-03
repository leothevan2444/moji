package indexer

import (
	"sync"

	"github.com/google/uuid"
)

type EventKind string

const (
	EventReady         EventKind = "READY"
	EventConfiguration EventKind = "CONFIGURATION"
	EventCatalog       EventKind = "CATALOG"
	EventSite          EventKind = "SITE"
	EventCache         EventKind = "CACHE"
)

type Event struct {
	StreamID string
	Sequence int64
	Kind     EventKind
}
type EventPublisher interface{ Publish(EventKind) }

// Events carries bounded invalidations, not business truth or an event journal.
// Each subscriber gets the latest sequence even when intermediate events merge.
// Initial READY plus a fresh snapshot also covers reconnect and process restart.
type Events struct {
	mu          sync.Mutex
	streamID    string
	sequence    int64
	closed      bool
	subscribers map[chan Event]struct{}
}

func NewEvents() *Events {
	return &Events{streamID: uuid.NewString(), subscribers: make(map[chan Event]struct{})}
}
func (e *Events) Subscribe() (<-chan Event, func(), error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, nil, ErrNotConfigured
	}
	if len(e.subscribers) >= 128 {
		return nil, nil, ErrBusy
	}
	ch := make(chan Event, 1)
	ch <- Event{StreamID: e.streamID, Sequence: e.sequence, Kind: EventReady}
	e.subscribers[ch] = struct{}{}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			e.mu.Lock()
			if _, ok := e.subscribers[ch]; ok {
				delete(e.subscribers, ch)
				close(ch)
			}
			e.mu.Unlock()
		})
	}, nil
}
func (e *Events) Publish(kind EventKind) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return
	}
	e.sequence++
	event := Event{StreamID: e.streamID, Sequence: e.sequence, Kind: kind}
	for ch := range e.subscribers {
		select {
		case <-ch:
		default:
		}
		ch <- event
	}
}
func (e *Events) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	for ch := range e.subscribers {
		delete(e.subscribers, ch)
		close(ch)
	}
}
