package indexercheck

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/indexer"
)

const maxRunningChecks = 4
const maxQueuedChecks = 1024

// Activity is transient request progress, never a persisted test result.
type Activity struct{ RequestID, InstanceID, State string }
type reservation struct {
	Activity
	clientID, key string
	ctx           context.Context
	ready         chan struct{}
	running       bool
}
type activitySubscriber struct {
	clientID string
	out      chan []Activity
}

func (s *Service) reserve(ctx context.Context, input Input) (*reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.owner.Err() != nil {
		return nil, context.Canceled
	}
	key := input.InstanceID
	if key == "" {
		key = "draft"
	}
	requestID := input.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}
	if s.active[key] != nil || s.requests[requestID] != nil {
		return nil, indexer.ErrBusy
	}
	waiting := s.running >= maxRunningChecks || len(s.queue) > 0
	if waiting && (!input.ConnectionOnly || len(s.queue) >= maxQueuedChecks) {
		return nil, indexer.ErrBusy
	}
	entry := &reservation{Activity: Activity{RequestID: requestID, InstanceID: input.InstanceID, State: "QUEUED"}, clientID: input.ClientID, key: key, ctx: ctx, ready: make(chan struct{})}
	s.active[key] = entry
	s.requests[requestID] = entry
	s.wg.Add(1)
	if waiting {
		s.queue = append(s.queue, entry)
	} else {
		entry.running = true
		entry.State = "RUNNING"
		s.running++
		close(entry.ready)
	}
	s.publishActivityLocked()
	return entry, nil
}
func (s *Service) release(entry *reservation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.active, entry.key)
	delete(s.requests, entry.RequestID)
	if entry.running {
		s.running--
	} else {
		for i, queued := range s.queue {
			if queued == entry {
				s.queue = append(s.queue[:i], s.queue[i+1:]...)
				break
			}
		}
	}
	for s.running < maxRunningChecks && len(s.queue) > 0 && !s.closed {
		next := s.queue[0]
		s.queue = s.queue[1:]
		if next.ctx.Err() != nil {
			continue
		}
		next.running = true
		next.State = "RUNNING"
		s.running++
		close(next.ready)
	}
	s.publishActivityLocked()
	s.wg.Done()
}
func (s *Service) activityLocked(clientID string) []Activity {
	result := []Activity{}
	for _, entry := range s.requests {
		if entry.clientID == clientID {
			result = append(result, entry.Activity)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RequestID < result[j].RequestID })
	return result
}
func (s *Service) publishActivityLocked() {
	for sub := range s.subscribers {
		snapshot := s.activityLocked(sub.clientID)
		select {
		case <-sub.out:
		default:
		}
		sub.out <- snapshot
	}
}

// SubscribeActivity replays the current snapshot, so subscribing after admission
// or reconnecting cannot lose the transition from queued to running.
func (s *Service) SubscribeActivity(clientID string) (<-chan []Activity, func(), error) {
	if clientID == "" || len(clientID) > 128 {
		return nil, nil, indexer.ErrInvalidSettings
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, nil, context.Canceled
	}
	if len(s.subscribers) >= 64 {
		return nil, nil, indexer.ErrBusy
	}
	sub := &activitySubscriber{clientID: clientID, out: make(chan []Activity, 1)}
	s.subscribers[sub] = true
	sub.out <- s.activityLocked(clientID)
	stop := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.subscribers[sub] {
			delete(s.subscribers, sub)
			close(sub.out)
		}
	}
	return sub.out, stop, nil
}
