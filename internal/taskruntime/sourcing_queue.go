package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
)

// QueueDownloadContext persists user intent before any external library lookup,
// torrent search or submission. The worker owns execution after the request ends.
func (s *Service) QueueDownloadContext(ctx context.Context, req DownloadRequest) (*Task, error) {
	code := extractCode(req.Code)
	if code == "" {
		return nil, ErrTaskCodeRequired
	}
	unlock := s.lockTask("sourcing-code:" + code)
	defer unlock()
	existing, err := s.store.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: task %q already uses code %s", ErrDuplicateCodeTask, existing.ID, code)
	}
	req.Code = code
	if req.Source == "" {
		req.Source = TaskSourceManual
	}
	request, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	task := &Task{ID: s.newID(), Code: code, Source: req.Source, SavePath: req.SavePath, Category: req.Category, Tags: req.Tags, SourcingRequestJSON: string(request), CreatedAt: now, UpdatedAt: now}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusPending)
	if err := s.store.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create queued task: %w", err)
	}
	select {
	case s.sourcingWake <- struct{}{}:
	default:
	}
	return cloneTask(task), nil
}

// RunSourcingWorker uses persisted SOURCING tasks as the queue, including work
// interrupted by a restart. A fixed worker pool bounds external requests.
func (s *Service) RunSourcingWorker(ctx context.Context) {
	s.sourcingRunMu.Lock()
	defer s.sourcingRunMu.Unlock()
	if ctx.Err() != nil {
		return
	}
	jobs := make(chan string, 2)
	done := make(chan string, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-jobs:
					if err := s.processQueuedSourcing(ctx, id); err != nil && !errors.Is(err, context.Canceled) {
						logging.Errorf("taskruntime: queued sourcing task=%s: %v", id, err)
					}
					select {
					case done <- id:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	defer workers.Wait()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	inflight := map[string]bool{}
	scan := func() {
		tasks, err := s.store.List(ctx)
		if err != nil {
			logging.Errorf("taskruntime: list sourcing queue: %v", err)
			return
		}
		for _, task := range tasks {
			if task.Stage != TaskStageSourcing || task.StageStatus == TaskStageStatusBlocked || task.StageStatus == TaskStageStatusDone || inflight[task.ID] {
				continue
			}
			// Old synchronous requests are not queue jobs; only resume persisted intents.
			if task.SourcingRequestJSON == "" {
				continue
			}
			select {
			case jobs <- task.ID:
				inflight[task.ID] = true
			case <-ctx.Done():
				return
			default:
				return
			}
		}
	}
	scan()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.sourcingWake:
			scan()
		case <-ticker.C:
			scan()
		case id := <-done:
			delete(inflight, id)
			scan()
		}
	}
}

func (s *Service) processQueuedSourcing(parent context.Context, id string) (resultErr error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	unlock := s.lockTask(id)
	defer unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return err
	}
	if task == nil || task.Stage != TaskStageSourcing || task.StageStatus == TaskStageStatusBlocked || task.SourcingRequestJSON == "" {
		return nil
	}
	// Shutdown leaves a resumable job; a live deadline becomes actionable even
	// when it expires during the library lookup rather than during torrent search.
	defer func() {
		if parent.Err() == nil && ctx.Err() != nil {
			blockTask(task, TaskStageErrorSearch, "sourcing timed out; retry this task", s.now().UTC())
			if err := s.store.Update(parent, task); err != nil {
				resultErr = err
			}
		}
	}()
	var req DownloadRequest
	if err := json.Unmarshal([]byte(task.SourcingRequestJSON), &req); err != nil {
		blockTask(task, TaskStageErrorSearch, "unable to read queued task request", s.now().UTC())
		return s.store.Update(ctx, task)
	}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusRunning)
	clearTaskStageError(task)
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		return err
	}
	if s.libraryCodeChecker != nil {
		exists, lookupErr := s.libraryCodeChecker.HasCode(ctx, task.Code)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if lookupErr != nil || exists {
			code, message := TaskStageErrorSearch, "unable to check Stash library"
			if exists {
				code, message = TaskStageErrorDuplicateLibrary, "Stash library already contains this code"
			}
			blockTask(task, code, message, s.now().UTC())
			if err := s.store.Update(ctx, task); err != nil {
				return err
			}
			return nil
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if task.TorrentURL != "" {
		// Resume the validated selection after interruption rather than selecting a
		// different torrent. Re-adding the same torrent is idempotent in qBittorrent.
		err = s.submitTaskTorrent(ctx, task, task.TorrentURL, req.SavePath, req.Category, req.Tags, req.Paused)
	} else {
		_, err = s.runSourcingFlow(ctx, task, req)
	}

	return err
}
