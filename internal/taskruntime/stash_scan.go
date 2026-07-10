package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
)

type StashScanner interface {
	MetadataScan(ctx context.Context, req stashsync.ScanRequest) (string, error)
	FindJob(ctx context.Context, id string) (*stashsync.Job, error)
	CurrentConfig() stashsync.IntegrationConfig
}

type StashIntegrationPlan struct {
	DeliveryMode         stashsync.DeliveryMode
	QBSourcePath         string
	RelativePath         string
	MojiSourcePath       string
	ResolvedTransferPath string
	ResolvedScanPath     string
	TransferAction       stashsync.TransferAction
	NeedsTransfer        bool
	ValidationError      error
	UserHint             string
}

func (s *Service) TriggerStashScans(ctx context.Context, scanner StashScanner) ([]*Task, error) {
	if scanner == nil {
		return nil, errors.New("taskruntime: stash scanner is required")
	}

	tasks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	updated := make([]*Task, 0, len(tasks))
	var firstErr error
	for _, task := range tasks {
		if task == nil {
			updated = append(updated, task)
			continue
		}
		if task.Stage == TaskStageScanning && task.StageStatus == TaskStageStatusRunning {
			next, pollErr := s.syncStashScanJob(ctx, cloneTask(task), scanner)
			if persistErr := s.store.Update(ctx, next); persistErr != nil {
				logging.Errorf("taskruntime: persist stash scan job state failed for task %s: %v", next.ID, persistErr)
				if firstErr == nil {
					firstErr = fmt.Errorf("update task %q: %w", next.ID, persistErr)
				}
			}
			if pollErr != nil && firstErr == nil {
				firstErr = pollErr
			}
			updated = append(updated, next)
			continue
		}
		if !shouldTriggerStashScan(task) {
			updated = append(updated, task)
			continue
		}

		next, execErr := s.executeTaskStashIntegration(ctx, cloneTask(task), scanner)
		if persistErr := s.store.Update(ctx, next); persistErr != nil {
			logging.Errorf("taskruntime: persist stash integration state failed for task %s: %v", next.ID, persistErr)
			if firstErr == nil {
				firstErr = fmt.Errorf("update task %q: %w", next.ID, persistErr)
			}
		}
		if execErr != nil && firstErr == nil {
			firstErr = execErr
		}
		updated = append(updated, next)
	}

	return updated, firstErr
}

func (s *Service) TriggerTaskStashScan(ctx context.Context, id string, scanner StashScanner) (*Task, error) {
	if scanner == nil {
		return nil, errors.New("taskruntime: stash scanner is required")
	}

	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	if !shouldAllowManualStashScan(task) {
		return nil, fmt.Errorf("taskruntime: task %q is not ready for stash scan", id)
	}

	next, execErr := s.executeTaskStashIntegration(ctx, cloneTask(task), scanner)
	if persistErr := s.store.Update(ctx, next); persistErr != nil {
		logging.Errorf("taskruntime: persist manual stash integration state failed for task %s: %v", next.ID, persistErr)
		return nil, fmt.Errorf("update task %q: %w", next.ID, persistErr)
	}
	if execErr != nil {
		return next, execErr
	}
	return next, nil
}

func (s *Service) executeTaskStashIntegration(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	cfg := scanner.CurrentConfig()
	qbSourcePath := resolveSource(task)
	plan := planDelivery(cfg, qbSourcePath)
	now := s.now().UTC()

	applyStashIntegrationPlan(task, plan, now)
	if plan.ValidationError != nil {
		recordStashIntegrationFailure(task, plan, now, plan.ValidationError)
		blockTask(task, TaskStageErrorTransferPlan, plan.ValidationError.Error(), now)
		logging.Errorf("taskruntime: stash integration planning failed for task %s delivery_mode=%s: %v", task.ID, plan.DeliveryMode, plan.ValidationError)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, plan.ValidationError)
	}

	if plan.NeedsTransfer {
		setTaskStage(task, TaskStageTransferring, TaskStageStatusRunning)
	} else {
		setTaskStage(task, TaskStageScanning, TaskStageStatusPending)
	}
	clearTaskStageError(task)
	if err := s.executeDelivery(ctx, task, plan, now); err != nil {
		recordTransferFailure(task, plan, s.now().UTC(), err)
		blockTask(task, TaskStageErrorTransfer, err.Error(), s.now().UTC())
		logging.Errorf("taskruntime: delivery failed for task %s qb_source=%s moji_source=%s target=%s: %v", task.ID, plan.QBSourcePath, plan.MojiSourcePath, plan.ResolvedTransferPath, err)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
	}

	setTaskStage(task, TaskStageScanning, TaskStageStatusRunning)
	jobID, err := s.triggerScan(ctx, scanner, plan)
	now = s.now().UTC()
	task.UpdatedAt = now
	task.StashScanStartedAt = &now
	if err != nil {
		task.StashScanError = err.Error()
		blockTask(task, TaskStageErrorScanTrigger, err.Error(), now)
		logging.Errorf("taskruntime: stash scan trigger failed for task %s delivery_mode=%s path=%s: %v", task.ID, plan.DeliveryMode, plan.ResolvedScanPath, err)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
	}

	task.StashScanJobID = jobID
	task.StashScanError = ""
	clearTaskStageError(task)
	logging.Infof("taskruntime: started stash scan for task %s delivery_mode=%s path=%s job=%s", task.ID, plan.DeliveryMode, plan.ResolvedScanPath, jobID)
	return task, nil
}

func (s *Service) syncStashScanJob(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	if task == nil || strings.TrimSpace(task.StashScanJobID) == "" {
		return task, nil
	}

	job, err := scanner.FindJob(ctx, task.StashScanJobID)
	if err != nil {
		task.StashScanError = err.Error()
		blockTask(task, TaskStageErrorScanTrigger, err.Error(), s.now().UTC())
		return task, fmt.Errorf("find stash job %q for task %q: %w", task.StashScanJobID, task.ID, err)
	}
	if job == nil {
		return task, nil
	}

	now := s.now().UTC()
	switch strings.ToUpper(strings.TrimSpace(job.Status)) {
	case "FINISHED":
		setTaskStage(task, TaskStageCompleted, TaskStageStatusDone)
		clearTaskStageError(task)
		task.StashScanError = ""
		task.UpdatedAt = now
	case "FAILED", "CANCELLED":
		task.StashScanError = firstNonEmpty([]string{stringValue(job.Error), "stash scan job did not complete successfully"})
		blockTask(task, TaskStageErrorScanTrigger, task.StashScanError, now)
	default:
		setTaskStage(task, TaskStageScanning, TaskStageStatusRunning)
		task.UpdatedAt = now
	}

	return task, nil
}

func (s *Service) executeDelivery(ctx context.Context, task *Task, plan StashIntegrationPlan, now time.Time) error {
	if !plan.NeedsTransfer {
		return nil
	}
	task.UpdatedAt = now
	if err := s.fileOps.Transfer(ctx, plan.MojiSourcePath, plan.TransferAction, plan.ResolvedTransferPath); err != nil {
		return err
	}
	task.TransferError = ""
	task.StashScanPath = plan.ResolvedScanPath
	task.UpdatedAt = s.now().UTC()
	return nil
}

func (s *Service) triggerScan(ctx context.Context, scanner StashScanner, plan StashIntegrationPlan) (string, error) {
	return scanner.MetadataScan(ctx, stashsync.ScanRequest{Paths: []string{plan.ResolvedScanPath}})
}

func planTaskStashIntegration(task *Task, cfg stashsync.IntegrationConfig) StashIntegrationPlan {
	return planDelivery(cfg, resolveSource(task))
}

func planDelivery(cfg stashsync.IntegrationConfig, qbSourcePath string) StashIntegrationPlan {
	deliveryMode := cfg.DeliveryMode
	if deliveryMode == "" {
		deliveryMode = stashsync.DeliveryModePathMap
	}

	plan := StashIntegrationPlan{
		DeliveryMode:   deliveryMode,
		QBSourcePath:   qbSourcePath,
		TransferAction: cfg.Transfer.Action,
	}
	plan.UserHint = pathMappingHint(deliveryMode)

	if plan.QBSourcePath == "" {
		plan.ValidationError = errors.New("taskruntime: qB source path is required for stash integration")
		plan.UserHint = "任务缺少 qB 原始内容路径，无法开始路径映射。"
		return plan
	}

	qbRoot := strings.TrimSpace(cfg.Downloads.QBRoot)
	if qbRoot == "" {
		plan.ValidationError = errors.New("taskruntime: qB downloads root is required")
		plan.UserHint = "请先配置 qB 视角下的下载根目录。"
		return plan
	}

	relative, err := relativePathWithin(qbRoot, plan.QBSourcePath)
	if err != nil {
		plan.ValidationError = fmt.Errorf("resolve qB relative path failed: %w", err)
		plan.UserHint = "当前任务的 qB 路径不在已配置的 qB 下载根目录下。"
		return plan
	}
	plan.RelativePath = relative

	scanPath, err := buildMappedPath(cfg.Library.StashRoot, relative, "build Stash scan path failed", "library.stash_root")
	if err != nil {
		plan.ValidationError = err
		plan.UserHint = "请先配置 Stash 视角下的媒体库根目录。"
		return plan
	}
	plan.ResolvedScanPath = scanPath

	switch deliveryMode {
	case stashsync.DeliveryModePathMap:
		return plan
	case stashsync.DeliveryModeTransfer:
		plan.NeedsTransfer = true
		if cfg.Transfer.Action != stashsync.TransferActionCopy &&
			cfg.Transfer.Action != stashsync.TransferActionMove &&
			cfg.Transfer.Action != stashsync.TransferActionSymlink {
			plan.ValidationError = errors.New("taskruntime: transfer delivery requires a transfer action of COPY, MOVE, or SYMLINK")
			plan.UserHint = "请先选择交付动作：复制、移动或符号链接。"
			return plan
		}

		mojiSourcePath, err := buildMappedPath(cfg.Downloads.MojiRoot, relative, "build Moji transfer source path failed", "downloads.moji_root")
		if err != nil {
			plan.ValidationError = err
			plan.UserHint = "请先配置 Moji 视角下的下载根目录。"
			return plan
		}
		plan.MojiSourcePath = mojiSourcePath

		transferPath, err := buildMappedPath(cfg.Library.MojiRoot, relative, "build Moji transfer target path failed", "library.moji_root")
		if err != nil {
			plan.ValidationError = err
			plan.UserHint = "请先配置 Moji 视角下的媒体库根目录。"
			return plan
		}
		plan.ResolvedTransferPath = transferPath
		return plan
	default:
		plan.ValidationError = fmt.Errorf("taskruntime: unsupported ingest delivery mode %q", deliveryMode)
		plan.UserHint = "当前入库方式无效，请重新保存设置。"
		return plan
	}
}

func pathMappingHint(mode stashsync.DeliveryMode) string {
	switch mode {
	case stashsync.DeliveryModeTransfer:
		return "Moji 会先基于 qB 下载根路径计算相对路径，再翻译成自己的源路径、交付目标路径和 Stash 扫描路径。"
	default:
		return "Moji 会先基于 qB 下载根路径计算相对路径，再翻译成 Stash 扫描路径。"
	}
}

func buildMappedPath(root string, relative string, label string, field string) (string, error) {
	cleanRoot := strings.TrimSpace(root)
	if cleanRoot == "" {
		return "", fmt.Errorf("%s: %s is required", label, field)
	}
	return joinRootAndRelative(cleanRoot, relative), nil
}

func applyStashIntegrationPlan(task *Task, plan StashIntegrationPlan, now time.Time) {
	task.DeliveryMode = string(plan.DeliveryMode)
	task.MojiSourcePath = plan.MojiSourcePath
	task.TransferAction = string(plan.TransferAction)
	task.MojiTransferPath = plan.ResolvedTransferPath
	task.TransferError = ""
	task.StashScanJobID = ""
	task.StashScanPath = plan.ResolvedScanPath
	task.StashScanError = ""
	task.StashScanHint = plan.UserHint
	task.StashScanStartedAt = nil
	task.UpdatedAt = now
	refreshTaskStageFields(task)
}

func recordStashIntegrationFailure(task *Task, plan StashIntegrationPlan, now time.Time, err error) {
	if plan.NeedsTransfer {
		task.TransferError = err.Error()
	}
	task.StashScanError = err.Error()
	task.UpdatedAt = now
	refreshTaskStageFields(task)
}

func recordTransferFailure(task *Task, plan StashIntegrationPlan, now time.Time, err error) {
	task.TransferError = err.Error()
	task.StashScanError = err.Error()
	task.StashScanHint = plan.UserHint
	task.UpdatedAt = now
	refreshTaskStageFields(task)
}

func resolveSource(task *Task) string {
	if task == nil {
		return ""
	}
	if path := strings.TrimSpace(task.ContentPath); path != "" {
		return path
	}
	return strings.TrimSpace(task.SavePath)
}

func relativePathWithin(root string, path string) (string, error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanRoot == "" || cleanPath == "" {
		return "", errors.New("root and path are required")
	}
	relative, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q is not within %q", cleanPath, cleanRoot)
	}
	return relative, nil
}

func joinRootAndRelative(root string, relative string) string {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	cleanRelative := filepath.Clean(strings.TrimSpace(relative))
	if cleanRelative == "." {
		return cleanRoot
	}
	return filepath.Join(cleanRoot, cleanRelative)
}

func shouldTriggerStashScan(task *Task) bool {
	if task == nil {
		return false
	}
	if task.Stage != TaskStagePendingIngest || task.StageStatus != TaskStageStatusPending {
		return false
	}
	if task.StashScanJobID != "" {
		return false
	}
	return true
}

func shouldAllowManualStashScan(task *Task) bool {
	if task == nil {
		return false
	}
	if task.Stage != TaskStagePendingIngest && task.Stage != TaskStageTransferring && task.Stage != TaskStageScanning {
		return false
	}
	if task.Stage == TaskStageScanning && task.StageStatus == TaskStageStatusRunning {
		return false
	}
	return task.StageStatus == TaskStageStatusPending || task.StageStatus == TaskStageStatusBlocked
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
