package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type capacityClient struct {
	*fakeClient
	space     *int64
	serverErr error
	released  int64
	hashes    []string
	prefs     qbittorrent.TorrentPreferences
	prefsErr  error
	patch     qbittorrent.TransferPreferences
}

func (f *capacityClient) GetDownloadServerState(context.Context) (*qbittorrent.DownloadServerState, error) {
	var space *int64
	if f.space != nil {
		v := *f.space
		space = &v
	}
	return &qbittorrent.DownloadServerState{FreeSpace: space}, f.serverErr
}
func (f *capacityClient) GetTorrentPreferences(context.Context) (*qbittorrent.TorrentPreferences, error) {
	return &f.prefs, f.prefsErr
}
func (f *capacityClient) PatchTransferPreferences(_ context.Context, p qbittorrent.TransferPreferences) error {
	f.patch = p
	if p.MaxRatio != nil {
		f.prefs.MaxRatio = p.MaxRatio
	}
	return nil
}
func (f *capacityClient) DeleteTorrents(ctx context.Context, h []string, files bool) error {
	f.hashes = append(f.hashes, h[0])
	err := f.fakeClient.DeleteTorrents(ctx, h, files)
	if err == nil && f.space != nil {
		*f.space += f.released
	}
	return err
}
func capacityFixture(t *testing.T) (*Service, *capacityClient, *fakeTasks) {
	t.Helper()
	s, q, tasks, _ := newTest(t)
	space := int64(40e9)
	f := &capacityClient{fakeClient: q, space: &space, released: 20e9, prefs: qbittorrent.TorrentPreferences{SavePath: "/downloads"}}
	tasks.tasks = nil
	q.torrents = nil
	for i := 0; i < 3; i++ {
		hash := strings.Repeat(string(rune('a'+i)), 40)
		source := filepath.Join(t.TempDir(), "source")
		target := filepath.Join(t.TempDir(), "library")
		if err := os.MkdirAll(source, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(target, 0700); err != nil {
			t.Fatal(err)
		}
		tasks.tasks = append(tasks.tasks, &taskruntime.Task{ID: hash, TorrentHash: hash, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone, TransferAction: "copy", MojiSourcePath: source, MojiTransferPath: target})
		q.torrents = append(q.torrents, qbittorrent.Torrent{Hash: hash, SavePath: "/downloads", ContentPath: "/downloads/" + hash, Progress: 1, State: qbittorrent.TorrentStateStalledUP, Completed: int64(3-i) * 1e9, Ratio: float64(i), SeedingTime: int64(2 - i)})
	}
	s.client = f
	p := DefaultPolicy()
	p.Enabled = true
	p.AfterIngest = false
	p.CapacityEnabled = true
	if err := s.SetPolicy(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return s, f, tasks
}
func TestCapacityCleanupOrdersAndStopsAfterActualSpaceRecovery(t *testing.T) {
	for _, tc := range []struct{ order, hash string }{{"DISK_USAGE", strings.Repeat("a", 40)}, {"RATIO", strings.Repeat("c", 40)}, {"SEED_TIME", strings.Repeat("a", 40)}} {
		t.Run(tc.order, func(t *testing.T) {
			s, q, _ := capacityFixture(t)
			order := tc.order
			if _, err := s.PatchPolicy(context.Background(), nil, nil, nil, nil, nil, &order); err != nil {
				t.Fatal(err)
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(q.hashes) != 1 || q.hashes[0] != tc.hash || !q.deleteFiles {
				t.Fatal(q.hashes)
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if q.deleted != 1 {
				t.Fatal("deleted past recovered threshold")
			}
			snap, _ := s.Snapshot(context.Background())
			if snap.FreeSpace == nil || *snap.FreeSpace != 60e9 || snap.Cleanup.Reason != "" {
				t.Fatal(snap)
			}
		})
	}
}
func TestCapacitySafetyAndUnknownState(t *testing.T) {
	for _, mode := range []string{"space unknown", "server error", "path unknown", "other disk", "active task", "unlinked", "shared source", "checking"} {
		t.Run(mode, func(t *testing.T) {
			s, q, tasks := capacityFixture(t)
			switch mode {
			case "space unknown":
				q.space = nil
			case "server error":
				q.serverErr = errors.New("offline")
			case "path unknown":
				q.prefsErr = errors.New("offline")
			case "other disk":
				for i := range q.torrents {
					q.torrents[i].SavePath = "/other"
				}
			case "active task":
				for _, task := range tasks.tasks {
					task.Stage = taskruntime.TaskStageScanning
				}
			case "unlinked":
				tasks.tasks = nil
			case "shared source":
				q.torrents = append(q.torrents, qbittorrent.Torrent{Hash: strings.Repeat("d", 40)})
			case "checking":
				for i := range q.torrents {
					q.torrents[i].State = qbittorrent.TorrentStateCheckingUP
				}
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if q.deleted != 0 {
				t.Fatal("unsafe cleanup", q.hashes)
			}
		})
	}
}
func TestCapacityObservesBrieflyThenContinuesAndSurvivesRestart(t *testing.T) {
	s, q, tasks := capacityFixture(t)
	q.released = 0
	clock := time.Now()
	s.now = func() time.Time { return clock }
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 {
		t.Fatal("did not observe before deleting again", q.deleted)
	}
	var path string
	if err := s.db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&path); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(q, tasks, path, s.settings)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopened.now = func() time.Time { return clock }
	if err := reopened.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 {
		t.Fatal("restart lost observation deadline")
	}
	clock = clock.Add(cleanupSettleTime)
	if err := reopened.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 2 {
		t.Fatal("no net space increase permanently blocked cleanup", q.deleted)
	}
}
func TestCleanupTriggersAreExclusiveAndImmediateDoesNotRequireDiskMetrics(t *testing.T) {
	s, q, _ := capacityFixture(t)
	q.space = nil
	yes, no := true, false
	if _, err := s.PatchPolicy(context.Background(), nil, &yes, nil, nil, nil, nil); !errors.Is(err, ErrInvalidPolicy) {
		t.Fatal("both cleanup triggers accepted", err)
	}
	if _, err := s.PatchPolicy(context.Background(), nil, &yes, nil, &no, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 3 {
		t.Fatal("capacity blocked immediate cleanup", q.deleted)
	}
}

func TestCapacityUncertainRemovalReservesTheSameHash(t *testing.T) {
	s, q, _ := capacityFixture(t)
	q.errorDelete = errors.New("response lost")
	q.released = 0
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	q.errorDelete = nil
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 1 || snap.Cleanup.Phase != "WARNING" || snap.Cleanup.Reason != "CONFIRMING" {
		t.Fatal("uncertain request was replayed", q.deleted, snap.Cleanup)
	}
	q.torrents = q.torrents[1:]
	*q.space = 60e9
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 {
		t.Fatal("deleted another torrent before observing capacity")
	}
}
func TestTransferPreferencesValidationAndReadback(t *testing.T) {
	s, q, _ := capacityFixture(t)
	ratio := 1.0
	yes := true
	q.prefs.MaxRatio = &ratio
	q.prefs.MaxRatioEnabled = &yes
	value := 2.5
	observed, err := s.PatchTransferPreferences(context.Background(), qbittorrent.TransferPreferences{MaxRatio: &value})
	if err != nil || *observed.MaxRatio != 2.5 || q.patch.MaxRatioEnabled != nil {
		t.Fatal(observed, err)
	}
	bad := -0.1
	if _, err := s.PatchTransferPreferences(context.Background(), qbittorrent.TransferPreferences{MaxRatio: &bad}); !errors.Is(err, ErrInvalidPreferences) {
		t.Fatal(err)
	}
	mode := "MatchAll"
	if _, err := s.PatchTransferPreferences(context.Background(), qbittorrent.TransferPreferences{ShareLimitMode: &mode}); !errors.Is(err, ErrPreferencesUnavailable) {
		t.Fatal("unsupported field accepted", err)
	}
	no := false
	if _, err := s.PatchTransferPreferences(context.Background(), qbittorrent.TransferPreferences{MaxRatioEnabled: &no}); !errors.Is(err, ErrObservationPending) {
		t.Fatal("unconfirmed value acknowledged", err)
	}
}
