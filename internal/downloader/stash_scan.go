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
	Mode                 stashsync.IntegrationMode
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
	plan := planTaskStashIntegration(task, cfg)
	now := s.now().UTC()

	applyStashIntegrationPlan(task, plan, now)
	if plan.ValidationError != nil {
		recordStashIntegrationFailure(task, plan, now, plan.ValidationError)
		logging.Errorf("downloader: stash integration planning failed for task %s mode=%s: %v", task.ID, plan.Mode, plan.ValidationError)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, plan.ValidationError)
	}

	if plan.NeedsTransfer {
		task.StashTransferStatus = StashTransferStatusStarted
		task.UpdatedAt = now
		if err := s.fileOps.Transfer(ctx, plan.SourcePath, plan.TransferAction, plan.ResolvedTransferPath); err != nil {
			recordTransferFailure(task, plan, s.now().UTC(), err)
			logging.Errorf("downloader: file transfer failed for task %s source=%s target=%s: %v", task.ID, plan.SourcePath, plan.ResolvedTransferPath, err)
			return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
		}
		task.StashTransferStatus = StashTransferStatusCompleted
		task.StashTransferError = ""
		task.StashScanPath = plan.ResolvedScanPath
		task.UpdatedAt = s.now().UTC()
	}

	jobID, err := scanner.MetadataScan(ctx, stashsync.ScanRequest{Paths: []string{plan.ResolvedScanPath}})
	now = s.now().UTC()
	task.UpdatedAt = now
	task.StashScanStartedAt = &now
	if err != nil {
		task.StashScanStatus = StashScanStatusFailed
		task.StashScanError = err.Error()
		logging.Errorf("downloader: stash scan trigger failed for task %s mode=%s path=%s: %v", task.ID, plan.Mode, plan.ResolvedScanPath, err)
		return task, fmt.Errorf("trigger stash scan for task %q: %w", task.ID, err)
	}

	task.StashJobID = jobID
	task.StashScanStatus = StashScanStatusStarted
	task.StashScanError = ""
	logging.Infof("downloader: started stash scan for task %s mode=%s path=%s job=%s", task.ID, plan.Mode, plan.ResolvedScanPath, jobID)
	return task, nil
}

func planTaskStashIntegration(task *Task, cfg stashsync.IntegrationConfig) StashIntegrationPlan {
	mode := cfg.Mode
	if mode == "" {
		mode = stashsync.IntegrationModeSharedStorage
	}

	plan := StashIntegrationPlan{
		Mode:           mode,
		SourcePath:     taskSourcePath(task),
		TransferAction: cfg.TransferAction,
	}

	switch mode {
	case stashsync.IntegrationModeSharedStorage:
		plan.UserHint = "将 qBittorrent 可见路径映射为 Stash 可见路径。"
		if plan.SourcePath == "" {
			plan.ValidationError = errors.New("downloader: shared storage mode requires a completed task content path or save path")
			plan.UserHint = "共享存储模式需要可用的内容路径或保存路径。"
			return plan
		}
		sourcePrefix := strings.TrimSpace(cfg.QBittorrentPathPrefix)
		targetPrefix := strings.TrimSpace(cfg.StashPathPrefix)
		if sourcePrefix == "" || targetPrefix == "" {
			plan.ValidationError = errors.New("downloader: shared storage mode requires qbittorrent and stash path prefixes")
			plan.UserHint = "请先配置 qBittorrent 路径前缀和 Stash 路径前缀。"
			return plan
		}
		mappedPath, ok := replacePathPrefix(plan.SourcePath, sourcePrefix, targetPrefix)
		if !ok {
			plan.ValidationError = fmt.Errorf("downloader: source path %q does not match qBittorrent path prefix %q", plan.SourcePath, sourcePrefix)
			plan.UserHint = "当前下载路径与配置的 qBittorrent 路径前缀不匹配。"
			return plan
		}
		plan.ResolvedScanPath = mappedPath
	case stashsync.IntegrationModeFileTransfer:
		plan.NeedsTransfer = true
		plan.UserHint = "由 Moji 执行文件搬运，成功后再触发 Stash 单文件扫描。"
		if plan.SourcePath == "" {
			plan.ValidationError = errors.New("downloader: file transfer mode requires a completed task content path or save path")
			plan.UserHint = "文件搬运模式需要可用的内容路径或保存路径。"
			return plan
		}
		targetDir := strings.TrimSpace(cfg.TransferTargetPath)
		if targetDir == "" {
			plan.ValidationError = errors.New("downloader: file transfer mode requires a transfer target path")
			plan.UserHint = "请先配置文件搬运目标目录。"
			return plan
		}
		if cfg.TransferAction != stashsync.TransferActionCopy && cfg.TransferAction != stashsync.TransferActionMove {
			plan.ValidationError = errors.New("downloader: file transfer mode requires a transfer action of COPY or MOVE")
			plan.UserHint = "请先选择文件搬运动作：复制或移动。"
			return plan
		}
		plan.ResolvedTransferPath = filepath.Join(targetDir, filepath.Base(plan.SourcePath))
		plan.ResolvedScanPath = plan.ResolvedTransferPath
	case stashsync.IntegrationModeLibraryScan:
		plan.UserHint = "当前模式会扫描整个 Stash 库目录，无法精确定位单个下载文件。"
		plan.ResolvedScanPath = strings.TrimSpace(cfg.LibraryPath)
		if plan.ResolvedScanPath == "" {
			plan.ValidationError = errors.New("downloader: library scan mode requires a stash library path")
			plan.UserHint = "请先配置 Stash 库目录。"
			return plan
		}
	default:
		plan.ValidationError = fmt.Errorf("downloader: unsupported stash integration mode %q", mode)
		plan.UserHint = "当前 Stash 集成模式无效，请重新保存设置。"
	}

	return plan
}

func applyStashIntegrationPlan(task *Task, plan StashIntegrationPlan, now time.Time) {
	task.StashMode = string(plan.Mode)
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

func taskSourcePath(task *Task) string {
	if task == nil {
		return ""
	}
	if path := strings.TrimSpace(task.ContentPath); path != "" {
		return path
	}
	return strings.TrimSpace(task.SavePath)
}

func replacePathPrefix(path string, sourcePrefix string, targetPrefix string) (string, bool) {
	cleanPath := filepath.Clean(path)
	cleanSource := filepath.Clean(sourcePrefix)
	if cleanPath == cleanSource {
		return filepath.Clean(targetPrefix), true
	}
	if !strings.HasPrefix(cleanPath, cleanSource+string(filepath.Separator)) {
		return "", false
	}
	relative := strings.TrimPrefix(cleanPath, cleanSource)
	return filepath.Clean(filepath.Clean(targetPrefix) + relative), true
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
