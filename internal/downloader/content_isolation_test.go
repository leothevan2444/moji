package downloader

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

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
func TestMetadataOnlyTorrentsDoNotBlockSafeCapacityCleanup(t *testing.T) {
	s, q, hashes := metadataFixture(t, 9)
	q.torrents[len(q.torrents)-1].State = qbittorrent.TorrentStateForcedMetaDL
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 || q.hashes[0] != strings.Repeat("a", 40) || !q.deleteFiles {
		t.Fatal("metadata downloads blocked eligible source", q.hashes)
	}
	for _, hash := range hashes {
		if q.calls[hash] != 2 {
			t.Fatal("expected one check per listing plus deletion recheck", hash, q.calls[hash])
		}
	}
}
func TestUnknownOrNonemptyTorrentContentStillBlocksDeletion(t *testing.T) {
	for _, mode := range []string{"files unavailable", "null file list", "files already known", "ordinary downloading", "stopped unknown", "data received", "size known", "progress nonzero"} {
		t.Run(mode, func(t *testing.T) {
			s, q, hashes := metadataFixture(t, 1)
			last := len(q.torrents) - 1
			switch mode {
			case "files unavailable":
				q.fileErr = errors.New("offline")
			case "null file list":
				q.files[hashes[0]] = nil
			case "files already known":
				q.files[hashes[0]] = []qbittorrent.TorrentContentFile{{}}
			case "ordinary downloading":
				q.torrents[last].State = qbittorrent.TorrentStateDownloading
			case "stopped unknown":
				q.torrents[last].State = qbittorrent.TorrentState("stoppedDL")
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
			if q.deleted != 0 || snap.Cleanup.Reason != string(contentUnknown) {
				t.Fatal("unknown content accepted or misreported", q.deleted, snap.Cleanup.Reason)
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
	s, q, _ := metadataFixture(t, 1)
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
}
func TestMetadataFileAPIFailureAtDeletionRecheckIsSafe(t *testing.T) {
	s, q, _ := metadataFixture(t, 1)
	q.onFile = func(_ string, n int) {
		if n == 2 {
			q.fileErr = errors.New("offline")
		}
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 0 || snap.Cleanup.Reason != string(contentUnknown) {
		t.Fatal(q.deleted, snap.Cleanup.Reason)
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
func TestMetadataVerificationHasBoundedRequests(t *testing.T) {
	s, q, _ := metadataFixture(t, metadataCheckLimit+1)
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, calls := range q.calls {
		count += calls
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 0 || count != metadataCheckLimit || snap.Cleanup.Reason != string(contentUnknown) {
		t.Fatal(count, q.deleted, snap.Cleanup.Reason)
	}
}
func TestManualFileRemovalUsesTheSameMetadataSafetyCheck(t *testing.T) {
	s, q, _ := metadataFixture(t, 1)
	if err := s.Action(context.Background(), strings.Repeat("a", 40), "REMOVE_FILES"); err != nil {
		t.Fatal(err)
	}
	if q.deleted != 1 || !q.deleteFiles {
		t.Fatal(q.hashes)
	}
}
