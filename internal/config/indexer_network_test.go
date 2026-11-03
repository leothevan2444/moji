package config

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestNetworkEditsPreserveCredentialsProtectReferencesAndRejectStaleWrites(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	proxy := "http://synthetic-user:synthetic-password@proxy.example:8080"
	_, ack, err := s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", true, IndexerNetworkProfilePatch{ProxyURL: &proxy})
	if err != nil || ack != s.Revision() {
		t.Fatal("profile creation", err)
	}
	// Preserve fields owned by the optional solver while editing only the proxy.
	profile := s.Config().Indexers.Builtin.NetworkProfiles["private"]
	profile.FlareSolverrURL = "http://solver.example:8191"
	if _, err = s.SetIndexerNetworkProfile(ctx, s.Revision(), "private", &profile); err != nil {
		t.Fatal(err)
	}
	blank := ""
	_, ack, err = s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", false, IndexerNetworkProfilePatch{ProxyURL: &blank})
	if err != nil || s.Config().Indexers.Builtin.NetworkProfiles["private"] != profile {
		t.Fatal("blank/omitted edit erased credentials or solver", err)
	}
	before, _ := os.ReadFile(s.Path())
	stale := ack
	definition, name := "example", "private"
	id, _, err := s.UpdateBuiltinInstance(ctx, "", ack, BuiltinInstancePatch{Definition: &definition, NetworkProfile: &name})
	if err != nil {
		t.Fatal(err)
	}
	protected, _ := os.ReadFile(s.Path())
	if _, _, err = s.RemoveIndexerNetworkProfile(ctx, s.Revision(), "private"); !errors.Is(err, ErrIndexerNetworkInUse) {
		t.Fatal("deleted referenced profile", err)
	}
	if _, _, err = s.SaveIndexerNetworkProfile(ctx, stale, "private", false, IndexerNetworkProfilePatch{ClearProxy: true}); !errors.Is(err, ErrConfigRevision) {
		t.Fatal("stale edit accepted", err)
	}
	for _, id := range []string{"direct", "missing"} {
		if _, _, err = s.SaveIndexerNetworkProfile(ctx, s.Revision(), id, false, IndexerNetworkProfilePatch{ProxyURL: &proxy}); err == nil {
			t.Fatal("updated protected/absent profile", id)
		}
	}
	invalid := "http://synthetic-user:synthetic-password@proxy.example/path?token=secret"
	if _, _, err = s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", false, IndexerNetworkProfilePatch{ProxyURL: &invalid}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatal("invalid endpoint accepted", err)
	}
	if _, _, err = s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", true, IndexerNetworkProfilePatch{ProxyURL: &proxy}); err == nil {
		t.Fatal("create overwrote existing profile")
	}
	after, _ := os.ReadFile(s.Path())
	if string(after) != string(protected) || string(before) == string(protected) {
		t.Fatal("rejected edits changed durable config")
	}
	_, _, err = s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", false, IndexerNetworkProfilePatch{ClearProxy: true})
	if err != nil || s.Config().Indexers.Builtin.NetworkProfiles["private"].ProxyURL != "" || s.Config().Indexers.Builtin.NetworkProfiles["private"].FlareSolverrURL != profile.FlareSolverrURL {
		t.Fatal("explicit clear changed unrelated fields", err)
	}
	direct := "direct"
	if _, _, err = s.UpdateBuiltinInstance(ctx, id, s.Revision(), BuiltinInstancePatch{NetworkProfile: &direct}); err != nil {
		t.Fatal(err)
	}
	if _, ack, err = s.RemoveIndexerNetworkProfile(ctx, s.Revision(), "private"); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(s.Path())
	if err != nil || reopened.Revision() != ack {
		t.Fatal("profile removal not durable", err)
	}
	if _, exists := reopened.Config().Indexers.Builtin.NetworkProfiles["private"]; exists {
		t.Fatal("removed profile returned after reopen")
	}
}
func TestNetworkTimeoutPatchIsAtomicAndLeavesOtherPoliciesIntact(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	before := s.Config()
	request := 45
	saved, ack, err := s.UpdateIndexerNetworkTimeouts(ctx, s.Revision(), IndexerNetworkTimeoutPatch{RequestTimeoutSeconds: &request})
	if err != nil || ack != s.Revision() || saved.Indexers.Builtin.RequestTimeoutSeconds != 45 || saved.Indexers.Builtin.SiteTimeoutSeconds != before.Indexers.Builtin.SiteTimeoutSeconds || saved.Indexers.SearchCache != before.Indexers.SearchCache {
		t.Fatal("timeout patch overwrote unrelated configuration", err)
	}
	bytes, _ := os.ReadFile(s.Path())
	zero := 0
	if _, _, err = s.UpdateIndexerNetworkTimeouts(ctx, ack, IndexerNetworkTimeoutPatch{RequestTimeoutSeconds: &zero}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatal("zero timeout accepted", err)
	}
	after, _ := os.ReadFile(s.Path())
	if string(bytes) != string(after) {
		t.Fatal("rejected timeout changed durable file")
	}
}

func TestSolverProfileAndChallengeTimeoutAreSparseDurableAndAtomic(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	solver := "http://solver.example:8191/private-service"
	_, ack, err := s.SaveIndexerNetworkProfile(ctx, s.Revision(), "browser", true, IndexerNetworkProfilePatch{FlareSolverrURL: &solver})
	if err != nil || ack != s.Revision() {
		t.Fatal("solver-only profile", err)
	}
	proxy := "http://synthetic-user:synthetic-password@proxy.example:8080"
	blank := ""
	_, ack, err = s.SaveIndexerNetworkProfile(ctx, ack, "browser", false, IndexerNetworkProfilePatch{ProxyURL: &proxy, FlareSolverrURL: &blank})
	if err != nil || s.Config().Indexers.Builtin.NetworkProfiles["browser"].FlareSolverrURL != solver {
		t.Fatal("sparse edit erased solver", err)
	}
	before, _ := os.ReadFile(s.Path())
	for _, patch := range []IndexerNetworkProfilePatch{
		{ClearFlareSolverr: true, FlareSolverrURL: &blank},
		{FlareSolverrURL: func() *string { v := "http://private-user:private-password@solver.example"; return &v }()},
	} {
		if _, _, err = s.SaveIndexerNetworkProfile(ctx, ack, "browser", false, patch); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("invalid solver edit accepted", err)
		}
	}
	for _, value := range []int{0, 9, 301} {
		if _, _, err = s.UpdateIndexerNetworkTimeouts(ctx, ack, IndexerNetworkTimeoutPatch{ChallengeTimeoutSeconds: &value}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("invalid challenge timeout", err)
		}
	}
	after, _ := os.ReadFile(s.Path())
	if string(before) != string(after) {
		t.Fatal("rejected edits changed file")
	}
	timeout := 150
	_, next, err := s.UpdateIndexerNetworkTimeouts(ctx, ack, IndexerNetworkTimeoutPatch{ChallengeTimeoutSeconds: &timeout})
	if err != nil || next != s.Revision() || s.Config().Indexers.Builtin.RequestTimeoutSeconds != 30 || s.Config().Indexers.Builtin.SiteTimeoutSeconds != 120 {
		t.Fatal("challenge timeout overwrote other timeouts", err)
	}
	if _, _, err = s.SaveIndexerNetworkProfile(ctx, ack, "browser", false, IndexerNetworkProfilePatch{ClearFlareSolverr: true}); !errors.Is(err, ErrConfigRevision) {
		t.Fatal("stale clear accepted", err)
	}
	reopened, err := OpenStore(s.Path())
	if err != nil || reopened.Revision() != next || reopened.Config().Indexers.Builtin.ChallengeTimeoutSeconds != timeout || reopened.Config().Indexers.Builtin.NetworkProfiles["browser"].FlareSolverrURL != solver {
		t.Fatal("solver settings did not survive reopen", err)
	}
	_, _, err = reopened.SaveIndexerNetworkProfile(ctx, next, "browser", false, IndexerNetworkProfilePatch{ClearFlareSolverr: true})
	profile := reopened.Config().Indexers.Builtin.NetworkProfiles["browser"]
	if err != nil || profile.FlareSolverrURL != "" || profile.ProxyURL != proxy {
		t.Fatal("clear solver changed proxy", err)
	}
}
