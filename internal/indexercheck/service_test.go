package indexercheck

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
)

type sourceFunc func(context.Context, Input) (Prepared, error)

func (f sourceFunc) PrepareCheck(ctx context.Context, in Input) (Prepared, error) { return f(ctx, in) }

type factoryFunc func(context.Context, indexer.InstanceSnapshot) (Execution, error)

func (f factoryFunc) Open(ctx context.Context, in indexer.InstanceSnapshot) (Execution, error) {
	return f(ctx, in)
}

type fakeExecution struct {
	connect, auth, search, fetch, closed int
	authErr, searchErr, fetchErr         error
	query                                indexer.Query
	batch                                indexer.Batch
	artifact                             indexer.Artifact
	wait                                 chan struct{}
}

func (e *fakeExecution) Authenticate(context.Context) error { e.auth++; return e.authErr }
func (e *fakeExecution) Search(ctx context.Context, q indexer.Query) (indexer.Batch, error) {
	e.search++
	e.query = q
	if e.wait != nil {
		e.wait <- struct{}{}
		<-ctx.Done()
		return indexer.Batch{}, ctx.Err()
	}
	return e.batch, e.searchErr
}
func (e *fakeExecution) Fetch(context.Context, indexer.ResourceRef) (indexer.Artifact, error) {
	e.fetch++
	return e.artifact, e.fetchErr
}
func (e *fakeExecution) Close() { e.closed++ }
func checkFixture(t *testing.T) (*Service, *fakeExecution, *atomic.Int32) {
	t.Helper()
	released := new(atomic.Int32)
	instance := indexer.InstanceSnapshot{ID: "instance", Kind: indexer.Builtin, DefinitionID: "rule", BundleRevision: "bundle", ConfigRevision: "account"}
	e := &fakeExecution{batch: indexer.Batch{Releases: []indexer.Release{{Resource: indexer.ResourceRef{Version: 1, InstanceID: instance.ID, Kind: instance.Kind, DefinitionID: instance.DefinitionID, BundleRevision: instance.BundleRevision, ConfigRevision: instance.ConfigRevision}}}}, artifact: indexer.Artifact{TorrentBytes: []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee")}}
	s, err := New(context.Background(), sourceFunc(func(context.Context, Input) (Prepared, error) {
		return Prepared{Instance: instance, AuthenticationRequired: true, ConfigRevision: "config", RuleRevision: "bundle", Release: func() { released.Add(1) }}, nil
	}), factoryFunc(func(context.Context, indexer.InstanceSnapshot) (Execution, error) { return e, nil }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, e, released
}
func TestCheckSeparatesStagesAndOnlyDownloadsWhenExplicitlyRequested(t *testing.T) {
	for _, mode := range []struct{ search, download bool }{{false, false}, {true, false}, {true, true}} {
		t.Run(map[bool]string{true: "search", false: "authentication"}[mode.search]+map[bool]string{true: "-download", false: ""}[mode.download], func(t *testing.T) {
			s, e, released := checkFixture(t)
			report, err := s.Check(context.Background(), Input{InstanceID: "instance", ExpectedRevision: "config", Keywords: "ABCD-123", Search: mode.search, Download: mode.download})
			if err != nil || report.ID == "" || report.FinishedAt.IsZero() || released.Load() != 1 || e.closed != 1 || e.auth != 1 {
				t.Fatal(report, err)
			}
			if e.search != boolInt(mode.search) || e.fetch != boolInt(mode.download) {
				t.Fatal("test scope performed an unrequested operation", e.search, e.fetch)
			}
			if mode.search && (e.query.CachePolicy != indexer.CacheRefresh || e.query.Limit != 20) {
				t.Fatal("diagnostic search reused cache/unbounded query")
			}
			if !mode.download && report.Stages[3].State != "NOT_REQUESTED" {
				t.Fatal("untested download was marked passed")
			}
		})
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func TestFailedStagesStopDependentOperationsAndRedactUnexpectedErrors(t *testing.T) {
	for _, stage := range []string{"authentication", "search", "download", "invalid-body", "foreign-reference"} {
		t.Run(stage, func(t *testing.T) {
			s, e, released := checkFixture(t)
			switch stage {
			case "authentication":
				e.authErr = indexer.ErrAuthentication
			case "search":
				e.searchErr = errors.New("https://private.example/?passkey=synthetic")
			case "download":
				e.fetchErr = indexer.ErrTransport
			case "invalid-body":
				e.artifact = indexer.Artifact{TorrentBytes: []byte("<html>login</html>")}
			case "foreign-reference":
				e.batch.Releases[0].Resource.InstanceID = "another-account"
			}
			report, err := s.Check(context.Background(), Input{InstanceID: "instance", Search: true, Download: true, Keywords: "ABCD-123"})
			if err != nil || released.Load() != 1 || e.closed != 1 {
				t.Fatal(err)
			}
			i := 3
			if stage == "authentication" {
				i = 1
				if e.search != 0 || e.fetch != 0 {
					t.Fatal("failed authentication continued")
				}
			}
			if stage == "search" {
				i = 2
				if e.fetch != 0 {
					t.Fatal("failed search fetched")
				}
				if report.Stages[i].ErrorCode != "INDEXER_OPERATION_FAILED" {
					t.Fatal("unexpected source error escaped")
				}
			}
			if report.Stages[i].State != "FAILED" {
				t.Fatal("failure was reported as passed", report)
			}
			if stage == "foreign-reference" && e.fetch != 0 {
				t.Fatal("foreign reference reached adapter")
			}
		})
	}
}
func TestEmptyPartialAndUnobservedAuthenticationAreDistinct(t *testing.T) {
	for _, partial := range []bool{false, true} {
		s, e, _ := checkFixture(t)
		e.batch = indexer.Batch{Partial: partial}
		report, err := s.Check(context.Background(), Input{InstanceID: "instance", Search: true, Download: true, Keywords: "ABCD-123"})
		want := "EMPTY"
		if partial {
			want = "PARTIAL"
		}
		if err != nil || report.Stages[2].State != want || report.Stages[3].State != "NO_CANDIDATE" || e.fetch != 0 {
			t.Fatal(report, err)
		}
	}
	s, e, _ := checkFixture(t)
	s.source = sourceFunc(func(context.Context, Input) (Prepared, error) {
		return Prepared{Instance: indexer.InstanceSnapshot{ID: "external", Kind: indexer.Jackett}, AuthenticationRequired: true}, nil
	})
	report, err := s.Check(context.Background(), Input{InstanceID: "external"})
	if err != nil || report.Stages[1].State != "NOT_OBSERVED" || e.auth != 0 {
		t.Fatal("Jackett gateway configuration claimed remote authentication", report, err)
	}
}
func TestChecksBoundConcurrencyAndCloseCancelsAndJoinsActiveOperations(t *testing.T) {
	entered := make(chan struct{}, 4)
	s, _, _ := checkFixture(t)
	var mu sync.Mutex
	closed := 0
	s.factory = factoryFunc(func(context.Context, indexer.InstanceSnapshot) (Execution, error) {
		return &joinExecution{fakeExecution: fakeExecution{wait: entered}, mu: &mu, closed: &closed}, nil
	})
	var wg sync.WaitGroup
	for _, id := range []string{"a", "b", "c", "d"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			report, err := s.Check(context.Background(), Input{InstanceID: id, Search: true, Keywords: "ABCD-123"})
			if err != nil || report.Stages[2].ErrorCode != "INDEXER_CANCELLED" {
				t.Error(report, err)
			}
		}(id)
	}
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("check did not enter bounded operation")
		}
	}
	for _, id := range []string{"a", "overflow"} {
		if _, err := s.Check(context.Background(), Input{InstanceID: id, Search: true, Keywords: "ABCD-123"}); !errors.Is(err, indexer.ErrBusy) {
			t.Fatal("overlapping/unbounded diagnostic was admitted", err)
		}
	}
	s.Close()
	wg.Wait()
	if closed != 4 {
		t.Fatal("close returned before sessions were released", closed)
	}
}

type joinExecution struct {
	fakeExecution
	mu     *sync.Mutex
	closed *int
}

func (e *joinExecution) Close() { e.mu.Lock(); defer e.mu.Unlock(); *e.closed++ }

func TestDiagnosticSearchBoundsAnAdapterThatIgnoresItsLimit(t *testing.T) {
	s, e, _ := checkFixture(t)
	candidate := e.batch.Releases[0]
	e.batch.Releases = make([]indexer.Release, 200)
	for i := range e.batch.Releases {
		e.batch.Releases[i] = candidate
	}
	report, err := s.Check(context.Background(), Input{InstanceID: "instance", Search: true, Keywords: "ABCD-123"})
	if err != nil || report.Results != 20 || !report.Truncated || report.Stages[2].State != "PASSED" || e.fetch != 0 {
		t.Fatal("test reported an unbounded/complete upstream batch", report, err)
	}
}

func (e *fakeExecution) Connect(context.Context) error { e.connect++; return nil }
func TestConnectionOnlyDoesNotAuthenticateSearchOrFetch(t *testing.T) {
	s, e, _ := checkFixture(t)
	report, err := s.Check(context.Background(), Input{InstanceID: "instance", ConnectionOnly: true})
	if err != nil || e.connect != 1 || e.auth != 0 || e.search != 0 || e.fetch != 0 || report.Stages[4].State != "PASSED" {
		t.Fatalf("%+v %+v %v", report, e, err)
	}
	if _, err = s.Check(context.Background(), Input{InstanceID: "instance", ConnectionOnly: true, Search: true}); err == nil {
		t.Fatal("mixed scope accepted")
	}
}
