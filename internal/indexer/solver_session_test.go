package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
	"github.com/leothevan2444/moji/pkg/flaresolverr"
)

func challengeResponse() *http.Response {
	return httpResponse(503, http.Header{"Server": []string{"cloudflare"}, "Cf-Mitigated": []string{"challenge"}}, `<title>Just a moment...</title>`)
}
func solverReply(t *testing.T, target, domain string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"status": "ok", "solution": flaresolverr.Solution{URL: target, Status: 200, UserAgent: "Synthetic solved browser", Cookies: []flaresolverr.Cookie{{Name: "cf_clearance", Value: "private-clearance", Domain: domain, Path: "/", Secure: true}}}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func TestSolverSearchBindsProxyCookiesAndUserAgentAndClearsOnlyItsOwn503(t *testing.T) {
	siteCalls, solverCalls := 0, 0
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.Host == "solver.internal:8191" {
			solverCalls++
			data, _ := io.ReadAll(r.Body)
			var request struct {
				Cmd, URL string
				Cookies  []flaresolverr.Cookie
				Proxy    *flaresolverr.Proxy
			}
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}
			if request.Cmd != "request.get" || request.URL != "https://site.invalid/search" || request.Proxy == nil || request.Proxy.URL != "socks5://proxy.example:1080" || request.Proxy.Username != "user" || request.Proxy.Password != "private-proxy" {
				t.Fatal("solver lost frozen egress or request scope")
			}
			found := false
			for _, cookie := range request.Cookies {
				if cookie.Name == "account" && cookie.Value == "private-account" {
					found = true
				}
			}
			if !found || r.Header.Get("Cookie") != "" {
				t.Fatal("account cookie missing from protocol or sent as service authentication")
			}
			return httpResponse(200, nil, solverReply(t, "https://site.invalid/search", "site.invalid")), nil
		}
		siteCalls++
		if siteCalls == 1 {
			return challengeResponse(), nil
		}
		if r.Header.Get("User-Agent") != "Synthetic solved browser" || !strings.Contains(r.Header.Get("Cookie"), "cf_clearance=private-clearance") {
			t.Fatal("clearance cookie and browser UA were not paired")
		}
		return httpResponse(200, nil, "search-result"), nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", ChallengeTimeout: time.Second, Network: NetworkProfile{ProxyURL: "socks5h://user:private-proxy@proxy.example:1080", FlareSolverrURL: "http://solver.internal:8191"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	out, err := session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/search", Headers: http.Header{"User-Agent": []string{"rule-agent"}, "Cookie": []string{"account=private-account"}}}, purposeSearch)
	if err != nil || out.Status != 200 || string(out.Body) != "search-result" || siteCalls != 2 || solverCalls != 1 {
		t.Fatal("search did not verify its original request after challenge", err, siteCalls, solverCalls)
	}
	if _, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/next"}, purposeSearch); err != nil {
		t.Fatal("own solved CF 503 left a false cooldown", err)
	}
}
func TestSolverNeverReplaysAcquisitionEvenForPlainHTMLBeforeTorrent(t *testing.T) {
	for _, torrent := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-page", true: "torrent"}[torrent], func(t *testing.T) {
			calls := 0
			network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Host != "site.invalid" {
					t.Fatal("acquisition delegated to solver")
				}
				return challengeResponse(), nil
			})
			session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if _, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/grab", Torrent: torrent}, purposeFetch); !errors.Is(err, ErrChallenge) || calls != 1 {
				t.Fatal("quota-bearing acquisition replayed", err, calls)
			}
		})
	}
}
func TestUnconfiguredChallengeAndOrdinaryFailureRemainDistinct(t *testing.T) {
	for _, protected := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary-403", true: "challenge"}[protected], func(t *testing.T) {
			calls := 0
			network := testNetwork(t, func(*http.Request) (*http.Response, error) {
				calls++
				if protected {
					return challengeResponse(), nil
				}
				return httpResponse(403, http.Header{"Server": []string{"cloudflare"}}, "Access denied"), nil
			})
			session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			_, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch)
			if protected && !errors.Is(err, ErrChallenge) || !protected && err != nil || calls != 1 {
				t.Fatal("ordinary error and challenge confused", err, calls)
			}
		})
	}
}
func TestSolverRejectsForeignReplyWithoutAdoptingCookiesOrReplaying(t *testing.T) {
	for _, foreign := range []string{"url", "cookie", "public-suffix"} {
		t.Run(foreign, func(t *testing.T) {
			siteCalls := 0
			network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
				if r.Host == "site.invalid" {
					siteCalls++
					return challengeResponse(), nil
				}
				target, domain := "https://site.invalid/search", "site.invalid"
				if foreign == "url" {
					target = "https://other.invalid/search"
				}
				if foreign == "cookie" {
					domain = "other.invalid"
				}
				if foreign == "public-suffix" {
					domain = "invalid"
				}
				return httpResponse(200, nil, solverReply(t, target, domain)), nil
			})
			session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if _, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch); !errors.Is(err, ErrUnsafeURL) || siteCalls != 1 || len(session.jar.Cookies(session.base)) != 0 || session.solverUserAgent {
				t.Fatal("foreign solver reply changed account or replayed", err)
			}
		})
	}
}
func TestSolverStopsAfterOneSolveAndOneOriginalRequestVerification(t *testing.T) {
	siteCalls, solverCalls := 0, 0
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.Host == "site.invalid" {
			siteCalls++
			return challengeResponse(), nil
		}
		solverCalls++
		return httpResponse(200, nil, solverReply(t, "https://site.invalid/search", "site.invalid")), nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch); !errors.Is(err, ErrChallenge) || siteCalls != 2 || solverCalls != 1 {
		t.Fatal("unsolved response caused a solver loop or empty success", err, siteCalls, solverCalls)
	}
}
func TestSolverCannotClearNewerRateLimit(t *testing.T) {
	gate := &originGate{gate: make(chan struct{}, 1), failureSequence: 2, failures: 2, blocked: time.Now().Add(time.Hour)}
	session := &HTTPSession{}
	if err := session.clearChallengePenalty(context.Background(), &responseReceipt{gate: gate, sequence: 1, status: 503}); err != nil {
		t.Fatal(err)
	}
	if gate.blocked.IsZero() || gate.failures != 2 {
		t.Fatal("an older challenge cleared a newer 429")
	}
	if err := session.clearChallengePenalty(context.Background(), &responseReceipt{gate: gate, sequence: 2, status: 429}); err != nil {
		t.Fatal(err)
	}
	if gate.blocked.IsZero() {
		t.Fatal("rate-limit receipt was cleared as challenge")
	}
}
func TestSolverHonorsRemainingParentBudgetWithoutCookieAdoption(t *testing.T) {
	var deadline time.Time
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.Host == "site.invalid" {
			return challengeResponse(), nil
		}
		actual, _ := r.Context().Deadline()
		if actual.Before(deadline.Add(-2 * time.Millisecond)) {
			t.Error("solver inherited the short raw HTTP timeout")
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", RequestTimeout: 10 * time.Millisecond, ChallengeTimeout: time.Minute, Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	deadline, _ = ctx.Deadline()
	defer cancel()
	if _, err = session.doScoped(ctx, cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch); !errors.Is(err, context.DeadlineExceeded) || session.solverUserAgent || len(session.jar.Cookies(session.base)) != 0 {
		t.Fatal("solver escaped parent budget or imported cancelled result", err)
	}
}

func TestSolverAuthenticationVerifiesOriginalPostAndRestoresPairedSession(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	logins, solves, searches := 0, 0, 0
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.Host == "solver.internal:8191" {
			solves++
			data, _ := io.ReadAll(r.Body)
			var request struct {
				Cmd, URL string
				PostData *string
			}
			if err := json.Unmarshal(data, &request); err != nil {
				t.Fatal(err)
			}
			if request.Cmd != "request.get" || request.PostData != nil || request.URL != "https://site.invalid/login" || strings.Contains(string(data), "synthetic-password") {
				t.Fatal("browser received auth form credentials or replayed the post")
			}
			return httpResponse(200, nil, solverReply(t, "https://site.invalid/login", "site.invalid")), nil
		}
		switch r.URL.Path {
		case "/login":
			logins++
			if !strings.Contains(r.Header.Get("Cookie"), "cf_clearance=private-clearance") {
				return challengeResponse(), nil
			}
			data, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || !strings.Contains(string(data), "password=synthetic-password") || r.Header.Get("User-Agent") != "Synthetic solved browser" {
				t.Fatal("original login post not verified with paired clearance")
			}
			return httpResponse(200, http.Header{"Set-Cookie": []string{"account=derived; Path=/; Secure; HttpOnly"}}, "ok"), nil
		case "/account", "/search":
			if !strings.Contains(r.Header.Get("Cookie"), "account=derived") || !strings.Contains(r.Header.Get("Cookie"), "cf_clearance=private-clearance") || r.Header.Get("User-Agent") != "Synthetic solved browser" {
				t.Fatal("restored account changed cookie/UA pairing")
			}
			if r.URL.Path == "/search" {
				searches++
			}
			return httpResponse(200, nil, `<div id="member"></div><article><h2>ABCD-123</h2><a href="/download">get</a></article>`), nil
		}
		t.Fatal("unexpected endpoint")
		return nil, nil
	})
	definition := strings.Replace(siteFixture, `search:
`, `search:
  headers: {User-Agent: [rule-agent]}
`, 1)
	plan, err := cardigann.Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	instance := fixtureInstance()
	instance.BaseURL = ""
	instance.Network.FlareSolverrURL = "http://solver.internal:8191"
	instance.ChallengeTimeout = 10 * time.Second
	bundles := fixedBundles{bundle: &indexerdefs.Bundle{Manifest: indexerdefs.Manifest{Revision: strings.Repeat("a", 40)}, Plans: map[string]*cardigann.Plan{"example": plan}}}
	sessions, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewBuiltinAdapter(bundles, sessions)
	if err != nil {
		t.Fatal(err)
	}
	if err = adapter.Authenticate(ctx, instance); err != nil {
		t.Fatal(err)
	}
	sessions.Close()
	var data string
	if err = manager.Runtime.Reader.Get(&data, "SELECT payload_json FROM indexer_sessions"); err != nil {
		t.Fatal(err)
	}
	var payload sessionPayload
	if err = json.Unmarshal([]byte(data), &payload); err != nil || payload.UserAgent != "Synthetic solved browser" {
		t.Fatal("clearance UA did not persist", err)
	}
	restored, err := NewSessions(network, manager)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	adapter, err = NewBuiltinAdapter(bundles, restored)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := adapter.Search(ctx, instance, Query{Keywords: "ABCD-123", Limit: 20})
	batch.Close()
	if err != nil || len(batch.Releases) != 1 || logins != 2 || solves != 1 || searches != 1 {
		t.Fatal("restored search reauthenticated or lost solved session", err, logins, solves, searches)
	}
}

func TestSolverDoesNotDelegateRateLimitedChallenge(t *testing.T) {
	calls := 0
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Host != "site.invalid" {
			t.Fatal("rate limit delegated to solver")
		}
		return httpResponse(429, http.Header{"Cf-Mitigated": []string{"challenge"}, "Retry-After": []string{"120"}}, "challenge"), nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid", Network: NetworkProfile{FlareSolverrURL: "http://solver.internal"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for i := 0; i < 2; i++ {
		if _, err = session.doScoped(context.Background(), cardigann.Request{URL: "https://site.invalid/search"}, purposeSearch); !errors.Is(err, ErrCoolingDown) {
			t.Fatal("rate limit lost", err)
		}
	}
	if calls != 1 {
		t.Fatal("rate limit was replayed", calls)
	}
}

func TestSolverCookieBatchCannotPartiallyChangeAccount(t *testing.T) {
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("cookie validation made HTTP request")
		return nil, nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err = session.SetCookieHeader("https://site.invalid", "account=original"); err != nil {
		t.Fatal(err)
	}
	solution := flaresolverr.Solution{UserAgent: "Synthetic browser", Cookies: []flaresolverr.Cookie{
		{Name: "account", Value: "changed", Domain: "site.invalid", Path: "/"},
		{Name: "foreign", Value: "bad", Domain: "other.invalid", Path: "/"},
	}}
	if err = session.adoptSolution(session.base, solution); !errors.Is(err, ErrUnsafeURL) {
		t.Fatal("invalid batch accepted", err)
	}
	cookies := session.jar.Cookies(session.base)
	if len(cookies) != 1 || cookies[0].Name != "account" || cookies[0].Value != "original" || session.solverUserAgent {
		t.Fatal("invalid batch partially changed account")
	}
}

func TestSolverFailureSurvivesAuthenticationBoundary(t *testing.T) {
	network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
		if r.Host == "site.invalid" {
			return challengeResponse(), nil
		}
		return httpResponse(500, nil, `{"status":"error","message":"private-upstream-details"}`), nil
	})
	sessions, err := NewTransientSessions(network)
	if err != nil {
		t.Fatal(err)
	}
	defer sessions.Close()
	plan, err := cardigann.Load([]byte(siteFixture))
	if err != nil {
		t.Fatal(err)
	}
	instance := fixtureInstance()
	instance.Network.FlareSolverrURL = "http://solver.internal"
	err = sessions.With(context.Background(), instance, plan, true, func(*cardigann.Execution) error { t.Fatal("operation ran after failed login"); return nil })
	if !errors.Is(err, ErrSolver) || ErrorCode(err) != "INDEXER_SOLVER_FAILED" || strings.Contains(err.Error(), "private-upstream-details") {
		t.Fatal("solver failure lost at login boundary", err)
	}
}
