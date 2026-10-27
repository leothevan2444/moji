package taskruntime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type rejectedTorrentAdder struct{ fakeTorrentAdder }

func (q *rejectedTorrentAdder) AddNewTorrent(context.Context, qbittorrent.AddTorrentOptions) error {
	return errors.New("add torrent rejected by qBittorrent")
}

func TestRejectedTorrentSubmissionAdoptsOnlyExactHash(t *testing.T) {
	const hash = "0123456789012345678901234567890123456789"
	for _, tc := range []struct {
		name, observedHash string
		success            bool
	}{
		{"already present", hash, true}, {"unrelated torrent", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false}, {"not present", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := &rejectedTorrentAdder{}
			if tc.observedHash != "" {
				q.torrents = []qbittorrent.Torrent{{Hash: tc.observedHash, Name: "ABCD-123", Progress: .5, State: qbittorrent.TorrentStateDownloading, SavePath: "/existing", ContentPath: "/existing/ABCD-123"}}
			}
			store := NewMemoryTaskStore()
			s, err := NewService(fakeTracker{}, q, store)
			if err != nil {
				t.Fatal(err)
			}
			at := time.Now().UTC()
			task := &Task{ID: "retry", Code: "ABCD-123", Candidate: Candidate{Title: "ABCD-123", InfoHash: hash}, Stage: TaskStageSourcing, StageStatus: TaskStageStatusRunning, CreatedAt: at, UpdatedAt: at}
			if err := store.Create(context.Background(), task); err != nil {
				t.Fatal(err)
			}
			err = s.submitTaskTorrent(context.Background(), task, "magnet:?xt=urn:btih:"+hash+"&dn=ABCD-123", "", "", "", nil)
			if tc.success {
				if err != nil || task.TorrentHash != hash || task.Progress != .5 || task.StageStatus != TaskStageStatusRunning || task.SavePath != "/existing" {
					t.Fatalf("existing torrent not reconciled: %+v %v", task, err)
				}
			} else if err == nil || task.StageErrorCode != TaskStageErrorTorrentSubmit {
				t.Fatalf("unrelated torrent was adopted: %+v %v", task, err)
			}
		})
	}
}
