package qbittorrent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type pieceResponseBody struct {
	io.Reader
	read int64
}

func (b *pieceResponseBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += int64(n)
	return n, err
}
func (b *pieceResponseBody) Close() error { return nil }
func TestPieceStatesUsesAuthenticatedTorrentEndpointAndBoundsResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{{"states", "[0,1,2,2]", true}, {"oversized", strings.Repeat(" ", maxPieceStateResponseBytes) + "[0]", false}} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(nil)
			client.baseURL = "http://qbt"
			body := &pieceResponseBody{Reader: strings.NewReader(tc.body)}
			client.httpClient = &http.Client{Transport: fileSelectionTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Path != "/api/v2/torrents/pieceStates" || r.URL.Query().Get("hash") != strings.Repeat("a", 40) {
					t.Fatalf("wrong piece request: %s", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
			})}
			states, err := client.GetTorrentPiecesStates(context.Background(), strings.Repeat("a", 40))
			if tc.valid {
				if err != nil || len(states) != 4 || states[1] != 1 || states[2] != 2 {
					t.Fatalf("real states lost: %v %v", states, err)
				}
			} else if err == nil {
				t.Fatal("oversized piece response accepted")
			}
			if body.read > maxPieceStateResponseBytes {
				t.Fatalf("unbounded piece response: %d", body.read)
			}
		})
	}
}
