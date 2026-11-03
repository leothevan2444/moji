package indexerlogin

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexer"
)

type testSource struct {
	revision atomic.Value
	releases atomic.Int32
	stale    atomic.Bool
}

func (s *testSource) PrepareLogin(ctx context.Context, in Input) (Prepared, error) {
	if err := ctx.Err(); err != nil {
		return Prepared{}, err
	}
	return Prepared{Instance: indexer.InstanceSnapshot{ID: in.InstanceID, Kind: indexer.Builtin}, ConfigRevision: in.ExpectedRevision, RuleRevision: "fixed-rules", Plan: &cardigann.Plan{Definition: cardigann.Definition{Login: &cardigann.Login{}}}, Release: func() { s.releases.Add(1) }}, nil
}
func (s *testSource) ValidateLogin(ctx context.Context, revision string) error {
	if current := s.revision.Load(); current != nil && current.(string) != revision {
		return indexer.ErrInvalidSettings
	}
	if s.stale.Load() {
		return indexer.ErrInvalidSettings
	}
	return ctx.Err()
}

type testExecution struct {
	authentications            atomic.Int32
	authErr                    error
	closeEntered, closeRelease chan struct{}
	closes, answers            atomic.Int32
	block                      bool
	entered                    chan struct{}
}

func (e *testExecution) Authenticate(context.Context) error {
	e.authentications.Add(1)
	return e.authErr
}
func (e *testExecution) Challenge() (*indexer.Challenge, error) {
	return &indexer.Challenge{ID: "internal-challenge", Prompt: "Synthetic question", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (e *testExecution) Answer(ctx context.Context, id, answer string) error {
	e.answers.Add(1)
	if id != "internal-challenge" || answer != "private-answer" {
		return indexer.ErrAuthentication
	}
	if e.block {
		close(e.entered)
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}
func (e *testExecution) Close() {
	e.closes.Add(1)
	if e.closeEntered != nil {
		close(e.closeEntered)
		<-e.closeRelease
	}
}

type testFactory struct{ execution *testExecution }

func (f testFactory) Open(Prepared) (Execution, error) { return f.execution, nil }
func testService(t *testing.T) (*Service, *testSource, *testExecution) {
	t.Helper()
	source := &testSource{}
	source.revision.Store(input.ExpectedRevision)
	execution := &testExecution{authErr: indexer.ErrChallenge}
	service, err := New(context.Background(), source, testFactory{execution})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	return service, source, execution
}

var input = Input{InstanceID: "saved-site", ExpectedRevision: "saved-config"}

func TestLoginIsScopedOneShotAndKeepsRulesUntilAnswer(t *testing.T) {
	s, source, e := testService(t)
	report, err := s.Begin(context.Background(), input)
	if err != nil || report.State != "WAITING" || report.RuleRevision != "fixed-rules" || report.Challenge == nil || source.releases.Load() != 0 {
		t.Fatal(report, err)
	}
	if _, err = s.Begin(context.Background(), input); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("duplicate login accepted", err)
	}
	foreign := input
	foreign.InstanceID = "another-site"
	if _, err = s.Answer(context.Background(), report.ID, foreign, "private-answer"); !errors.Is(err, indexer.ErrInvalidScope) {
		t.Fatal("foreign answer accepted", err)
	}
	foreign = input
	foreign.ExpectedRevision = "another-config"
	if err = s.Cancel(context.Background(), report.ID, foreign); !errors.Is(err, indexer.ErrInvalidScope) {
		t.Fatal("foreign cancel accepted", err)
	}
	ready, err := s.Answer(context.Background(), report.ID, input, "private-answer")
	if err != nil || ready.State != "READY" || ready.Challenge != nil || e.answers.Load() != 1 || source.releases.Load() != 1 || e.closes.Load() != 1 {
		t.Fatal(ready, err)
	}
	if _, err = s.Answer(context.Background(), report.ID, input, "private-answer"); !errors.Is(err, indexer.ErrExpired) {
		t.Fatal("replayed login answer", err)
	}
}
func TestLoginRevalidatesConfigurationBeforeSubmittingAnswer(t *testing.T) {
	s, source, e := testService(t)
	report, err := s.Begin(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	source.stale.Store(true)
	if _, err = s.Answer(context.Background(), report.ID, input, "private-answer"); !errors.Is(err, indexer.ErrInvalidSettings) {
		t.Fatal(err)
	}
	if e.answers.Load() != 0 || e.closes.Load() != 1 || source.releases.Load() != 1 {
		t.Fatal("changed configuration submitted answer or retained lease")
	}
}
func TestLoginExpiryAndIdleCancelReleaseAllState(t *testing.T) {
	for _, mode := range []string{"expiry", "cancel", "close"} {
		t.Run(mode, func(t *testing.T) {
			s, source, e := testService(t)
			report, err := s.Begin(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "expiry":
				s.expire(time.Now().Add(6 * time.Minute))
			case "cancel":
				if err = s.Cancel(context.Background(), report.ID, input); err != nil {
					t.Fatal(err)
				}
			case "close":
				s.Close()
			}
			if e.answers.Load() != 0 || e.closes.Load() != 1 || source.releases.Load() != 1 {
				t.Fatal("idle login retained resources or submitted an answer")
			}
			if _, err = s.Answer(context.Background(), report.ID, input, "private-answer"); !errors.Is(err, indexer.ErrExpired) {
				t.Fatal("retired login accepted an answer", err)
			}
		})
	}
}
func TestLoginCancelAndCloseJoinAnActiveAnswer(t *testing.T) {
	for _, mode := range []string{"cancel", "close"} {
		t.Run(mode, func(t *testing.T) {
			s, source, e := testService(t)
			e.block = true
			e.entered = make(chan struct{})
			report, err := s.Begin(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, err := s.Answer(context.Background(), report.ID, input, "private-answer"); done <- err }()
			<-e.entered
			if _, err = s.Answer(context.Background(), report.ID, input, "private-answer"); !errors.Is(err, indexer.ErrBusy) {
				t.Fatal("concurrent answer accepted", err)
			}
			if mode == "cancel" {
				if err = s.Cancel(context.Background(), report.ID, input); err != nil {
					t.Fatal(err)
				}
			} else {
				s.Close()
			}
			if err = <-done; !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if e.closes.Load() != 1 || source.releases.Load() != 1 {
				t.Fatal("close returned before work released resources")
			}
		})
	}
}
func TestLoginGlobalBudgetIncludesWaitingChallenges(t *testing.T) {
	s, _, _ := testService(t)
	for _, id := range []string{"one", "two", "three", "four"} {
		if _, err := s.Begin(context.Background(), Input{InstanceID: id, ExpectedRevision: "config"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Begin(context.Background(), Input{InstanceID: "five", ExpectedRevision: "config"}); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("global challenge budget ignored", err)
	}
}

func TestLoginImmediateSuccessAndAuthenticationFailureReleaseLeases(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "ready", true: "failed"}[failed], func(t *testing.T) {
			s, source, e := testService(t)
			e.authErr = nil
			if failed {
				e.authErr = indexer.ErrAuthentication
			}
			report, err := s.Begin(context.Background(), input)
			if failed {
				if !errors.Is(err, indexer.ErrAuthentication) {
					t.Fatal(err)
				}
			} else if err != nil || report.State != "READY" || report.Challenge != nil {
				t.Fatal(report, err)
			}
			if source.releases.Load() != 1 || e.closes.Load() != 1 {
				t.Fatal("completed login retained state")
			}
		})
	}
}
func TestLoginKeepsAdmissionReservedUntilCleanupFinishes(t *testing.T) {
	s, _, e := testService(t)
	e.closeEntered = make(chan struct{})
	e.closeRelease = make(chan struct{})
	report, err := s.Begin(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	old := s.entries[report.ID]
	s.mu.Unlock()
	done := make(chan error, 1)
	go func() { done <- s.Cancel(context.Background(), report.ID, input) }()
	<-e.closeEntered
	if _, err = s.Begin(context.Background(), input); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("same account admitted before cleanup", err)
	}
	close(e.closeRelease)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	e.closeEntered = nil
	e.closeRelease = nil
	if _, err = s.Begin(context.Background(), input); err != nil {
		t.Fatal("cleanup did not release account slot", err)
	}
	s.remove(old)
	if _, err = s.Begin(context.Background(), input); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("old cleanup removed the newer account reservation", err)
	}
}

func TestPendingLoginRestoresChallengeWithoutAuthenticationOrAnswer(t *testing.T) {
	s, source, e := testService(t)
	if pending, err := s.Pending(context.Background(), input); err != nil || pending != nil {
		t.Fatal(pending, err)
	}
	report, err := s.Begin(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := s.Pending(context.Background(), input)
	if err != nil || pending == nil || pending.ID != report.ID || pending.Challenge == nil || pending.Challenge.Prompt != report.Challenge.Prompt || e.authentications.Load() != 1 || e.answers.Load() != 0 || source.releases.Load() != 0 {
		t.Fatal("read-only challenge restoration performed login work or lost its scope", pending, err)
	}
	foreign := input
	foreign.ExpectedRevision = "different-config"
	if _, err = s.Pending(context.Background(), foreign); err == nil {
		t.Fatal("another configuration recovered the challenge", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.Pending(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if pending, err = s.Pending(context.Background(), input); err != nil || pending == nil {
		t.Fatal("cancelled read discarded challenge", err)
	}
	source.stale.Store(true)
	if _, err = s.Pending(context.Background(), input); !errors.Is(err, indexer.ErrInvalidSettings) {
		t.Fatal("changed configuration recovered challenge", err)
	}
	if e.closes.Load() != 1 || source.releases.Load() != 1 {
		t.Fatal("stale pending challenge retained resources")
	}
}

func TestCurrentConfigurationCanRetireObsoleteLoginButStaleRequestsCannot(t *testing.T) {
	for _, mode := range []string{"begin", "restore"} {
		t.Run(mode, func(t *testing.T) {
			s, source, e := testService(t)
			first, err := s.Begin(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			next := input
			next.ExpectedRevision = "new-current-config"
			if _, err = s.Begin(context.Background(), next); err == nil {
				t.Fatal("unconfirmed configuration cancelled an active login")
			}
			if e.closes.Load() != 0 || source.releases.Load() != 0 {
				t.Fatal("stale request destroyed an active login")
			}
			source.revision.Store(next.ExpectedRevision)
			if mode == "restore" {
				if pending, err := s.Pending(context.Background(), next); err != nil || pending != nil {
					t.Fatal("new configuration restored an old challenge", pending, err)
				}
			}
			current, err := s.Begin(context.Background(), next)
			if err != nil || current.ID == first.ID || current.ConfigRevision != next.ExpectedRevision {
				t.Fatal("new configuration remained blocked by old challenge", current, err)
			}
			if _, err = s.Answer(context.Background(), first.ID, input, "private-answer"); !errors.Is(err, indexer.ErrExpired) {
				t.Fatal("obsolete flow retained an answer capability", err)
			}
			if e.closes.Load() != 1 || source.releases.Load() != 1 {
				t.Fatal("old configuration retained its challenge or rule lease")
			}
		})
	}
}
