package downloader

import (
	"context"
	"strings"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type isolationResult string

const (
	contentIsolated     isolationResult = ""
	contentUnknown      isolationResult = "CONTENT_PATH_UNKNOWN"
	contentShared       isolationResult = "SHARED_SOURCE_FILES"
	metadataCheckLimit                  = 16
	metadataCheckBudget                 = 2 * time.Second
)

type torrentFileReader interface {
	GetTorrentContents(context.Context, string, []string) ([]qbittorrent.TorrentContentFile, error)
}

// Evidence belongs to one successful listing only. Never cache empty file lists
// across refreshes: metadata may arrive and reveal a shared content path.
type contentIsolation struct {
	ctx      context.Context
	client   Client
	torrents []qbittorrent.Torrent
	checked  map[string]bool
	requests int
	deadline time.Time
}

func newContentIsolation(ctx context.Context, client Client, torrents []qbittorrent.Torrent) *contentIsolation {
	return &contentIsolation{ctx: ctx, client: client, torrents: torrents, checked: make(map[string]bool)}
}
func metadataWithoutData(t qbittorrent.Torrent) bool {
	return validHash(strings.ToLower(t.Hash)) && (t.State == qbittorrent.TorrentStateMetaDL || t.State == qbittorrent.TorrentStateForcedMetaDL) && t.ContentPath == "" && t.Progress == 0 && t.Completed == 0 && t.Size == 0
}
func (c *contentIsolation) emptyMetadataFiles(t qbittorrent.Torrent) bool {
	hash := strings.ToLower(t.Hash)
	if empty, ok := c.checked[hash]; ok {
		return empty
	}
	c.checked[hash] = false
	reader, ok := c.client.(torrentFileReader)
	if !ok || c.requests >= metadataCheckLimit {
		return false
	}
	if c.deadline.IsZero() {
		c.deadline = time.Now().Add(metadataCheckBudget)
	}
	if c.ctx.Err() != nil || !time.Now().Before(c.deadline) {
		return false
	}
	c.requests++
	ctx, cancel := context.WithDeadline(c.ctx, c.deadline)
	defer cancel()
	files, err := reader.GetTorrentContents(ctx, t.Hash, nil)
	// JSON null is not a confirmed empty list. Require the explicit [] response.
	empty := err == nil && ctx.Err() == nil && files != nil && len(files) == 0
	c.checked[hash] = empty
	return empty
}
func (c *contentIsolation) check(target qbittorrent.Torrent) isolationResult {
	if target.ContentPath == "" {
		return contentUnknown
	}
	// Known overlaps always win, regardless of unrelated metadata requests.
	for _, other := range c.torrents {
		if strings.EqualFold(other.Hash, target.Hash) || other.ContentPath == "" {
			continue
		}
		if within(target.ContentPath, other.ContentPath) || within(other.ContentPath, target.ContentPath) {
			return contentShared
		}
	}
	// Unknown ordinary torrents remain protected; only positively identified,
	// data-free metadata downloads can be exempted after checking their file API.
	for _, other := range c.torrents {
		if strings.EqualFold(other.Hash, target.Hash) || other.ContentPath != "" {
			continue
		}
		if !metadataWithoutData(other) {
			return contentUnknown
		}
	}
	for _, other := range c.torrents {
		if strings.EqualFold(other.Hash, target.Hash) || other.ContentPath != "" {
			continue
		}
		if !c.emptyMetadataFiles(other) {
			return contentUnknown
		}
	}
	return contentIsolated
}

// Recheck the complete qB listing just before an irreversible file removal.
// In particular, a formerly metadata-only torrent may now share source files.
func (s *Service) recheckFileRemoval(ctx context.Context, expected qbittorrent.Torrent, afterIngest bool) isolationResult {
	all, err := s.client.GetTorrentList(ctx, nil)
	if err != nil {
		return contentUnknown
	}
	for _, current := range all {
		if !strings.EqualFold(current.Hash, expected.Hash) {
			continue
		}
		if current.ContentPath != expected.ContentPath || current.SavePath != expected.SavePath {
			return contentUnknown
		}
		if !removableState(string(current.State)) && !(afterIngest && current.State == qbittorrent.TorrentStateMissingFiles) {
			return contentUnknown
		}
		if !afterIngest && current.Progress < 1 {
			return contentUnknown
		}
		return newContentIsolation(ctx, s.client, all).check(current)
	}
	return contentUnknown
}
