package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

// This is never part of the default offline suite. The explicit config supplies
// the authorized site scope; all state and config writes use private copies.
func TestOptInLiveIndexerSearchAndFetch(t *testing.T) {
	sourcePath := os.Getenv("MOJI_INDEXER_LIVE_CONFIG")
	if sourcePath == "" {
		t.Skip("requires explicit live search/fetch authorization and MOJI_INDEXER_LIVE_CONFIG")
	}
	archivePath := os.Getenv("MOJI_INDEXER_LIVE_RULES_ARCHIVE")
	if archivePath == "" {
		t.Fatal("MOJI_INDEXER_LIVE_RULES_ARCHIVE must name the pinned corpus archive")
	}
	keywords := os.Getenv("MOJI_INDEXER_LIVE_QUERY")
	if keywords == "" {
		t.Fatal("MOJI_INDEXER_LIVE_QUERY must explicitly select the live search")
	}
	original, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal("cannot read authorized configuration")
	}
	copyPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(copyPath, original, 0600); err != nil {
		t.Fatal("cannot prepare private configuration copy")
	}
	store, err := config.OpenStore(copyPath)
	if err != nil {
		t.Fatal("cannot parse private configuration copy")
	}
	cfg := store.Config()
	ids := append([]string(nil), cfg.Indexers.QueryOrder...)
	if len(ids) == 0 || len(ids) > 16 {
		t.Fatal("live check requires 1..16 explicitly configured sites")
	}
	if selected := os.Getenv("MOJI_INDEXER_LIVE_SITE_IDS"); selected != "" {
		wanted := map[string]bool{}
		for _, id := range strings.Split(selected, ",") {
			wanted[strings.TrimSpace(id)] = true
		}
		var filtered []string
		for _, id := range ids {
			if wanted[strings.TrimPrefix(id, "jackett/")] {
				filtered = append(filtered, id)
				delete(wanted, strings.TrimPrefix(id, "jackett/"))
			}
		}
		if len(wanted) != 0 || len(filtered) == 0 {
			t.Fatal("live site filter must be a subset of the authorized configuration")
		}
		ids = filtered
	}
	kind := os.Getenv("MOJI_INDEXER_LIVE_KIND")
	if kind != "" && kind != "BUILTIN" && kind != "JACKETT" {
		t.Fatal("live kind must be BUILTIN, JACKETT or unset")
	}
	var reportFile *os.File
	if target := os.Getenv("MOJI_INDEXER_LIVE_REPORT"); target != "" {
		reportFile, err = os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal("live report must be a new file; existing files are never overwritten")
		}
		defer reportFile.Close()
	}
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal("cannot read pinned rule archive")
	}
	digest := sha256.Sum256(archive)
	if hex.EncodeToString(digest[:]) != "b266646ab2662c260ac6b0fad5f99794b200784f3ac865b521820e8e6814a472" {
		t.Fatal("live rule archive differs from the reviewed corpus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 26*time.Minute)
	defer cancel()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal("cannot open isolated storage")
	}
	defer manager.Close()
	source := &runtimeRuleSource{revision: "5f5db4e58d06be18619c3a858b9a91ade6f341ca", archive: archive}
	runtime, err := newIndexerRuntime(ctx, cfg, store, manager, source)
	if err != nil {
		t.Fatal("cannot assemble isolated indexer runtime")
	}
	defer runtime.Close()
	if err := runtime.Start(ctx); err != nil {
		t.Fatal("cannot start isolated indexer runtime")
	}
	if _, err := runtime.definitions.Check(ctx); err != nil {
		t.Fatal("cannot discover pinned local rules")
	}
	job, err := runtime.definitions.Begin(ctx, "INSTALL", source.revision, "")
	if err != nil {
		t.Fatal("cannot install pinned local rules")
	}
	if result := awaitRuntimeRuleJob(t, ctx, runtime.definitions, job.ID); result.State != indexerdefs.Completed {
		t.Fatal("pinned local rule installation failed")
	}
	networkProfile := "direct"
	solver := os.Getenv("MOJI_INDEXER_LIVE_SOLVER")
	if solver != "" {
		networkProfile = "live-solver"
		if _, err := store.SetIndexerNetworkProfile(ctx, store.Revision(), networkProfile, &config.IndexerNetworkProfile{FlareSolverrURL: solver}); err != nil {
			t.Fatal("cannot prepare solver in isolated configuration")
		}
	}
	type sitePair struct{ site, builtin, external string }
	var pairs []sitePair
	for _, id := range ids {
		if !strings.HasPrefix(id, "jackett/") {
			t.Fatal("this comparison requires explicit Jackett site scope")
		}
		site := strings.TrimPrefix(id, "jackett/")
		enabled := true
		builtin, _, err := store.UpdateBuiltinInstance(ctx, "", store.Revision(), config.BuiltinInstancePatch{Definition: &site, Name: &site, Enabled: &enabled, NetworkProfile: &networkProfile})
		if err != nil {
			t.Fatal("cannot prepare isolated builtin site")
		}
		pairs = append(pairs, sitePair{site, builtin, id})
	}
	type result struct {
		Site, Kind, ErrorCode string
		Report                indexercheck.Report
	}
	type originComparison struct {
		Site                                            string
		ReferenceConfigured, OriginCompared, SameOrigin bool
	}
	output := struct {
		RuleRevision, Keywords, CatalogErrorCode string
		Results                                  []result
		Tasks                                    int
		SourceConfigurationUnchanged             bool
		CheckBudgetSeconds                       int
		SolverConfigured, CatalogRequested       bool
		CatalogKnown                             bool
		Origins                                  []originComparison
	}{RuleRevision: source.revision, Keywords: keywords, CheckBudgetSeconds: 180, SolverConfigured: solver != "", CatalogRequested: kind != "BUILTIN"}
	if output.CatalogRequested {
		registry, err := runtime.registry.Refresh(ctx)
		if err != nil {
			output.CatalogErrorCode = indexer.ErrorCode(err)
		}
		output.CatalogKnown = registry.JackettCatalog.Known
		for _, diagnostic := range registry.Diagnostics {
			if diagnostic.InstanceID == "jackett" && diagnostic.Code != "" {
				output.CatalogErrorCode = diagnostic.Code
			}
		}
		lease, err := runtime.definitions.Acquire(ctx, "")
		if err != nil {
			t.Fatal("cannot inspect pinned rule origins")
		}
		for _, id := range cfg.Indexers.QueryOrder {
			site := strings.TrimPrefix(id, "jackett/")
			plan := lease.Bundle.Plans[site]
			if plan == nil {
				continue
			}
			row := originComparison{Site: site}
			for _, remote := range registry.Instances {
				if remote.ID == id && remote.Kind == indexer.Jackett {
					row.ReferenceConfigured = true
					builtinURL, e1 := url.Parse(plan.Definition.Links[0])
					remoteURL, e2 := url.Parse(remote.SiteURL)
					row.OriginCompared = e1 == nil && e2 == nil && builtinURL.Host != "" && remoteURL.Host != ""
					row.SameOrigin = row.OriginCompared && strings.EqualFold(builtinURL.Scheme, remoteURL.Scheme) && strings.EqualFold(builtinURL.Host, remoteURL.Host)
				}
			}
			output.Origins = append(output.Origins, row)
		}
		lease.Release()
	}
	revision := store.Revision()
	var mu sync.Mutex
	writeReport := func() {
		if reportFile != nil {
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil || reportFile.Truncate(0) != nil {
				t.Error("cannot write sanitized live report")
				return
			}
			if _, err := reportFile.Seek(0, 0); err != nil {
				t.Error("cannot write sanitized live report")
				return
			}
			if _, err := reportFile.Write(append(data, '\n')); err != nil {
				t.Error("cannot write sanitized live report")
			}
		}
	}
	// Two sites at a time, with sequential builtin/Jackett requests per site.
	// Each check may fetch at most one artifact and cannot create a Moji task.
	slots := make(chan struct{}, 2)
	var workers sync.WaitGroup
	for _, pair := range pairs {
		slots <- struct{}{}
		workers.Add(1)
		go func(pair sitePair) {
			defer workers.Done()
			defer func() { <-slots }()
			for _, entry := range []struct{ kind, id string }{{"BUILTIN", pair.builtin}, {"JACKETT", pair.external}} {
				if kind != "" && kind != entry.kind {
					continue
				}
				work, stop := context.WithTimeout(ctx, 3*time.Minute)
				report, err := runtime.checks.Check(work, indexercheck.Input{InstanceID: entry.id, ExpectedRevision: revision, Keywords: keywords, Search: true, Download: true})
				stop()
				row := result{Site: pair.site, Kind: entry.kind, Report: report}
				if err != nil {
					row.ErrorCode = indexer.ErrorCode(err)
				}
				mu.Lock()
				output.Results = append(output.Results, row)
				writeReport()
				mu.Unlock()
				t.Logf("live check completed: %s %s results=%d", pair.site, entry.kind, report.Results)
			}
		}(pair)
	}
	workers.Wait()
	if err := manager.Core.Reader.GetContext(ctx, &output.Tasks, "SELECT count(*) FROM tasks"); err != nil || output.Tasks != 0 {
		t.Error("live diagnostics created task state")
	}
	after, err := os.ReadFile(sourcePath)
	output.SourceConfigurationUnchanged = err == nil && bytes.Equal(original, after)
	if !output.SourceConfigurationUnchanged {
		t.Error("source configuration changed during the check")
	}
	writeReport()
	// A successful harness exit means observations were collected safely. It
	// does not turn FAILED/EMPTY/PARTIAL site stages into availability passes.
}
