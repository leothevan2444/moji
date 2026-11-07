package config

import (
	"context"
	"errors"
	"testing"
)

func TestGlobalNetworkEditsPreserveSecretsAndRejectStaleOrInvalidWrites(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	enabled := true
	endpoint := "http://proxy.example:8080"
	user := "synthetic-user"
	password := "synthetic-secret"
	solver := "http://solver.example:8191"
	maxTimeout := 45000
	_, ack, err := s.UpdateIndexerNetwork(ctx, s.Revision(), IndexerNetworkPatch{ProxyEnabled: &enabled, ProxyURL: &endpoint, ProxyUsername: &user, ProxyPassword: &password, FlareSolverrEnabled: &enabled, FlareSolverrURL: &solver, FlareSolverrMaxTimeoutMS: &maxTimeout})
	if err != nil || ack != s.Revision() {
		t.Fatal("save", err)
	}
	invalid, replacement := 0, "replacement-secret"
	originalRevision := s.Revision()
	if _, _, err = s.UpdateIndexerNetwork(ctx, originalRevision, IndexerNetworkPatch{ProxyPassword: &replacement, RequestTimeoutSeconds: &invalid}); !errors.Is(err, ErrInvalidSettings) || s.Revision() != originalRevision || s.Config().Indexers.Builtin.Network.Proxy.Password != password {
		t.Fatal("invalid mixed patch partially changed credentials", err)
	}
	for _, invalidMax := range []int{0, 999, 300001} {
		if _, _, err = s.UpdateIndexerNetwork(ctx, s.Revision(), IndexerNetworkPatch{FlareSolverrMaxTimeoutMS: &invalidMax}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("invalid solver timeout accepted", invalidMax, err)
		}
	}
	frozen := s.Config().Indexers.Builtin.Network
	disabled := false
	blank := ""
	if _, _, err = s.UpdateIndexerNetwork(ctx, ack, IndexerNetworkPatch{ProxyEnabled: &disabled, FlareSolverrEnabled: &disabled, ProxyPassword: &blank}); err != nil {
		t.Fatal(err)
	}
	next := s.Config().Indexers.Builtin.Network
	if next.Proxy.Password != password || next.Proxy.URL != endpoint || next.FlareSolverr.URL != solver || next.ProxyURL() != "" || next.SolverURL() != "" || frozen.ProxyURL() == "" {
		t.Fatal("disable erased saved settings or changed old work")
	}
	if _, _, err = s.UpdateIndexerNetwork(ctx, ack, IndexerNetworkPatch{ProxyPassword: &password}); !errors.Is(err, ErrConfigRevision) {
		t.Fatal("stale write accepted", err)
	}
	for _, raw := range []string{"ftp://proxy.example", "http://user:secret@proxy.example", "http://proxy.example/path"} {
		if _, _, err = s.UpdateIndexerNetwork(ctx, s.Revision(), IndexerNetworkPatch{ProxyURL: &raw}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("invalid endpoint accepted", err)
		}
	}
	if _, _, err = s.UpdateIndexerNetwork(ctx, s.Revision(), IndexerNetworkPatch{ClearProxyPassword: true, ProxyPassword: &blank}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatal("ambiguous clear", err)
	}
	_, ack, err = s.UpdateIndexerNetwork(ctx, s.Revision(), IndexerNetworkPatch{ClearProxyPassword: true})
	if err != nil || s.Config().Indexers.Builtin.Network.Proxy.Password != "" {
		t.Fatal("clear", err)
	}
	reopened, err := OpenStore(s.path)
	if err != nil || reopened.Revision() != ack || reopened.Config().Indexers.Builtin.Network.Proxy.Password != "" || reopened.Config().Indexers.Builtin.Network.FlareSolverr.MaxTimeoutMS != 45000 {
		t.Fatal("reopen", err)
	}
	timeout := 180
	if _, _, err = s.UpdateIndexerNetwork(ctx, ack, IndexerNetworkPatch{SiteTimeoutSeconds: &timeout}); err != nil || s.Config().Indexers.Builtin.Network.Proxy.URL != endpoint {
		t.Fatal("timeout overwrote network", err)
	}
}
