package taskruntime

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
)

type sourcingScopeProbe struct{ queries []indexer.Query }

func (p *sourcingScopeProbe) SearchSequential(_ context.Context, query indexer.Query, _ func(context.Context, indexer.Batch) (bool, error)) (indexer.SearchReport, error) {
	p.queries = append(p.queries, query)
	return indexer.SearchReport{}, nil
}
func (*sourcingScopeProbe) FetchReference(context.Context, string) (indexer.Artifact, error) {
	panic("empty search must not fetch")
}

func TestLegacyQueuedScopeAlwaysNamesJackettEvenForUUIDShapedIDs(t *testing.T) {
	ctx := context.Background()
	store, _ := relationalStore(t)
	now := time.Now().UTC()
	// This is the unversioned payload shape written by 502cfa9, before builtin
	// instance IDs existed. No shape of its Tracker strings can mean builtin.
	task := &Task{ID: "old-scope", Source: TaskSourceSearch, Code: "UPGRADE-123", Stage: TaskStageSourcing, StageStatus: TaskStageStatusPending, CreatedAt: now, UpdatedAt: now,
		SourcingRequestJSON: `{"Source":"SEARCH","Code":"UPGRADE-123","Trackers":["legacy-site","33333333-3333-4333-8333-333333333333"],"Categories":[2000],"Limit":17,"SavePath":"/downloads/upgrade","Category":"upgrade","Tags":"preserved","Paused":true}`}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	probe := &sourcingScopeProbe{}
	service, err := NewService(nil, &fakeTorrentAdder{}, store, WithIndexers(probe))
	if err != nil {
		t.Fatal(err)
	}
	_ = service.processQueuedSourcing(ctx, task.ID)
	want := []string{"jackett/legacy-site", "jackett/33333333-3333-4333-8333-333333333333"}
	if len(probe.queries) != 1 || !reflect.DeepEqual(probe.queries[0].InstanceIDs, want) || probe.queries[0].Limit != 17 || !reflect.DeepEqual(probe.queries[0].Categories, []int{2000}) {
		t.Fatalf("legacy scope was reinterpreted: %+v", probe.queries)
	}
}

func TestUnknownQueuedRequestVersionIsBlockedBeforeSearching(t *testing.T) {
	ctx := context.Background()
	store, _ := relationalStore(t)
	now := time.Now().UTC()
	task := &Task{ID: "unknown-payload", Source: TaskSourceSearch, Code: "UPGRADE-124", Stage: TaskStageSourcing, StageStatus: TaskStageStatusPending, CreatedAt: now, UpdatedAt: now, SourcingRequestJSON: `{"version":99,"Code":"UPGRADE-124","Trackers":[]}`}
	if err := store.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	probe := &sourcingScopeProbe{}
	service, err := NewService(nil, &fakeTorrentAdder{}, store, WithIndexers(probe))
	if err != nil {
		t.Fatal(err)
	}
	_ = service.processQueuedSourcing(ctx, task.ID)
	saved, err := store.Find(ctx, task.ID)
	if err != nil || len(probe.queries) != 0 || saved.StageErrorCode != TaskStageErrorSourcingRequest || saved.SourcingRequestJSON != task.SourcingRequestJSON {
		t.Fatalf("unknown request executed or lost its saved intent: %+v %+v %v", probe.queries, saved, err)
	}
}

func TestNewQueuedRequestHasAnExplicitSourceIdentityVersion(t *testing.T) {
	ctx := context.Background()
	store, _ := relationalStore(t)
	probe := &sourcingScopeProbe{}
	service, err := NewService(nil, &fakeTorrentAdder{}, store, WithIndexers(probe))
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"jackett/legacy-site", "33333333-3333-4333-8333-333333333333"}
	paused := false
	task, err := service.QueueDownloadContext(ctx, DownloadRequest{Code: "UPGRADE-125", Trackers: ids, Paused: &paused})
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Version int `json:"version"`
		DownloadRequest
	}
	if err := json.Unmarshal([]byte(task.SourcingRequestJSON), &saved); err != nil || saved.Version != 1 || !reflect.DeepEqual(saved.Trackers, ids) || saved.Paused == nil || *saved.Paused {
		t.Fatalf("new request has ambiguous source identities: %+v, %v", saved, err)
	}
}

func TestSourcingRequestRejectsUnknownVersionsAndUnqualifiedNewIdentities(t *testing.T) {
	for _, input := range []string{
		`null`, `[]`,
		`{"version":null}`, `{"version":"1"}`, `{"version":2}`,
		`{"version":1,"Trackers":["legacy-site"]}`,
		`{"version":1,"Trackers":["jackett/"]}`,
		`{"version":1,"Tags":"` + strings.Repeat("x", maxSourcingRequestBytes) + `"}`,
	} {
		_, err := unmarshalSourcingRequest([]byte(input))
		if err == nil {
			t.Fatalf("invalid sourcing request accepted: %.80s", input)
		}
	}
	if _, err := unmarshalSourcingRequest([]byte(`{}`)); err != nil {
		t.Fatal("empty legacy scope did not retain all-sites meaning", err)
	}
	if _, err := marshalSourcingRequest(DownloadRequest{Trackers: []string{"unknown/source"}}); err == nil {
		t.Fatal("accepted a new request which cannot be resumed")
	}
}
