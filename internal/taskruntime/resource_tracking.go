package taskruntime

import (
	"context"
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
	case TaskStageErrorNoCandidate, TaskStageErrorNoDownloadCandidate, TaskStageErrorCandidateMismatch, TaskStageErrorCandidateUnverified, TaskStageErrorTorrentMetadata:
		return true
	}
	return false
}

func ensureSourcingRequest(task *Task) error {
	if task.SourcingRequestJSON != "" {
		return nil
	}
	request, err := marshalSourcingRequest(DownloadRequest{Source: task.Source, Code: task.Code, SavePath: task.SavePath, Category: task.Category, Tags: task.Tags})
	if err == nil {
		task.SourcingRequestJSON = string(request)
	}
	return err
}

// Persist expected resource absence directly as waiting work, without emitting
// a temporary BLOCKED event to connected task views.
func (s *Service) recordSourcingFailure(ctx context.Context, task *Task, code, message string) error {
	blockTask(task, code, message, s.now().UTC())
	if task.SourcingMode != "USER_SELECTED" && task.SelectedResourceID == "" && task.SourcingRequestJSON != "" && resourceUnavailable(code) {
		if err := s.scheduleResourceCheck(task, s.now().UTC()); err != nil {
			return err
		}
	}
	return s.store.Update(ctx, task)
}

// Automatic sourcing searches afresh; an explicit selection keeps its frozen
// acquisition intent. Both wait without carrying an indexer failure on the task.
func (s *Service) scheduleResourceCheck(task *Task, checkedAt time.Time) error {
	if err := ensureSourcingRequest(task); err != nil {
		return err
	}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusPending)
	clearTaskStageError(task)
	task.SourcingLastCheckedAt = cloneTime(&checkedAt)
	next := checkedAt.Add(s.resourceCheckInterval())
	task.SourcingNextCheckAt = &next
	if task.SelectedResourceID != "" {
		task.UpdatedAt = s.now().UTC()
		return nil
	}
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
	if snapshot.SelectedResourceID != "" && snapshot.SourcingNextCheckAt == nil {
		return snapshot, nil
	}
	if snapshot.Stage != TaskStageSourcing || (snapshot.SourcingNextCheckAt == nil && (snapshot.StageStatus != TaskStageStatusBlocked || !resourceUnavailable(snapshot.StageErrorCode))) {
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
	if task.Stage != TaskStageSourcing || (task.SelectedResourceID != "" && task.SourcingNextCheckAt == nil) {
		return task, nil
	}
	if task.SourcingNextCheckAt == nil {
		if task.StageStatus != TaskStageStatusBlocked || !resourceUnavailable(task.StageErrorCode) {
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
	if task.SelectedResourceID == "" && task.Stage == TaskStageSourcing && task.StageStatus == TaskStageStatusBlocked && resourceUnavailable(task.StageErrorCode) {
		if err := s.scheduleResourceCheck(task, checkedAt); err != nil {
			return err
		}
	}
	return s.store.Update(ctx, task)
}

// Even a site deadline must not hide a failed diagnostic write. Only this
// bounded local persistence may outlive the expired operation context.
func (s *Service) blockSourcingState(ctx context.Context, task *Task) error {
	blockTask(task, TaskStageErrorSourcingState, "unable to persist indexer diagnostics", s.now().UTC())
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	return s.store.Update(writeCtx, task)
}
