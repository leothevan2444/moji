package flaresolverr

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func success() []byte {
	return []byte(`{"status":"ok","solution":{"url":"https://site.example/search","status":200,"headers":{"content-type":"text/html"},"response":"<html>result</html>","cookies":[{"name":"cf_clearance","value":"private-clearance","domain":"site.example","path":"/","secure":true}],"userAgent":"Synthetic browser"}}`)
}
func decode(t *testing.T, body []byte) command {
	t.Helper()
	var c command
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	return c
}
func TestRequestUsesTemporaryBrowserAndReturnsCookieAndUserAgentWithoutRetry(t *testing.T) {
	calls := 0
	client, _ := New(CallFunc(func(ctx context.Context, body []byte) (int, []byte, error) {
		calls++
		c := decode(t, body)
		if c.Cmd != "request.get" || c.URL != "https://site.example/search" || c.MaxTimeout <= 0 || c.MaxTimeout > 90000 || len(c.Cookies) != 1 || c.Cookies[0].Name != "account" {
			t.Fatal("request scope lost", c.Cmd)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("no request deadline")
		}
		return 200, success(), nil
	}))
	solution, err := client.Solve(context.Background(), Request{URL: "https://site.example/search", Timeout: 90 * time.Second, Cookies: []Cookie{{Name: "account", Value: "private-account"}}})
	if err != nil || calls != 1 || solution.UserAgent != "Synthetic browser" || solution.Cookies[0].Value != "private-clearance" {
		t.Fatal("protocol solution lost", err, calls)
	}
}
func TestProtocolErrorsAreBoundedAndNeverExposeUpstreamMessage(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		status int
		body   []byte
		want   error
	}{
		{"rejected", 500, []byte(`{"status":"error","message":"private-secret-url"}`), ErrRejected},
		{"html", 503, []byte("private-error-page"), ErrProtocol},
		{"invalid-json", 200, []byte("<html>private</html>"), ErrProtocol},
		{"no-solution", 200, []byte(`{"status":"ok"}`), ErrProtocol},
		{"oversized", 200, []byte(strings.Repeat("x", MaxResponseBytes+1)), ErrBudget},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			calls := 0
			client, _ := New(CallFunc(func(context.Context, []byte) (int, []byte, error) {
				calls++
				return scenario.status, scenario.body, nil
			}))
			_, err := client.Solve(context.Background(), Request{URL: "https://site.example/", Timeout: time.Second})
			if !errors.Is(err, scenario.want) || calls != 1 || strings.Contains(err.Error(), "private") {
				t.Fatal(err, calls)
			}
		})
	}
}
func TestInvalidRequestDoesNotCallEndpoint(t *testing.T) {
	for _, request := range []Request{{URL: "file:///etc/passwd", Timeout: time.Second}, {URL: "https://user:secret@site.example/", Timeout: time.Second}, {URL: "https://site.example/", Method: "DELETE", Timeout: time.Second}, {URL: "https://site.example/", Timeout: 301 * time.Second}, {URL: "https://site.example/", PostData: "unexpected", Timeout: time.Second}, {URL: "https://site.example/", Timeout: time.Second, Proxy: &Proxy{URL: "http://user:secret@proxy.example"}}} {
		client, _ := New(CallFunc(func(context.Context, []byte) (int, []byte, error) {
			t.Fatal("invalid request reached endpoint")
			return 0, nil, nil
		}))
		if _, err := client.Solve(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatal(err)
		}
	}
}
func TestCanceledCallerCannotReturnLateSuccessfulResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client, _ := New(CallFunc(func(context.Context, []byte) (int, []byte, error) { cancel(); return 200, success(), nil }))
	solution, err := client.Solve(ctx, Request{URL: "https://site.example/", Timeout: time.Second})
	if !errors.Is(err, context.Canceled) || solution.UserAgent != "" {
		t.Fatal("cancelled request returned a late solution", err)
	}
}
func TestUnconfiguredClientRejectsWithoutPanic(t *testing.T) {
	for _, client := range []*Client{nil, {}} {
		if _, err := client.Solve(context.Background(), Request{}); !errors.Is(err, ErrInvalidRequest) {
			t.Fatal(err)
		}
	}
}
func TestEndpointSerializesEntireSolveAndWaitingConsumesCallerBudget(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	calls := make(chan struct{}, 2)
	client, _ := New(CallFunc(func(context.Context, []byte) (int, []byte, error) {
		calls <- struct{}{}
		close(entered)
		<-release
		return 200, success(), nil
	}))
	done := make(chan error, 1)
	go func() {
		_, err := client.Solve(context.Background(), Request{URL: "https://one.example/", Timeout: time.Second})
		done <- err
	}()
	<-entered
	if _, err := client.Solve(context.Background(), Request{URL: "https://two.example/", Timeout: 5 * time.Millisecond}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("waiting solve escaped its budget", err)
	}
	if len(calls) != 1 {
		t.Fatal("independent browser created before previous solve cleanup")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticatedProxyUsesOneTemporaryPostWithoutPersistentSession(t *testing.T) {
	calls := 0
	client, _ := New(CallFunc(func(_ context.Context, body []byte) (int, []byte, error) {
		calls++
		c := decode(t, body)
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatal(err)
		}
		if _, ok := raw["session"]; ok {
			t.Fatal("persistent remote browser was created")
		}
		if c.Cmd != "request.post" || c.Proxy == nil || c.Proxy.Username != "proxy-user" || c.Proxy.Password != "private-proxy-password" || c.PostData == nil || *c.PostData != "query=a%26b" {
			t.Fatal("post or proxy scope lost")
		}
		return 200, success(), nil
	}))
	for i := 0; i < 2; i++ {
		_, err := client.Solve(context.Background(), Request{URL: "https://site.example/search", Method: "POST", PostData: "query=a%26b", Timeout: time.Second, Proxy: &Proxy{URL: "http://proxy.example", Username: "proxy-user", Password: "private-proxy-password"}})
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("solver created a remote session or replayed the post")
	}
}
func TestProxyAuthenticationIncludesBothKeysOnlyWhenConfigured(t *testing.T) {
	for _, p := range []Proxy{{URL: "http://proxy.example"}, {URL: "http://proxy.example", Username: "user"}, {URL: "http://proxy.example", Password: "private-password"}} {
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]string
		if err = json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		_, username := fields["username"]
		_, password := fields["password"]
		want := p.Username != "" || p.Password != ""
		if username != want || password != want {
			t.Fatal("proxy authentication became anonymous or enabled unintentionally")
		}
	}
}
func TestCancellationDoesNotRetryOrCreateRemoteSession(t *testing.T) {
	entered := make(chan struct{})
	calls := 0
	client, _ := New(CallFunc(func(ctx context.Context, body []byte) (int, []byte, error) {
		calls++
		if decode(t, body).Cmd != "request.get" {
			t.Error("persistent session command emitted")
		}
		close(entered)
		<-ctx.Done()
		return 0, nil, ctx.Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.Solve(ctx, Request{URL: "https://site.example/", Timeout: time.Minute, Proxy: &Proxy{URL: "http://proxy.example", Username: "user"}})
		done <- err
	}()
	<-entered
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("cancellation replayed a request")
	}
}
func TestTransportErrorDetailsDoNotEscape(t *testing.T) {
	calls := 0
	client, _ := New(CallFunc(func(context.Context, []byte) (int, []byte, error) {
		calls++
		return 0, nil, errors.New("https://private-user:secret@service.example/")
	}))
	_, err := client.Solve(context.Background(), Request{URL: "https://site.example/", Timeout: time.Second})
	if !errors.Is(err, ErrProtocol) || strings.Contains(err.Error(), "secret") || calls != 1 {
		t.Fatal("transport details exposed or retried", err)
	}
}
