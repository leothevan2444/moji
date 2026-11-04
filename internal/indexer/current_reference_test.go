package indexer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type currentReferenceRegistry struct{ registry Registry }

func (r *currentReferenceRegistry) Snapshot(context.Context) (Registry, error) {
	return Registry{}, errors.New("reference admission must not refresh the remote catalog")
}
func (r *currentReferenceRegistry) CachedSnapshot(ctx context.Context) (Registry, error) {
	return r.registry, ctx.Err()
}

type currentReferenceResources struct {
	record   ResourceRecord
	released int
}

func (*currentReferenceResources) SaveBatch(context.Context, InstanceSnapshot, []ResourceRef) error {
	return nil
}
func (r *currentReferenceResources) Resolve(context.Context, string) (*ResourceLease, error) {
	return &ResourceLease{Record: r.record.Clone(), release: func() { r.released++ }}, nil
}

type currentReferenceAdapter struct{ fetches int }

func (*currentReferenceAdapter) Search(context.Context, InstanceSnapshot, Query) (Batch, error) {
	return Batch{}, ErrUnsupported
}
func (a *currentReferenceAdapter) Fetch(context.Context, InstanceSnapshot, ResourceRef) (Artifact, error) {
	a.fetches++
	return Artifact{Magnet: "magnet:?xt=urn:btih:" + strings.Repeat("d", 40)}, nil
}

func currentReferenceFixture(t *testing.T) (*Service, *currentReferenceRegistry, *currentReferenceResources, *currentReferenceAdapter, *cardigann.Plan) {
	t.Helper()
	instance := InstanceSnapshot{ID: "builtin-one", Kind: Builtin, Enabled: true, DefinitionID: "example", ConfigRevision: "original-account", BundleRevision: strings.Repeat("a", 40)}
	ref := ResourceRef{Version: 1, ID: "opaque", InstanceID: instance.ID, Kind: Builtin, ConfigRevision: instance.ConfigRevision, DefinitionID: instance.DefinitionID, BundleRevision: instance.BundleRevision, DefinitionDigest: strings.Repeat("b", 64)}
	resources := &currentReferenceResources{record: ResourceRecord{Version: 1, Instance: instance, Ref: ref}}
	registry := &currentReferenceRegistry{registry: Registry{Instances: []InstanceSnapshot{instance}}}
	adapter := &currentReferenceAdapter{}
	service, err := NewService(registry, resources, adapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan := &cardigann.Plan{Digest: ref.DefinitionDigest}
	service.bundles = fixedBundles{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("c", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}
	return service, registry, resources, adapter, plan
}

func TestCurrentReferenceAdmissionRejectsChangedSourceAndRetainsUnchangedRule(t *testing.T) {
	for _, change := range []string{"unchanged-rule-new-bundle", "disabled", "removed", "account", "rule"} {
		t.Run(change, func(t *testing.T) {
			service, registry, resources, adapter, plan := currentReferenceFixture(t)
			switch change {
			case "disabled":
				registry.registry.Instances[0].Enabled = false
			case "removed":
				registry.registry.Instances = nil
			case "account":
				registry.registry.Instances[0].ConfigRevision = "replacement-account"
			case "rule":
				plan.Digest = strings.Repeat("e", 64)
			}
			lease, err := service.ResolveReference(context.Background(), resources.record.Ref.ID)
			if change == "unchanged-rule-new-bundle" {
				if err != nil || lease == nil || lease.Record.Ref.BundleRevision != resources.record.Ref.BundleRevision {
					t.Fatal("unchanged rule did not retain its original snapshot", err)
				}
				lease.Release()
			} else if !errors.Is(err, ErrReference) || lease != nil {
				t.Fatal("new admission accepted a changed source", err)
			}
			if resources.released != 1 || adapter.fetches != 0 {
				t.Fatal("admission leaked its lease or fetched a resource")
			}
		})
	}
}

func TestCurrentReferenceDoesNotRevokeAlreadyAcceptedAcquisition(t *testing.T) {
	service, registry, resources, adapter, _ := currentReferenceFixture(t)
	registry.registry.Instances[0].Enabled = false
	artifact, err := service.FetchRecord(context.Background(), resources.record)
	if err != nil || artifact.Magnet == "" || adapter.fetches != 1 {
		t.Fatal("accepted task lost its frozen acquisition after disabling the source", err)
	}
}

func TestCurrentReferenceFetchValidatesBeforeCachedArtifact(t *testing.T) {
	service, registry, resources, adapter, _ := currentReferenceFixture(t)
	cache, err := NewArtifactCache(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	service.SetArtifactCache(cache)
	if _, err := service.FetchCurrentReference(context.Background(), resources.record.Ref.ID); err != nil {
		t.Fatal(err)
	}
	registry.registry.Instances[0].Enabled = false
	if _, err := service.FetchCurrentReference(context.Background(), resources.record.Ref.ID); !errors.Is(err, ErrReference) {
		t.Fatal("cached artifact bypassed current admission policy", err)
	}
	if _, err := service.FetchRecord(context.Background(), resources.record); err != nil {
		t.Fatal("accepted snapshot could not reuse the same acquired bytes", err)
	}
	if adapter.fetches != 1 || resources.released != 2 {
		t.Fatal("new admission fetched again or leaked a reference lease")
	}
}
