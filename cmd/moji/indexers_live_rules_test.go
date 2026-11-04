package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexercheck"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

// Exercise the real distribution client in isolation, including an update of
// the whole v11 library and a local rollback without replacing the program.
func TestOptInLiveOfficialRuleUpdate(t *testing.T) {
	sourcePath := os.Getenv("MOJI_INDEXER_LIVE_CONFIG")
	if sourcePath == "" || os.Getenv("MOJI_INDEXER_LIVE_RULE_UPDATES") != "1" {
		t.Skip("requires explicit authorized configuration and live rule-update opt-in")
	}
	original, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal("cannot read authorized configuration")
	}
	copyPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(copyPath, original, 0600); err != nil {
		t.Fatal("cannot prepare isolated configuration")
	}
	store, err := config.OpenStore(copyPath)
	if err != nil {
		t.Fatal("cannot parse isolated configuration")
	}
	archive, err := os.ReadFile(os.Getenv("MOJI_INDEXER_LIVE_RULES_ARCHIVE"))
	if err != nil {
		t.Fatal("cannot read bootstrap archive")
	}
	digest := sha256.Sum256(archive)
	if hex.EncodeToString(digest[:]) != liveRulesArchiveSHA256 {
		t.Fatal("bootstrap differs from pinned corpus")
	}
	target := os.Getenv("MOJI_INDEXER_LIVE_REPORT")
	if target == "" {
		t.Fatal("official update check requires a private report path")
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal("report must be a new file")
	}
	defer file.Close()
	output := struct {
		Bootstrap, Available, Updated, RolledBack string
		Update, Rollback                          *indexerdefs.Job
		Definitions                               int
		UpdatedSite                               *indexercheck.Report
		CheckOnlyPreservedActive, LinkChanged     bool
		SolverConfigured                          bool
		OriginalConfigurationUnchanged            bool
		Tasks                                     int
	}{Bootstrap: liveRulesRevision, Tasks: -1}
	defer func() {
		after, err := os.ReadFile(sourcePath)
		output.OriginalConfigurationUnchanged = err == nil && bytes.Equal(original, after)
		if !output.OriginalConfigurationUnchanged {
			t.Error("source configuration changed during rule check")
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			t.Error("cannot serialize rule check")
			return
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			t.Error("cannot write rule check report")
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal("cannot open isolated storage")
	}
	defer manager.Close()
	var runtime *indexerRuntime
	defer func() {
		if runtime != nil {
			runtime.Close()
		}
	}()
	await := func(id string) *indexerdefs.Job {
		work, stop := context.WithTimeout(ctx, 3*time.Minute)
		defer stop()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			job, err := runtime.definitions.Job(work, id)
			if err != nil {
				t.Fatal("cannot read isolated rule job")
			}
			if job.State.Terminal() {
				return job
			}
			select {
			case <-work.Done():
				t.Fatal("isolated rule job exceeded budget")
			case <-ticker.C:
			}
		}
	}
	bootstrap := &runtimeRuleSource{revision: liveRulesRevision, archive: archive}
	runtime, err = newIndexerRuntime(ctx, store.Config(), store, manager, bootstrap)
	if err != nil || runtime.Start(ctx) != nil {
		t.Fatal("cannot start isolated bootstrap runtime")
	}
	if _, err := runtime.definitions.Check(ctx); err != nil {
		t.Fatal("cannot discover bootstrap rules")
	}
	job, err := runtime.definitions.Begin(ctx, "INSTALL", liveRulesRevision, "")
	if err != nil || await(job.ID).State != indexerdefs.Completed {
		t.Fatal("cannot install bootstrap rules")
	}
	before, err := runtime.definitions.Acquire(ctx, "")
	if err != nil || before.Bundle.Plans["52bt"] == nil {
		t.Fatal("bootstrap rule missing")
	}
	oldLink := before.Bundle.Plans["52bt"].Definition.Links[0]
	before.Release()
	runtime.Close()
	runtime = nil
	// A nil source assembles the actual official GitHub API/codeload client.
	runtime, err = newIndexerRuntime(ctx, store.Config(), store, manager, nil)
	if err != nil || runtime.Start(ctx) != nil {
		t.Fatal("cannot reopen runtime with official source")
	}
	state, err := runtime.definitions.Check(ctx)
	if err != nil {
		t.Fatal("official rule discovery failed")
	}
	output.Available = state.Available
	output.CheckOnlyPreservedActive = state.Active == liveRulesRevision
	if !output.CheckOnlyPreservedActive || state.Available == liveRulesRevision {
		t.Fatal("check changed active rules or no newer revision was available")
	}
	job, err = runtime.definitions.Begin(ctx, "UPDATE", state.Available, liveRulesRevision)
	if err != nil {
		t.Fatal("official update was not admitted")
	}
	output.Update = await(job.ID)
	if output.Update.State != indexerdefs.Completed {
		t.Fatal("official update failed; see sanitized job state")
	}
	updated, err := runtime.definitions.Acquire(ctx, "")
	if err != nil || updated.Bundle.Plans["52bt"] == nil {
		t.Fatal("updated rule missing")
	}
	output.Updated = updated.Bundle.Manifest.Revision
	output.Definitions = updated.Bundle.Manifest.DefinitionCount
	output.LinkChanged = updated.Bundle.Plans["52bt"].Definition.Links[0] != oldLink
	updated.Release()
	// Only perform the already-authorized site operation when explicitly asked
	// and the site is present in the original configuration scope.
	if query := os.Getenv("MOJI_INDEXER_LIVE_QUERY"); query != "" {
		allowed := false
		for _, id := range store.Config().Indexers.QueryOrder {
			allowed = allowed || id == "jackett/52bt"
		}
		if !allowed {
			t.Fatal("updated site is outside authorized scope")
		}
		definition, enabled := "52bt", true
		profile := "direct"
		if solver := os.Getenv("MOJI_INDEXER_LIVE_SOLVER"); solver != "" {
			output.SolverConfigured = true
			profile = "live-solver"
			if _, err := store.SetIndexerNetworkProfile(ctx, store.Revision(), profile, &config.IndexerNetworkProfile{FlareSolverrURL: solver}); err != nil {
				t.Fatal("cannot prepare solver in isolated configuration")
			}
		}
		id, _, err := store.UpdateBuiltinInstance(ctx, "", store.Revision(), config.BuiltinInstancePatch{Definition: &definition, Enabled: &enabled, NetworkProfile: &profile})
		if err != nil {
			t.Fatal("cannot configure updated site in isolated copy")
		}
		report, err := runtime.checks.Check(ctx, indexercheck.Input{InstanceID: id, ExpectedRevision: store.Revision(), Keywords: query, Search: true, Download: true})
		if err != nil {
			t.Fatal("updated site check was not admitted")
		}
		output.UpdatedSite = &report
	}
	job, err = runtime.definitions.Begin(ctx, "ROLLBACK", liveRulesRevision, output.Updated)
	if err != nil {
		t.Fatal("local rollback was not admitted")
	}
	output.Rollback = await(job.ID)
	if output.Rollback.State != indexerdefs.Completed {
		t.Fatal("local rollback failed")
	}
	status, err := runtime.definitions.Status(ctx)
	if err != nil {
		t.Fatal("cannot read final rule state")
	}
	output.RolledBack = status.State.Active
	if output.RolledBack != liveRulesRevision || status.State.Previous != output.Updated {
		t.Error("rollback did not preserve both installed revisions")
	}
	if err := manager.Core.Reader.GetContext(ctx, &output.Tasks, "SELECT count(*) FROM tasks"); err != nil || output.Tasks != 0 {
		t.Error("rule acceptance created task state")
	}
}
