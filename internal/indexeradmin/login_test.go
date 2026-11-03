package indexeradmin

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerlogin"
)

func TestPrepareLoginPinsSavedAccountWithoutConfigurationWrites(t *testing.T) {
	service, store, defs, registry := managementFixture(t)
	defs.bundle.Plans["example"].Definition.Login = &cardigann.Login{Method: "post", Path: "/login"}
	definition, secret := "example", "saved-synthetic-password"
	saved, err := service.SaveBuiltin(context.Background(), "", store.Revision(), BuiltinPatch{Definition: &definition, Settings: map[string]config.IndexerSettingChange{"password": {Value: &config.IndexerSettingValue{String: &secret}}}})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	revision := store.Revision()
	prepared, err := service.PrepareLogin(context.Background(), indexerlogin.Input{InstanceID: saved.Instance.ID, ExpectedRevision: revision})
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Release()
	if prepared.Instance.ID != saved.Instance.ID || *prepared.Instance.Settings["password"].String != secret || prepared.RuleRevision != defs.bundle.Manifest.Revision || prepared.Plan != defs.bundle.Plans[definition] || prepared.ConfigRevision != revision {
		t.Fatal("login did not freeze the saved account and rule")
	}
	if err = service.ValidateLogin(context.Background(), revision); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(store.Path())
	if err != nil || !bytes.Equal(before, after) || registry.refreshes != 0 {
		t.Fatal("login preparation changed configuration or refreshed gateway", err)
	}
	name := "Changed account"
	if _, err = service.SaveBuiltin(context.Background(), saved.Instance.ID, revision, BuiltinPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err = service.ValidateLogin(context.Background(), revision); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal("stale login could submit an answer", err)
	}
	if _, err = service.PrepareLogin(context.Background(), indexerlogin.Input{InstanceID: saved.Instance.ID, ExpectedRevision: revision}); !errors.Is(err, config.ErrConfigRevision) {
		t.Fatal(err)
	}
	if _, err = service.PrepareLogin(context.Background(), indexerlogin.Input{InstanceID: "jackett/one", ExpectedRevision: store.Revision()}); !errors.Is(err, indexer.ErrUnsupported) {
		t.Fatal("external gateway login accepted", err)
	}
	if _, err = service.PrepareLogin(context.Background(), indexerlogin.Input{ExpectedRevision: store.Revision()}); !errors.Is(err, indexer.ErrInvalidScope) {
		t.Fatal("unsaved login accepted", err)
	}
}
