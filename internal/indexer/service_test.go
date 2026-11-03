package indexer

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type registryFunc func(context.Context) (Registry, error)

func (f registryFunc) Snapshot(ctx context.Context) (Registry, error) { return f(ctx) }

type fakeAdapter struct {
	search func(context.Context, InstanceSnapshot, Query) (Batch, error)
}

type testResources struct{}

func (testResources) SaveBatch(context.Context, InstanceSnapshot, []ResourceRef) error { return nil }
func (testResources) Resolve(context.Context, string) (*ResourceLease, error)          { return nil, ErrExpired }

func (f fakeAdapter) Search(ctx context.Context, instance InstanceSnapshot, query Query) (Batch, error) {
	return f.search(ctx, instance, query)
}
func (f fakeAdapter) Fetch(context.Context, InstanceSnapshot, ResourceRef) (Artifact, error) {
	return Artifact{}, ErrUnsupported
}
func candidateBatch(instance InstanceSnapshot) Batch {
	return Batch{InstanceID: instance.ID, Releases: []Release{{Title: "ABCD-123", InstanceID: instance.ID, Resource: ResourceRef{Version: 1, ID: instance.ID + "-resource", InstanceID: instance.ID, Kind: instance.Kind, ConfigRevision: instance.ConfigRevision}}}}
}

func TestSequentialSearchUsesUnifiedOrderAndStopsOnlyWhenAccepted(t *testing.T) {
	instances := []InstanceSnapshot{{ID: "last", Kind: Builtin, Enabled: true}, {ID: "jackett/remote", Kind: Jackett, Enabled: true}, {ID: "first", Kind: Builtin, Enabled: true}, {ID: "after", Kind: Builtin, Enabled: true}}
	registry := Registry{Instances: instances, Order: []string{"first", "jackett/remote", "last", "after"}}
	var calls []string
	adapter := fakeAdapter{search: func(ctx context.Context, instance InstanceSnapshot, q Query) (Batch, error) {
		calls = append(calls, instance.ID)
		if instance.ID == "first" {
			return Batch{}, ErrTransport
		}
		return candidateBatch(instance), nil
	}}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) { return registry, nil }), testResources{}, adapter, adapter)
	if err != nil {
		t.Fatal(err)
	}
	var inspected []string
	report, err := service.SearchSequential(context.Background(), Query{Keywords: "ABCD-123"}, func(ctx context.Context, batch Batch) (bool, error) {
		inspected = append(inspected, batch.InstanceID)
		return batch.InstanceID == "last", nil
	})
	if err != nil || !reflect.DeepEqual(calls, []string{"first", "jackett/remote", "last"}) || !reflect.DeepEqual(inspected, []string{"jackett/remote", "last"}) || !report.Partial || report.AcceptedInstanceID != "last" {
		t.Fatalf("incorrect sequential behavior: calls=%v inspected=%v report=%+v err=%v", calls, inspected, report, err)
	}
}

func TestSearchPartialFailureIsNotCompleteEmptyAndScopeCannotWiden(t *testing.T) {
	registry := Registry{Instances: []InstanceSnapshot{{ID: "one", Kind: Builtin, Enabled: true}, {ID: "two", Kind: Jackett, Enabled: true}, {ID: "off", Kind: Builtin, Enabled: false}}}
	calls := 0
	adapter := fakeAdapter{search: func(ctx context.Context, instance InstanceSnapshot, q Query) (Batch, error) {
		calls++
		if instance.ID == "one" {
			return Batch{}, ErrTransport
		}
		return Batch{}, nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return registry, nil }), testResources{}, adapter, adapter)
	result, err := service.Search(context.Background(), Query{Keywords: "ABCD-123"})
	if !errors.Is(err, ErrIncomplete) || !result.Report.Partial || len(result.Report.Sites) != 2 {
		t.Fatal("upstream failure presented as empty success", err)
	}
	calls = 0
	if _, err := service.Search(context.Background(), Query{InstanceIDs: []string{"missing"}}); !errors.Is(err, ErrInvalidScope) || calls != 0 {
		t.Fatal("invalid explicit scope widened search", err)
	}
	if _, err := service.Search(context.Background(), Query{InstanceIDs: []string{"off"}}); !errors.Is(err, ErrInvalidScope) || calls != 0 {
		t.Fatal("disabled explicit source queried", err)
	}
	result, err = service.Search(context.Background(), Query{InstanceIDs: []string{"two"}})
	if err != nil || result.Report.Partial || calls != 1 {
		t.Fatal("successful empty scope misreported", err)
	}
}

func TestSearchFreezesSettingsAndMarksTotalResultTruncation(t *testing.T) {
	password := "original"
	registry := Registry{Instances: []InstanceSnapshot{{ID: "one", Kind: Builtin, Enabled: true}, {ID: "two", Kind: Builtin, Enabled: true, Settings: map[string]SettingValue{"password": {String: &password}}}}}
	adapter := fakeAdapter{search: func(ctx context.Context, instance InstanceSnapshot, q Query) (Batch, error) {
		if instance.ID == "one" {
			password = "changed"
		} else if *instance.Settings["password"].String != "original" {
			t.Error("in-flight settings changed")
		}
		batch := candidateBatch(instance)
		for len(batch.Releases) < 1000 {
			batch.Releases = append(batch.Releases, batch.Releases[0])
		}
		return batch, nil
	}}
	service, _ := NewService(registryFunc(func(context.Context) (Registry, error) { return registry, nil }), testResources{}, adapter, nil)
	result, err := service.Search(context.Background(), Query{Limit: 1000})
	if err != nil || len(result.Releases) != 1000 || !result.Report.Truncated {
		t.Fatal("total response bound not explicit", err)
	}
}
