package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/leothevan2444/moji/internal/stashsync"
)

const stashCodeIdentifyCommand = "STASH_CODE_IDENTIFY"

type stashCodeIdentifier interface {
	PrepareCodeIdentify(context.Context, string, string, []string) (*stashsync.CodeIdentifyPlan, error)
	ApplyCodeIdentify(context.Context, string, []string, *stashsync.CodeIdentifyPlan) error
}

func (s *Service) tryStashCodeIdentify(ctx context.Context, task *Task, ingestor StashIngestor, allowPrepare bool) (bool, error) {
	identifier, ok := ingestor.(stashCodeIdentifier)
	if !ok {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var plan *stashsync.CodeIdentifyPlan
	if reader, ok := s.store.(taskCommandReader); ok {
		raw, err := reader.TaskCommandPayload(ctx, task, stashCodeIdentifyCommand)
		if err != nil {
			return false, err
		}
		if len(raw) > 0 {
			var envelope struct {
				Plan *stashsync.CodeIdentifyPlan `json:"plan"`
			}
			if len(raw) > stashsync.MaxCodeIdentifyPlanBytes+64*1024 || json.Unmarshal(raw, &envelope) != nil || envelope.Plan == nil || (envelope.Plan.Version != 1 && envelope.Plan.Version != 2) {
				return false, stashsync.ErrCodePlanChanged
			}
			plan = envelope.Plan
		}
	}
	if plan != nil && plan.Version == 1 {
		upgrader, ok := ingestor.(interface {
			UpgradeCodeIdentify(context.Context, string, []string, *stashsync.CodeIdentifyPlan) (*stashsync.CodeIdentifyPlan, error)
		})
		if !ok {
			return false, stashsync.ErrCodePlanChanged
		}
		var err error
		plan, err = upgrader.UpgradeCodeIdentify(ctx, task.StashScanPath, task.StashSceneIDs(), plan)
		if err != nil {
			return false, err
		}
	}
	if plan == nil && !allowPrepare {
		return false, nil
	}
	if plan == nil {
		var err error
		plan, err = identifier.PrepareCodeIdentify(ctx, task.Code, task.StashScanPath, task.StashSceneIDs())
		if err != nil || plan == nil {
			return false, err
		}
	}
	if !exactTaskCode(plan.Code, task.Code) {
		return false, stashsync.ErrCodePlanChanged
	}
	err := s.runTaskCommand(ctx, task, stashCodeIdentifyCommand, taskCommandTarget(task, stashCodeIdentifyCommand), map[string]any{"plan": plan}, func() error {
		return identifier.ApplyCodeIdentify(ctx, task.StashScanPath, task.StashSceneIDs(), plan)
	})
	if err == nil && plan.BlockReason == "multiple" {
		return false, stashsync.ErrCodeAmbiguous
	}
	return err == nil, err
}

func exactTaskCode(a, b string) bool {
	return normalizeCode(a) != "" && normalizeCode(a) == normalizeCode(b)
}

func stashCodeErrorCode(err error) string {
	switch {
	case errors.Is(err, stashsync.ErrStudioConflict):
		return TaskStageErrorStudioConflict
	case errors.Is(err, stashsync.ErrCodeNoMatch):
		return "STASH_CODE_NOT_FOUND"
	case errors.Is(err, stashsync.ErrCodeAmbiguous):
		return "STASH_CODE_AMBIGUOUS"
	case errors.Is(err, stashsync.ErrCodeRelations):
		return "STASH_CODE_RELATIONS_REQUIRED"
	case errors.Is(err, stashsync.ErrCodePlanChanged):
		return "STASH_CODE_PLAN_CHANGED"
	default:
		return "STASH_CODE_IDENTIFY_FAILED"
	}
}
