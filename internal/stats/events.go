package stats

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type ExternalService string

const (
	ExternalServiceStash       ExternalService = "STASH"
	ExternalServiceJackett     ExternalService = "JACKETT"
	ExternalServiceQBittorrent ExternalService = "QBITTORRENT"
)

type ServiceStatusEvent struct {
	Sequence   int
	Services   []ExternalService
	ObservedAt time.Time
}

type ServiceStatusEventSource interface {
	Subscribe(ctx context.Context) <-chan *ServiceStatusEvent
}

type ServiceStatusEventPublisher interface {
	Publish(event *ServiceStatusEvent)
}

type serviceStatusSubscriber struct {
	channel chan *ServiceStatusEvent
	done    chan struct{}
}

type ServiceStatusEventBus struct {
	mu          sync.RWMutex
	subscribers map[uint64]serviceStatusSubscriber
	nextID      atomic.Uint64
	sequence    atomic.Uint64
	published   atomic.Uint64
	dropped     atomic.Uint64
	bufferSize  int
	closed      bool
}

type ServiceStatusEventBusStats struct {
	Subscribers int
	Published   uint64
	Dropped     uint64
	Sequence    uint64
}

func NewServiceStatusEventBus(bufferSize int) *ServiceStatusEventBus {
	if bufferSize <= 0 {
		bufferSize = 8
	}
	return &ServiceStatusEventBus{
		subscribers: make(map[uint64]serviceStatusSubscriber),
		bufferSize:  bufferSize,
	}
}

func (b *ServiceStatusEventBus) Subscribe(ctx context.Context) <-chan *ServiceStatusEvent {
	channel := make(chan *ServiceStatusEvent, b.bufferSize)
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
	subscriber := serviceStatusSubscriber{channel: channel, done: make(chan struct{})}
	b.subscribers[id] = subscriber
	b.mu.Unlock()
	slog.Info("service status events: subscriber connected", "subscriber_id", id)

	go func() {
		select {
		case <-ctx.Done():
			b.unsubscribe(id)
		case <-subscriber.done:
		}
	}()
	return channel
}

func (b *ServiceStatusEventBus) unsubscribe(id uint64) {
	b.mu.Lock()
	subscriber, ok := b.subscribers[id]
	if ok {
		delete(b.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	b.mu.Unlock()
	if ok {
		slog.Info("service status events: subscriber disconnected", "subscriber_id", id)
	}
}

func (b *ServiceStatusEventBus) Publish(event *ServiceStatusEvent) {
	if event == nil || len(event.Services) == 0 {
		return
	}
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}
	next := cloneServiceStatusEvent(event)
	next.Sequence = int(b.sequence.Add(1))
	b.published.Add(1)
	if next.ObservedAt.IsZero() {
		next.ObservedAt = time.Now().UTC()
	}
	for id, subscriber := range b.subscribers {
		select {
		case subscriber.channel <- cloneServiceStatusEvent(next):
		default:
			b.dropped.Add(1)
			slog.Warn("service status events: event dropped because subscriber buffer is full", "subscriber_id", id, "sequence", next.Sequence)
		}
	}
	b.mu.RUnlock()
	slog.Debug("service status events: event published", "sequence", next.Sequence, "services", next.Services)
}

func (b *ServiceStatusEventBus) Close() {
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
	slog.Info("service status events: shutdown summary", "published", stats.Published, "dropped", stats.Dropped, "sequence", stats.Sequence, "disconnected", disconnected)
}

func (b *ServiceStatusEventBus) Stats() ServiceStatusEventBusStats {
	b.mu.RLock()
	subscribers := len(b.subscribers)
	b.mu.RUnlock()
	return ServiceStatusEventBusStats{Subscribers: subscribers, Published: b.published.Load(), Dropped: b.dropped.Load(), Sequence: b.sequence.Load()}
}

func (b *ServiceStatusEventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

func cloneServiceStatusEvent(event *ServiceStatusEvent) *ServiceStatusEvent {
	if event == nil {
		return nil
	}
	next := *event
	next.Services = append([]ExternalService(nil), event.Services...)
	return &next
}
