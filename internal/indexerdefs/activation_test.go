package indexerdefs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type activationFunc func(context.Context, func() error) error

func (f activationFunc) WithCurrent(ctx context.Context, commit func() error) error {
	return f(ctx, commit)
}

type validatorFunc func(context.Context, *Bundle) (ActivationGuard, error)

func (f validatorFunc) Validate(ctx context.Context, bundle *Bundle) (ActivationGuard, error) {
	return f(ctx, bundle)
}

func emptyConfigValidator() Validator {
	return validatorFunc(func(ctx context.Context, bundle *Bundle) (ActivationGuard, error) {
		return activationFunc(func(ctx context.Context, commit func() error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return commit()
		}), nil
	})
}

func TestUnknownEncodingUpdatePreservesInstalledLibrary(t *testing.T) {
	_, svc, source := serviceFixture(t)
	ctx := context.Background()
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	installNext(t, svc, source, a)
	data := validArchive(t, strings.Replace(fixtureDefinition, "encoding: UTF-8", "encoding: moji-unknown-codepage", 1))
	source.mu.Lock()
	source.archives[b] = data
	source.latest = b
	source.mu.Unlock()
	if _, err := svc.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := svc.Begin(ctx, "UPDATE", b, a)
	if err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, svc, job.ID); finished.State != Failed || finished.ErrorCode != ErrIncompatible.Error() {
		t.Fatalf("unknown encoding update accepted: %+v", finished)
	}
	status, err := svc.Status(ctx)
	if err != nil || status.State.Active != a || !status.Available {
		t.Fatalf("invalid encoding damaged installed library: %+v, %v", status, err)
	}
	if _, err := os.Stat(filepath.Join(svc.root, b)); !os.IsNotExist(err) {
		t.Fatal("invalid encoding candidate was published", err)
	}
}

func TestInvalidSearchFilterUpdatePreservesInstalledLibrary(t *testing.T) {
	for _, pipeline := range []string{"keywordsfilters", "preprocessingfilters"} {
		t.Run(pipeline, func(t *testing.T) {
			_, svc, source := serviceFixture(t)
			ctx := context.Background()
			a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
			installNext(t, svc, source, a)
			definition := strings.Replace(fixtureDefinition, "  rows: {selector: article}", "  "+pipeline+": [{name: regexp, args: '('}]\n  rows: {selector: article}", 1)
			archive := validArchive(t, definition)
			source.mu.Lock()
			source.archives[b] = archive
			source.latest = b
			source.mu.Unlock()
			if _, err := svc.Check(ctx); err != nil {
				t.Fatal(err)
			}
			job, err := svc.Begin(ctx, "UPDATE", b, a)
			if err != nil {
				t.Fatal(err)
			}
			if finished := awaitJob(t, svc, job.ID); finished.State != Failed || finished.ErrorCode != ErrIncompatible.Error() {
				t.Fatalf("invalid executable filter update accepted: %+v", finished)
			}
			status, err := svc.Status(ctx)
			if err != nil || status.State.Active != a || !status.Available {
				t.Fatalf("bad filter damaged active library: %+v, %v", status, err)
			}
			if _, err := os.Stat(filepath.Join(svc.root, b)); !os.IsNotExist(err) {
				t.Fatal("bad filter candidate was published", err)
			}
		})
	}
}

func TestCandidateValidationFailurePreservesActiveAndRechecksConfiguration(t *testing.T) {
	manager, initial, source := serviceFixture(t)
	ctx := context.Background()
	a, b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	installNext(t, initial, source, a)
	initial.Close()
	var validateCalls atomic.Int32
	validator := validatorFunc(func(ctx context.Context, bundle *Bundle) (ActivationGuard, error) {
		call := validateCalls.Add(1)
		if call == 1 {
			return nil, ErrIncompatible
		}
		if call == 2 {
			return activationFunc(func(context.Context, func() error) error { return ErrConflict }), nil
		}
		return activationFunc(func(ctx context.Context, commit func() error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return commit()
		}), nil
	})
	svc, err := NewService(manager, source, validator)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	source.setLatest(b)
	if _, err := svc.Check(ctx); err != nil {
		t.Fatal(err)
	}
	job, err := svc.Begin(ctx, "UPDATE", b, a)
	if err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, svc, job.ID); finished.State != Failed || finished.ErrorCode != ErrIncompatible.Error() {
		t.Fatalf("incorrect rejected job %+v", finished)
	}
	status, err := svc.Status(ctx)
	if err != nil || status.State.Active != a || !status.Available {
		t.Fatal("failed candidate changed active", err)
	}
	if _, err := os.Stat(filepath.Join(svc.root, b)); !os.IsNotExist(err) {
		t.Fatal("incompatible candidate published", err)
	}
	job, err = svc.Begin(ctx, "UPDATE", b, a)
	if err != nil {
		t.Fatal(err)
	}
	if finished := awaitJob(t, svc, job.ID); finished.State != Completed {
		t.Fatalf("configuration recheck failed %+v", finished)
	}
	if validateCalls.Load() != 3 {
		t.Fatalf("expected initial validation plus bounded revision recheck, got %d", validateCalls.Load())
	}
}

func TestApplicationShutdownInterruptsUpdateWithoutActivating(t *testing.T) {
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
		t.Fatal("update not started")
	}
	svc.Close()
	finished, err := svc.Job(ctx, job.ID)
	if err != nil || finished.State != Interrupted {
		t.Fatalf("shutdown not persisted %+v %v", finished, err)
	}
	status, err := svc.Status(ctx)
	if err != nil || status.State.Active != a {
		t.Fatal("shutdown changed active", err)
	}
	entries, err := os.ReadDir(svc.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".download-") || strings.HasPrefix(entry.Name(), ".staging-") {
			t.Fatal("shutdown left temporary assets", entry.Name())
		}
	}
}
