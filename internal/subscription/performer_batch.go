package subscription

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/performer"
)

const (
	MaxPerformerBatchSize        = 100
	defaultPerformerBatchWorkers = 4
)

var (
	ErrPerformerBatchEmpty    = errors.New("performer batch requires at least one performer")
	ErrPerformerBatchTooLarge = errors.New("performer batch exceeds maximum size")
)

type performerOperationLock struct {
	mu   sync.Mutex
	refs int
}

func (s *Service) SubscribePerformers(ctx context.Context, ids []string) (PerformerBatchPayload, error) {
	return s.runPerformerBatch(ctx, "subscribe", ids, func(ctx context.Context, id string) PerformerBatchResult {
		unlock := s.lockPerformerOperation(id)
		defer unlock()
		item, found, err := s.loadBatchPerformer(ctx, id)
		if err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonStashUpdateFailed}
		}
		if !found {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonPerformerNotFound}
		}
		if item.Subscribed {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonAlreadySubscribed, Performer: &item}
		}
		state, err := s.subscribePerformer(ctx, id)
		if err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonStashUpdateFailed, Performer: &item}
		}
		return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSucceeded, ReasonCode: PerformerBatchReasonSubscribed, Performer: &state.Performer, State: &state}
	})
}

func (s *Service) UnsubscribePerformers(ctx context.Context, ids []string) (PerformerBatchPayload, error) {
	return s.runPerformerBatch(ctx, "unsubscribe", ids, func(ctx context.Context, id string) PerformerBatchResult {
		unlock := s.lockPerformerOperation(id)
		defer unlock()
		item, found, err := s.loadBatchPerformer(ctx, id)
		if err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonStashUpdateFailed}
		}
		if !found {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonPerformerNotFound}
		}
		if !item.Subscribed {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonNotSubscribed, Performer: &item}
		}
		if err := s.unsubscribePerformer(ctx, id); err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonStashUpdateFailed, Performer: &item}
		}
		item.Subscribed = false
		return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSucceeded, ReasonCode: PerformerBatchReasonUnsubscribed, Performer: &item}
	})
}

func (s *Service) RefreshSubscribedPerformers(ctx context.Context, ids []string) (PerformerBatchPayload, error) {
	return s.runPerformerBatch(ctx, "refresh", ids, func(ctx context.Context, id string) PerformerBatchResult {
		unlock := s.lockPerformerOperation(id)
		defer unlock()
		item, found, err := s.loadBatchPerformer(ctx, id)
		if err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonRefreshFailed}
		}
		if !found {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonPerformerNotFound}
		}
		if !item.Subscribed {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSkipped, ReasonCode: PerformerBatchReasonNotSubscribed, Performer: &item}
		}
		state, err := s.refreshSubscribedPerformer(ctx, id)
		if err != nil {
			return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonRefreshFailed, Performer: &item, State: &state}
		}
		return PerformerBatchResult{PerformerID: id, Status: PerformerBatchStatusSucceeded, ReasonCode: PerformerBatchReasonRefreshed, Performer: &state.Performer, State: &state}
	})
}

func (s *Service) loadBatchPerformer(ctx context.Context, id string) (performer.Performer, bool, error) {
	raw, err := s.stash.FindPerformerByID(ctx, id)
	if err != nil {
		return performer.Performer{}, false, err
	}
	if raw == nil {
		return performer.Performer{}, false, nil
	}
	item := performerFromStash(raw, s.customFieldKey)
	item.ImagePath = s.proxyStashImage(ctx, item.ImagePath)
	return item, true, nil
}

func (s *Service) runPerformerBatch(ctx context.Context, action string, ids []string, operation func(context.Context, string) PerformerBatchResult) (PerformerBatchPayload, error) {
	cleaned, err := normalizePerformerBatchIDs(ids)
	if err != nil {
		return PerformerBatchPayload{}, err
	}
	results := make([]PerformerBatchResult, len(cleaned))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for range min(defaultPerformerBatchWorkers, len(cleaned)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				if ctx.Err() != nil {
					results[index] = PerformerBatchResult{PerformerID: cleaned[index], Status: PerformerBatchStatusFailed, ReasonCode: PerformerBatchReasonCancelled}
					continue
				}
				results[index] = operation(ctx, cleaned[index])
			}
		}()
	}
	for index := range cleaned {
		jobs <- index
	}
	close(jobs)
	workers.Wait()

	payload := PerformerBatchPayload{BatchID: "performer-batch-" + uuid.NewString(), Results: results}
	payload.Summary.RequestedCount = len(results)
	for _, result := range results {
		switch result.Status {
		case PerformerBatchStatusSucceeded:
			payload.Summary.SucceededCount++
		case PerformerBatchStatusSkipped:
			payload.Summary.SkippedCount++
		default:
			payload.Summary.FailedCount++
		}
	}
	logging.Infof("subscription: performer batch completed batch_id=%s action=%s requested=%d succeeded=%d skipped=%d failed=%d", payload.BatchID, action, payload.Summary.RequestedCount, payload.Summary.SucceededCount, payload.Summary.SkippedCount, payload.Summary.FailedCount)
	return payload, nil
}

func normalizePerformerBatchIDs(ids []string) ([]string, error) {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, ErrPerformerBatchEmpty
	}
	if len(out) > MaxPerformerBatchSize {
		return nil, fmt.Errorf("%w: maximum is %d", ErrPerformerBatchTooLarge, MaxPerformerBatchSize)
	}
	return out, nil
}

func (s *Service) lockPerformerOperation(id string) func() {
	id = strings.TrimSpace(id)
	s.operationMu.Lock()
	lock := s.operations[id]
	if lock == nil {
		lock = &performerOperationLock{}
		s.operations[id] = lock
	}
	lock.refs++
	s.operationMu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		s.operationMu.Lock()
		lock.refs--
		if lock.refs == 0 && s.operations[id] == lock {
			delete(s.operations, id)
		}
		s.operationMu.Unlock()
	}
}
