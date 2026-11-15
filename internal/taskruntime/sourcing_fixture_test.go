package taskruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/tracker"
)

// runSourcingFixture exercises the execution engine from an already admitted
// film. Queue admission, confirmation and worker recovery have separate tests.
func (s *Service) runSourcingFixture(ctx context.Context, req DownloadRequest) (*Task, error) {
	if req.Film == nil || req.Film.Validate() != nil {
		return nil, film.ErrRequired
	}
	req.Code = req.Film.Code
	code := normalizeCode(req.Code)
	if code == "" {
		return nil, ErrTaskCodeRequired
	}
	if err := s.ensureTaskCodeCanBeCreated(ctx, code); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	source := req.Source
	if source == "" {
		source = TaskSourceManual
	}
	task := &Task{Film: req.Film, SourcingMode: "AUTO",
		ID:        s.newID(),
		Source:    source,
		Code:      code,
		SavePath:  req.SavePath,
		Category:  req.Category,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusRunning)

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("taskruntime: create task failed for code %q: %v", code, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof("taskruntime: created %s task %s for code %q", strings.ToLower(string(source)), task.ID, code)

	if s.indexers != nil {
		req.Trackers = tracker.InstanceIDsFromLegacyTrackers(req.Trackers)
	}
	return s.runSourcingFlow(ctx, task, req)
}
