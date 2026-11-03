package indexer

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

// ChallengeFor resolves only the frozen account, origin, network and login rule.
// Listing by instance ID alone is insufficient after account or rule changes.
func (s *Sessions) ChallengeFor(instance InstanceSnapshot, plan *cardigann.Plan) (*Challenge, error) {
	key, err := sessionKey(instance, plan)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, context.Canceled
	}
	s.expireChallengesLocked()
	entry := s.entries[key]
	if entry == nil || entry.challenge == nil || !time.Now().Before(entry.challenge.ExpiresAt) {
		return nil, nil
	}
	result := *entry.challenge
	result.Image = append([]byte(nil), result.Image...)
	return &result, nil
}

func (s *Sessions) AnswerFor(ctx context.Context, instance InstanceSnapshot, plan *cardigann.Plan, id, answer string) error {
	challenge, err := s.ChallengeFor(instance, plan)
	if err != nil {
		return err
	}
	if challenge == nil || challenge.ID != id {
		return ErrInvalidScope
	}
	return s.Answer(ctx, id, answer)
}

// DiscardChallenge drops a scoped pending login without submitting any answer.
// It serializes with authentication and answer operations on the same session.
func (s *Sessions) DiscardChallenge(ctx context.Context, instance InstanceSnapshot, plan *cardigann.Plan, id string) error {
	key, err := sessionKey(instance, plan)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return context.Canceled
	}
	entry := s.entries[key]
	if entry == nil || s.challenges[id] != key {
		s.mu.Unlock()
		return ErrInvalidScope
	}
	entry.refs++
	s.mu.Unlock()
	defer func() { s.mu.Lock(); entry.refs--; s.mu.Unlock() }()
	select {
	case entry.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-entry.gate }()
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry.challenge == nil || entry.challenge.ID != id {
		return ErrExpired
	}
	delete(s.challenges, id)
	entry.pending = nil
	entry.challenge = nil
	entry.authenticated = false
	return nil
}
