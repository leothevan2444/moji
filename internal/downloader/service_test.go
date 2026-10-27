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

const testHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type fakeClient struct {
	torrents             []qbittorrent.Torrent
	listErr, errorDelete error
	deleted              int
	deleteFiles          bool
	lastAction           string
}

func (f *fakeClient) GetTorrentList(context.Context, *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	return f.torrents, f.listErr
}
func (f *fakeClient) GetDownloadServerState(context.Context) (*qbittorrent.DownloadServerState, error) {
	space := int64(1024)
	return &qbittorrent.DownloadServerState{FreeSpace: &space}, nil
}
func (f *fakeClient) PauseTorrents(context.Context, []string) error {
	f.lastAction = "pause"
	return nil
}
func (f *fakeClient) ResumeTorrents(context.Context, []string) error {
	f.lastAction = "resume"
	return nil
}
func (f *fakeClient) RecheckTorrents(context.Context, []string) error {
	f.lastAction = "recheck"
	return nil
}
func (f *fakeClient) ReannounceTorrents(context.Context, []string) error {
	f.lastAction = "reannounce"
	return nil
}
func (f *fakeClient) SetDownloadSpeedLimit(context.Context, bool, int64) error { return nil }
func (f *fakeClient) DeleteTorrents(_ context.Context, hashes []string, files bool) error {
	f.deleted++
	f.deleteFiles = files
	if f.errorDelete != nil {
		return f.errorDelete
	}
	for i, t := range f.torrents {
		if t.Hash == hashes[0] {
			f.torrents = append(f.torrents[:i], f.torrents[i+1:]...)
			break
		}
	}
	return nil
}

type fakeTasks struct{ tasks []*taskruntime.Task }

func (f *fakeTasks) ListTasks(context.Context) ([]*taskruntime.Task, error) { return f.tasks, nil }
func newTest(t *testing.T) (*Service, *fakeClient, *fakeTasks, string) {
	t.Helper()
	client := &fakeClient{torrents: []qbittorrent.Torrent{{Hash: testHash, Name: "ABCD-123", Progress: 1, State: qbittorrent.TorrentStateStalledUP, Ratio: 2, SeedingTime: 7200, ContentPath: "/downloads/ABCD-123"}}}
	tasks := &fakeTasks{tasks: []*taskruntime.Task{{ID: "task", Code: "ABCD-123", TorrentHash: testHash, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone}}}
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := New(client, tasks, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, client, tasks, path
}
func TestRetentionRequiresCompletedManagedTasksAndThresholds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		p     Policy
		stage taskruntime.TaskStage
		other bool
		want  int
	}{
		{"disabled", Policy{Ratio: 1}, taskruntime.TaskStageCompleted, false, 0},
		{"any", Policy{Enabled: true, Ratio: 1, SeedHours: 10}, taskruntime.TaskStageCompleted, false, 1},
		{"all not met", Policy{Enabled: true, Ratio: 1, SeedHours: 10, MatchAll: true}, taskruntime.TaskStageCompleted, false, 0},
		{"all met", Policy{Enabled: true, Ratio: 1, SeedHours: 1, MatchAll: true}, taskruntime.TaskStageCompleted, false, 1},
		{"ingest active", Policy{Enabled: true, Ratio: 1}, taskruntime.TaskStageScanning, false, 0},
		{"shared active task", Policy{Enabled: true, Ratio: 1}, taskruntime.TaskStageCompleted, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, q, tasks, _ := newTest(t)
			tasks.tasks[0].Stage = tc.stage
			if tc.other {
				tasks.tasks = append(tasks.tasks, &taskruntime.Task{ID: "active", TorrentHash: testHash, Stage: taskruntime.TaskStageDownloading})
			}
			ctx := context.Background()
			if err := s.SetPolicy(ctx, tc.p); err != nil {
				t.Fatal(err)
			}
			if err := s.Cycle(ctx); err != nil {
				t.Fatal(err)
			}
			if q.deleted != tc.want {
				t.Fatalf("deleted=%d want=%d", q.deleted, tc.want)
			}
			if tc.want > 0 {
				snap, _ := s.Snapshot(ctx)
				if snap.Torrents[0].RemovalReason != "POLICY" {
					t.Fatal("removal provenance lost")
				}
				if err := s.Cycle(ctx); err != nil {
					t.Fatal(err)
				}
				snap, _ = s.Snapshot(ctx)
				if snap.Torrents[0].State != "removed" {
					t.Fatal("policy removal became missing")
				}
			}
		})
	}
}
func TestOutageRetainsLastSeenAndDoesNotMarkTorrentsMissing(t *testing.T) {
	s, q, _, path := newTest(t)
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Snapshot(ctx)
	q.listErr = errors.New("offline")
	if err := s.Cycle(ctx); err == nil {
		t.Fatal("outage ignored")
	}
	after, _ := s.Snapshot(ctx)
	if after.Connected || !after.Torrents[0].Present || after.CheckedAt != before.CheckedAt || q.deleted != 0 {
		t.Fatal("outage changed last confirmed data")
	}
	q.listErr = nil
	q.torrents = nil
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 2}); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(q, s.tasks, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	snap, _ := reopened.Snapshot(ctx)
	p, _ := reopened.Policy(ctx)
	if snap.Torrents[0].State != "missing" || snap.Torrents[0].LastSeenAt != before.CheckedAt || !p.Enabled {
		t.Fatal("restart lost policy or last observation")
	}
}
func TestFileCleanupRequiresIndependentCopyAndNoCrossSeed(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	ctx := context.Background()
	if err := s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 1, DeleteFiles: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 0 {
		t.Fatal("deleted mapped library data")
	}
	source, target := filepath.Join(t.TempDir(), "downloads"), filepath.Join(t.TempDir(), "library")
	os.MkdirAll(source, 0755)
	os.MkdirAll(target, 0755)
	tasks.tasks[0].TransferAction = "copy"
	tasks.tasks[0].MojiSourcePath = source
	tasks.tasks[0].MojiTransferPath = target
	q.torrents = append(q.torrents, qbittorrent.Torrent{Hash: strings.Repeat("b", 40), ContentPath: q.torrents[0].ContentPath})
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 0 {
		t.Fatal("deleted cross-seeded files")
	}
	if err := s.Action(ctx, testHash, "REMOVE_FILES"); !errors.Is(err, ErrUnsafeFiles) {
		t.Fatal("manual removal bypassed cross-seed guard", err)
	}
	q.torrents = q.torrents[:1]
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 || !q.deleteFiles {
		t.Fatal("eligible files not deleted")
	}
}
func TestActionValidationFailureAndRecovery(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	ctx := context.Background()
	for _, hash := range []string{"all", "", testHash + "|" + testHash} {
		if s.Action(ctx, hash, "PAUSE") == nil {
			t.Fatal("accepted unsafe hash")
		}
	}
	tasks.tasks[0].Stage = taskruntime.TaskStageDownloading
	if err := s.Action(ctx, testHash, "REMOVE"); !errors.Is(err, ErrIngestActive) {
		t.Fatal("unfinished ingest removed", err)
	}
	if err := s.Action(ctx, testHash, "PAUSE"); err != nil || q.lastAction != "pause" {
		t.Fatal(err)
	}
	tasks.tasks[0].Stage = taskruntime.TaskStageCompleted
	q.errorDelete = errors.New("delete failed")
	s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 1})
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(ctx)
	if !snap.Torrents[0].CleanupError || !snap.Torrents[0].Present {
		t.Fatal("failed cleanup claimed success")
	}
	q.errorDelete = nil
	s.now = func() time.Time { return time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC) }
	if err := s.Action(ctx, testHash, "REMOVE"); err != nil {
		t.Fatal(err)
	}
	snap, _ = s.Snapshot(ctx)
	if snap.Torrents[0].RemovalReason != "MANUAL" || snap.Torrents[0].RemovedAt == "" {
		t.Fatal("manual removal provenance missing")
	}
}
func TestUnmanagedAndCheckingTorrentsNeverAutoRemoved(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	ctx := context.Background()
	s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 1})
	tasks.tasks = nil
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 0 {
		t.Fatal("unmanaged torrent removed")
	}
	tasks.tasks = []*taskruntime.Task{{ID: "task", TorrentHash: testHash, Stage: taskruntime.TaskStageCompleted, StageStatus: taskruntime.TaskStageStatusDone}}
	q.torrents[0].State = qbittorrent.TorrentStateCheckingUP
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 0 {
		t.Fatal("checking torrent removed")
	}
}

func TestImmediateRetentionAndDefaultFilePolicy(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	ctx := context.Background()
	p, err := s.Policy(ctx)
	if err != nil || !p.DeleteFiles || p.Enabled {
		t.Fatal("wrong first-run policy", p, err)
	}
	if SyncInterval != 3*time.Second {
		t.Fatal("monitor must sync every three seconds")
	}
	q.torrents[0].Ratio = 0
	q.torrents[0].SeedingTime = 0
	if err := s.SetPolicy(ctx, Policy{Enabled: true, AfterIngest: true}); err != nil {
		t.Fatal(err)
	}
	tasks.tasks[0].Stage = taskruntime.TaskStageScanning
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 0 {
		t.Fatal("immediate cleanup ran before ingest")
	}
	tasks.tasks[0].Stage = taskruntime.TaskStageCompleted
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 {
		t.Fatal("immediate cleanup waited for a threshold")
	}
}
func TestCompletedSourceAbsenceIsNotAnIssue(t *testing.T) {
	for _, state := range []string{"missing", "missingFiles", "removed"} {
		if (Torrent{State: state, AllCompleted: true, CleanupError: true}).NeedsAttention() {
			t.Fatal("completed source cleanup counted as an error", state)
		}
	}
	for _, state := range []string{"missing", "missingFiles", "error"} {
		if !(Torrent{State: state}).NeedsAttention() {
			t.Fatal("active download error ignored", state)
		}
	}
	if !(Torrent{State: "error", AllCompleted: true}).NeedsAttention() {
		t.Fatal("unexpected qB error hidden")
	}
}

func TestImmediateCleanupHandlesCompletedMoveWithMissingSourceFiles(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	ctx := context.Background()
	source, target := filepath.Join(t.TempDir(), "downloads"), filepath.Join(t.TempDir(), "library")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	task := tasks.tasks[0]
	task.TransferAction = "move"
	task.MojiSourcePath = source
	task.MojiTransferPath = target
	q.torrents[0].State = qbittorrent.TorrentStateMissingFiles
	q.torrents[0].Progress = 0
	if err := s.SetPolicy(ctx, Policy{Enabled: true, AfterIngest: true, DeleteFiles: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 || !q.deleteFiles {
		t.Fatal("completed move retained its unusable torrent")
	}
}

type unavailableTasks struct{ fakeTasks }

func (f *unavailableTasks) ListTasks(context.Context) ([]*taskruntime.Task, error) {
	return nil, errors.New("task database unavailable")
}

func TestFreshQBListingSurvivesTaskDatabaseFailureWithoutCleanup(t *testing.T) {
	s, q, _, _ := newTest(t)
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 1}); err != nil {
		t.Fatal(err)
	}
	s.tasks = &unavailableTasks{}
	q.torrents[0].DLSpeed = 12345
	if err := s.Cycle(ctx); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot(ctx)
	if err != nil || !snap.Connected || snap.Torrents[0].DownloadSpeed != 12345 {
		t.Fatalf("fresh qB observation lost: %+v %v", snap, err)
	}
	if q.deleted != 0 || snap.Torrents[0].CanDeleteFiles || snap.Torrents[0].AllCompleted {
		t.Fatal("cleanup allowed without authoritative tasks")
	}
	if len(snap.Torrents[0].TaskIDs) != 1 {
		t.Fatal("last known associations lost")
	}
}

type blockedObserver struct {
	fakeTasks
	entered chan struct{}
}

func (f *blockedObserver) ObserveTorrentStates(ctx context.Context, _ []qbittorrent.Torrent) error {
	select {
	case f.entered <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}
func TestTaskObserverDoesNotBlockFreshSnapshots(t *testing.T) {
	s, q, tasks, _ := newTest(t)
	observer := &blockedObserver{fakeTasks: *tasks, entered: make(chan struct{}, 1)}
	s.tasks = observer
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.runTaskObserver(ctx) }()
	defer func() { cancel(); <-done }()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-observer.entered:
	case <-time.After(time.Second):
		t.Fatal("observer did not start")
	}
	q.torrents[0].Progress = .42
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot(ctx)
	if err != nil || snap.Torrents[0].Progress != .42 {
		t.Fatalf("blocked observer prevented fresh listing: %+v %v", snap, err)
	}
	// Responses must not expose mutable cached slices.
	snap.Torrents[0].TaskIDs[0] = "changed"
	next, _ := s.Snapshot(ctx)
	if next.Torrents[0].TaskIDs[0] == "changed" {
		t.Fatal("snapshot shares mutable state")
	}
}

type waitingClient struct {
	*fakeClient
	entered, release chan struct{}
}

func (f *waitingClient) GetTorrentList(ctx context.Context, opts *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	close(f.entered)
	select {
	case <-f.release:
		return f.fakeClient.GetTorrentList(ctx, opts)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func TestReadingSnapshotDoesNotWaitForQBRequest(t *testing.T) {
	s, q, _, _ := newTest(t)
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	client := &waitingClient{fakeClient: q, entered: make(chan struct{}), release: make(chan struct{})}
	s.client = client
	done := make(chan error, 1)
	go func() { done <- s.Refresh(context.Background()) }()
	<-client.entered
	defer func() {
		close(client.release)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	readDone := make(chan error, 1)
	go func() { _, err := s.Snapshot(context.Background()); readDone <- err }()
	select {
	case err := <-readDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot blocked behind qB network request")
	}
}

func TestFreshObservationRemainsReadableWhenPersistenceFails(t *testing.T) {
	s, q, _, _ := newTest(t)
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Close(); err != nil {
		t.Fatal(err)
	}
	q.torrents[0].DLSpeed = 9876
	if err := s.Refresh(ctx); err == nil || !strings.Contains(err.Error(), "persist downloader snapshot") {
		t.Fatalf("persistence failure was not reported separately: %v", err)
	}
	snap, err := s.Snapshot(ctx)
	if err != nil || !snap.Connected || snap.Torrents[0].DownloadSpeed != 9876 {
		t.Fatalf("persistence failure discarded live observation: %+v %v", snap, err)
	}
}

type persistenceFailureClient struct {
	*fakeClient
	fail func()
}

func (q *persistenceFailureClient) GetDownloadServerState(ctx context.Context) (*qbittorrent.DownloadServerState, error) {
	q.fail()
	return q.fakeClient.GetDownloadServerState(ctx)
}
func TestCleanupNeverRunsWithoutDurableIntent(t *testing.T) {
	s, q, _, _ := newTest(t)
	ctx := context.Background()
	if err := s.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPolicy(ctx, Policy{Enabled: true, Ratio: 1}); err != nil {
		t.Fatal(err)
	}
	s.client = &persistenceFailureClient{fakeClient: q, fail: func() { _ = s.db.Close() }}
	if err := s.Cycle(ctx); err == nil {
		t.Fatal("expected failed intent persistence")
	}
	if q.deleted != 0 {
		t.Fatal("qB deletion executed before intent was durable")
	}
	snap, _ := s.Snapshot(ctx)
	if snap.Torrents[0].RemovalReason != "" {
		t.Fatal("failed intent was published as accepted")
	}
}
