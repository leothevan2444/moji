package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/graphqlapi"
)

func TestVRSettingsPreserveOmittedFieldAndPersistExplicitChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("automation:\n  subscription_release_policy:\n    vr_behavior: BLOCK\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := config.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	editor := newRuntimeSettingsEditor(store, "test", false, false, nil, nil, nil)
	input := graphqlapi.UpdateAutomationSettingsInput{TaskProgressSyncIntervalSeconds: 60, SubscriptionPollIntervalHours: 1,
		SubscriptionReleasePolicy: graphqlapi.SubscriptionReleasePolicySnapshot{SoloBehavior: "DOWNLOAD", GroupBehavior: "REVIEW", CompilationBehavior: "BLOCK"}}
	for _, behavior := range []string{"", "DOWNLOAD", "REVIEW", "BLOCK"} {
		input.SubscriptionReleasePolicy.VRBehavior = behavior
		snapshot, err := editor.UpdateAutomationSettings(input)
		if err != nil {
			t.Fatal(err)
		}
		want := behavior
		if want == "" {
			want = "BLOCK"
		}
		if snapshot.Automation.SubscriptionReleasePolicy.VRBehavior != want {
			t.Fatalf("snapshot VR policy = %s, want %s", snapshot.Automation.SubscriptionReleasePolicy.VRBehavior, want)
		}
		reloaded, err := config.OpenStore(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(reloaded.Config().Automation.SubscriptionReleasePolicy.VRBehavior) != want {
			t.Fatal("VR policy was not persisted")
		}
	}
}
