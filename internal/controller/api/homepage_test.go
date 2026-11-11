package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

type homeStatsFunc func(context.Context, time.Time) (taskruntime.HomeStats, error)

func (f homeStatsFunc) HomeStats(ctx context.Context, now time.Time) (taskruntime.HomeStats, error) {
	return f(ctx, now)
}

func widgetMux(provider HomeStatsProvider) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(nil, WithHomeStats(provider)).Register(mux)
	return mux
}

func TestHomepageWidget(t *testing.T) {
	for _, stats := range []taskruntime.HomeStats{{}, {Total: 99, Attention: 3, Active: 8, Waiting: 12, CompletedLast24Hours: 5}} {
		mux := widgetMux(homeStatsFunc(func(ctx context.Context, now time.Time) (taskruntime.HomeStats, error) {
			deadline, ok := ctx.Deadline()
			if !ok || deadline.Sub(now) > 5*time.Second || time.Since(now) > time.Second {
				t.Fatal("missing bounded query deadline")
			}
			return stats, nil
		}))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/widget/homepage", nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Fatalf("unexpected response: %+v", w)
		}
		var got map[string]int
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		want := map[string]int{"attention": stats.Attention, "active": stats.Active, "waiting": stats.Waiting, "completedLast24Hours": stats.CompletedLast24Hours}
		if len(got) != 4 {
			t.Fatalf("unexpected fields: %v", got)
		}
		for key, value := range want {
			if got[key] != value {
				t.Fatalf("%s: got %d want %d", key, got[key], value)
			}
		}
	}
}

func TestHomepageWidgetErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider HomeStatsProvider
		status   int
		code     string
	}{
		{"unavailable", nil, 503, "WIDGET_UNAVAILABLE"},
		{"read failure", homeStatsFunc(func(context.Context, time.Time) (taskruntime.HomeStats, error) {
			return taskruntime.HomeStats{}, errors.New("private SQL path secret")
		}), 500, "WIDGET_QUERY_FAILED"},
		{"timeout", homeStatsFunc(func(context.Context, time.Time) (taskruntime.HomeStats, error) {
			return taskruntime.HomeStats{}, context.DeadlineExceeded
		}), 504, "WIDGET_QUERY_TIMEOUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			widgetMux(tc.provider).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/widget/homepage", nil))
			var body struct {
				Error widgetError `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || body.Error.Code != tc.code || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected failure: %d %s", w.Code, w.Body.String())
			}
			if _, err := uuid.Parse(body.Error.CorrelationID); err != nil {
				t.Fatal("missing correlation ID", err)
			}
			if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "attention") {
				t.Fatal("error leaked data or success fields")
			}
		})
	}
}

func TestHomepageWidgetCancellationAndMethods(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := homeStatsFunc(func(queryCtx context.Context, _ time.Time) (taskruntime.HomeStats, error) {
		cancel()
		<-queryCtx.Done()
		return taskruntime.HomeStats{}, queryCtx.Err()
	})
	w := httptest.NewRecorder()
	widgetMux(provider).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/widget/homepage", nil).WithContext(ctx))
	if w.Body.Len() != 0 {
		t.Fatalf("canceled request wrote a response: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	widgetMux(nil).ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/widget/homepage", nil))
	if w.Code != 405 || !strings.Contains(w.Header().Get("Allow"), "GET") {
		t.Fatalf("write method was accepted: %d", w.Code)
	}
}
