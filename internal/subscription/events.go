package subscription

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

type PerformerSubscriptionEventType string

const (
	PerformerSubscriptionEventCreated PerformerSubscriptionEventType = "CREATED"
	PerformerSubscriptionEventUpdated PerformerSubscriptionEventType = "UPDATED"
	PerformerSubscriptionEventDeleted PerformerSubscriptionEventType = "DELETED"
)

type PerformerSubscriptionEvent struct {
	Sequence    int
	Type        PerformerSubscriptionEventType
	PerformerID string
	State       *SubscribedPerformer
}

type PerformerSubscriptionEventSource interface {
	Subscribe(ctx context.Context) <-chan *PerformerSubscriptionEvent
}

type PerformerSubscriptionEventPublisher interface {
	Publish(event *PerformerSubscriptionEvent)
}

type PerformerSubscriptionEventBusStats struct {
	Subscribers int
	Published   uint64
	Dropped     uint64
	Sequence    uint64
}

type performerSubscriptionSubscriber struct {
	channel chan *PerformerSubscriptionEvent
	done    chan struct{}
}

type PerformerSubscriptionEventBus struct {
	mu          sync.RWMutex
	subscribers map[uint64]performerSubscriptionSubscriber
	nextID      atomic.Uint64
	sequence    atomic.Uint64
	published   atomic.Uint64
	dropped     atomic.Uint64
	bufferSize  int
	closed      bool
}

func NewPerformerSubscriptionEventBus(bufferSize int) *PerformerSubscriptionEventBus {
	if bufferSize <= 0 {
		bufferSize = 16
	}
	return &PerformerSubscriptionEventBus{
		subscribers: make(map[uint64]performerSubscriptionSubscriber),
		bufferSize:  bufferSize,
	}
}

func (b *PerformerSubscriptionEventBus) Subscribe(ctx context.Context) <-chan *PerformerSubscriptionEvent {
	channel := make(chan *PerformerSubscriptionEvent, b.bufferSize)
	if ctx == nil {
		ctx = context.Background()
	}
	b.mu.Lock()
	if b.closed {
		close(channel)
		b.mu.Unlock()
		return channel
	}
	id := b.nextID.Add(1)
	subscriber := performerSubscriptionSubscriber{channel: channel, done: make(chan struct{})}
	b.subscribers[id] = subscriber
	b.mu.Unlock()
	slog.Info("performer subscription events: subscriber connected", "subscriber_id", id)

	go func() {
		select {
		case <-ctx.Done():
			b.unsubscribe(id)
		case <-subscriber.done:
		}
	}()
	return channel
}

func (b *PerformerSubscriptionEventBus) unsubscribe(id uint64) {
	b.mu.Lock()
	subscriber, ok := b.subscribers[id]
	if ok {
		delete(b.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	b.mu.Unlock()
	if ok {
		slog.Info("performer subscription events: subscriber disconnected", "subscriber_id", id)
	}
}

func (b *PerformerSubscriptionEventBus) Publish(event *PerformerSubscriptionEvent) {
	if event == nil || event.PerformerID == "" {
		return
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}
	next := clonePerformerSubscriptionEvent(event)
	next.Sequence = int(b.sequence.Add(1))
	b.published.Add(1)
	for id, subscriber := range b.subscribers {
		select {
		case subscriber.channel <- clonePerformerSubscriptionEvent(next):
		default:
			b.dropped.Add(1)
			slog.Warn("performer subscription events: event dropped because subscriber buffer is full", "subscriber_id", id, "sequence", next.Sequence, "performer_id", next.PerformerID)
		}
	}
	b.mu.RUnlock()
	slog.Info("performer subscription events: event published", "sequence", next.Sequence, "type", next.Type, "performer_id", next.PerformerID)
}

func (b *PerformerSubscriptionEventBus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	disconnected := len(b.subscribers)
	for id, subscriber := range b.subscribers {
		delete(b.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	b.mu.Unlock()
	stats := b.Stats()
	slog.Info("performer subscription events: shutdown summary", "published", stats.Published, "dropped", stats.Dropped, "sequence", stats.Sequence, "disconnected", disconnected)
}

func (b *PerformerSubscriptionEventBus) Stats() PerformerSubscriptionEventBusStats {
	b.mu.RLock()
	subscribers := len(b.subscribers)
	b.mu.RUnlock()
	return PerformerSubscriptionEventBusStats{
		Subscribers: subscribers,
		Published:   b.published.Load(),
		Dropped:     b.dropped.Load(),
		Sequence:    b.sequence.Load(),
	}
}

func clonePerformerSubscriptionEvent(event *PerformerSubscriptionEvent) *PerformerSubscriptionEvent {
	if event == nil {
		return nil
	}
	next := *event
	if event.State != nil {
		state := cloneSubscribedPerformer(*event.State)
		next.State = &state
	}
	return &next
}

func cloneSubscribedPerformer(item SubscribedPerformer) SubscribedPerformer {
	next := item
	next.Performer.AliasList = append([]string(nil), item.Performer.AliasList...)
	if item.LastCheckedAt != nil {
		checked := *item.LastCheckedAt
		next.LastCheckedAt = &checked
	}
	next.RecentReleases = append([]RecordedRelease(nil), item.RecentReleases...)
	for index := range next.RecentReleases {
		next.RecentReleases[index].PerformerNames = append([]string(nil), item.RecentReleases[index].PerformerNames...)
	}
	return next
}
