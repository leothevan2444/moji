package taskruntime

import (
	"context"
	"strings"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

// ObserveTorrentStates consumes the monitor's authoritative listing. It updates
// download progress and problems without repeating file selection or ingest work.
func (s *Service) ObserveTorrentStates(ctx context.Context, torrents []qbittorrent.Torrent) error {
	byHash := make(map[string]qbittorrent.Torrent, len(torrents))
	for _, torrent := range torrents {
		byHash[strings.ToLower(torrent.Hash)] = torrent
	}
	return walkTaskWork(ctx, s.store, "MONITOR", func(snapshot *Task) error {
		if snapshot == nil {
			return nil
		}
		if snapshot.Stage != "" {
			next := applyTorrentObservation(snapshot, byHash, s.now().UTC())
			if sameTaskProgress(snapshot, next) && snapshot.ContentPath == next.ContentPath {
				return nil
			}
		}
		return s.observeTorrentState(ctx, snapshot.ID, byHash)
	})
}
func (s *Service) observeTorrentState(ctx context.Context, id string, byHash map[string]qbittorrent.Torrent) error {
	unlock := s.lockTask(id)
	defer unlock()
	task, err := s.store.Find(ctx, id)
	if err != nil {
		// A task can be removed after the monitor lists it.
		return nil
	}
	hash := strings.ToLower(taskHashForMonitoring(task))
	if hash == "" || task.Stage == TaskStageSourcing || task.RestartStateJSON != "" {
		return nil
	}
	next := applyTorrentObservation(task, byHash, s.now().UTC())
	if sameTaskProgress(task, next) && task.ContentPath == next.ContentPath {
		return nil
	}
	next.UpdatedAt = s.now().UTC()
	return s.store.Update(ctx, next)
}

func applyTorrentObservation(task *Task, byHash map[string]qbittorrent.Torrent, now time.Time) *Task {
	hash := strings.ToLower(taskHashForMonitoring(task))
	torrent, exists := byHash[hash]
	state := "missing"
	if exists {
		state = string(torrent.State)
	}
	next := cloneTask(task)
	next.QBittorrentState = state
	if exists && torrent.ContentPath != "" {
		next.ContentPath = torrent.ContentPath
	}
	if task.Stage == TaskStageDownloading {
		if exists {
			next.Progress = torrent.Progress
		}
		ownError := task.StageErrorCode == "TORRENT_MISSING" || task.StageErrorCode == "TORRENT_ERROR"
		if task.StageStatus != TaskStageStatusBlocked || ownError {
			if !exists {
				blockTask(next, "TORRENT_MISSING", "TORRENT_MISSING", now)
			} else if torrent.State == qbittorrent.TorrentStateError || torrent.State == qbittorrent.TorrentStateMissingFiles {
				blockTask(next, "TORRENT_ERROR", "TORRENT_ERROR", now)
			} else if ownError {
				clearTaskStageError(next)
				setTaskStage(next, TaskStageDownloading, TaskStageStatusRunning)
			}
		}
	}
	return next
}
