package downloader

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

type StashScanner interface {
	MetadataScan(ctx context.Context, req stashsync.ScanRequest) (string, error)
}

const (
	StashScanStatusPending = ""
	StashScanStatusStarted = "started"
	StashScanStatusFailed  = "failed"
)

func (s *Service) TriggerStashScans(ctx context.Context, scanner StashScanner) ([]*Task, error) {
	if scanner == nil {
		return nil, errors.New("downloader: stash scanner is required")
	}

	tasks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	updated := make([]*Task, 0, len(tasks))
	var firstErr error
	for _, task := range tasks {
		if task == nil || !shouldTriggerStashScan(task) {
			updated = append(updated, task)
			continue
		}

		next := cloneTask(task)
		jobID, err := scanner.MetadataScan(ctx, stashsync.ScanRequest{
			Paths: scanPathsForTask(next),
		})
		now := s.now().UTC()
		next.UpdatedAt = now
		next.StashScanStartedAt = &now
		if err != nil {
			next.StashScanStatus = StashScanStatusFailed
			next.StashScanError = err.Error()
			if firstErr == nil {
				firstErr = fmt.Errorf("trigger stash scan for task %q: %w", next.ID, err)
			}
		} else {
			next.StashJobID = jobID
			next.StashScanStatus = StashScanStatusStarted
			next.StashScanError = ""
		}

		if updateErr := s.store.Update(ctx, next); updateErr != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("update task %q: %w", next.ID, updateErr)
			}
		}
		updated = append(updated, next)
	}

	return updated, firstErr
}

func shouldTriggerStashScan(task *Task) bool {
	if task.Status != TaskStatusCompleted {
		return false
	}
	if task.StashJobID != "" {
		return false
	}
	return task.StashScanStatus == StashScanStatusPending
}

func scanPathsForTask(task *Task) []string {
	if task.ContentPath != "" {
		return []string{task.ContentPath}
	}
	if task.SavePath != "" {
		return []string{task.SavePath}
	}
	return nil
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}
