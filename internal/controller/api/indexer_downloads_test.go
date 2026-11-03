package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdownloads"
)

type downloadFetchFunc func(context.Context, string) (indexer.Artifact, error)

func (f downloadFetchFunc) FetchReference(ctx context.Context, id string) (indexer.Artifact, error) {
	return f(ctx, id)
}
func TestPreparedDownloadOnlyServesStoredAttachmentAndHeadDoesNotAcquire(t *testing.T) {
	for _, magnet := range []bool{false, true} {
		t.Run(map[bool]string{false: "torrent", true: "magnet"}[magnet], func(t *testing.T) {
			calls := 0
			data := []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee")
			uri := "magnet:?xt=urn:btih:synthetic&tr=https%3A%2F%2Ftracker.example%2Fsynthetic-passkey"
			store, err := indexerdownloads.New(context.Background(), downloadFetchFunc(func(context.Context, string) (indexer.Artifact, error) {
				calls++
				if magnet {
					return indexer.Artifact{Magnet: uri}, nil
				}
				return indexer.Artifact{TorrentBytes: data}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			receipt, err := store.Prepare(context.Background(), "private-reference")
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			NewHandler(nil, WithIndexerDownloads(store)).Register(mux)
			for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodGet} {
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, httptest.NewRequest(method, "/api/indexers/downloads/"+receipt.Handle+"?url=https://other.example/private", nil))
				if response.Code != http.StatusOK || calls != 1 || response.Header().Get("Location") != "" {
					t.Fatal("read performed an acquisition or redirect", response.Code, calls)
				}
				if response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Referrer-Policy") != "no-referrer" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
					t.Fatal("download response can leak through cache/referrer")
				}
				expected, filename, content := string(data), "resource.torrent", "application/x-bittorrent"
				if magnet {
					expected, filename, content = uri, "resource.magnet", "text/uri-list; charset=utf-8"
				}
				if method == http.MethodHead {
					expected = ""
				}
				if response.Body.String() != expected || response.Header().Get("Content-Disposition") != `attachment; filename="`+filename+`"` || response.Header().Get("Content-Type") != content {
					t.Fatal("wrong attachment representation", response.Header())
				}
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/indexers/downloads/"+strings.Repeat("a", 64), nil))
			if response.Code != http.StatusGone || calls != 1 {
				t.Fatal("unknown handle fetched a source", response.Code, calls)
			}
		})
	}
}

type failedDownloads struct{ err error }

func (f failedDownloads) Open(string) (*indexerdownloads.Lease, error) { return nil, f.err }
func TestPreparedDownloadErrorsAreBoundedAndRedacted(t *testing.T) {
	for _, item := range []struct {
		store  preparedDownloads
		status int
		code   string
	}{{nil, 503, "INDEXER_NOT_CONFIGURED"}, {failedDownloads{indexer.ErrBusy}, 429, "INDEXER_BUSY"}, {failedDownloads{errors.New("https://private.example/secret?passkey=synthetic")}, 503, "INDEXER_DOWNLOAD_UNAVAILABLE"}} {
		mux := http.NewServeMux()
		NewHandler(nil, WithIndexerDownloads(item.store)).Register(mux)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/indexers/downloads/"+strings.Repeat("a", 64), nil))
		if w.Code != item.status || !strings.Contains(w.Body.String(), item.code) || strings.Contains(w.Body.String(), "synthetic") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
