// Package indexerlogin owns bounded, ephemeral login challenges for saved sites.
package indexerlogin

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexer"
)

type Input struct{ InstanceID, ExpectedRevision string }
type Prepared struct {
	Instance                     indexer.InstanceSnapshot
	ConfigRevision, RuleRevision string
	Plan                         *cardigann.Plan
	Release                      func()
}
type Source interface {
	PrepareLogin(context.Context, Input) (Prepared, error)
	ValidateLogin(context.Context, string) error
}
type Execution interface {
	Authenticate(context.Context) error
	Challenge() (*indexer.Challenge, error)
	Answer(context.Context, string, string) error
	Close()
}
type Factory interface {
	Open(Prepared) (Execution, error)
}
type Report struct {
	ID, InstanceID, ConfigRevision, RuleRevision, State string
	ExpiresAt                                           time.Time
	Challenge                                           *indexer.Challenge
}
type flow struct {
	revision  string
	id, key   string
	ctx       context.Context
	cancel    context.CancelFunc
	expires   time.Time
	prepared  Prepared
	execution Execution
	busy      bool
	wg        sync.WaitGroup
	once      sync.Once
}

func (f *flow) close() {
	f.cancel()
	f.once.Do(func() {
		f.wg.Wait()
		if f.execution != nil {
			f.execution.Close()
		}
		if f.prepared.Release != nil {
			f.prepared.Release()
		}
	})
}

type Service struct {
	owner    context.Context
	cancel   context.CancelFunc
	source   Source
	factory  Factory
	mu       sync.Mutex
	entries  map[string]*flow
	keys     map[string]*flow
	closed   bool
	wg       sync.WaitGroup
	loopDone chan struct{}
}

func New(owner context.Context, source Source, factory Factory) (*Service, error) {
	if owner == nil || source == nil || factory == nil {
		return nil, indexer.ErrNotConfigured
	}
	ctx, cancel := context.WithCancel(owner)
	s := &Service{owner: ctx, cancel: cancel, source: source, factory: factory, entries: map[string]*flow{}, keys: map[string]*flow{}, loopDone: make(chan struct{})}
	go func() {
		defer close(s.loopDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				s.expire(now)
			}
		}
	}()
	return s, nil
}
func (s *Service) remove(f *flow) {
	s.mu.Lock()
	if s.entries[f.id] == f {
		delete(s.entries, f.id)
	}
	s.mu.Unlock()
	f.close()
	s.mu.Lock()
	if s.keys[f.key] == f {
		delete(s.keys, f.key)
	}
	s.mu.Unlock()
}
func (s *Service) expire(now time.Time) {
	s.mu.Lock()
	var expired []*flow
	for _, f := range s.entries {
		if !now.Before(f.expires) {
			expired = append(expired, f)
		}
	}
	s.mu.Unlock()
	for _, f := range expired {
		s.remove(f)
	}
}
func requestContext(parent, caller context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(caller, cancel)
	return ctx, func() { stop(); cancel() }
}
func (s *Service) Begin(ctx context.Context, input Input) (Report, error) {
	return s.begin(ctx, input, false)
}
func (s *Service) begin(ctx context.Context, input Input, retried bool) (Report, error) {
	if input.InstanceID == "" || len(input.InstanceID) > 128 || input.ExpectedRevision == "" || len(input.ExpectedRevision) > 128 {
		return Report{}, indexer.ErrInvalidSettings
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	s.mu.Lock()
	if s.closed || s.owner.Err() != nil {
		s.mu.Unlock()
		return Report{}, context.Canceled
	}
	if old := s.keys[input.InstanceID]; old != nil && old.revision != input.ExpectedRevision {
		s.mu.Unlock()
		if retried {
			return Report{}, indexer.ErrBusy
		}
		if err := s.source.ValidateLogin(ctx, input.ExpectedRevision); err != nil {
			return Report{}, err
		}
		s.remove(old)
		return s.begin(ctx, input, true)
	}
	if len(s.keys) >= 4 || s.keys[input.InstanceID] != nil {
		s.mu.Unlock()
		return Report{}, indexer.ErrBusy
	}
	deadline := time.Now().Add(5 * time.Minute)
	work, cancel := context.WithDeadline(s.owner, deadline)
	f := &flow{id: uuid.NewString(), key: input.InstanceID, revision: input.ExpectedRevision, ctx: work, cancel: cancel, expires: deadline, busy: true}
	f.wg.Add(1)
	s.wg.Add(1)
	s.entries[f.id] = f
	s.keys[f.key] = f
	s.mu.Unlock()
	keep := false
	defer func() {
		f.wg.Done()
		if !keep {
			s.remove(f)
		}
		s.wg.Done()
	}()
	request, done := requestContext(f.ctx, ctx)
	defer done()
	var err error
	f.prepared, err = s.source.PrepareLogin(request, input)
	if err != nil {
		return Report{}, err
	}
	p := f.prepared
	if p.ConfigRevision != input.ExpectedRevision || p.Instance.ID != input.InstanceID {
		return Report{}, indexer.ErrInvalidScope
	}
	report := Report{ID: f.id, InstanceID: p.Instance.ID, ConfigRevision: p.ConfigRevision, RuleRevision: p.RuleRevision, State: "NOT_REQUIRED", ExpiresAt: f.expires}
	if p.Instance.Kind != indexer.Builtin || p.Plan == nil {
		return Report{}, indexer.ErrUnsupported
	}
	if p.Plan.Definition.Login == nil {
		return report, nil
	}
	f.execution, err = s.factory.Open(p)
	if err != nil {
		return Report{}, err
	}
	if f.execution == nil {
		return Report{}, indexer.ErrNotConfigured
	}
	err = f.execution.Authenticate(request)
	if err == nil {
		report.State = "READY"
		return report, nil
	}
	if !errors.Is(err, indexer.ErrChallenge) {
		return Report{}, err
	}
	challenge, err := f.execution.Challenge()
	if err != nil {
		return Report{}, err
	}
	if challenge == nil {
		return Report{}, indexer.ErrExpired
	}
	if challenge.ExpiresAt.Before(report.ExpiresAt) {
		report.ExpiresAt = challenge.ExpiresAt
	}
	if err := request.Err(); err != nil {
		return Report{}, err
	}
	report.State = "WAITING"
	report.Challenge = challenge
	s.mu.Lock()
	if s.entries[f.id] != f || s.closed {
		s.mu.Unlock()
		return Report{}, context.Canceled
	}
	f.expires = report.ExpiresAt
	f.busy = false
	keep = true
	s.mu.Unlock()
	return report, nil
}
func (s *Service) Answer(ctx context.Context, id string, input Input, answer string) (Report, error) {
	if answer == "" || len(answer) > 1024 {
		return Report{}, indexer.ErrInvalidSettings
	}
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	s.mu.Lock()
	f := s.entries[id]
	if f == nil {
		s.mu.Unlock()
		return Report{}, indexer.ErrExpired
	}
	if f.key != input.InstanceID || f.revision != input.ExpectedRevision {
		s.mu.Unlock()
		return Report{}, indexer.ErrInvalidScope
	}
	if f.busy {
		s.mu.Unlock()
		return Report{}, indexer.ErrBusy
	}
	if !time.Now().Before(f.expires) {
		s.mu.Unlock()
		s.remove(f)
		return Report{}, indexer.ErrExpired
	}
	f.busy = true
	f.wg.Add(1)
	s.wg.Add(1)
	s.mu.Unlock()
	defer func() { f.wg.Done(); s.remove(f); s.wg.Done() }()
	work, done := requestContext(f.ctx, ctx)
	defer done()
	if err := s.source.ValidateLogin(work, input.ExpectedRevision); err != nil {
		return Report{}, err
	}
	challenge, err := f.execution.Challenge()
	if err != nil {
		return Report{}, err
	}
	if challenge == nil {
		return Report{}, indexer.ErrExpired
	}
	if err = f.execution.Answer(work, challenge.ID, answer); err != nil {
		return Report{}, err
	}
	return Report{ID: f.id, InstanceID: f.key, ConfigRevision: f.prepared.ConfigRevision, RuleRevision: f.prepared.RuleRevision, State: "READY", ExpiresAt: f.expires}, nil
}
func (s *Service) Cancel(ctx context.Context, id string, input Input) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	f := s.entries[id]
	if f == nil {
		s.mu.Unlock()
		return indexer.ErrExpired
	}
	if f.key != input.InstanceID || f.revision != input.ExpectedRevision {
		s.mu.Unlock()
		return indexer.ErrInvalidScope
	}
	s.wg.Add(1)
	s.mu.Unlock()
	defer s.wg.Done()
	s.remove(f)
	return nil
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	var entries []*flow
	for _, f := range s.entries {
		entries = append(entries, f)
		f.cancel()
	}
	s.mu.Unlock()
	<-s.loopDone
	for _, f := range entries {
		s.remove(f)
	}
	s.wg.Wait()
}

// Pending restores an existing challenge after a page refresh without starting
// authentication, changing cookies or issuing any request to the indexer.
func (s *Service) Pending(ctx context.Context, input Input) (*Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	f := s.keys[input.InstanceID]
	if f == nil || s.entries[f.id] != f {
		s.mu.Unlock()
		return nil, nil
	}
	if f.revision != input.ExpectedRevision {
		s.mu.Unlock()
		if err := s.source.ValidateLogin(ctx, input.ExpectedRevision); err != nil {
			return nil, err
		}
		s.remove(f)
		return nil, nil
	}
	if f.busy {
		s.mu.Unlock()
		return nil, indexer.ErrBusy
	}
	if !time.Now().Before(f.expires) {
		s.mu.Unlock()
		s.remove(f)
		return nil, nil
	}
	f.busy = true
	f.wg.Add(1)
	s.wg.Add(1)
	s.mu.Unlock()
	retire := false
	defer func() {
		f.wg.Done()
		if retire {
			s.remove(f)
		} else {
			s.mu.Lock()
			if s.entries[f.id] == f {
				f.busy = false
			}
			s.mu.Unlock()
		}
		s.wg.Done()
	}()
	work, done := requestContext(f.ctx, ctx)
	defer done()
	if err := s.source.ValidateLogin(work, input.ExpectedRevision); err != nil {
		retire = !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
		return nil, err
	}
	challenge, err := f.execution.Challenge()
	if err != nil {
		retire = true
		return nil, err
	}
	if challenge == nil {
		retire = true
		return nil, nil
	}
	return &Report{ID: f.id, InstanceID: f.key, ConfigRevision: f.prepared.ConfigRevision, RuleRevision: f.prepared.RuleRevision, State: "WAITING", ExpiresAt: f.expires, Challenge: challenge}, nil
}
