package taskruntime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMetadataCooldownSharesJackettIndexerAcrossTasks(t *testing.T) {
	for _, status := range []int{429, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			baseURL := "https://jackett.test"
			s, err := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore(), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
			})}))
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.fetchTorrentMetadata(context.Background(), baseURL+"/dl/indexer/?jackett_apikey=secret&path=one")
			if err == nil {
				t.Fatal("expected first request failure")
			}
			_, err = s.fetchTorrentMetadata(context.Background(), baseURL+"/dl/indexer/?jackett_apikey=secret&path=two")
			if !errors.Is(err, errTorrentMetadataCoolingDown) || calls.Load() != 1 {
				t.Fatalf("cooldown did not suppress another candidate: calls=%d err=%v", calls.Load(), err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("cooldown leaked token")
			}
			target, _ := url.Parse(baseURL + "/dl/other/")
			finish, err := s.acquireTorrentMetadataHost(context.Background(), target)
			if err != nil {
				t.Fatalf("unrelated indexer blocked: %v", err)
			}
			finish(200, "", nil)
			for key, h := range s.metadataHosts {
				if strings.HasSuffix(key, "/dl/indexer") && time.Until(h.blockedUntil) < 119*time.Second {
					t.Fatal("Retry-After was ignored")
				}
			}
		})
	}
}
func TestMetadataHostWaitRespectsCancellation(t *testing.T) {
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore())
	target, _ := url.Parse("https://example.test/dl/indexer/")
	finish, err := s.acquireTorrentMetadataHost(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.acquireTorrentMetadataHost(ctx, target); !errors.Is(err, context.Canceled) {
		t.Fatalf("queued request ignored cancellation: %v", err)
	}
	finish(200, "", nil)
	// The pacing timer, as well as the concurrency gate, must be cancellable.
	if _, err := s.acquireTorrentMetadataHost(ctx, target); !errors.Is(err, context.Canceled) {
		t.Fatalf("pacing ignored cancellation: %v", err)
	}
}
func TestMetadataNetworkErrorsRedactSignedURL(t *testing.T) {
	s, _ := NewService(fakeTracker{}, &fakeTorrentAdder{}, NewMemoryTaskStore())
	s.httpClient = &http.Client{Transport: metadataErrorTransport{}}
	_, err := s.fetchTorrentMetadata(context.Background(), "http://example.test/dl/indexer/?jackett_apikey=secret")
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("network error contains signed URL: %v", err)
	}
}

type metadataErrorTransport struct{}

func (metadataErrorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}
