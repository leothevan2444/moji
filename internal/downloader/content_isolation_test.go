package downloader

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type metadataClient struct {
	*capacityClient
	files     map[string][]qbittorrent.TorrentContentFile
	fileErr   error
	calls     map[string]int
	listings  int
	onListing func(int)
	onFile    func(string, int)
}

func (m *metadataClient) GetTorrentContents(_ context.Context, hash string, _ []string) ([]qbittorrent.TorrentContentFile, error) {
	m.calls[hash]++
	if m.onFile != nil {
		m.onFile(hash, m.calls[hash])
	}
	return m.files[hash], m.fileErr
}
func (m *metadataClient) GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error) {
	m.listings++
	if m.onListing != nil {
		m.onListing(m.listings)
	}
	return m.fakeClient.GetTorrentList(ctx, options)
}
func metadataFixture(t *testing.T, n int) (*Service, *metadataClient, []string) {
	t.Helper()
	s, q, _ := capacityFixture(t)
	client := &metadataClient{capacityClient: q, files: make(map[string][]qbittorrent.TorrentContentFile), calls: make(map[string]int)}
	hashes := []string{}
	for i := 0; i < n; i++ {
		hash := fmt.Sprintf("%040x", i+100)
		q.torrents = append(q.torrents, qbittorrent.Torrent{Hash: hash, State: qbittorrent.TorrentStateMetaDL, SavePath: "/downloads"})
		client.files[hash] = []qbittorrent.TorrentContentFile{}
		hashes = append(hashes, hash)
	}
	s.client = client
	return s, client, hashes
}
func TestEmptyDownloadsDoNotBlockSafeCapacityCleanup(t *testing.T) {
	for _, state := range []qbittorrent.TorrentState{qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentStateForcedMetaDL, qbittorrent.TorrentState("stoppedDL"), qbittorrent.TorrentStatePausedDL} {
		t.Run(string(state), func(t *testing.T) {
			s, q, hashes := metadataFixture(t, 27)
			for i := len(q.torrents) - 27; i < len(q.torrents); i++ {
				q.torrents[i].State = state
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if q.deleted != 1 || q.hashes[0] != strings.Repeat("a", 40) || !q.deleteFiles {
				t.Fatal("empty downloads blocked eligible source", q.hashes)
			}
			for _, hash := range hashes {
				if q.calls[hash] != 2 {
					t.Fatal("expected one check per listing plus deletion recheck", hash, q.calls[hash])
				}
			}
		})
	}
}

func TestUnknownOrNonemptyTorrentContentStillBlocksDeletion(t *testing.T) {
	for _, state := range []qbittorrent.TorrentState{qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentState("stoppedDL"), qbittorrent.TorrentStatePausedDL} {
		t.Run(string(state), func(t *testing.T) {
			for _, mode := range []string{"files unavailable", "null file list", "files already known", "ordinary downloading", "unknown state", "data received", "size known", "progress nonzero"} {
				t.Run(mode, func(t *testing.T) {
					s, q, hashes := metadataFixture(t, 1)
					last := len(q.torrents) - 1
					q.torrents[last].State = state
					wantReason := contentUnknown
					switch mode {
					case "files unavailable":
						q.fileErr = errors.New("offline")
						wantReason = contentCheckFailed
					case "null file list":
						q.files[hashes[0]] = nil
					case "files already known":
						q.files[hashes[0]] = []qbittorrent.TorrentContentFile{{}}
					case "ordinary downloading":
						q.torrents[last].State = qbittorrent.TorrentStateDownloading
					case "unknown state":
						q.torrents[last].State = qbittorrent.TorrentState("unknown")
					case "data received":
						q.torrents[last].Completed = 1
					case "size known":
						q.torrents[last].Size = 1
					case "progress nonzero":
						q.torrents[last].Progress = .1
					}
					if err := s.Cycle(context.Background()); err != nil {
						t.Fatal(err)
					}
					snap, _ := s.Snapshot(context.Background())
					if q.deleted != 0 || snap.Cleanup.Reason != string(wantReason) {
						t.Fatal("unknown content accepted or misreported", q.deleted, snap.Cleanup.Reason)
					}
				})
			}
		})
	}
}
func TestKnownSharedContentIsReportedWithoutMetadataRequests(t *testing.T) {
	s, q, _ := metadataFixture(t, 1)
	q.torrents = append(q.torrents, qbittorrent.Torrent{Hash: strings.Repeat("e", 40), ContentPath: "/downloads"})
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 0 || snap.Cleanup.Reason != string(contentShared) || len(q.calls) != 0 {
		t.Fatal(q.deleted, snap.Cleanup.Reason, q.calls)
	}
}
func TestMetadataArrivalIsRecheckedBeforeDeletingFiles(t *testing.T) {
	for _, state := range []qbittorrent.TorrentState{qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentState("stoppedDL"), qbittorrent.TorrentStatePausedDL} {
		t.Run(string(state), func(t *testing.T) {
			s, q, _ := metadataFixture(t, 1)
			q.torrents[len(q.torrents)-1].State = state
			q.onListing = func(n int) {
				if n == 2 {
					q.torrents[len(q.torrents)-1].ContentPath = "/downloads"
					q.torrents[len(q.torrents)-1].State = qbittorrent.TorrentStateDownloading
				}
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			snap, _ := s.Snapshot(context.Background())
			if q.deleted != 0 || snap.Cleanup.Reason != string(contentShared) {
				t.Fatal("newly shared path was deleted", q.deleted, snap.Cleanup.Reason)
			}
		})
	}
}
func TestMetadataFileAPIFailureAtDeletionRecheckIsSafe(t *testing.T) {
	for _, state := range []qbittorrent.TorrentState{qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentState("stoppedDL"), qbittorrent.TorrentStatePausedDL} {
		t.Run(string(state), func(t *testing.T) {
			s, q, _ := metadataFixture(t, 1)
			q.torrents[len(q.torrents)-1].State = state
			q.onFile = func(_ string, n int) {
				if n == 2 {
					q.fileErr = errors.New("offline")
				}
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			snap, _ := s.Snapshot(context.Background())
			if q.deleted != 0 || snap.Cleanup.Reason != string(contentCheckFailed) {
				t.Fatal(q.deleted, snap.Cleanup.Reason)
			}
		})
	}
}
func TestMetadataProofIsNotReusedAcrossCycles(t *testing.T) {
	s, q, hashes := metadataFixture(t, 1)
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	*q.space = 40e9
	q.files[hashes[0]] = []qbittorrent.TorrentContentFile{{}}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 1 || snap.Cleanup.Reason != string(contentUnknown) {
		t.Fatal("stale file proof reused", q.deleted, snap.Cleanup.Reason)
	}
}
func TestMetadataVerificationBudgetAndCancellation(t *testing.T) {
	for _, mode := range []string{"budget expired", "request deadline", "parent canceled"} {
		t.Run(mode, func(t *testing.T) {
			_, q, hashes := metadataFixture(t, 27)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			isolation := newContentIsolation(ctx, q, q.torrents)
			want := contentCheckTimeout
			switch mode {
			case "budget expired":
				isolation.deadline = time.Now().Add(-time.Second)
			case "request deadline":
				q.fileErr = context.DeadlineExceeded
			case "parent canceled":
				cancel()
				want = contentCheckFailed
			}
			if got := isolation.check(q.torrents[0]); got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
			if mode != "request deadline" && len(q.calls) != 0 {
				t.Fatal("queried files after cancellation or budget expiry", q.calls)
			}
			// No later unknown torrent can be accepted after an incomplete check.
			if _, checked := isolation.checked[hashes[1]]; checked {
				t.Fatal("continued checking after verification failed")
			}
		})
	}
}

func TestMetadataVerificationTimeoutBlocksCleanup(t *testing.T) {
	for _, recheck := range []bool{false, true} {
		t.Run(fmt.Sprintf("deletion recheck=%v", recheck), func(t *testing.T) {
			s, q, _ := metadataFixture(t, 27)
			q.onFile = func(_ string, n int) {
				if !recheck || n == 2 {
					q.fileErr = context.DeadlineExceeded
				}
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			snap, _ := s.Snapshot(context.Background())
			if q.deleted != 0 || snap.Cleanup.Reason != string(contentCheckTimeout) {
				t.Fatal("timeout did not block cleanup", q.deleted, snap.Cleanup.Reason)
			}
		})
	}
}

func TestMetadataVerificationSharesDeadlineAcrossRequests(t *testing.T) {
	_, q, hashes := metadataFixture(t, 27)
	isolation := newContentIsolation(context.Background(), q, q.torrents)
	q.onFile = func(hash string, _ int) {
		if hash == hashes[0] {
			// Expire the shared budget while handling the first file response.
			isolation.deadline = time.Now().Add(-time.Second)
		}
	}
	if got := isolation.check(q.torrents[0]); got != contentCheckTimeout {
		t.Fatal("did not enforce shared deadline", got)
	}
	if len(q.calls) != 1 || q.calls[hashes[0]] != 1 {
		t.Fatal("continued requesting files after shared deadline", q.calls)
	}
}

func TestManualFileRemovalUsesTheSameMetadataSafetyCheck(t *testing.T) {
	for _, state := range []qbittorrent.TorrentState{qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentState("stoppedDL"), qbittorrent.TorrentStatePausedDL} {
		t.Run(string(state), func(t *testing.T) {
			s, q, _ := metadataFixture(t, 1)
			q.torrents[len(q.torrents)-1].State = state
			if err := s.Action(context.Background(), strings.Repeat("a", 40), "REMOVE_FILES"); err != nil {
				t.Fatal(err)
			}
			if q.deleted != 1 || !q.deleteFiles {
				t.Fatal(q.hashes)
			}
		})
	}
}
