package torrentsearch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/storage"
)

type Searcher interface {
	SearchWithProgress(context.Context, indexer.Query, func(context.Context, indexer.SearchProgress) error, func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error)
}
type Event struct {
	StreamID, SearchID string
	Sequence           int64
}
type subscriber struct {
	id      string
	channel chan Event
}
type Service struct {
	store               *Store
	searcher            Searcher
	ctx                 context.Context
	cancel              context.CancelFunc
	runMu               sync.Mutex
	mu                  sync.Mutex
	started, closed     bool
	subscriptionsClosed bool
	active              map[string]context.CancelFunc
	subscribers         map[*subscriber]bool
	stream              string
	sequence            int64
	wake                chan struct{}
	now                 func() time.Time
	onError             func(error)
}

func NewService(owner context.Context, manager *storage.Manager, searcher Searcher) (*Service, error) {
	if owner == nil || searcher == nil {
		return nil, indexer.ErrNotConfigured
	}
	store, err := NewStore(manager)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(owner)
	return &Service{store: store, searcher: searcher, ctx: ctx, cancel: cancel, active: map[string]context.CancelFunc{}, subscribers: map[*subscriber]bool{}, stream: uuid.NewString(), wake: make(chan struct{}, 1), now: time.Now}, nil
}
func (s *Service) SetErrorHandler(handler func(error)) { s.onError = handler }
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrNotRunning
	}
	if s.started {
		return nil
	}
	if err := s.store.interrupt(ctx, s.now()); err != nil {
		return err
	}
	s.started = true
	return nil
}
func (s *Service) StartSearch(ctx context.Context, input Input) (Snapshot, error) {
	query, err := indexer.ValidateQuery(input.Query)
	if err != nil || strings.TrimSpace(query.Keywords) == "" {
		return Snapshot{}, ErrInvalid
	}
	input.Query = query
	s.mu.Lock()
	ready := s.started && !s.closed && s.ctx.Err() == nil
	s.mu.Unlock()
	if !ready {
		return Snapshot{}, ErrNotRunning
	}
	id, err := s.store.create(ctx, input, s.now())
	if err != nil {
		return Snapshot{}, err
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	s.publish(id)
	return s.Get(ctx, id, Page{})
}
func (s *Service) Get(ctx context.Context, id string, page Page) (Snapshot, error) {
	return s.store.Get(ctx, id, page, s.now())
}
func (s *Service) ResourceID(ctx context.Context, id, candidate string) (string, error) {
	return s.store.ResourceID(ctx, id, candidate, s.now())
}
func (s *Service) CancelSearch(ctx context.Context, id string) (Snapshot, error) {
	if _, err := s.Get(ctx, id, Page{Limit: 1}); err != nil {
		return Snapshot{}, err
	}
	if err := s.store.cancel(ctx, id, s.now()); err != nil {
		return Snapshot{}, err
	}
	s.mu.Lock()
	cancel := s.active[id]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.publish(id)
	return s.Get(ctx, id, Page{})
}
func (s *Service) publish(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.sequence++
	event := Event{StreamID: s.stream, Sequence: s.sequence, SearchID: id}
	for sub := range s.subscribers {
		if sub.id != id {
			continue
		}
		select {
		case <-sub.channel:
		default:
		}
		sub.channel <- event
	}
}
func (s *Service) Subscribe(ctx context.Context, id string) (<-chan Event, func(), error) {
	if _, err := s.Get(ctx, id, Page{Limit: 1}); err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.subscriptionsClosed {
		return nil, nil, ErrNotRunning
	}
	if len(s.subscribers) >= 128 {
		return nil, nil, ErrBusy
	}
	sub := &subscriber{id: id, channel: make(chan Event, 1)}
	sub.channel <- Event{StreamID: s.stream, Sequence: s.sequence, SearchID: id}
	s.subscribers[sub] = true
	var once sync.Once
	return sub.channel, func() {
		once.Do(func() {
			s.mu.Lock()
			if s.subscribers[sub] {
				delete(s.subscribers, sub)
				close(sub.channel)
			}
			s.mu.Unlock()
		})
	}, nil
}
func (s *Service) report(err error) {
	if err != nil && !errors.Is(err, context.Canceled) && s.onError != nil {
		s.onError(err)
	}
}

// Run owns a two-worker interactive search pool. Persisted QUEUED rows are the
// admission queue; wakeups/done merely trigger a bounded scan, never replay work.
func (s *Service) Run(ctx context.Context) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.ctx.Err() != nil {
		return
	}
	stop := context.AfterFunc(ctx, s.cancel)
	defer stop()
	jobs := make(chan string, 2)
	done := make(chan string, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-s.ctx.Done():
					return
				case id := <-jobs:
					s.execute(id)
					select {
					case done <- id:
					case <-s.ctx.Done():
						return
					}
				}
			}
		}()
	}
	defer workers.Wait()
	inflight := map[string]bool{}
	scan := func() {
		ids, err := s.store.queued(s.ctx, s.now())
		if err != nil {
			s.report(err)
			return
		}
		for _, id := range ids {
			if inflight[id] {
				continue
			}
			select {
			case jobs <- id:
				inflight[id] = true
			default:
				return
			}
		}
	}
	scan()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.wake:
			scan()
		case id := <-done:
			delete(inflight, id)
			scan()
		}
	}
}
func (s *Service) execute(id string) {
	claimed, err := s.store.claim(s.ctx, id, s.now())
	if err != nil || !claimed {
		s.report(err)
		return
	}
	snapshot, err := s.Get(s.ctx, id, Page{Limit: 1})
	if err != nil {
		s.report(err)
		cleanup, stop := context.WithTimeout(context.WithoutCancel(s.ctx), 2*time.Second)
		defer stop()
		state, code := "FAILED", indexer.ErrorCode(err)
		if s.ctx.Err() != nil {
			state = "INTERRUPTED"
			code = "TORRENT_SEARCH_INTERRUPTED"
		}
		if finishErr := s.store.finish(cleanup, id, state, code, false, false, s.now()); finishErr != nil {
			s.report(finishErr)
		} else {
			s.publish(id)
		}
		return
	}
	budget := 10 * time.Minute
	if remaining := snapshot.ExpiresAt.Sub(s.now()); remaining < budget {
		budget = remaining
	}
	worker, cancel := context.WithTimeout(s.ctx, budget)
	s.mu.Lock()
	s.active[id] = cancel
	s.mu.Unlock()
	defer func() { cancel(); s.mu.Lock(); delete(s.active, id); s.mu.Unlock() }()
	// A durable cancellation can race registration of the in-memory cancel func.
	current, err := s.Get(worker, id, Page{Limit: 1})
	if err != nil || current.CancelRequested {
		cancel()
	}
	s.publish(id)
	var report indexer.SearchReport
	if worker.Err() == nil {
		report, err = s.searcher.SearchWithProgress(worker, snapshot.Query, func(ctx context.Context, event indexer.SearchProgress) error {
			if err := s.store.progress(ctx, id, event, s.now()); err != nil {
				return err
			}
			s.publish(id)
			return nil
		}, func(ctx context.Context, batch indexer.Batch) (bool, error) {
			truncated, err := s.store.appendBatch(ctx, id, batch, s.now())
			if err == nil {
				s.publish(id)
			}
			return truncated, err
		})
	} else {
		err = worker.Err()
	}
	cleanup, finishCancel := context.WithTimeout(context.WithoutCancel(s.ctx), 2*time.Second)
	defer finishCancel()
	latest, readErr := s.Get(cleanup, id, Page{Limit: 1})
	if readErr != nil {
		s.report(readErr)
		return
	}
	state, code := "COMPLETED", ""
	if latest.CancelRequested {
		state = "CANCELLED"
	} else if s.ctx.Err() != nil {
		state = "INTERRUPTED"
		code = "TORRENT_SEARCH_INTERRUPTED"
	} else if err != nil && !errors.Is(err, indexer.ErrIncomplete) {
		state = "FAILED"
		code = indexer.ErrorCode(err)
		if latest.Total > 0 {
			state = "PARTIAL"
		}
	} else if err != nil || report.Partial || report.Truncated || latest.Truncated {
		state = "PARTIAL"
		if err != nil {
			code = indexer.ErrorCode(err)
		}
	}
	if err = s.store.finish(cleanup, id, state, code, report.Partial || state == "PARTIAL", report.Truncated, s.now()); err != nil {
		s.report(err)
		return
	}
	s.publish(id)
}
func (s *Service) Close() {
	s.cancel()
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.CloseSubscriptions()
	s.runMu.Lock()
	s.runMu.Unlock()
}

// CloseSubscriptions releases HTTP streams before the application drains handlers.
func (s *Service) CloseSubscriptions() {
	s.mu.Lock()
	s.subscriptionsClosed = true
	for sub := range s.subscribers {
		delete(s.subscribers, sub)
		close(sub.channel)
	}
	s.mu.Unlock()
}
