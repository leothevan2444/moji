package downloader

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type isolationResult string

const (
	contentIsolated     isolationResult = ""
	contentUnknown      isolationResult = "CONTENT_PATH_UNKNOWN"
	contentShared       isolationResult = "SHARED_SOURCE_FILES"
	contentCheckTimeout isolationResult = "CONTENT_CHECK_TIMEOUT"
	contentCheckFailed  isolationResult = "CONTENT_CHECK_FAILED"
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
	checked  map[string]isolationResult
	deadline time.Time
}

func newContentIsolation(ctx context.Context, client Client, torrents []qbittorrent.Torrent) *contentIsolation {
	return &contentIsolation{ctx: ctx, client: client, torrents: torrents, checked: make(map[string]isolationResult)}
}
func canVerifyEmptyFiles(t qbittorrent.Torrent) bool {
	switch t.State {
	case qbittorrent.TorrentStateMetaDL, qbittorrent.TorrentStateForcedMetaDL,
		qbittorrent.TorrentStatePausedDL, qbittorrent.TorrentState("stoppedDL"):
		return validHash(strings.ToLower(t.Hash)) && t.ContentPath == "" && t.Progress == 0 && t.Completed == 0 && t.Size == 0
	default:
		return false
	}
}
func (c *contentIsolation) emptyFiles(t qbittorrent.Torrent) isolationResult {
	hash := strings.ToLower(t.Hash)
	if result, ok := c.checked[hash]; ok {
		return result
	}
	result := c.verifyEmptyFiles(t)
	c.checked[hash] = result
	return result
}

func (c *contentIsolation) verifyEmptyFiles(t qbittorrent.Torrent) isolationResult {
	reader, ok := c.client.(torrentFileReader)
	if !ok {
		return contentCheckFailed
	}
	if c.deadline.IsZero() {
		c.deadline = time.Now().Add(metadataCheckBudget)
	}
	ctx, cancel := context.WithDeadline(c.ctx, c.deadline)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return contentCheckError(err)
	}
	files, err := reader.GetTorrentContents(ctx, t.Hash, nil)
	if ctx.Err() != nil {
		return contentCheckError(ctx.Err())
	}
	if err != nil {
		return contentCheckError(err)
	}
	// JSON null is not a confirmed empty list. Require the explicit [] response.
	if files == nil || len(files) != 0 {
		return contentUnknown
	}
	return contentIsolated
}

func contentCheckError(err error) isolationResult {
	if errors.Is(err, context.DeadlineExceeded) {
		return contentCheckTimeout
	}
	return contentCheckFailed
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
	// Unknown ordinary torrents remain protected. Metadata downloads and stopped
	// downloads without data require an explicitly empty file list to be exempted.
	for _, other := range c.torrents {
		if strings.EqualFold(other.Hash, target.Hash) || other.ContentPath != "" {
			continue
		}
		if !canVerifyEmptyFiles(other) {
			return contentUnknown
		}
	}
	for _, other := range c.torrents {
		if strings.EqualFold(other.Hash, target.Hash) || other.ContentPath != "" {
			continue
		}
		if result := c.emptyFiles(other); result != contentIsolated {
			return result
		}
	}
	return contentIsolated
}

// Recheck the complete qB listing just before an irreversible file removal.
// In particular, a formerly metadata-only torrent may now share source files.
func (s *Service) recheckFileRemoval(ctx context.Context, expected qbittorrent.Torrent, afterIngest bool) isolationResult {
	all, err := s.client.GetTorrentList(ctx, nil)
	if err != nil {
		return contentCheckError(err)
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
