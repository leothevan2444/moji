package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

var ErrRestartUnavailable = errors.New("task cannot restart sourcing while completed or processing media")
var ErrRestartCleanup = errors.New("task restart cleanup is incomplete; restart sourcing again to resume")

const TaskStageErrorRestartPending = "TASK_RESTART_CLEANUP_PENDING"
const TaskStageErrorManualSource = "MANUAL_SOURCE_REQUIRED"

type restartState struct {
	Version           int
	Generation        int64 `json:",omitempty"`
	Delivery          *mediafiles.DeliveryRemoval
	Downloads         *mediafiles.DownloadRemoval
	DownloaderRemoved bool
}

type restartSceneCleaner interface {
	RemoveIngestScenes(context.Context, string, []string) error
}

// RestartTaskSourcing removes all task-owned execution artifacts and returns the
// original task to manual selection. Cleanup checkpoints survive partial deletion.
func (s *Service) RestartTaskSourcing(ctx context.Context, id string, scanner StashScanner) (*Task, error) {
	unlock := s.lockTask(id)
	defer unlock()
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrRestartUnavailable
	}
	return s.restartTaskSourcingLocked(ctx, task, scanner)
}

func (s *Service) restartTaskSourcingLocked(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	if task.Stage == TaskStageSourcing && task.StageErrorCode == TaskStageErrorManualSource && task.RestartStateJSON == "" {
		return task, nil
	}
	s.mediaMu.Lock()
	_, active := s.mediaJobs[task.ID]
	s.mediaMu.Unlock()
	if active || task.Code == "" || task.Stage == TaskStageCompleted || (task.Stage == TaskStageSourcing && task.RestartStateJSON == "") ||
		(task.RestartStateJSON == "" && task.Stage != TaskStageDownloading && task.StageStatus != TaskStageStatusBlocked && !(task.Stage == TaskStagePendingIngest && task.StageStatus == TaskStageStatusPending)) {
		return task, ErrRestartUnavailable
	}
	if task.RestartStateJSON == "" && task.Stage == TaskStageScanning {
		if scanner == nil {
			return task, ErrRestartUnavailable
		}
		if id := stashJobID(task); id != "" {
			job, err := scanner.FindJob(ctx, id)
			if err != nil {
				return task, err
			}
			if job != nil && job.Status != "FINISHED" && job.Status != "FAILED" && job.Status != "CANCELLED" {
				return task, ErrRestartUnavailable
			}
		}
	}
	state := restartState{Version: 1}
	if task.RestartStateJSON != "" {
		state.Version = 0
		if err := json.Unmarshal([]byte(task.RestartStateJSON), &state); err != nil {
			return task, err
		}
		if state.Version != 1 || (task.Revision > 0 && state.Generation != task.Generation) {
			return task, fmt.Errorf("%w: unsupported cleanup checkpoint", ErrRestartCleanup)
		}
	} else {
		// Recover publication which succeeded before its task checkpoint, such as
		// a worker failure while cleaning MOVE sources.
		if task.MojiTransferPath == "" && s.mediaConfig != nil {
			cfg := s.mediaConfig()
			if cfg.Library.MojiRoot != "" {
				target := filepath.Join(cfg.Library.MojiRoot, task.Code)
				if _, err := os.Lstat(target); err == nil {
					task.MojiTransferPath = target
					if cfg.Library.StashRoot != "" {
						task.StashScanPath = filepath.Join(cfg.Library.StashRoot, task.Code)
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					return task, err
				}
			}
		}
		var err error
		if task.MojiTransferPath != "" {
			state.Delivery, err = mediafiles.PlanDeliveryRemoval(ctx, task.ID, task.Code, task.MojiTransferPath)
			if err != nil {
				return task, fmt.Errorf("%w: %v", ErrRestartCleanup, err)
			}
			if task.StashScanPath != "" {
				if _, ok := scanner.(restartSceneCleaner); !ok {
					return task, ErrRestartUnavailable
				}
			}
		} else if task.StashScanPath != "" || len(task.StashSceneIDs()) > 0 {
			return task, fmt.Errorf("%w: cannot establish delivery ownership", ErrRestartCleanup)
		}
		state.Downloads, err = s.planRestartDownloads(ctx, task)
		if err != nil {
			return task, fmt.Errorf("%w: %v", ErrRestartCleanup, err)
		}
		// A new durable generation invalidates old worker/file-selection snapshots.
		// Legacy development stores have no revision protocol; imports start at zero.
		if task.Revision > 0 {
			task.Generation++
			state.Generation = task.Generation
		}
		data, _ := json.Marshal(state)
		task.RestartStateJSON = string(data)
		blockTask(task, TaskStageErrorRestartPending, ErrRestartCleanup.Error(), s.now().UTC())
		if err := s.store.Update(ctx, task); err != nil {
			return task, err
		}
	}
	fail := func(err error) (*Task, error) {
		blockTask(task, TaskStageErrorRestartPending, err.Error(), s.now().UTC())
		if persistErr := s.store.Update(ctx, task); persistErr != nil {
			return task, persistErr
		}
		return task, fmt.Errorf("%w: %v", ErrRestartCleanup, err)
	}
	if !state.DownloaderRemoved {
		if hash := restartTorrentHash(task); hash != "" {
			// A restart always deletes downloaded bytes, independently of the
			// ordinary task deletion policy.
			if err := s.runTaskCommand(ctx, task, "TASK_RESTART_DELETE", hash, map[string]any{"delete_files": true}, func() error { return s.qbt.DeleteTorrents(ctx, []string{hash}, true) }); err != nil {
				return fail(err)
			}
			if err := s.waitRestartTorrentRemoval(ctx, hash); err != nil {
				return fail(err)
			}
		}
		state.DownloaderRemoved = true
		data, _ := json.Marshal(state)
		task.RestartStateJSON = string(data)
		if err := s.store.Update(ctx, task); err != nil {
			return task, err
		}
	}
	if state.Downloads != nil {
		if err := mediafiles.RemoveDownloads(ctx, *state.Downloads); err != nil {
			return fail(err)
		}
	}
	if state.Delivery != nil {
		if task.StashScanPath != "" {
			cleaner, ok := scanner.(restartSceneCleaner)
			if !ok {
				return fail(errors.New("Stash scene cleanup is unavailable"))
			}
			if err := cleaner.RemoveIngestScenes(ctx, task.StashScanPath, task.StashSceneIDs()); err != nil {
				return fail(err)
			}
		}
		if err := mediafiles.RemoveDelivery(ctx, *state.Delivery); err != nil {
			return fail(err)
		}
	}
	now := s.now().UTC()
	next := &Task{Revision: task.Revision, Generation: task.Generation, ID: task.ID, Source: task.Source, Code: task.Code, SavePath: task.SavePath,
		Category: task.Category, Tags: task.Tags, CreatedAt: task.CreatedAt, UpdatedAt: now,
		LastProgressAt: &now, SourcingRequestJSON: task.SourcingRequestJSON}
	if err := ensureSourcingRequest(next); err != nil {
		return fail(err)
	}
	setTaskStage(next, TaskStageSourcing, TaskStageStatusBlocked)
	blockTask(next, TaskStageErrorManualSource, "choose a torrent to restart this task", now)
	if err := s.store.Update(ctx, next); err != nil {
		return task, err
	}
	logging.Infof("taskruntime: cleared execution artifacts and restarted sourcing task=%s", task.ID)
	return next, nil
}

func restartTorrentHash(task *Task) string {
	return strings.TrimSpace(firstNonEmpty([]string{task.TorrentHash, task.TorrentIdentityHash, task.Candidate.InfoHash, infoHashFromMagnet(task.TorrentURL)}))
}

func (s *Service) waitRestartTorrentRemoval(ctx context.Context, hash string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for {
		torrents, err := s.qbt.GetTorrentList(ctx, nil)
		if err != nil {
			return err
		}
		found := false
		for _, torrent := range torrents {
			if strings.EqualFold(torrent.Hash, hash) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for qBittorrent deletion: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *Service) planRestartDownloads(ctx context.Context, task *Task) (*mediafiles.DownloadRemoval, error) {
	root := task.MojiSourcePath
	if root == "" && s.mediaConfig != nil {
		cfg := s.mediaConfig()
		if cfg.Downloads.MojiRoot != "" && cfg.Downloads.QBRoot != "" && task.SavePath != "" {
			relative, err := relativePathWithin(cfg.Downloads.QBRoot, task.SavePath)
			if err != nil {
				return nil, err
			}
			root = filepath.Join(cfg.Downloads.MojiRoot, relative)
		}
	}
	if root == "" {
		// qBittorrent owns file removal when no local path mapping exists.
		return nil, nil
	}
	names := map[string]bool{}
	for _, file := range task.FilePlan().Files {
		names[file.Path] = true
	}
	if client, ok := s.qbt.(interface {
		GetTorrentContents(context.Context, string, []string) ([]qbittorrent.TorrentContentFile, error)
	}); ok && restartTorrentHash(task) != "" {
		files, err := client.GetTorrentContents(ctx, restartTorrentHash(task), nil)
		if err != nil {
			torrents, lookupErr := s.qbt.GetTorrentList(ctx, nil)
			if lookupErr != nil {
				return nil, lookupErr
			}
			for _, torrent := range torrents {
				if strings.EqualFold(torrent.Hash, restartTorrentHash(task)) {
					return nil, err
				}
			}
		} else {
			for _, file := range files {
				names[file.Name] = true
			}
		}
	}
	plan := &mediafiles.DownloadRemoval{Root: root}
	for name := range names {
		plan.Paths = append(plan.Paths, name)
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	return plan, nil
}
