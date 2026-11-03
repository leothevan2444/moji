package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

// LoginSession binds an interactive login to one frozen saved-site session.
// Closing it discards only the challenge observed by this login, not another
// challenge created later by normal sourcing.
type LoginSession struct {
	sessions    *Sessions
	instance    InstanceSnapshot
	plan        *cardigann.Plan
	challengeID string
}

func NewLoginSession(sessions *Sessions, instance InstanceSnapshot, plan *cardigann.Plan) (*LoginSession, error) {
	if sessions == nil || plan == nil {
		return nil, ErrNotConfigured
	}
	return &LoginSession{sessions: sessions, instance: instance, plan: plan}, nil
}
func (e *LoginSession) Authenticate(ctx context.Context) error {
	err := e.sessions.With(ctx, e.instance, e.plan, true, func(_ *cardigann.Execution) error { return nil })
	if errors.Is(err, ErrChallenge) {
		challenge, lookupErr := e.sessions.ChallengeFor(e.instance, e.plan)
		if lookupErr != nil {
			return lookupErr
		}
		if challenge == nil {
			return ErrExpired
		}
		e.challengeID = challenge.ID
	}
	return err
}
func (e *LoginSession) Challenge() (*Challenge, error) {
	challenge, err := e.sessions.ChallengeFor(e.instance, e.plan)
	if err != nil {
		return nil, err
	}
	if challenge == nil || challenge.ID != e.challengeID {
		return nil, nil
	}
	return challenge, nil
}
func (e *LoginSession) Answer(ctx context.Context, id, answer string) error {
	if id != e.challengeID {
		return ErrInvalidScope
	}
	return e.sessions.AnswerFor(ctx, e.instance, e.plan, id, answer)
}
func (e *LoginSession) Close() {
	if e.challengeID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = e.sessions.DiscardChallenge(ctx, e.instance, e.plan, e.challengeID)
}
