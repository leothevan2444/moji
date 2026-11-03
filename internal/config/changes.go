package config

import (
	"errors"
	"sync"
)

var ErrChangeSubscribersFull = errors.New("configuration change subscriber limit reached")

// SubscribeChanges sends invalidations only, never configuration or secrets.
// Consumers reread their own scoped snapshot. Writers never wait for readers;
// one pending notification covers all changes until that consumer rereads.
func (s *Store) SubscribeChanges() (<-chan struct{}, func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.changes) >= 128 {
		return nil, nil, ErrChangeSubscribersFull
	}
	if s.changes == nil {
		s.changes = make(map[chan struct{}]struct{})
	}
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	s.changes[ch] = struct{}{}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.changes, ch)
			close(ch)
			s.mu.Unlock()
		})
	}, nil
}

// Caller holds the writer lock and has reconciled the actual durable file.
func (s *Store) changedLocked() {
	for ch := range s.changes {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
