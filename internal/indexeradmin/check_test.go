package indexeradmin

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexercheck"
)

func TestPrepareCheckUsesDraftRetainsSavedSecretsAndDoesNotWriteConfiguration(t *testing.T) {
	service, store, _, registry := managementFixture(t)
	ctx := context.Background()
	definition, secret, cookie := "example", "synthetic-password", "synthetic-cookie"
	saved, err := service.SaveBuiltin(ctx, "", store.Revision(), BuiltinPatch{Definition: &definition, Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &secret}}, "cookie": {Value: &config.IndexerSettingValue{String: &cookie}}}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	revision := store.Revision()
	name, newSecret := "Unsaved name", "synthetic-new-password"
	prepared, err := service.PrepareCheck(ctx, indexercheck.Input{InstanceID: saved.Instance.ID, ExpectedRevision: revision, Draft: &config.BuiltinInstancePatch{Name: &name, Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &newSecret}}}}})
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Release()
	if prepared.Instance.Name != name || *prepared.Instance.Settings["password"].String != newSecret || *prepared.Instance.Settings["cookie"].String != cookie || prepared.ConfigRevision != revision || prepared.RuleRevision == "" || prepared.Instance.BundleRevision != prepared.RuleRevision {
		t.Fatal("diagnostic did not own the real merged draft")
	}
	// Mutating a caller's prepared snapshot cannot alter saved credentials.
	*prepared.Instance.Settings["cookie"].String = "changed-copy"
	if *store.Config().Indexers.Builtin.Instances[saved.Instance.ID].Settings["cookie"].String != cookie {
		t.Fatal("prepared draft shared saved credential pointers")
	}
	draft, err := service.PrepareCheck(ctx, indexercheck.Input{ExpectedRevision: revision, Draft: &config.BuiltinInstancePatch{Definition: &definition}})
	if err != nil || draft.Instance.ID == "" {
		t.Fatal("new unsaved draft did not prepare", err)
	}
	draft.Release()
	after, err := os.ReadFile(store.Path())
	if err != nil || !bytes.Equal(before, after) || store.Revision() != revision || len(store.Config().Indexers.Builtin.Instances) != 1 || registry.refreshes != 0 {
		t.Fatal("test preparation wrote config/created instance/refreshed gateway", err)
	}
}
func TestPrepareCheckRejectsStaleDraftsAndRemoteOverrides(t *testing.T) {
	service, store, _, registry := managementFixture(t)
	ctx := context.Background()
	definition := "example"
	if _, err := service.PrepareCheck(ctx, indexercheck.Input{ExpectedRevision: "stale", Draft: &config.BuiltinInstancePatch{Definition: &definition}}); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal(err)
	}
	if _, err := service.PrepareCheck(ctx, indexercheck.Input{InstanceID: "jackett/one", ExpectedRevision: store.Revision(), Draft: &config.BuiltinInstancePatch{}}); !errors.Is(err, indexer.ErrUnsupported) {
		t.Fatal("remote-owned settings accepted local overrides", err)
	}
	prepared, err := service.PrepareCheck(ctx, indexercheck.Input{InstanceID: "jackett/one", ExpectedRevision: store.Revision()})
	if err != nil || prepared.Instance.Kind != indexer.Jackett || prepared.Release != nil || registry.refreshes != 0 {
		t.Fatal("external test changed remote catalog or claimed local rule ownership", err)
	}
	network := "missing"
	if _, err = service.PrepareCheck(ctx, indexercheck.Input{ExpectedRevision: store.Revision(), Draft: &config.BuiltinInstancePatch{Definition: &definition, NetworkProfile: &network}}); !errors.Is(err, config.ErrInvalidSettings) {
		t.Fatal("invalid draft network profile accepted", err)
	}
}

func TestSavedConnectionChecksUseAccountVersionRatherThanUnrelatedGlobalEdits(t *testing.T) {
	service, store, _, _ := managementFixture(t)
	ctx := context.Background()
	definition := "example"
	first, err := service.SaveBuiltin(ctx, "", store.Revision(), BuiltinPatch{Definition: &definition})
	if err != nil {
		t.Fatal(err)
	}
	oldRevision := store.Revision()
	if _, err = service.SaveBuiltin(ctx, "", store.Revision(), BuiltinPatch{Definition: &definition}); err != nil {
		t.Fatal(err)
	}
	input := indexercheck.Input{InstanceID: first.Instance.ID, ExpectedRevision: oldRevision, ExpectedConnectionRevision: first.Instance.ConnectionRevision, ConnectionOnly: true}
	prepared, err := service.PrepareCheck(ctx, input)
	if err != nil {
		t.Fatal("an unrelated addition invalidated the account", err)
	}
	prepared.Release()
	if prepared.ConfigRevision != store.Revision() {
		t.Fatal("did not capture current global snapshot")
	}
	diagnostic := input
	diagnostic.ConnectionOnly = false
	if _, err = service.PrepareCheck(ctx, diagnostic); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal("ordinary diagnostics bypassed revision checks", err)
	}
	draft := input
	draft.Draft = &BuiltinPatch{}
	if _, err = service.PrepareCheck(ctx, draft); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal("draft bypassed global revision checks", err)
	}
	secret := "changed-synthetic-secret"
	if _, err = service.SaveBuiltin(ctx, first.Instance.ID, store.Revision(), BuiltinPatch{Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &secret}}}}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.PrepareCheck(ctx, input); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal("changed account accepted an old connection version", err)
	}
}
