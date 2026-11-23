package downloader

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type pieceTestClient struct {
	*fakeClient
	states  []int
	failure error
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (c *pieceTestClient) GetTorrentPiecesStates(ctx context.Context, _ string) ([]int, error) {
	c.calls.Add(1)
	if c.entered != nil {
		c.entered <- struct{}{}
	}
	if c.release != nil {
		select {
		case <-c.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return c.states, c.failure
}
func pieceTestService(t *testing.T) (*Service, *pieceTestClient) {
	t.Helper()
	s, base, _, _ := newTest(t)
	c := &pieceTestClient{fakeClient: base, states: []int{0, 1, 2, 2}}
	s.client = c
	s.snapshot = Snapshot{CheckedAt: time.Now().UTC().Format(time.RFC3339), Connected: true, Torrents: []Torrent{{Hash: testHash, Present: true}}}
	return s, c
}
func TestPieceAggregationPreservesCountsAndBounds(t *testing.T) {
	states := make([]int, 6001)
	want := [3]int{}
	for i := range states {
		states[i] = i % 3
		want[i%3]++
	}
	buckets, err := aggregatePieces(states)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) > maxPieceBuckets {
		t.Fatalf("unbounded response: %d", len(buckets))
	}
	got := [3]int{}
	next := 0
	for _, b := range buckets {
		if b.Start != next || b.Count != b.Missing+b.Downloading+b.Downloaded {
			t.Fatalf("lost piece range: %+v", b)
		}
		next += b.Count
		got[0] += b.Missing
		got[1] += b.Downloading
		got[2] += b.Downloaded
	}
	if next != len(states) || got != want {
		t.Fatalf("lost real states: %v / %v", got, want)
	}
	for _, invalid := range [][]int{{3}, make([]int, maxTorrentPieces+1)} {
		if _, err := aggregatePieces(invalid); !errors.Is(err, ErrPiecesUnavailable) {
			t.Fatal("invalid map accepted")
		}
	}
	empty, err := aggregatePieces(nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("empty map should be a known empty response")
	}
}
func TestPiecesRequiresFreshPresenceAndValidHash(t *testing.T) {
	s, c := pieceTestService(t)
	out, err := s.Pieces(context.Background(), strings.ToUpper(testHash))
	if err != nil || out.Total != 4 || len(out.Buckets) != 4 {
		t.Fatalf("actual piece read failed: %+v %v", out, err)
	}
	if c.calls.Load() != 1 {
		t.Fatal("wrong query count")
	}
	for _, tc := range []struct {
		name, hash         string
		connected, present bool
		at                 string
	}{
		{"invalid", "not-a-hash", true, true, time.Now().Format(time.RFC3339)},
		{"absent", testHash, true, false, time.Now().Format(time.RFC3339)},
		{"offline", testHash, false, true, time.Now().Format(time.RFC3339)},
		{"stale", testHash, true, true, time.Now().Add(-time.Minute).Format(time.RFC3339)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.snapshot.Connected = tc.connected
			s.snapshot.CheckedAt = tc.at
			s.snapshot.Torrents[0].Present = tc.present
			if _, err := s.Pieces(context.Background(), tc.hash); err == nil {
				t.Fatal("unconfirmed pieces accepted")
			}
		})
	}
	if c.calls.Load() != 1 {
		t.Fatal("invalid/absent/offline reads contacted qB")
	}
}
func TestPiecesFailureAndCancellation(t *testing.T) {
	s, c := pieceTestService(t)
	c.failure = errors.New("private upstream detail")
	if _, err := s.Pieces(context.Background(), testHash); !errors.Is(err, ErrPiecesUnavailable) || strings.Contains(err.Error(), "private") {
		t.Fatal("raw upstream error exposed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := c.calls.Load()
	if _, err := s.Pieces(ctx, testHash); !errors.Is(err, context.Canceled) || c.calls.Load() != before {
		t.Fatal("cancelled read contacted qB")
	}
}
func TestPiecesConcurrencyIsBoundedAndCancellable(t *testing.T) {
	s, c := pieceTestService(t)
	c.entered = make(chan struct{}, 2)
	c.release = make(chan struct{})
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, err := s.Pieces(context.Background(), testHash); done <- err }()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-c.entered:
		case <-time.After(time.Second):
			t.Fatal("piece reader did not start")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := s.Pieces(ctx, testHash); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("third query did not respect its deadline")
	}
	if c.calls.Load() != 2 {
		t.Fatal("piece reader concurrency exceeded limit")
	}
	close(c.release)
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}
func TestTrackerCountsAndUpstreamTimesRemainDistinct(t *testing.T) {
	s, q, _, _ := newTest(t)
	q.torrents[0].NumComplete = 17
	q.torrents[0].NumIncomplete = -1
	q.torrents[0].TimeActive = 10800
	q.torrents[0].SeenComplete = 1700000000
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := s.Snapshot(context.Background())
	tor := snapshot.Torrents[0]
	if tor.SeedsTotal == nil || *tor.SeedsTotal != 17 || tor.PeersTotal != nil || tor.ActiveSeconds == nil || *tor.ActiveSeconds != 10800 || tor.LastSeenCompleteAt != time.Unix(1700000000, 0).UTC().Format(time.RFC3339) || tor.LastSeenAt == tor.LastSeenCompleteAt {
		t.Fatalf("incorrect observed fields: %+v", tor)
	}
	*tor.SeedsTotal = 999
	again, _ := s.Snapshot(context.Background())
	if *again.Torrents[0].SeedsTotal != 17 {
		t.Fatal("snapshot pointers were not cloned")
	}
	if knownCount(-1) != nil || completeTime(0) != "" {
		t.Fatal("unknown values fabricated")
	}
	q.torrents = []qbittorrent.Torrent{} // Presence transitions preserve the previous confirmed map fields.
	if err := s.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
}
