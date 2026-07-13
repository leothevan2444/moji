package stashbox

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/leothevan2444/moji/pkg/stashbox/graphql"
)

func TestClientUsesCallerContext(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	tests := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "find performer",
			call: func(ctx context.Context) error {
				_, err := client.FindPerformerByID(ctx, "performer-1")
				return err
			},
		},
		{
			name: "query scenes",
			call: func(ctx context.Context) error {
				_, err := client.QueryScenes(ctx, graphql.SceneQueryInput{Page: 1, PerPage: 40})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := test.call(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("upstream requests = %d, want 0", got)
	}
}
