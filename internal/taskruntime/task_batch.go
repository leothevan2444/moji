package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/leothevan2444/moji/internal/logging"
)

const (
	MaxTaskBatchSize        = 100
	defaultTaskBatchWorkers = 4
)

var (
	ErrTaskBatchEmpty    = errors.New("task batch requires at least one task")
	ErrTaskBatchTooLarge = errors.New("task batch exceeds maximum size")
)

type TaskBatchStatus string

const (
	TaskBatchStatusSucceeded TaskBatchStatus = "SUCCEEDED"
	TaskBatchStatusSkipped   TaskBatchStatus = "SKIPPED"
	TaskBatchStatusFailed    TaskBatchStatus = "FAILED"
)

const (
	TaskBatchReasonRetried           = "RETRIED"
	TaskBatchReasonIngestStarted     = "INGEST_STARTED"
	TaskBatchReasonDeleted           = "DELETED"
	TaskBatchReasonTaskNotFound      = "TASK_NOT_FOUND"
	TaskBatchReasonNotRetryable      = "NOT_RETRYABLE"
	TaskBatchReasonNotReadyForIngest = "NOT_READY_FOR_INGEST"
	TaskBatchReasonAlreadyRunning    = "ALREADY_RUNNING"
	TaskBatchReasonRetryFailed       = "RETRY_FAILED"
	TaskBatchReasonIngestFailed      = "INGEST_FAILED"
	TaskBatchReasonDeleteFailed      = "DELETE_FAILED"
)

type TaskBatchResult struct {
	TaskID     string
	Status     TaskBatchStatus
	ReasonCode string
	Task       *Task
}

type TaskBatchSummary struct {
	RequestedCount int
	SucceededCount int
	SkippedCount   int
	FailedCount    int
}

type TaskBatchPayload struct {
	BatchID string
	Summary TaskBatchSummary
	Results []TaskBatchResult
}

type taskOperationLock struct {
	mu   sync.Mutex
	refs int
}

func (s *Service) RetryTasks(ctx context.Context, ids []string, scanner StashScanner) (TaskBatchPayload, error) {
	return s.runTaskBatch(ctx, "retry", ids, func(ctx context.Context, id string) TaskBatchResult {
		task, err := s.FindTask(ctx, id)
		if err != nil || task == nil {
			return missingOrFailedBatchResult(id, err, TaskBatchReasonRetryFailed)
		}
		if task.StageStatus != TaskStageStatusBlocked || task.Stage == TaskStageCompleted {
			return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSkipped, ReasonCode: TaskBatchReasonNotRetryable, Task: task}
		}
		next, err := s.RetryTask(ctx, id, scanner)
		if err != nil {
			return TaskBatchResult{TaskID: id, Status: TaskBatchStatusFailed, ReasonCode: TaskBatchReasonRetryFailed, Task: next}
		}
		return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSucceeded, ReasonCode: TaskBatchReasonRetried, Task: next}
	})
}

func (s *Service) ProcessTaskIngest(ctx context.Context, ids []string, scanner StashScanner) (TaskBatchPayload, error) {
	if scanner == nil {
		return TaskBatchPayload{}, errors.New("taskruntime: stash scanner is required")
	}
	return s.runTaskBatch(ctx, "ingest", ids, func(ctx context.Context, id string) TaskBatchResult {
		task, err := s.FindTask(ctx, id)
		if err != nil || task == nil {
			return missingOrFailedBatchResult(id, err, TaskBatchReasonIngestFailed)
		}
		if task.Stage == TaskStageScanning && task.StageStatus == TaskStageStatusRunning {
			return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSkipped, ReasonCode: TaskBatchReasonAlreadyRunning, Task: task}
		}
		if !shouldAllowManualStashScan(task) {
			return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSkipped, ReasonCode: TaskBatchReasonNotReadyForIngest, Task: task}
		}
		next, err := s.TriggerTaskStashScan(ctx, id, scanner)
		if err != nil {
			return TaskBatchResult{TaskID: id, Status: TaskBatchStatusFailed, ReasonCode: TaskBatchReasonIngestFailed, Task: next}
		}
		return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSucceeded, ReasonCode: TaskBatchReasonIngestStarted, Task: next}
	})
}

func (s *Service) DeleteTasks(ctx context.Context, ids []string) (TaskBatchPayload, error) {
	return s.runTaskBatch(ctx, "delete", ids, func(ctx context.Context, id string) TaskBatchResult {
		task, err := s.DeleteTask(ctx, id)
		if err != nil {
			return missingOrFailedBatchResult(id, err, TaskBatchReasonDeleteFailed)
		}
		return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSucceeded, ReasonCode: TaskBatchReasonDeleted, Task: task}
	})
}

func (s *Service) runTaskBatch(ctx context.Context, actionName string, ids []string, action func(context.Context, string) TaskBatchResult) (TaskBatchPayload, error) {
	cleaned, err := normalizeTaskBatchIDs(ids)
	if err != nil {
		return TaskBatchPayload{}, err
	}
	results := make([]TaskBatchResult, len(cleaned))
	jobs := make(chan int)
	var workers sync.WaitGroup
	workerCount := min(defaultTaskBatchWorkers, len(cleaned))
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				if ctx.Err() != nil {
					results[index] = TaskBatchResult{TaskID: cleaned[index], Status: TaskBatchStatusFailed, ReasonCode: "CANCELLED"}
					continue
				}
				results[index] = action(ctx, cleaned[index])
			}
		}()
	}
	for index := range cleaned {
		jobs <- index
	}
	close(jobs)
	workers.Wait()

	payload := TaskBatchPayload{BatchID: "batch-" + strings.TrimPrefix(newTaskID(), "task-"), Results: results}
	payload.Summary.RequestedCount = len(results)
	for _, result := range results {
		switch result.Status {
		case TaskBatchStatusSucceeded:
			payload.Summary.SucceededCount++
		case TaskBatchStatusSkipped:
			payload.Summary.SkippedCount++
		default:
			payload.Summary.FailedCount++
		}
	}
	logging.Infof("taskruntime: batch completed batch_id=%s action=%s requested=%d succeeded=%d skipped=%d failed=%d", payload.BatchID, actionName, payload.Summary.RequestedCount, payload.Summary.SucceededCount, payload.Summary.SkippedCount, payload.Summary.FailedCount)
	return payload, nil
}

func normalizeTaskBatchIDs(ids []string) ([]string, error) {
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
		return nil, ErrTaskBatchEmpty
	}
	if len(out) > MaxTaskBatchSize {
		return nil, fmt.Errorf("%w: maximum is %d", ErrTaskBatchTooLarge, MaxTaskBatchSize)
	}
	return out, nil
}

func missingOrFailedBatchResult(id string, err error, failureCode string) TaskBatchResult {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "not found") {
		return TaskBatchResult{TaskID: id, Status: TaskBatchStatusSkipped, ReasonCode: TaskBatchReasonTaskNotFound}
	}
	return TaskBatchResult{TaskID: id, Status: TaskBatchStatusFailed, ReasonCode: failureCode}
}

func (s *Service) lockTask(id string) func() {
	id = strings.TrimSpace(id)
	s.taskLocksMu.Lock()
	lock := s.taskLocks[id]
	if lock == nil {
		lock = &taskOperationLock{}
		s.taskLocks[id] = lock
	}
	lock.refs++
	s.taskLocksMu.Unlock()
	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		s.taskLocksMu.Lock()
		lock.refs--
		if lock.refs == 0 && s.taskLocks[id] == lock {
			delete(s.taskLocks, id)
		}
		s.taskLocksMu.Unlock()
	}
}
