package logging

import "context"

const defaultLogEventBufferSize = 64

type LogEvent struct {
	Sequence int
	Entry    Entry
}

type LogEventBusStats struct {
	Subscribers int
	Published   uint64
	Dropped     uint64
	Sequence    uint64
}

type LogEventSource interface {
	Subscribe(ctx context.Context) <-chan *LogEvent
}

type logEventSubscriber struct {
	channel chan *LogEvent
	done    chan struct{}
}

func (l *Logger) Subscribe(ctx context.Context) <-chan *LogEvent {
	channel := make(chan *LogEvent, defaultLogEventBufferSize)
	if l == nil {
		close(channel)
		return channel
	}
	if ctx == nil {
		ctx = context.Background()
	}

	l.eventMu.Lock()
	if l.eventsClosed {
		close(channel)
		l.eventMu.Unlock()
		return channel
	}
	l.publishedEvents.Add(1)
	id := l.nextSubscriberID.Add(1)
	subscriber := logEventSubscriber{channel: channel, done: make(chan struct{})}
	l.eventSubscribers[id] = subscriber
	l.eventMu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			l.unsubscribe(id)
		case <-subscriber.done:
		}
	}()
	return channel
}

func (l *Logger) unsubscribe(id uint64) {
	l.eventMu.Lock()
	subscriber, ok := l.eventSubscribers[id]
	if ok {
		delete(l.eventSubscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	l.eventMu.Unlock()
}

// publishEvent intentionally does not log delivery or drop information. Any
// log written here would itself create another LogEvent and recurse forever.
func (l *Logger) publishEvent(event *LogEvent) {
	if event == nil {
		return
	}
	l.eventMu.RLock()
	if l.eventsClosed {
		l.eventMu.RUnlock()
		return
	}
	for _, subscriber := range l.eventSubscribers {
		next := *event
		select {
		case subscriber.channel <- &next:
		default:
			l.droppedEvents.Add(1)
		}
	}
	l.eventMu.RUnlock()
}

func (l *Logger) closeEventSubscribers() {
	l.eventMu.Lock()
	if l.eventsClosed {
		l.eventMu.Unlock()
		return
	}
	l.eventsClosed = true
	for id, subscriber := range l.eventSubscribers {
		delete(l.eventSubscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
	l.eventMu.Unlock()
}

func (l *Logger) SubscriberCount() int {
	if l == nil {
		return 0
	}
	l.eventMu.RLock()
	defer l.eventMu.RUnlock()
	return len(l.eventSubscribers)
}

func (l *Logger) DroppedEventCount() uint64 {
	if l == nil {
		return 0
	}
	return l.droppedEvents.Load()
}

func (l *Logger) EventStats() LogEventBusStats {
	if l == nil {
		return LogEventBusStats{}
	}
	l.eventMu.RLock()
	subscribers := len(l.eventSubscribers)
	l.eventMu.RUnlock()
	l.mu.RLock()
	sequence := l.sequence
	l.mu.RUnlock()
	return LogEventBusStats{Subscribers: subscribers, Published: l.publishedEvents.Load(), Dropped: l.droppedEvents.Load(), Sequence: sequence}
}
