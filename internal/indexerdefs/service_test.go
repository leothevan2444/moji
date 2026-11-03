package indexerdefs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

type fixtureSource struct {
	mu       sync.Mutex
	latest   string
	archives map[string][]byte
	block    chan struct{}
	started  chan struct{}
}

func (s *fixtureSource) Latest(ctx context.Context, etag string) (Revision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Revision{Commit: s.latest, ETag: `"` + s.latest + `"`}, false, ctx.Err()
}
func (s *fixtureSource) Download(ctx context.Context, revision string, w io.Writer) error {
	s.mu.Lock()
	block, started, data := s.block, s.started, append([]byte(nil), s.archives[revision]...)
	s.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	_, err := w.Write(data)
	return err
}
func (s *fixtureSource) setLatest(revision string) { s.mu.Lock(); s.latest = revision; s.mu.Unlock() }

func serviceFixture(t *testing.T) (*storage.Manager, *Service, *fixtureSource) {
	t.Helper()
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Close() })
	source := &fixtureSource{latest: strings.Repeat("a", 40), archives: map[string][]byte{}}
	for _, letter := range []string{"a", "b", "c", "d"} {
		source.archives[strings.Repeat(letter, 40)] = validArchive(t, fixtureDefinition)
	}
	svc, err := NewService(manager, source, emptyConfigValidator())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	return manager, svc, source
}

func awaitJob(t *testing.T, svc *Service, id string) *Job {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	changed, unsubscribe, err := svc.Subscribe()
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	for {
		job, err := svc.Job(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if job != nil && job.State.Terminal() {
			return job
		}
		select {
		case <-changed:
		case <-deadline.C:
			t.Fatal("job did not terminate")
		}
	}
}

func installNext(t *testing.T, svc *Service, source *fixtureSource, revision string) *Job {
	t.Helper()
	source.setLatest(revision)
	state, err := svc.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	kind := "UPDATE"
	if state.Active == "" {
		kind = "INSTALL"
	}
	job, err := svc.Begin(context.Background(), kind, revision, state.Active)
	if err != nil {
		t.Fatal(err)
	}
	finished := awaitJob(t, svc, job.ID)
	if finished.State != Completed {
		t.Fatalf("job failed: %+v", finished)
	}
	return finished
}

func TestRuntimeInstallUpdateRollbackAndReferenceRetention(t *testing.T) {
	manager, svc, source := serviceFixture(t)
	ctx := context.Background()
	a, b, c := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	installNext(t, svc, source, a)
	lease, err := svc.Acquire(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Release()
	if err := manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error { return PinInTx(ctx, tx, "task", "pending", a) }); err != nil {
		t.Fatal(err)
	}
	installNext(t, svc, source, b)
	installNext(t, svc, source, c)
	if err := svc.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(svc.root, a)); err != nil {
		t.Fatal("referenced old bundle removed", err)
	}
	if err := manager.Core.WithWrite(ctx, func(tx *sqlx.Tx) error { return UnpinInTx(ctx, tx, "task", "pending") }); err != nil {
		t.Fatal(err)
	}
	if err := svc.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(svc.root, a)); err != nil {
		t.Fatal("live lease ignored", err)
	}
	lease.Release()
	if err := svc.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(svc.root, a)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unreferenced old bundle retained", err)
	}
	job, err := svc.Begin(ctx, "ROLLBACK", b, c)
	if err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, svc, job.ID); finished.State != Completed {
		t.Fatalf("rollback: %+v", finished)
	}
	status, err := svc.Status(ctx)
	if err != nil || status.State.Active != b || status.State.Previous != c || !status.Available {
		t.Fatalf("wrong rollback status %+v %v", status, err)
	}
	if _, err := svc.Begin(ctx, "UPDATE", c, a); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale active accepted: %v", err)
	}
}

func TestUpdateCancelAndShutdownKeepActiveBundle(t *testing.T) {
	_, svc, source := serviceFixture(t)
	ctx := context.Background()
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	installNext(t, svc, source, a)
	source.mu.Lock()
	source.latest = b
	source.block = make(chan struct{})
	source.started = make(chan struct{}, 1)
	started := source.started
	source.mu.Unlock()
	if _, err := svc.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := svc.Begin(ctx, "UPDATE", b, a)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download not started")
	}
	duplicate, err := svc.Begin(ctx, "UPDATE", b, a)
	if err != nil || duplicate.ID != job.ID {
		t.Fatal("duplicate update not idempotent", err)
	}
	if err := svc.Cancel(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, svc, job.ID); finished.State != Cancelled {
		t.Fatalf("cancel overwritten: %+v", finished)
	}
	// Close waits for cancelled work before a new service owns this generation.
	svc.Close()
	status, err := svc.Status(ctx)
	if err != nil || status.State.Active != a {
		t.Fatal("cancel changed active", err)
	}
	if _, err := svc.Begin(ctx, "UPDATE", b, a); !errors.Is(err, ErrNotRunning) {
		t.Fatal("closed service accepted job", err)
	}
}

func TestInterruptedJobRecoveryAndCorruptBundleRepair(t *testing.T) {
	manager, svc, source := serviceFixture(t)
	ctx := context.Background()
	a := strings.Repeat("a", 40)
	installNext(t, svc, source, a)
	svc.Close()
	if err := os.WriteFile(filepath.Join(svc.root, a, "example.yml"), []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(svc.root, ".staging-interrupted")
	if err := os.Mkdir(partial, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(partial, "partial.yml"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	pending := &Job{ID: "interrupted-job", Kind: "UPDATE", State: Downloading, Target: strings.Repeat("b", 40), ExpectedActive: a, CorrelationID: "fixture", StartedAt: time.Now()}
	if err := svc.repo.begin(ctx, pending); err != nil {
		t.Fatal(err)
	}
	restarted, err := NewService(manager, source, emptyConfigValidator())
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Start(ctx); err != nil {
		t.Fatal("damaged rule bundle blocked app startup", err)
	}
	defer restarted.Close()
	if _, err := os.Stat(partial); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("interrupted stage not reclaimed", err)
	}
	status, err := restarted.Status(ctx)
	if err != nil || status.Available || status.ErrorCode != ErrUnavailable.Error() || status.State.Active != a {
		t.Fatalf("corruption hidden %+v %v", status, err)
	}
	job, err := restarted.Job(ctx, pending.ID)
	if err != nil || job.State != Interrupted {
		t.Fatal("old job not interrupted", err)
	}
	job, err = restarted.Begin(ctx, "REPAIR", a, a)
	if err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, restarted, job.ID); finished.State != Completed {
		t.Fatalf("repair failed %+v", finished)
	}
	status, err = restarted.Status(ctx)
	if err != nil || !status.Available || status.State.Active != a {
		t.Fatal("repair did not restore availability", err)
	}
	if _, err := LoadBundle(ctx, filepath.Join(restarted.root, a), ""); err != nil {
		t.Fatal(err)
	}
}

func TestRuleBackupRestoreIsOfflineAndRejectsMissingAssets(t *testing.T) {
	manager, svc, source := serviceFixture(t)
	ctx := context.Background()
	a := strings.Repeat("a", 40)
	installNext(t, svc, source, a)
	saved, err := manager.Backup(ctx, "rules", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Artifacts) != 3 {
		t.Fatalf("incomplete bundle backup %d", len(saved.Artifacts))
	}
	manifestPath := filepath.Join(manager.BackupDirectory, saved.Database+".json")
	if _, err := storage.VerifyBackup(ctx, manifestPath); err != nil {
		t.Fatal(err)
	}
	result, err := storage.RestoreCore(ctx, t.TempDir(), manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBundle(ctx, filepath.Join(result.Directory, "indexers", "bundles", a), ""); err != nil {
		t.Fatal("offline restore lost rules", err)
	}
	if err := os.Remove(filepath.Join(manager.BackupDirectory, saved.Artifacts[0].Path)); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.VerifyBackup(ctx, manifestPath); err == nil {
		t.Fatal("missing rule artifact accepted")
	}
}
