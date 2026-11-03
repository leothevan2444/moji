package indexer

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/storage"
)

func persistentTestNetwork(t *testing.T, manager *storage.Manager, respond func(*http.Request) (*http.Response, error), warn func(error)) *Network {
	t.Helper()
	n, err := NewPersistentNetwork(context.Background(), manager, warn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(n.Close)
	n.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("1.1.1.1")}}, nil
	})
	n.roundTrip = respond
	return n
}
func cooldownSession(t *testing.T, n *Network, base, proxy string) *HTTPSession {
	t.Helper()
	s, e := n.NewSession(InstanceSnapshot{BaseURL: base, Network: NetworkProfile{ProxyURL: proxy}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s
}
func TestPersistentCooldownSurvivesReopenAndSharesOnlyItsOriginEgress(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(dir))
	if e != nil {
		t.Fatal(e)
	}
	calls := 0
	n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(429, http.Header{"Retry-After": []string{"120"}}, "limited"), nil
	}, nil)
	s := cooldownSession(t, n, "https://site.invalid", "socks5://synthetic-user:synthetic-password@proxy.invalid:1080")
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); !errors.Is(e, ErrCoolingDown) || calls != 1 {
		t.Fatal("initial rate limit", e, calls)
	}
	s.Close()
	n.Close()
	if e = m.Close(); e != nil {
		t.Fatal(e)
	}
	m, e = storage.OpenManager(ctx, storage.DefaultManagerOptions(dir))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	n = persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { calls++; return httpResponse(200, nil, "ok"), nil }, nil)
	// A different account on the same gateway cannot bypass an egress limit.
	s = cooldownSession(t, n, "https://site.invalid", "socks5://another-account:other-password@proxy.invalid:1080")
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); !errors.Is(e, ErrCoolingDown) || calls != 1 {
		t.Fatal("restart/account change bypassed shared cooldown", e, calls)
	}
	for _, pair := range [][2]string{{"https://other.invalid", "socks5://proxy.invalid:1080"}, {"https://site.invalid", ""}, {"https://site.invalid", "socks5://second.invalid:1080"}} {
		other := cooldownSession(t, n, pair[0], pair[1])
		if _, e = other.Do(ctx, cardigann.Request{URL: pair[0] + "/search"}); e != nil {
			t.Fatal("unrelated origin or egress inherited cooldown", e)
		}
	}
	var keys []string
	if e = m.Runtime.Reader.Select(&keys, `SELECT scope_key FROM indexer_network_cooldowns`); e != nil || len(keys) != 1 || len(keys[0]) != 64 || strings.Contains(keys[0], "synthetic") {
		t.Fatal("cooldown leaked identity or wrong count", e, keys)
	}
	n.Close()
}
func TestPersistentCooldownRetainsExponentThenSuccessClearsIt(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	key := cooldownKey("https://site.invalid:443|direct")
	// Simulate a previous process whose first 30s wait has elapsed.
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO indexer_network_cooldowns VALUES(?,?,?,?)`, key, 1, time.Now().Add(-time.Second).UnixMilli(), time.Now().Add(time.Hour).UnixMilli())
		return err
	}); e != nil {
		t.Fatal(e)
	}
	n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { return httpResponse(429, nil, "unavailable"), nil }, nil)
	s := cooldownSession(t, n, "https://site.invalid", "")
	_, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"})
	var cooling *CoolingDownError
	if !errors.As(e, &cooling) || time.Until(cooling.Until) < 59*time.Second {
		t.Fatal("reported cooldown ignored restored exponent", e)
	}
	var row persistedCooldown
	if e = m.Runtime.Reader.Get(&row, `SELECT failures,blocked_until_ms,expires_at_ms FROM indexer_network_cooldowns WHERE scope_key=?`, key); e != nil || row.Failures != 2 || time.Until(time.UnixMilli(row.Blocked)) < 59*time.Second {
		t.Fatal("restart reset failure exponent", e, row)
	}
	s.Close()
	n.Close()
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE indexer_network_cooldowns SET blocked_until_ms=?`, time.Now().Add(-time.Second).UnixMilli())
		return err
	}); e != nil {
		t.Fatal(e)
	}
	n = persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { return httpResponse(200, nil, "ok"), nil }, nil)
	s = cooldownSession(t, n, "https://site.invalid", "")
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = m.Runtime.Reader.Get(&count, `SELECT count(*) FROM indexer_network_cooldowns`); e != nil || count != 0 {
		t.Fatal("success did not clear persisted cooldown", e, count)
	}
	n.Close()
}
func TestUnavailableCooldownReadStopsBeforeHTTPAndRecovers(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	calls, warnings := 0, 0
	n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { calls++; return httpResponse(200, nil, "ok"), nil }, func(e error) {
		warnings++
		if !errors.Is(e, ErrStateUnavailable) {
			t.Error("unsafe persistence error projection")
		}
	})
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`ALTER TABLE indexer_network_cooldowns RENAME TO temporarily_unavailable`)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	s := cooldownSession(t, n, "https://site.invalid", "")
	for i := 0; i < 2; i++ {
		if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); !errors.Is(e, ErrStateUnavailable) {
			t.Fatal("storage failure not distinct", e)
		}
	}
	if calls != 0 || warnings != 1 {
		t.Fatal("unavailable store allowed requests or flooded warnings", calls, warnings)
	}
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`ALTER TABLE temporarily_unavailable RENAME TO indexer_network_cooldowns`)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); e != nil || calls != 1 {
		t.Fatal("storage recovery did not resume admission", e, calls)
	}
	n.Close()
}

func TestFailedCooldownWriteRetainsResultAndStopsReplayUntilStorageRecovers(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	key := cooldownKey("https://site.invalid:443|direct")
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO indexer_network_cooldowns VALUES(?,?,?,?)`, key, 1, time.Now().Add(-time.Second).UnixMilli(), time.Now().Add(time.Hour).UnixMilli())
		return err
	}); e != nil {
		t.Fatal(e)
	}
	calls, warnings := 0, 0
	n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if e := m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
				_, err := tx.Exec(`ALTER TABLE indexer_network_cooldowns RENAME TO unavailable_write`)
				return err
			}); e != nil {
				t.Fatal(e)
			}
		}
		return httpResponse(200, nil, "received-resource"), nil
	}, func(error) { warnings++ })
	s := cooldownSession(t, n, "https://site.invalid", "")
	out, e := s.Do(ctx, cardigann.Request{URL: "https://site.invalid/download", Torrent: true})
	if e != nil || string(out.Body) != "received-resource" || warnings != 1 {
		t.Fatal("persistence error discarded received resource", e, warnings)
	}
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/download", Torrent: true}); !errors.Is(e, ErrStateUnavailable) || calls != 1 {
		t.Fatal("dirty state allowed repeated acquisition", e, calls)
	}
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`ALTER TABLE unavailable_write RENAME TO indexer_network_cooldowns`)
		return err
	}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/next"}); e != nil || calls != 2 {
		t.Fatal("recovered dirty state did not resume", e, calls)
	}
	n.Close()
}
func TestSolvedChallengeRemovesOnlyItsPersistedPenalty(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	siteCalls := 0
	n := persistentTestNetwork(t, m, func(r *http.Request) (*http.Response, error) {
		if r.Host == "solver.internal" {
			return httpResponse(200, nil, solverReply(t, "https://site.invalid/search", "site.invalid")), nil
		}
		siteCalls++
		if siteCalls == 1 {
			return challengeResponse(), nil
		}
		return httpResponse(200, nil, "ok"), nil
	}, nil)
	s, e := n.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.doScoped(ctx, cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = m.Runtime.Reader.Get(&count, `SELECT count(*) FROM indexer_network_cooldowns`); e != nil || count != 0 {
		t.Fatal("solved challenge left persisted cooldown", e, count)
	}
	s.Close()
	n.Close()
	n = persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { siteCalls++; return httpResponse(200, nil, "ok"), nil }, nil)
	s = cooldownSession(t, n, "https://site.invalid", "")
	if _, e = s.Do(ctx, cardigann.Request{URL: "https://site.invalid/search"}); e != nil {
		t.Fatal("solved challenge cooldown returned after restart", e)
	}
	n.Close()
}
func TestCooldownCapacityNeverEvictsActivePenalties(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	until := time.Now().Add(time.Hour)
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`WITH RECURSIVE sequence(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM sequence WHERE n<8192) INSERT INTO indexer_network_cooldowns SELECT printf('%064x',n),1,?,? FROM sequence`, until.UnixMilli(), until.Add(time.Hour).UnixMilli())
		return err
	}); e != nil {
		t.Fatal(e)
	}
	repo := cooldownStore{db: m.Runtime}
	gate := &originGate{failures: 1, blocked: until}
	if e = repo.save(ctx, cooldownKey("new-origin"), gate); !errors.Is(e, ErrBusy) {
		t.Fatal("active row budget bypassed", e)
	}
	var count int
	if e = m.Runtime.Reader.Get(&count, `SELECT count(*) FROM indexer_network_cooldowns`); e != nil || count != cooldownRows {
		t.Fatal("active penalty evicted", e, count)
	}
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`UPDATE indexer_network_cooldowns SET expires_at_ms=? WHERE scope_key=printf('%064x',1)`, time.Now().Add(-time.Second).UnixMilli())
		return err
	}); e != nil {
		t.Fatal(e)
	}
	if e = repo.save(ctx, cooldownKey("new-origin"), gate); e != nil {
		t.Fatal("expired row was not reclaimed", e)
	}
}

func TestJackettDoesNotMisclassifyUnavailableCooldownStore(t *testing.T) {
	ctx := context.Background()
	m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if e != nil {
		t.Fatal(e)
	}
	defer m.Close()
	calls := 0
	n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) { calls++; return httpResponse(200, nil, "ok"), nil }, nil)
	if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error { _, err := tx.Exec(`DROP TABLE indexer_network_cooldowns`); return err }); e != nil {
		t.Fatal(e)
	}
	adapter, e := NewJackettAdapter(n)
	if e != nil {
		t.Fatal(e)
	}
	_, e = adapter.Catalog(ctx, InstanceSnapshot{ID: "jackett", BaseURL: "http://jackett.invalid", JackettAPIKey: "synthetic"})
	if !errors.Is(e, ErrStateUnavailable) || calls != 0 {
		t.Fatal("external indexer storage failure lost its boundary", e, calls)
	}
	n.Close()
}

func TestJackettFirstRateLimitReportsActualRestoredDeadlineForSearchAndCatalog(t *testing.T) {
	for _, op := range []string{"search", "catalog"} {
		t.Run(op, func(t *testing.T) {
			ctx := context.Background()
			m, e := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
			if e != nil {
				t.Fatal(e)
			}
			defer m.Close()
			key := cooldownKey("http://jackett.invalid:80|direct")
			if e = m.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
				_, err := tx.Exec(`INSERT INTO indexer_network_cooldowns VALUES(?,?,?,?)`, key, 1, time.Now().Add(-time.Second).UnixMilli(), time.Now().Add(time.Hour).UnixMilli())
				return err
			}); e != nil {
				t.Fatal(e)
			}
			calls := 0
			n := persistentTestNetwork(t, m, func(*http.Request) (*http.Response, error) {
				calls++
				return httpResponse(429, http.Header{"Retry-After": []string{"5"}}, "limited"), nil
			}, nil)
			defer n.Close()
			adapter, e := NewJackettAdapter(n)
			if e != nil {
				t.Fatal(e)
			}
			defer adapter.Close()
			instance := InstanceSnapshot{ID: "jackett/one", Kind: Jackett, RemoteID: "one", BaseURL: "http://jackett.invalid", JackettAPIKey: "synthetic"}
			run := func() error {
				if op == "search" {
					_, err := adapter.Search(ctx, instance, Query{Keywords: "ABCD-123"})
					return err
				}
				_, err := adapter.Catalog(ctx, instance)
				return err
			}
			e = run()
			var cooling *CoolingDownError
			if !errors.As(e, &cooling) || time.Until(cooling.Until) < 59*time.Second || calls != 1 {
				t.Fatal("first upstream rate limit did not use actual gate deadline", e, calls)
			}
			first := cooling.Until
			if e = run(); !errors.As(e, &cooling) || !cooling.Until.Equal(first) || calls != 1 {
				t.Fatal("admission deadline differs from first reported deadline or called upstream again", e, calls)
			}
		})
	}
}
