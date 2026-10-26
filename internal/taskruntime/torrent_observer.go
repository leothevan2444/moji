package taskruntime

import (
	"context"
	"strings"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

// ObserveTorrentStates consumes the monitor's authoritative listing. It updates
// download progress and problems without repeating file selection or ingest work.
func (s *Service) ObserveTorrentStates(ctx context.Context, torrents []qbittorrent.Torrent) error {
	tasks, err := s.store.List(ctx)
	if err != nil {
		return err
	}
	byHash := make(map[string]qbittorrent.Torrent, len(torrents))
	for _, torrent := range torrents {
		byHash[strings.ToLower(torrent.Hash)] = torrent
	}
	for _, snapshot := range tasks {
		if err := s.observeTorrentState(ctx, snapshot.ID, byHash); err != nil {
			return err
		}
	}
	return nil
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
	torrent, exists := byHash[hash]
	state := "missing"
	if exists {
		state = string(torrent.State)
	}
	next := cloneTask(task)
	next.QBittorrentState = state
	if task.Stage == TaskStageDownloading {
		if exists {
			next.Progress = torrent.Progress
		}
		ownError := task.StageErrorCode == "TORRENT_MISSING" || task.StageErrorCode == "TORRENT_ERROR"
		if task.StageStatus != TaskStageStatusBlocked || ownError {
			if !exists {
				blockTask(next, "TORRENT_MISSING", "TORRENT_MISSING", s.now().UTC())
			} else if torrent.State == qbittorrent.TorrentStateError || torrent.State == qbittorrent.TorrentStateMissingFiles {
				blockTask(next, "TORRENT_ERROR", "TORRENT_ERROR", s.now().UTC())
			} else if ownError {
				clearTaskStageError(next)
				setTaskStage(next, TaskStageDownloading, TaskStageStatusRunning)
			}
		}
	}
	if sameTaskProgress(task, next) {
		return nil
	}
	next.UpdatedAt = s.now().UTC()
	return s.store.Update(ctx, next)
}
