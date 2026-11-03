package indexer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

type resolverFunc func(context.Context, string) ([]net.IPAddr, error)

func (f resolverFunc) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	return f(ctx, host)
}

func testNetwork(t *testing.T, respond func(*http.Request) (*http.Response, error)) *Network {
	t.Helper()
	network := NewNetwork(context.Background())
	t.Cleanup(network.Close)
	network.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("1.1.1.1")}}, nil
	})
	network.roundTrip = respond
	return network
}
func httpResponse(status int, headers http.Header, body string) *http.Response {
	if headers == nil {
		headers = http.Header{}
	}
	return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body))}
}

func TestSessionPinsPublicDNSAndDoesNotLeakCredentialsAcrossRedirects(t *testing.T) {
	var hosts []string
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "1.1.1.1:443" {
			t.Errorf("DNS not pinned: %s", request.URL.Host)
		}
		hosts = append(hosts, request.Host)
		if request.Host == "site.invalid" {
			if request.Header.Get("Authorization") != "Bearer synthetic" || !strings.Contains(request.Header.Get("Cookie"), "session=private") {
				t.Error("site authentication lost")
			}
			return httpResponse(302, http.Header{"Location": []string{"https://cdn.invalid/file"}}, ""), nil
		}
		if request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.Header.Get("X-Api-Key") != "" {
			t.Error("site credentials reached CDN")
		}
		return httpResponse(200, nil, "payload"), nil
	})
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.SetCookieHeader("https://site.invalid/", "session=private"); err != nil {
		t.Fatal(err)
	}
	response, err := session.Do(context.Background(), cardigann.Request{URL: "https://site.invalid/start", FollowRedirect: true, Headers: http.Header{"Authorization": []string{"Bearer synthetic"}, "X-Api-Key": []string{"private"}}})
	if err != nil || response.URL != "https://cdn.invalid/file" || string(response.Body) != "payload" || len(hosts) != 2 {
		t.Fatalf("redirect execution failed: %v", err)
	}
	other, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(other.jar.Cookies(other.base)) != 0 {
		t.Fatal("account sessions share cookies")
	}
}

func TestSessionRejectsPrivateTargetsRebindingAndCrossOriginPost(t *testing.T) {
	var calls atomic.Int32
	network := testNetwork(t, func(*http.Request) (*http.Response, error) { calls.Add(1); return httpResponse(200, nil, "ok"), nil })
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/", TrustedOrigins: []string{"http://127.0.0.1", "https://rebound.invalid"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"http://127.0.0.1/", "http://169.254.169.254/", "http://[::ffff:127.0.0.1]/", "http://100.64.0.1/", "http://[64:ff9b::a9fe:a9fe]/", "http://localhost/", "file:///etc/passwd", "https://user:pass@site.invalid/"} {
		if _, err := session.Do(context.Background(), cardigann.Request{URL: target}); !errors.Is(err, ErrUnsafeURL) {
			t.Errorf("unsafe target accepted %s: %v", target, err)
		}
	}
	if _, err := session.Do(context.Background(), cardigann.Request{URL: "https://other.invalid/", Method: http.MethodPost, Body: []byte("password=private")}); !errors.Is(err, ErrUnsafeURL) {
		t.Fatal("cross-origin credentials could be posted", err)
	}
	network.resolver = resolverFunc(func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("1.1.1.1")}, {IP: net.ParseIP("10.0.0.1")}}, nil
	})
	if _, err := session.Do(context.Background(), cardigann.Request{URL: "https://rebound.invalid/"}); !errors.Is(err, ErrUnsafeURL) {
		t.Fatal("mixed DNS response accepted", err)
	}
	if calls.Load() != 0 {
		t.Fatal("unsafe requests reached socket transport")
	}
}

func TestNetworkCapacityIncludesResponseBodyAndShutdownCancelsRequests(t *testing.T) {
	network := testNetwork(t, func(*http.Request) (*http.Response, error) { return httpResponse(200, nil, "body"), nil })
	session, err := network.NewSession(InstanceSnapshot{BaseURL: "https://site.invalid/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodGet, "https://site.invalid/", nil)
	response, err := session.transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(network.slots) != 1 || len(network.queue) != 1 {
		t.Fatal("request slot released before body consumption")
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if len(network.slots) != 0 || len(network.queue) != 0 {
		t.Fatal("body close leaked request slot")
	}
	started := make(chan struct{})
	network.roundTrip = func(request *http.Request) (*http.Response, error) {
		close(started)
		<-request.Context().Done()
		return nil, request.Context().Err()
	}
	finished := make(chan error, 1)
	go func() {
		_, err := session.Do(context.Background(), cardigann.Request{URL: "https://other.invalid/"})
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	network.Close()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("app cancellation not propagated", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("network shutdown leaked worker")
	}
}

func TestNetworkCooldownSharesOriginAcrossInstances(t *testing.T) {
	var calls atomic.Int32
	network := testNetwork(t, func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return httpResponse(429, http.Header{"Retry-After": []string{"120"}}, "rate limit"), nil
	})
	for _, id := range []string{"one", "two"} {
		session, err := network.NewSession(InstanceSnapshot{ID: id, BaseURL: "https://site.invalid/"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = session.Do(context.Background(), cardigann.Request{URL: "https://site.invalid/"})
		var cooldown *CoolingDownError
		if !errors.As(err, &cooldown) || time.Until(cooldown.Until) < 110*time.Second {
			t.Fatal("cooldown lost", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("cooldown did not stop second account request")
	}
}

func TestGlobalNetworkAdmissionIsBounded(t *testing.T) {
	gate := make(chan struct{})
	var active, maximum, busy atomic.Int32
	network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
		count := active.Add(1)
		for old := maximum.Load(); count > old; old = maximum.Load() {
			if maximum.CompareAndSwap(old, count) {
				break
			}
		}
		select {
		case <-gate:
		case <-request.Context().Done():
			active.Add(-1)
			return nil, request.Context().Err()
		}
		active.Add(-1)
		return httpResponse(200, nil, "ok"), nil
	})
	var wg sync.WaitGroup
	ready := make(chan struct{})
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-ready
			instance := InstanceSnapshot{BaseURL: "https://site" + strconv.Itoa(id) + ".invalid/"}
			session, err := network.NewSession(instance, nil)
			if err != nil {
				t.Error(err)
				return
			}
			_, err = session.Do(context.Background(), cardigann.Request{URL: instance.BaseURL})
			if errors.Is(err, ErrBusy) {
				busy.Add(1)
			} else if err != nil {
				t.Error(err)
			}
		}(i)
	}
	close(ready)
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for busy.Load() < 32 {
		select {
		case <-deadline.C:
			t.Fatal("queue failed to reject overload")
		case <-time.After(time.Millisecond):
		}
	}
	if maximum.Load() > 4 || len(network.queue) > 68 {
		t.Fatal("global request bounds exceeded")
	}
	close(gate)
	wg.Wait()
}

func TestTrustBoundaryAppliesToDomainCookiesAndResponseCookieAdoption(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(strconv.FormatBool(trusted), func(t *testing.T) {
			network := testNetwork(t, func(r *http.Request) (*http.Response, error) {
				if r.Host == "site.invalid" {
					return httpResponse(302, http.Header{"Location": []string{"https://cdn.site.invalid/file"}, "Set-Cookie": []string{"account=private; Domain=site.invalid; Path=/; Secure"}}, ""), nil
				}
				hasAccount := strings.Contains(r.Header.Get("Cookie"), "account=private")
				if r.URL.Path == "/direct" && (r.Header.Get("Authorization") != "") != trusted {
					t.Error("direct request credential boundary was bypassed")
				}
				if hasAccount != trusted {
					t.Errorf("domain cookie crossed unexpected trust boundary: trusted=%v cookiePresent=%v", trusted, hasAccount)
				}
				return httpResponse(200, http.Header{"Set-Cookie": []string{"account=replaced; Domain=site.invalid; Path=/; Secure"}}, "ok"), nil
			})
			instance := InstanceSnapshot{BaseURL: "https://site.invalid"}
			if trusted {
				instance.TrustedOrigins = []string{"https://cdn.site.invalid"}
			}
			session, err := network.NewSession(instance, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if _, err = session.Do(context.Background(), cardigann.Request{URL: "https://site.invalid/start", FollowRedirect: true}); err != nil {
				t.Fatal(err)
			}
			if _, err = session.Do(context.Background(), cardigann.Request{URL: "https://cdn.site.invalid/direct", Headers: http.Header{"Cookie": []string{"account=private"}, "Authorization": []string{"Bearer synthetic"}}}); err != nil {
				t.Fatal(err)
			}
			cookies := session.jar.Cookies(session.base)
			want := "private"
			if trusted {
				want = "replaced"
			}
			if len(cookies) != 1 || cookies[0].Value != want {
				t.Fatal("response changed cookies outside its trust boundary")
			}
		})
	}
}
