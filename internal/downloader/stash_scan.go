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
	SourcePath           string
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
	sourcePath := resolveSource(task)
	plan := planDelivery(task, cfg, sourcePath)
	now := s.now().UTC()

	applyStashIntegrationPlan(task, plan, now)
	if plan.ValidationError != nil {
		recordStashIntegrationFailure(task, plan, now, plan.ValidationError)
		logging.Errorf("downloader: stash integration planning failed for task %s delivery_mode=%s: %v", task.ID, plan.DeliveryMode, plan.ValidationError)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, plan.ValidationError)
	}

	if err := s.executeDelivery(ctx, task, plan, now); err != nil {
		recordTransferFailure(task, plan, s.now().UTC(), err)
		logging.Errorf("downloader: delivery failed for task %s source=%s target=%s: %v", task.ID, plan.SourcePath, plan.ResolvedTransferPath, err)
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
	if err := s.fileOps.Transfer(ctx, plan.SourcePath, plan.TransferAction, plan.ResolvedTransferPath); err != nil {
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
	return planDelivery(task, cfg, resolveSource(task))
}

func planDelivery(task *Task, cfg stashsync.IntegrationConfig, sourcePath string) StashIntegrationPlan {
	deliveryMode := cfg.DeliveryMode
	if deliveryMode == "" {
		deliveryMode = stashsync.DeliveryModePathMap
	}

	plan := StashIntegrationPlan{
		DeliveryMode:   deliveryMode,
		SourcePath:     sourcePath,
		TransferAction: cfg.Transfer.Action,
	}

	switch deliveryMode {
	case stashsync.DeliveryModePathMap:
		plan.UserHint = "基于下载任务的实际保存路径，自动换算为 Stash 媒体库中的扫描路径。"
		if plan.SourcePath == "" {
			plan.ValidationError = errors.New("downloader: path map delivery requires a completed task content path or save path")
			plan.UserHint = "路径映射方式需要可用的内容路径或保存路径。"
			return plan
		}
		if task == nil || strings.TrimSpace(task.SavePath) == "" {
			plan.ValidationError = errors.New("downloader: path map delivery requires a resolved qBittorrent save path")
			plan.UserHint = "路径映射方式需要任务记录包含 qBittorrent 实际保存目录。"
			return plan
		}
		stashLibraryPath := strings.TrimSpace(cfg.StashLibraryPath)
		if stashLibraryPath == "" {
			plan.ValidationError = errors.New("downloader: path map delivery requires a selected stash library path")
			plan.UserHint = "请先选择目标 Stash 媒体库。"
			return plan
		}
		relative, err := relativePathWithin(task.SavePath, plan.SourcePath)
		if err != nil {
			plan.ValidationError = fmt.Errorf("downloader: resolve path map relative path: %w", err)
			plan.UserHint = "当前下载路径不在任务记录的保存目录下，无法自动换算到 Stash 媒体库。"
			return plan
		}
		plan.ResolvedScanPath = joinRootAndRelative(stashLibraryPath, relative)
	case stashsync.DeliveryModeTransfer:
		plan.NeedsTransfer = true
		plan.UserHint = "由 Moji 交付文件到媒体库，成功后再触发 Stash 扫描。"
		if plan.SourcePath == "" {
			plan.ValidationError = errors.New("downloader: transfer delivery requires a completed task content path or save path")
			plan.UserHint = "文件交付方式需要可用的内容路径或保存路径。"
			return plan
		}
		stashLibraryPath := strings.TrimSpace(cfg.StashLibraryPath)
		if stashLibraryPath == "" {
			plan.ValidationError = errors.New("downloader: transfer delivery requires a selected stash library path")
			plan.UserHint = "请先选择目标 Stash 媒体库。"
			return plan
		}
		sourceRoot := strings.TrimSpace(cfg.Transfer.MojiSourceRoot)
		targetRoot := strings.TrimSpace(cfg.Transfer.MojiTargetRoot)
		if sourceRoot == "" || targetRoot == "" {
			plan.ValidationError = errors.New("downloader: transfer delivery requires moji source and target roots")
			plan.UserHint = "请先配置 Moji 可访问的下载区目录和媒体库目录。"
			return plan
		}
		if cfg.Transfer.Action != stashsync.TransferActionCopy &&
			cfg.Transfer.Action != stashsync.TransferActionMove &&
			cfg.Transfer.Action != stashsync.TransferActionSymlink {
			plan.ValidationError = errors.New("downloader: transfer delivery requires a transfer action of COPY, MOVE, or SYMLINK")
			plan.UserHint = "请先选择交付动作：复制、移动或符号链接。"
			return plan
		}
		relative, err := relativePathWithin(sourceRoot, plan.SourcePath)
		if err != nil {
			plan.ValidationError = fmt.Errorf("downloader: resolve transfer relative path: %w", err)
			plan.UserHint = "当前下载路径不在 Moji 可访问的下载区目录下，无法执行文件交付。"
			return plan
		}
		plan.ResolvedTransferPath = joinRootAndRelative(targetRoot, relative)
		plan.ResolvedScanPath = joinRootAndRelative(stashLibraryPath, relative)
	default:
		plan.ValidationError = fmt.Errorf("downloader: unsupported ingest delivery mode %q", deliveryMode)
		plan.UserHint = "当前入库方式无效，请重新保存设置。"
	}

	return plan
}

func applyStashIntegrationPlan(task *Task, plan StashIntegrationPlan, now time.Time) {
	task.StashMode = string(plan.DeliveryMode)
	task.StashSourcePath = plan.SourcePath
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
