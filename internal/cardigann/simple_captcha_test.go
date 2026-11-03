package cardigann

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestSimpleCaptchaPreservesFormAndStopsOnInvalidOrCancelledResponse(t *testing.T) {
	for _, scenario := range []string{"success", "invalid", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			definition := minimalDefinition + `login:
  method: form
  path: /login
  inputs: {username: alice, password: synthetic-secret}
`
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			posts := 0
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				switch r.URL {
				case "https://example.invalid/login":
					return HTTPResponse{Status: 200, Body: []byte(`<form action='/submit'><input name='csrf' value='token'><input name='username'><input name='password'><input name='captchaSelection'><input name='submitme' value='old'></form><script src='/simpleCaptcha.js'></script>`)}, nil
				case "https://example.invalid/simpleCaptcha.php?numImages=1":
					if r.Headers.Get("Referer") != "https://example.invalid/login" || r.Headers.Get("Content-Type") != "" {
						t.Error("captcha GET inherited form headers or lost referer")
					}
					if scenario == "invalid" {
						return HTTPResponse{Status: 200, Body: []byte(`{"images":[]}`)}, nil
					}
					if scenario == "cancelled" {
						cancel()
					}
					return HTTPResponse{Status: 200, Body: []byte(`{"images":[{"hash":"synthetic+hash"}]}`)}, nil
				case "https://example.invalid/submit":
					posts++
					if r.Method != http.MethodPost || string(r.Body) != "csrf=token&username=alice&password=synthetic-secret&captchaSelection=synthetic%2Bhash&submitme=X" {
						t.Error("captcha form order/value/encoding changed", string(r.Body))
					}
					return HTTPResponse{Status: 200, Body: []byte("ok")}, nil
				default:
					t.Error("unexpected login request", r.URL)
					return HTTPResponse{}, nil
				}
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			challenge, err := execution.Authenticate(ctx)
			if challenge != nil {
				t.Fatal("simpleCaptcha became a manual challenge")
			}
			if scenario == "success" {
				if err != nil || posts != 1 || len(session.requests) != 3 {
					t.Fatal("simpleCaptcha did not submit once", err, posts)
				}
			} else {
				if err == nil || posts != 0 || len(session.requests) != 2 {
					t.Fatal("failed/cancelled captcha continued or retried", err, posts)
				}
				if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation was hidden", err)
				}
			}
		})
	}
}
