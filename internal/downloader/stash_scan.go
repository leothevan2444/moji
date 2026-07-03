package downloader

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
	CurrentConfig() stashsync.IntegrationConfig
}

const (
	StashScanStatusPending = ""
	StashScanStatusStarted = "started"
	StashScanStatusFailed  = "failed"

	StashTransferStatusPending   = ""
	StashTransferStatusStarted   = "started"
	StashTransferStatusCompleted = "completed"
	StashTransferStatusFailed    = "failed"
)

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

		next, execErr := s.executeTaskStashIntegration(ctx, cloneTask(task), scanner)
		if persistErr := s.store.Update(ctx, next); persistErr != nil {
			logging.Errorf("downloader: persist stash integration state failed for task %s: %v", next.ID, persistErr)
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
		return nil, errors.New("downloader: stash scanner is required")
	}

	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("downloader: task %q not found", id)
	}
	if !shouldAllowManualStashScan(task) {
		return nil, fmt.Errorf("downloader: task %q is not ready for stash scan", id)
	}

	next, execErr := s.executeTaskStashIntegration(ctx, cloneTask(task), scanner)
	if persistErr := s.store.Update(ctx, next); persistErr != nil {
		logging.Errorf("downloader: persist manual stash integration state failed for task %s: %v", next.ID, persistErr)
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
		logging.Errorf("downloader: stash integration planning failed for task %s delivery_mode=%s: %v", task.ID, plan.DeliveryMode, plan.ValidationError)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, plan.ValidationError)
	}

	if err := s.executeDelivery(ctx, task, plan, now); err != nil {
		recordTransferFailure(task, plan, s.now().UTC(), err)
		logging.Errorf("downloader: delivery failed for task %s qb_source=%s moji_source=%s target=%s: %v", task.ID, plan.QBSourcePath, plan.MojiSourcePath, plan.ResolvedTransferPath, err)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
	}

	jobID, err := s.triggerScan(ctx, scanner, plan)
	now = s.now().UTC()
	task.UpdatedAt = now
	task.StashScanStartedAt = &now
	if err != nil {
		task.StashScanStatus = StashScanStatusFailed
		task.StashScanError = err.Error()
		logging.Errorf("downloader: stash scan trigger failed for task %s delivery_mode=%s path=%s: %v", task.ID, plan.DeliveryMode, plan.ResolvedScanPath, err)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
	}

	task.StashJobID = jobID
	task.StashScanStatus = StashScanStatusStarted
	task.StashScanError = ""
	logging.Infof("downloader: started stash scan for task %s delivery_mode=%s path=%s job=%s", task.ID, plan.DeliveryMode, plan.ResolvedScanPath, jobID)
	return task, nil
}

func (s *Service) executeDelivery(ctx context.Context, task *Task, plan StashIntegrationPlan, now time.Time) error {
	if !plan.NeedsTransfer {
		return nil
	}
	task.StashTransferStatus = StashTransferStatusStarted
	task.UpdatedAt = now
	if err := s.fileOps.Transfer(ctx, plan.MojiSourcePath, plan.TransferAction, plan.ResolvedTransferPath); err != nil {
		return err
	}
	task.StashTransferStatus = StashTransferStatusCompleted
	task.StashTransferError = ""
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
		plan.ValidationError = errors.New("downloader: qB source path is required for stash integration")
		plan.UserHint = "任务缺少 qB 原始内容路径，无法开始路径映射。"
		return plan
	}

	qbRoot := strings.TrimSpace(cfg.Downloads.QBRoot)
	if qbRoot == "" {
		plan.ValidationError = errors.New("downloader: qB downloads root is required")
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
			plan.ValidationError = errors.New("downloader: transfer delivery requires a transfer action of COPY, MOVE, or SYMLINK")
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
		plan.ValidationError = fmt.Errorf("downloader: unsupported ingest delivery mode %q", deliveryMode)
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
	task.StashMode = string(plan.DeliveryMode)
	task.StashSourcePath = plan.MojiSourcePath
	task.StashTransferAction = string(plan.TransferAction)
	task.StashTransferPath = plan.ResolvedTransferPath
	task.StashTransferStatus = StashTransferStatusPending
	task.StashTransferError = ""
	task.StashJobID = ""
	task.StashScanPath = plan.ResolvedScanPath
	task.StashScanStatus = StashScanStatusPending
	task.StashScanError = ""
	task.StashScanHint = plan.UserHint
	task.StashScanStartedAt = nil
	task.UpdatedAt = now
}

func recordStashIntegrationFailure(task *Task, plan StashIntegrationPlan, now time.Time, err error) {
	if plan.NeedsTransfer {
		task.StashTransferStatus = StashTransferStatusFailed
		task.StashTransferError = err.Error()
	}
	task.StashScanStatus = StashScanStatusFailed
	task.StashScanError = err.Error()
	task.UpdatedAt = now
}

func recordTransferFailure(task *Task, plan StashIntegrationPlan, now time.Time, err error) {
	task.StashTransferStatus = StashTransferStatusFailed
	task.StashTransferError = err.Error()
	task.StashScanStatus = StashScanStatusFailed
	task.StashScanError = err.Error()
	task.StashScanHint = plan.UserHint
	task.UpdatedAt = now
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
	if task.Status != TaskStatusCompleted {
		return false
	}
	if task.StashJobID != "" {
		return false
	}
	if task.StashTransferStatus == StashTransferStatusStarted {
		return false
	}
	return task.StashScanStatus == StashScanStatusPending
}

func shouldAllowManualStashScan(task *Task) bool {
	if task == nil || task.Status != TaskStatusCompleted {
		return false
	}
	if task.StashScanStatus == StashScanStatusStarted || task.StashTransferStatus == StashTransferStatusStarted {
		return false
	}
	return task.StashJobID == "" || task.StashScanStatus == StashScanStatusFailed
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}
