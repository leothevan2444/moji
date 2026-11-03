package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/storage"
)

type siteSession struct {
	gate          chan struct{}
	http          *HTTPSession
	authenticated bool
	loaded        bool
	pending       *cardigann.Execution
	challenge     *Challenge
	refs          int
	lastUsed      time.Time
}

type Challenge struct {
	ID, InstanceID, Type, Prompt, ContentType string
	Image                                     []byte
	ExpiresAt                                 time.Time
}

// Sessions serializes a frozen account's login/search/fetch while retaining its
// cookies across independent operations and process restarts. Challenges only
// live in memory; no submitted answer is saved.
type Sessions struct {
	network    *Network
	repo       sessionRepository
	mu         sync.Mutex
	entries    map[string]*siteSession
	challenges map[string]string
	transient  bool
	closed     bool
}

func (s *Sessions) expireChallengesLocked() {
	for id, key := range s.challenges {
		entry := s.entries[key]
		if entry == nil {
			delete(s.challenges, id)
			continue
		}
		if entry.refs == 0 && entry.challenge != nil && time.Now().After(entry.challenge.ExpiresAt) {
			entry.pending = nil
			entry.challenge = nil
			delete(s.challenges, id)
		}
	}
}

func NewSessions(network *Network, manager *storage.Manager) (*Sessions, error) {
	if network == nil || manager == nil || manager.Runtime == nil {
		return nil, ErrNotConfigured
	}
	return &Sessions{network: network, repo: sessionRepository{db: manager.Runtime}, entries: map[string]*siteSession{}, challenges: map[string]string{}}, nil
}

// NewTransientSessions isolates explicit diagnostic requests from production
// cookies and runtime storage. The owner must Close after all operations exit.
func NewTransientSessions(network *Network) (*Sessions, error) {
	if network == nil {
		return nil, ErrNotConfigured
	}
	return &Sessions{network: network, transient: true, entries: map[string]*siteSession{}, challenges: map[string]string{}}, nil
}

// Close releases account transports and ephemeral challenges. Call only after
// the owner has cancelled and joined operations using this session collection.
func (s *Sessions) Close() {
	s.mu.Lock()
	s.closed = true
	entries := s.entries
	s.entries = map[string]*siteSession{}
	s.challenges = map[string]string{}
	s.mu.Unlock()
	for _, entry := range entries {
		entry.http.Close()
	}
}

func sessionKey(instance InstanceSnapshot, plan *cardigann.Plan) (string, error) {
	if plan == nil {
		return "", ErrUnsupported
	}
	if instance.BaseURL == "" {
		if len(plan.Definition.Links) == 0 {
			return "", ErrInvalidSettings
		}
		instance.BaseURL = plan.Definition.Links[0]
	}
	// A result-selector change must not discard a still valid login. Include
	// authentication-affecting rule data, not the complete definition digest.
	value := struct {
		InstanceID, Revision, BaseURL string
		Settings                      map[string]SettingValue
		Network                       NetworkProfile
		Login                         *cardigann.Login
		Headers                       map[string][]string
		Certificates                  []string
		Origins                       []string
	}{instance.ID, instance.ConfigRevision, instance.BaseURL, instance.Settings, instance.Network, plan.Definition.Login, plan.Definition.Search.Headers, plan.Definition.Certificates, instance.TrustedOrigins}
	data, err := json.Marshal(value)
	if err != nil {
		return "", ErrInvalidSettings
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Sessions) entry(ctx context.Context, instance InstanceSnapshot, plan *cardigann.Plan) (string, *siteSession, func(), error) {
	key, err := sessionKey(instance, plan)
	if err != nil {
		return "", nil, nil, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", nil, nil, context.Canceled
	}
	s.expireChallengesLocked()
	entry := s.entries[key]
	if entry == nil {
		if len(s.entries) >= 128 {
			oldest := ""
			var at time.Time
			for id, item := range s.entries {
				if item.refs == 0 && item.pending == nil && (oldest == "" || item.lastUsed.Before(at)) {
					oldest = id
					at = item.lastUsed
				}
			}
			if oldest == "" {
				s.mu.Unlock()
				return "", nil, nil, ErrBusy
			}
			s.entries[oldest].http.Close()
			delete(s.entries, oldest)
		}
		instance.RequestDelay = max(instance.RequestDelay, time.Duration(plan.Definition.RequestDelay*float64(time.Second)))
		httpSession, err := s.network.NewSession(instance, plan.Definition.Certificates)
		if err != nil {
			s.mu.Unlock()
			return "", nil, nil, err
		}
		entry = &siteSession{gate: make(chan struct{}, 1), http: httpSession}
		s.entries[key] = entry
	}
	entry.refs++
	s.mu.Unlock()
	releaseRef := func() { s.mu.Lock(); entry.refs--; entry.lastUsed = time.Now(); s.mu.Unlock() }
	select {
	case entry.gate <- struct{}{}:
	case <-ctx.Done():
		releaseRef()
		return "", nil, nil, ctx.Err()
	}
	return key, entry, func() { <-entry.gate; releaseRef() }, nil
}

func settingsFor(instance InstanceSnapshot) (map[string]cardigann.Scalar, error) {
	settings := map[string]cardigann.Scalar{}
	for key, value := range instance.Settings {
		text, err := value.Text()
		if err != nil {
			return nil, err
		}
		settings[key] = cardigann.Scalar(text)
	}
	return settings, nil
}

func (s *Sessions) With(ctx context.Context, instance InstanceSnapshot, plan *cardigann.Plan, readOnly bool, operation func(*cardigann.Execution) error) error {
	if plan == nil || operation == nil {
		return ErrUnsupported
	}
	// The execution and its guarded HTTP session must agree on the rule's
	// default origin when the user has not selected a mirror.
	if instance.BaseURL == "" {
		if len(plan.Definition.Links) == 0 {
			return ErrInvalidSettings
		}
		instance.BaseURL = plan.Definition.Links[0]
	}
	key, entry, release, err := s.entry(ctx, instance, plan)
	if err != nil {
		return err
	}
	defer release()
	if entry.pending != nil {
		if time.Now().Before(entry.challenge.ExpiresAt) {
			return ErrChallenge
		}
		s.mu.Lock()
		delete(s.challenges, entry.challenge.ID)
		entry.pending = nil
		entry.challenge = nil
		s.mu.Unlock()
	}
	jar := entry.http.jar.(*persistentJar)
	if !entry.loaded {
		var payload *sessionPayload
		if !s.transient {
			payload, err = s.repo.load(ctx, key)
			if err != nil {
				return err
			}
		}
		if payload != nil {
			if err := jar.restore(payload.Cookies); err != nil {
				return err
			}
			entry.authenticated = payload.Authenticated
			if payload.UserAgent != "" {
				entry.http.userAgent = payload.UserAgent
				entry.http.solverUserAgent = payload.UserAgent != "Moji/Indexer"
			}
		}
		entry.loaded = true
	}
	settings, err := settingsFor(instance)
	if err != nil {
		return err
	}
	var execution *cardigann.Execution
	scoped := &purposeHTTP{session: entry.http, purpose: purposeAuth}
	runOperation := func() error {
		scoped.purpose = purposeFetch
		if readOnly {
			scoped.purpose = purposeSearch
		}
		return operation(execution)
	}
	authenticate := func() error {
		var err error
		scoped.purpose = purposeAuth
		execution, err = plan.NewExecution(scoped, instance.BaseURL, settings, time.Now())
		if err != nil {
			return ErrInvalidSettings
		}
		if entry.authenticated || plan.Definition.Login == nil {
			return nil
		}
		challenge, err := execution.Authenticate(ctx)
		if err != nil {
			return loginError(err)
		}
		if challenge != nil {
			s.mu.Lock()
			if len(s.challenges) >= 32 {
				s.mu.Unlock()
				return ErrBusy
			}
			entry.pending = execution
			entry.challenge = &Challenge{ID: uuid.NewString(), InstanceID: instance.ID, Type: challenge.Type, Prompt: challenge.Prompt, ContentType: challenge.ContentType, Image: append([]byte(nil), challenge.Image...), ExpiresAt: time.Now().Add(5 * time.Minute)}
			s.challenges[entry.challenge.ID] = key
			s.mu.Unlock()
			return ErrChallenge
		}
		entry.authenticated = true
		return nil
	}
	err = authenticate()
	if err == nil {
		err = runOperation()
	}
	if errors.Is(err, cardigann.ErrAuthentication) {
		entry.authenticated = false
		if readOnly && plan.Definition.Login != nil {
			err = authenticate()
			if err == nil {
				err = runOperation()
			}
		}
		if errors.Is(err, cardigann.ErrAuthentication) {
			err = ErrAuthentication
			entry.authenticated = false
		}
	}
	records, dirty, snapshotErr := jar.snapshot()
	if snapshotErr != nil {
		return snapshotErr
	}
	if dirty || plan.Definition.Login != nil {
		payload := sessionPayload{Version: 1, Cookies: records, Authenticated: entry.authenticated && entry.pending == nil, UserAgent: entry.http.userAgent}
		if !s.transient {
			if saveErr := s.repo.save(ctx, key, instance.ID, payload); saveErr != nil && err == nil {
				return saveErr
			}
		}
		jar.mu.Lock()
		jar.dirty = false
		jar.mu.Unlock()
	}
	return err
}

func (s *Sessions) Challenges(instanceID string) []Challenge {
	s.mu.Lock()
	s.expireChallengesLocked()
	defer s.mu.Unlock()
	out := []Challenge{}
	for _, entry := range s.entries {
		if entry.challenge != nil && entry.challenge.InstanceID == instanceID && time.Now().Before(entry.challenge.ExpiresAt) {
			copy := *entry.challenge
			copy.Image = append([]byte(nil), copy.Image...)
			out = append(out, copy)
		}
	}
	return out
}

func (s *Sessions) Answer(ctx context.Context, id, answer string) error {
	if answer == "" || len(answer) > 1024 {
		return ErrInvalidSettings
	}
	s.mu.Lock()
	key := s.challenges[id]
	entry := s.entries[key]
	if entry != nil {
		entry.refs++
	}
	s.mu.Unlock()
	if entry == nil {
		return ErrExpired
	}
	defer func() { s.mu.Lock(); entry.refs--; s.mu.Unlock() }()
	select {
	case entry.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-entry.gate }()
	if entry.challenge == nil || entry.challenge.ID != id || time.Now().After(entry.challenge.ExpiresAt) {
		return ErrExpired
	}
	err := entry.pending.AnswerChallenge(ctx, answer)
	instanceID := entry.challenge.InstanceID
	s.mu.Lock()
	delete(s.challenges, id)
	entry.pending = nil
	entry.challenge = nil
	s.mu.Unlock()
	entry.authenticated = err == nil
	records, _, snapshotErr := entry.http.jar.(*persistentJar).snapshot()
	if snapshotErr != nil {
		return snapshotErr
	}
	if !s.transient {
		if saveErr := s.repo.save(ctx, key, instanceID, sessionPayload{Version: 1, Cookies: records, Authenticated: entry.authenticated, UserAgent: entry.http.userAgent}); saveErr != nil && err == nil {
			return saveErr
		}
	}
	if err != nil {
		return loginError(err)
	}
	return nil
}

func loginError(err error) error {
	for _, known := range []error{context.Canceled, context.DeadlineExceeded, ErrUnsafeURL, ErrTransport, ErrCoolingDown, ErrBusy, ErrStateUnavailable, ErrResponseBudget, ErrChallenge, ErrSolver} {
		if errors.Is(err, known) {
			return unwrapNetworkError(err, known)
		}
	}
	return ErrAuthentication
}
