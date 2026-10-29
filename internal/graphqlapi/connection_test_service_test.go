package graphqlapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
)

func TestServiceConnectionUsesSuppliedCredentialsAndRealProtocol(t *testing.T) {
	for _, service := range model.AllSettingsConnectionService {
		t.Run(string(service), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch service {
				case model.SettingsConnectionServiceStash:
					if r.URL.Path != "/prefix/graphql" || r.Header.Get("ApiKey") != "test-key" {
						t.Errorf("wrong Stash request: %s", r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"data":{"version":{"version":"v0.28.0","hash":"abc","build_time":"today"}}}`))
				case model.SettingsConnectionServiceJackett:
					if r.URL.Path != "/prefix/api/v2.0/indexers/all/results/torznab/api" || r.URL.Query().Get("t") != "indexers" || r.URL.Query().Get("configured") != "true" || r.URL.Query().Get("apikey") != "test-key" {
						t.Errorf("wrong Jackett request")
					}
					_, _ = w.Write([]byte(`<indexers><indexer id="test" configured="true"><title>Test</title><link>https://tracker.example</link><type>private</type><language>en-US</language></indexer></indexers>`))
				case model.SettingsConnectionServiceQbittorrent:
					if strings.HasSuffix(r.URL.Path, "/auth/login") {
						_ = r.ParseForm()
						if r.Form.Get("username") != "user" || r.Form.Get("password") != " secret " {
							t.Error("credentials changed")
						}
						http.SetCookie(w, &http.Cookie{Name: "SID", Value: "isolated", Path: "/"})
						_, _ = w.Write([]byte("Ok."))
					} else {
						cookie, e := r.Cookie("SID")
						if e != nil || cookie.Value != "isolated" {
							t.Error("login session missing")
						}
						if r.URL.Path != "/prefix/api/v2/app/version" {
							t.Error("unexpected endpoint")
						}
						_, _ = w.Write([]byte("v5.1.0beta1"))
					}
				}
			}))
			defer server.Close()
			key, user, password := "test-key", "user", " secret "
			got, err := (&mutationResolver{&Resolver{}}).TestServiceConnection(context.Background(), model.TestServiceConnectionInput{Service: service, URL: server.URL + "/prefix/", APIKey: &key, Username: &user, Password: &password})
			if err != nil || got.Code != model.ServiceConnectionTestCodeOk || calls == 0 {
				t.Fatalf("result=%v err=%v calls=%d", got, err, calls)
			}
		})
	}
}

func TestServiceConnectionRejectsInvalidResponsesAndHasBoundedCancellation(t *testing.T) {
	for _, service := range model.AllSettingsConnectionService {
		for _, mode := range []string{"unauthorized", "wrong-service", "timeout"} {
			t.Run(string(service)+"/"+mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if mode == "timeout" {
						time.Sleep(100 * time.Millisecond)
						return
					}
					if mode == "unauthorized" {
						w.WriteHeader(http.StatusUnauthorized)
					}
					_, _ = w.Write([]byte("<html>secret-credential must not be exposed</html>"))
				}))
				defer server.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				got := testServiceConnection(ctx, model.TestServiceConnectionInput{Service: service, URL: server.URL})
				want := model.ServiceConnectionTestCodeFailed
				if mode == "timeout" {
					want = model.ServiceConnectionTestCodeTimeout
				}
				if got.Code != want {
					t.Fatalf("got %v want %v", got.Code, want)
				}
			})
		}
	}
	for _, endpoint := range []string{"", "file:///etc/passwd", "localhost:9999", "https://user:secret@example.test", "https://example.test/?secret=x"} {
		got := testServiceConnection(context.Background(), model.TestServiceConnectionInput{Service: model.SettingsConnectionServiceStash, URL: endpoint})
		if got.Code != model.ServiceConnectionTestCodeInvalidURL {
			t.Errorf("invalid URL accepted: %v", got.Code)
		}
	}
}

func TestServiceConnectionReportsRejectedQBLoginWithoutCallingVersion(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v2/auth/login" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("Fails."))
	}))
	defer server.Close()
	user, password := "wrong", "secret"
	got := testServiceConnection(context.Background(), model.TestServiceConnectionInput{Service: model.SettingsConnectionServiceQbittorrent, URL: server.URL, Username: &user, Password: &password})
	if got.Code != model.ServiceConnectionTestCodeFailed || calls != 1 {
		t.Fatalf("result=%v calls=%d", got.Code, calls)
	}
}
