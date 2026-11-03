package cardigann

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestLoginFormPreservesCSRFAndResumesTextChallengeWithoutRefetch(t *testing.T) {
	definition := minimalDefinition + `settings:
  - {name: username, type: text}
  - {name: password, type: password}
login:
  method: form
  path: /login
  form: form
  inputs:
    username: '{{ .Config.username }}'
    password: '{{ .Config.password }}'
  captcha:
    type: text
    selector: .challenge
    input: answer
  test:
    path: /account
    selector: a.logout
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		switch r.URL {
		case "https://example.invalid/login":
			return HTTPResponse{Status: 200, Body: []byte(`<form action="/authenticate"><input name="csrf" value="token"><input name="username"><input name="password"><input name="answer"></form><p class="challenge">What is 1 + 2?</p>`)}, nil
		case "https://example.invalid/authenticate":
			values, err := url.ParseQuery(string(r.Body))
			if err != nil || r.Method != http.MethodPost || values.Get("csrf") != "token" || values.Get("username") != "alice" || values.Get("password") != "private" || values.Get("answer") != "3" {
				t.Fatalf("incorrect form request: %#v", r)
			}
			return HTTPResponse{Status: 200, Body: []byte("ok")}, nil
		case "https://example.invalid/account":
			return HTTPResponse{Status: 200, Body: []byte(`<a class="logout">logout</a>`)}, nil
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return HTTPResponse{}, nil
		}
	}}
	execution, err := plan.NewExecution(session, "", map[string]Scalar{"username": "alice", "password": "private"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := execution.Authenticate(context.Background())
	if err != nil || challenge == nil || challenge.Type != "text" || challenge.Prompt != "What is 1 + 2?" || len(session.requests) != 1 {
		t.Fatalf("challenge: %#v %v", challenge, err)
	}
	if err := execution.AnswerChallenge(context.Background(), "3"); err != nil {
		t.Fatal(err)
	}
	if len(session.requests) != 3 {
		t.Fatal("landing form refetched or login replayed")
	}
	if err := execution.AnswerChallenge(context.Background(), "3"); err == nil {
		t.Fatal("challenge replay accepted")
	}
}

func TestCookieLoginAndFailedVerification(t *testing.T) {
	definition := minimalDefinition + `settings:
  - {name: cookie, type: text}
login:
  method: cookie
  test: {path: /account, selector: a.logout}
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		return HTTPResponse{Status: 200, Headers: http.Header{}, Body: []byte(`<p>not logged in</p>`)}, nil
	}}
	execution, err := plan.NewExecution(session, "", map[string]Scalar{"cookie": "session=secret"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execution.Authenticate(context.Background()); err == nil {
		t.Fatal("login test accepted missing selector")
	}
	if session.cookieOrigin != "https://example.invalid/" || session.cookieHeader != "session=secret" {
		t.Fatal("cookie did not enter isolated site session")
	}
}
