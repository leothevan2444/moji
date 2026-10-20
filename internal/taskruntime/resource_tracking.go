package taskruntime

import (
	"context"
	"encoding/json"
	"time"
)

func WithSourcingRetryInterval(provider func() time.Duration) Option {
	return func(s *Service) {
		if provider != nil {
			s.sourcingRetryInterval = provider
		}
	}
}

func (s *Service) resourceCheckInterval() time.Duration {
	interval := s.sourcingRetryInterval()
	if interval < time.Hour {
		return 6 * time.Hour
	}
	return interval
}

func waitingForResource(task *Task) bool {
	return task.Stage == TaskStageSourcing && task.StageStatus == TaskStageStatusPending && task.SourcingNextCheckAt != nil
}

func resourceUnavailable(code string) bool {
	switch code {
	case TaskStageErrorNoCandidate, TaskStageErrorNoDownloadCandidate, TaskStageErrorCandidateMismatch, TaskStageErrorCandidateUnverified:
		return true
	}
	return false
}

func retryableSourcingFailure(code string) bool {
	return resourceUnavailable(code) || code == TaskStageErrorSearch || code == TaskStageErrorTorrentMetadata
}

func ensureSourcingRequest(task *Task) error {
	if task.SourcingRequestJSON != "" {
		return nil
	}
	request, err := json.Marshal(DownloadRequest{Source: task.Source, Code: task.Code, SavePath: task.SavePath, Category: task.Category, Tags: task.Tags})
	if err == nil {
		task.SourcingRequestJSON = string(request)
	}
	return err
}

// Persist expected resource absence directly as waiting work, without emitting
// a temporary BLOCKED event to connected task views.
func (s *Service) recordSourcingFailure(ctx context.Context, task *Task, code, message string) error {
	blockTask(task, code, message, s.now().UTC())
	if task.SourcingRequestJSON != "" && resourceUnavailable(code) {
		if err := s.scheduleResourceCheck(task, s.now().UTC()); err != nil {
			return err
		}
	}
	return s.store.Update(ctx, task)
}

// A future check always searches again. An expired or fuzzy selection is not
// reused, and expected resource absence is represented as pending work.
func (s *Service) scheduleResourceCheck(task *Task, checkedAt time.Time) error {
	if err := ensureSourcingRequest(task); err != nil {
		return err
	}
	if resourceUnavailable(task.StageErrorCode) {
		setTaskStage(task, TaskStageSourcing, TaskStageStatusPending)
		clearTaskStageError(task)
	}
	task.SourcingLastCheckedAt = cloneTime(&checkedAt)
	next := checkedAt.Add(s.resourceCheckInterval())
	task.SourcingNextCheckAt = &next
	task.TorrentURL = ""
	task.Candidate = Candidate{}
	task.TorrentIdentityHash, task.TorrentIdentityMagnet = "", ""
	task.FilePlanJSON = ""
	task.TorrentFileData = nil
	task.UpdatedAt = s.now().UTC()
	return nil
}

// Adopt historical blocked searches and apply changes to the configured
// interval. Each snapshot is re-read under the same lock used by manual retry.
func (s *Service) prepareResourceTracking(ctx context.Context, snapshot *Task) (*Task, error) {
	if snapshot.Stage != TaskStageSourcing || (snapshot.SourcingNextCheckAt == nil && (snapshot.StageStatus != TaskStageStatusBlocked || !retryableSourcingFailure(snapshot.StageErrorCode))) {
		return snapshot, nil
	}
	// Most scans see unchanged future schedules. Avoid a query and task lock
	// per waiting movie on every queue tick.
	if snapshot.SourcingNextCheckAt != nil && snapshot.SourcingLastCheckedAt != nil && snapshot.SourcingNextCheckAt.Equal(snapshot.SourcingLastCheckedAt.Add(s.resourceCheckInterval())) {
		return snapshot, nil
	}
	unlock := s.lockTask(snapshot.ID)
	defer unlock()
	task, err := s.store.Find(ctx, snapshot.ID)
	if err != nil {
		return nil, err
	}
	if task.Stage != TaskStageSourcing {
		return task, nil
	}
	if task.SourcingNextCheckAt == nil {
		if task.StageStatus != TaskStageStatusBlocked || !retryableSourcingFailure(task.StageErrorCode) {
			return task, nil
		}
		checkedAt := task.UpdatedAt
		if err := s.scheduleResourceCheck(task, checkedAt); err != nil {
			return nil, err
		}
	} else {
		if task.SourcingLastCheckedAt == nil {
			return task, nil
		}
		next := task.SourcingLastCheckedAt.Add(s.resourceCheckInterval())
		if next.Equal(*task.SourcingNextCheckAt) {
			return task, nil
		}
		task.SourcingNextCheckAt = &next
		task.UpdatedAt = s.now().UTC()
	}
	return task, s.store.Update(ctx, task)
}

func (s *Service) finishResourceCheck(ctx context.Context, id string) error {
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return err
	}
	if waitingForResource(task) {
		return nil
	}
	checkedAt := s.now().UTC()
	task.SourcingLastCheckedAt = &checkedAt
	task.SourcingNextCheckAt = nil
	if task.Stage == TaskStageSourcing && task.StageStatus == TaskStageStatusBlocked && retryableSourcingFailure(task.StageErrorCode) {
		if err := s.scheduleResourceCheck(task, checkedAt); err != nil {
			return err
		}
	}
	return s.store.Update(ctx, task)
}
