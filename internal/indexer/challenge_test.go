package indexer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestChallengeResumesLoginOnceAndDoesNotPersistAnswer(t *testing.T) {
	for _, mode := range []string{"persistent", "transient"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close()
			definition := strings.Replace(siteFixture, `  path: /login
  method: post
  inputs: {username: '{{ .Config.username }}', password: '{{ .Config.password }}'}
  test: {path: /account, selector: '#member'}`, `  path: /login
  method: form
  form: form
  inputs: {username: '{{ .Config.username }}', password: '{{ .Config.password }}'}
  captcha: {type: text, selector: .challenge, input: answer}
  test: {path: /account, selector: '#member'}`, 1)
			plan, err := cardigann.Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			landings, submits := 0, 0
			network := testNetwork(t, func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/login":
					landings++
					return httpResponse(200, nil, `<form action="/authenticate"><input name="csrf" value="synthetic-csrf"><input name="username"><input name="password"><input name="answer"></form><p class="challenge">Synthetic prompt</p>`), nil
				case "/authenticate":
					submits++
					data, _ := io.ReadAll(request.Body)
					values, _ := url.ParseQuery(string(data))
					if values.Get("csrf") != "synthetic-csrf" || values.Get("answer") != "session-only-answer" {
						t.Error("challenge context lost")
					}
					return httpResponse(200, http.Header{"Set-Cookie": []string{"derived=synthetic; Path=/; Secure; HttpOnly"}}, "ok"), nil
				case "/account":
					return httpResponse(200, nil, `<div id="member">logged in</div>`), nil
				default:
					t.Error("unexpected endpoint")
					return httpResponse(404, nil, ""), nil
				}
			})
			var sessions *Sessions
			if mode == "transient" {
				sessions, err = NewTransientSessions(network)
			} else {
				sessions, err = NewSessions(network, manager)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer sessions.Close()
			instance := fixtureInstance()
			instance.BaseURL = ""
			operations := 0
			login, err := NewLoginSession(sessions, instance, plan)
			if err != nil {
				t.Fatal(err)
			}
			defer login.Close()
			err = login.Authenticate(ctx)
			if !errors.Is(err, ErrChallenge) || operations != 0 {
				t.Fatal("challenge not surfaced", err)
			}
			challenges := sessions.Challenges(instance.ID)
			if len(challenges) != 1 || challenges[0].Prompt != "Synthetic prompt" {
				t.Fatal("challenge presentation missing")
			}

			scoped, err := sessions.ChallengeFor(instance, plan)
			if err != nil || scoped == nil || scoped.ID != challenges[0].ID {
				t.Fatal("rule-default scope did not resolve its challenge", err)
			}
			unstarted, err := NewLoginSession(sessions, instance, plan)
			if err != nil {
				t.Fatal(err)
			}
			unstarted.Close()
			if current, err := login.Challenge(); err != nil || current == nil || current.ID != scoped.ID {
				t.Fatal("a login that observed no challenge discarded another pending login", err)
			}
			if err := login.Answer(ctx, "foreign-challenge", "session-only-answer"); !errors.Is(err, ErrInvalidScope) {
				t.Fatal("login answered another challenge", err)
			}
			changedRule := *plan
			changedLogin := *plan.Definition.Login
			changedLogin.Path = "/another-login"
			changedRule.Definition.Login = &changedLogin
			if current, err := sessions.ChallengeFor(instance, &changedRule); err != nil || current != nil {
				t.Fatal("changed login rule observed old challenge", err)
			}
			if err := sessions.AnswerFor(ctx, instance, &changedRule, scoped.ID, "session-only-answer"); !errors.Is(err, ErrInvalidScope) {
				t.Fatal("changed login rule answered old challenge", err)
			}
			resultOnlyRule := *plan
			resultOnlyRule.Digest = "different-result-parser"
			if current, err := sessions.ChallengeFor(instance, &resultOnlyRule); err != nil || current == nil || current.ID != scoped.ID {
				t.Fatal("result-only revision invalidated authentication context", err)
			}
			foreign := instance
			foreign.ConfigRevision += "-different-account"
			if other, err := sessions.ChallengeFor(foreign, plan); err != nil || other != nil {
				t.Fatal("different account observed challenge", err)
			}
			if err := sessions.AnswerFor(ctx, foreign, plan, scoped.ID, "wrong-account-answer"); !errors.Is(err, ErrInvalidScope) {
				t.Fatal("different account answered challenge", err)
			}
			if err := sessions.DiscardChallenge(ctx, foreign, plan, scoped.ID); !errors.Is(err, ErrInvalidScope) {
				t.Fatal("different account discarded challenge", err)
			}
			if err := login.Answer(ctx, scoped.ID, "session-only-answer"); err != nil {
				t.Fatal(err)
			}
			if err := sessions.Answer(ctx, challenges[0].ID, "session-only-answer"); !errors.Is(err, ErrExpired) {
				t.Fatal("challenge replay accepted", err)
			}
			if err := sessions.With(ctx, instance, plan, true, func(*cardigann.Execution) error { operations++; return nil }); err != nil {
				t.Fatal(err)
			}
			if landings != 1 || submits != 1 || operations != 1 || len(sessions.Challenges(instance.ID)) != 0 {
				t.Fatal("challenge repeated landing or submit")
			}
			var records []string
			if err := manager.Runtime.Reader.SelectContext(ctx, &records, `SELECT payload_json FROM indexer_sessions`); err != nil {
				t.Fatal(err)
			}
			if mode == "transient" && len(records) != 0 {
				t.Fatal("transient challenge wrote runtime cookies")
			}
			if mode == "persistent" && (len(records) != 1 || strings.Contains(records[0], "session-only-answer") || strings.Contains(records[0], "synthetic-password")) {
				t.Fatal("answer or configured password persisted as derived session")
			}
			restarted, err := NewSessions(network, manager)
			if err != nil {
				t.Fatal(err)
			}
			defer restarted.Close()
			err = restarted.With(ctx, instance, plan, true, func(*cardigann.Execution) error { return nil })
			if mode == "persistent" && (err != nil || landings != 1) {
				t.Fatal("solved persistent session not restored", err)
			}
			if mode == "transient" && (!errors.Is(err, ErrChallenge) || landings != 2) {
				t.Fatal("transient session leaked into a fresh production collection", err)
			}
			if mode == "transient" {
				challenge, err := restarted.ChallengeFor(instance, plan)
				if err != nil || challenge == nil {
					t.Fatal("fresh production challenge missing", err)
				}
				pendingLogin, err := NewLoginSession(restarted, instance, plan)
				if err != nil {
					t.Fatal(err)
				}
				if err = pendingLogin.Authenticate(ctx); !errors.Is(err, ErrChallenge) {
					t.Fatal(err)
				}
				pendingLogin.Close()
				pendingLogin.Close()
				if current, err := restarted.ChallengeFor(instance, plan); err != nil || current != nil {
					t.Fatal("discarded challenge retained", err)
				}
				if err := restarted.Answer(ctx, challenge.ID, "session-only-answer"); !errors.Is(err, ErrExpired) {
					t.Fatal("discarded challenge accepted an answer", err)
				}
				if submits != 1 {
					t.Fatal("discard submitted an answer")
				}
			}
		})
	}
}
