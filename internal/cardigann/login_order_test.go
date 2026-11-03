package cardigann

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoginUsesDeclaredOrderEncodingAndFormSafeCharacters(t *testing.T) {
	for _, method := range []string{"get", "post"} {
		t.Run(method, func(t *testing.T) {
			definition := strings.Replace(minimalDefinition, "encoding: UTF-8", "encoding: windows-1251", 1) + `login:
  method: ` + method + `
  path: /login
  inputs:
    z: 'Тест'
    a: "~!*()'"
`
			plan, err := Load([]byte(definition))
			if err != nil {
				t.Fatal(err)
			}
			session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
				u, _ := url.Parse(r.URL)
				encoded := u.RawQuery
				if r.Method == http.MethodPost {
					encoded = string(r.Body)
				}
				if r.Method != strings.ToUpper(method) || encoded != "z=%D2%E5%F1%F2&a=%7E!*()%27" {
					t.Errorf("login order or encoding changed: %s %s", r.Method, encoded)
				}
				return HTTPResponse{Status: 200, URL: r.URL, Body: []byte("ok")}, nil
			}}
			execution, err := plan.NewExecution(session, "", nil, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if challenge, err := execution.Authenticate(context.Background()); err != nil || challenge != nil || len(session.requests) != 1 {
				t.Fatalf("login: %v %v; requests %d", challenge, err, len(session.requests))
			}
		})
	}
}

func TestFormLoginKeepsDOMAndDefinitionOrderAcrossChallenge(t *testing.T) {
	definition := minimalDefinition + `login:
  method: form
  path: /login
  inputs:
    second: overridden
    new: appended
  selectorinputs:
    zsel: {selector: '#z'}
    asel: {selector: '#a'}
  getselectorinputs:
    zq: {selector: '#z'}
    aq: {selector: '#a'}
  captcha: {type: text, selector: .question, input: answer}
`
	plan, err := Load([]byte(definition))
	if err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{respond: func(r Request) (HTTPResponse, error) {
		if r.Method == http.MethodGet {
			return HTTPResponse{Status: 200, URL: r.URL, Body: []byte(`<form action='/auth?existing=1'><input name='token' value='old'><input name='second' value='original'><input name='token' value='new'><input name='answer' value=''></form><b id='z'>Z</b><b id='a'>A</b><p class='question'>1+2?</p>`)}, nil
		}
		if r.URL != "https://example.invalid/auth?existing=1&zq=Z&aq=A" || string(r.Body) != "token=new&second=overridden&answer=3&new=appended&zsel=Z&asel=A" {
			t.Errorf("form order or overriding changed: %s %s", r.URL, r.Body)
		}
		return HTTPResponse{Status: 200, URL: r.URL, Body: []byte("ok")}, nil
	}}
	execution, err := plan.NewExecution(session, "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := execution.Authenticate(context.Background())
	if err != nil || challenge == nil || len(session.requests) != 1 {
		t.Fatalf("challenge setup: %v %v", challenge, err)
	}
	if err := execution.AnswerChallenge(context.Background(), "3"); err != nil {
		t.Fatal(err)
	}
	if len(session.requests) != 2 {
		t.Fatal("challenge refetched or submitted twice")
	}
	if err := execution.AnswerChallenge(context.Background(), "3"); err == nil || len(session.requests) != 2 {
		t.Fatal("challenge answer replayed")
	}
}
