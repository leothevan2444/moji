// Package indexerdownloads owns short-lived, bounded handles for explicitly
// acquired artifacts. Handles never locate arbitrary URLs or create tasks.
package indexerdownloads

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/torrentmeta"
)

const (
	lifetime     = 5 * time.Minute
	maxBytes     = 64 << 20
	maxArtifact  = 8 << 20
	maxEntries   = 256
	maxPreparing = 8
	maxReaders   = 16
)

type Fetcher interface {
	FetchReference(context.Context, string) (indexer.Artifact, error)
}
type Receipt struct {
	Handle, Kind string
	ExpiresAt    time.Time
}
type entry struct {
	receipt Receipt
	body    []byte
	readers int
}
type Service struct {
	owner              context.Context
	cancel             context.CancelFunc
	fetcher            Fetcher
	mu                 sync.Mutex
	wg                 sync.WaitGroup
	entries            map[string]*entry
	used               int
	preparing, readers int
	closed             bool
	now                func() time.Time
}

func New(owner context.Context, fetcher Fetcher) (*Service, error) {
	if owner == nil || fetcher == nil {
		return nil, indexer.ErrNotConfigured
	}
	ctx, cancel := context.WithCancel(owner)
	return &Service{owner: ctx, cancel: cancel, fetcher: fetcher, entries: map[string]*entry{}, now: time.Now}, nil
}

// Prepare acquires through the existing resource registry/cache. Admission
// reserves the maximum artifact size before network work; live handles are not
// evicted to make room for later requests.
func (s *Service) Prepare(ctx context.Context, resourceID string) (Receipt, error) {
	if len(resourceID) == 0 || len(resourceID) > 128 {
		return Receipt{}, indexer.ErrReference
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	s.mu.Lock()
	s.pruneLocked()
	if s.closed || s.owner.Err() != nil {
		s.mu.Unlock()
		return Receipt{}, context.Canceled
	}
	if s.preparing >= maxPreparing || len(s.entries)+s.preparing >= maxEntries || s.used+(s.preparing+1)*maxArtifact > maxBytes {
		s.mu.Unlock()
		return Receipt{}, indexer.ErrBusy
	}
	s.preparing++
	s.wg.Add(1)
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.preparing--; s.mu.Unlock(); s.wg.Done() }()
	work, cancel := context.WithTimeout(ctx, 2*time.Minute)
	stop := context.AfterFunc(s.owner, cancel)
	defer stop()
	defer cancel()
	artifact, err := s.fetcher.FetchReference(work, resourceID)
	if err != nil {
		return Receipt{}, err
	}
	var body []byte
	kind := "TORRENT"
	if len(artifact.TorrentBytes) > 0 && artifact.Magnet == "" && len(artifact.TorrentBytes) <= maxArtifact {
		if _, err := torrentmeta.Parse(artifact.TorrentBytes); err != nil {
			return Receipt{}, indexer.ErrReference
		}
		body = append([]byte(nil), artifact.TorrentBytes...)
	} else if len(artifact.TorrentBytes) == 0 && strings.HasPrefix(artifact.Magnet, "magnet:?") && len(artifact.Magnet) <= 64<<10 && !strings.ContainsAny(artifact.Magnet, "\r\n\x00") {
		kind = "MAGNET"
		target, err := url.Parse(artifact.Magnet)
		if err != nil || target.Host != "" || target.Fragment != "" {
			return Receipt{}, indexer.ErrReference
		}
		query, err := url.ParseQuery(target.RawQuery)
		if err != nil || query.Get("xt") == "" {
			return Receipt{}, indexer.ErrReference
		}
		body = []byte(artifact.Magnet)
	} else {
		return Receipt{}, indexer.ErrReference
	}
	var token [32]byte
	if _, err = rand.Read(token[:]); err != nil {
		return Receipt{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.owner.Err() != nil {
		return Receipt{}, context.Canceled
	}
	if err = work.Err(); err != nil {
		return Receipt{}, err
	}
	receipt := Receipt{Handle: hex.EncodeToString(token[:]), Kind: kind, ExpiresAt: s.now().UTC().Add(lifetime)}
	if _, exists := s.entries[receipt.Handle]; exists {
		return Receipt{}, indexer.ErrBusy
	}
	s.entries[receipt.Handle] = &entry{receipt: receipt, body: body}
	s.used += len(body)
	return receipt, nil
}

type Lease struct {
	Reader                io.Reader
	ContentType, Filename string
	release               func()
	once                  sync.Once
}

func (l *Lease) Release() {
	if l != nil {
		l.once.Do(l.release)
	}
}

// Open exposes an immutable reader, never the stored backing slice. The caller
// must release it after writing the response. Active readers retain their byte
// budget even when the handle expires.
func (s *Service) Open(handle string) (*Lease, error) {
	if len(handle) != 64 {
		return nil, indexer.ErrExpired
	}
	if _, err := hex.DecodeString(handle); err != nil {
		return nil, indexer.ErrExpired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	if s.closed || s.owner.Err() != nil {
		return nil, context.Canceled
	}
	e := s.entries[handle]
	if e == nil || !s.now().Before(e.receipt.ExpiresAt) {
		return nil, indexer.ErrExpired
	}
	if s.readers >= maxReaders {
		return nil, indexer.ErrBusy
	}
	s.readers++
	e.readers++
	content, filename := "application/x-bittorrent", "resource.torrent"
	if e.receipt.Kind == "MAGNET" {
		content, filename = "text/uri-list; charset=utf-8", "resource.magnet"
	}
	return &Lease{Reader: bytes.NewReader(e.body), ContentType: content, Filename: filename, release: func() { s.mu.Lock(); defer s.mu.Unlock(); s.readers--; e.readers--; s.pruneLocked() }}, nil
}
func (s *Service) pruneLocked() {
	now := s.now()
	for id, e := range s.entries {
		if e.readers == 0 && !now.Before(e.receipt.ExpiresAt) {
			s.used -= len(e.body)
			delete(s.entries, id)
		}
	}
}
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.wg.Wait()
	s.mu.Lock()
	s.entries = map[string]*entry{}
	s.used = 0
	s.mu.Unlock()
}
