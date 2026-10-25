package stashboxcache

import (
	"context"
	"time"
)

// Keep browsing responsive while bounding retries and shutting down cleanly.
func (s *Service) refresh(key string, fetch func(context.Context)) {
	s.refreshMu.Lock()
	if s.closed {
		s.refreshMu.Unlock()
		return
	}
	if last, exists := s.refreshes[key]; exists && time.Since(last) < 30*time.Second {
		s.refreshMu.Unlock()
		return
	}
	s.refreshes[key] = time.Now()
	s.refreshWG.Add(1)
	s.refreshMu.Unlock()
	go func() {
		defer s.refreshWG.Done()
		ctx, cancel := context.WithTimeout(s.refreshCtx, 20*time.Second)
		defer cancel()
		fetch(ctx)
	}()
}
