package indexer

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/torrentmeta"
)

const artifactTTL = 10 * time.Minute
const artifactMaxBytes = 64 << 20
const artifactMaxEntries = 512
const artifactMaxOperations = 64

type artifactBlob struct {
	bytes            []byte
	magnet, infoHash string
	refs             int
	size             int64
}
type artifactEntry struct {
	blob                        string
	acquired, expires, lastUsed time.Time
}
type artifactFlight struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	epoch   uint64
	result  Artifact
	err     error
}

// ArtifactCache owns bounded shared acquisition work and successful original
// data. It contains no task decisions and never creates a downloader effect.
// Keys and blobs stay private to the process and to the frozen source identity.
type ArtifactCache struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	wg      sync.WaitGroup
	secret  [32]byte
	entries map[string]*artifactEntry
	blobs   map[string]*artifactBlob
	flights map[string]*artifactFlight
	used    int64
	waiters int
	closed  bool
	epoch   uint64
	now     func() time.Time
}

func NewArtifactCache(owner context.Context) (*ArtifactCache, error) {
	if owner == nil {
		return nil, ErrNotConfigured
	}
	ctx, cancel := context.WithCancel(owner)
	c := &ArtifactCache{ctx: ctx, cancel: cancel, entries: map[string]*artifactEntry{}, blobs: map[string]*artifactBlob{}, flights: map[string]*artifactFlight{}, now: time.Now}
	if _, err := rand.Read(c.secret[:]); err != nil {
		cancel()
		return nil, err
	}
	return c, nil
}
func (c *ArtifactCache) digest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", ErrReference
	}
	mac := hmac.New(sha256.New, c.secret[:])
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
func (c *ArtifactCache) keys(instance InstanceSnapshot, ref ResourceRef) (string, string, error) {
	ref.ID = ""
	key, err := c.digest(ResourceRecord{Version: 1, Instance: instance, Ref: ref})
	if err != nil {
		return "", "", err
	}
	ref.Locator = Locator{}
	scope, err := c.digest(ResourceRecord{Version: 1, Instance: instance, Ref: ref})
	return key, scope, err
}
func cloneArtifact(source Artifact, ref ResourceRef) Artifact {
	out := source
	out.TorrentBytes = append([]byte(nil), source.TorrentBytes...)
	out.Resource = ref
	if ref.Locator.Fields != nil {
		out.Resource.Locator.Fields = map[string]string{}
		for k, v := range ref.Locator.Fields {
			out.Resource.Locator.Fields[k] = v
		}
	}
	return out
}
func (c *ArtifactCache) Fetch(ctx context.Context, instance InstanceSnapshot, ref ResourceRef, run func(context.Context) (Artifact, error)) (Artifact, error) {
	if run == nil {
		return Artifact{}, ErrNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	// Own mutable caller configuration and locator maps before crossing goroutines.
	instance = cloneInstance(instance)
	owned := ResourceRecord{Ref: ref}.Clone()
	ref = owned.Ref
	key, scope, err := c.keys(instance, ref)
	if err != nil {
		return Artifact{}, err
	}
	c.mu.Lock()
	if c.closed || c.ctx.Err() != nil {
		c.mu.Unlock()
		return Artifact{}, context.Canceled
	}
	c.pruneLocked(c.now())
	if entry := c.entries[key]; entry != nil {
		entry.lastUsed = c.now()
		blob := c.blobs[entry.blob]
		result := cloneArtifact(Artifact{TorrentBytes: blob.bytes, Magnet: blob.magnet, infoHash: blob.infoHash, AcquiredAt: entry.acquired}, ref)
		c.mu.Unlock()
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		return result, nil
	}
	if c.waiters >= artifactMaxOperations {
		c.mu.Unlock()
		return Artifact{}, ErrBusy
	}
	flight := c.flights[key]
	if flight != nil && flight.waiters == 0 {
		c.mu.Unlock()
		return Artifact{}, ErrBusy
	}
	if flight == nil {
		if len(c.flights) >= artifactMaxOperations {
			c.mu.Unlock()
			return Artifact{}, ErrBusy
		}
		timeout := instance.SiteTimeout
		if timeout <= 0 || timeout > 10*time.Minute {
			timeout = 2 * time.Minute
		}
		worker, cancel := context.WithTimeout(c.ctx, timeout)
		flight = &artifactFlight{done: make(chan struct{}), cancel: cancel, epoch: c.epoch}
		c.flights[key] = flight
		c.wg.Add(1)
		go c.execute(worker, key, scope, flight, run)
	}
	flight.waiters++
	c.waiters++
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.waiters--
		flight.waiters--
		if flight.waiters == 0 {
			flight.cancel()
		}
		c.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return Artifact{}, ctx.Err()
	case <-flight.done:
		if err := c.ctx.Err(); err != nil {
			return Artifact{}, err
		}
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		return cloneArtifact(flight.result, ref), flight.err
	}
}
func (c *ArtifactCache) execute(ctx context.Context, key, scope string, f *artifactFlight, run func(context.Context) (Artifact, error)) {
	defer c.wg.Done()
	defer f.cancel()
	result, err := run(ctx)
	if err == nil {
		result, err = ValidateArtifact(result)
	}
	hash := result.infoHash
	if err != nil {
		result = Artifact{}
	} else {
		result = cloneArtifact(result, ResourceRef{})
	}
	if result.AcquiredAt.IsZero() && err == nil {
		result.AcquiredAt = c.now().UTC()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	f.result, f.err = result, err
	// A known successful late response can still prevent another grab after its
	// final waiter left. Shutdown/clear fences prevent retaining obsolete data.
	if !c.closed && c.ctx.Err() == nil && f.epoch == c.epoch && err == nil {
		sum := sha256.Sum256(result.TorrentBytes)
		blobKey, _ := c.digest(struct{ Scope, Hash, Bytes, Magnet string }{scope, hash, hex.EncodeToString(sum[:]), result.Magnet})
		if c.now().Before(result.AcquiredAt.Add(artifactTTL)) {
			if c.entries[key] != nil {
				c.removeLocked(key)
			}
			blob := c.blobs[blobKey]
			if blob == nil {
				blob = &artifactBlob{bytes: result.TorrentBytes, magnet: result.Magnet, infoHash: hash, size: int64(len(result.TorrentBytes) + len(result.Magnet) + len(hash) + 128)}
				c.blobs[blobKey] = blob
				c.used += blob.size
			}
			blob.refs++
			c.entries[key] = &artifactEntry{blob: blobKey, acquired: result.AcquiredAt, expires: result.AcquiredAt.Add(artifactTTL), lastUsed: c.now()}
			for c.used > artifactMaxBytes || len(c.entries) > artifactMaxEntries {
				c.removeOldestLocked()
			}
		}
	}
	delete(c.flights, key)
	close(f.done)
}
func (c *ArtifactCache) removeLocked(key string) {
	entry := c.entries[key]
	if entry == nil {
		return
	}
	delete(c.entries, key)
	blob := c.blobs[entry.blob]
	blob.refs--
	if blob.refs == 0 {
		c.used -= blob.size
		delete(c.blobs, entry.blob)
	}
}
func (c *ArtifactCache) removeOldestLocked() {
	oldest := ""
	for key, e := range c.entries {
		if oldest == "" || e.lastUsed.Before(c.entries[oldest].lastUsed) {
			oldest = key
		}
	}
	if oldest != "" {
		c.removeLocked(oldest)
	}
}
func (c *ArtifactCache) pruneLocked(now time.Time) {
	for key, e := range c.entries {
		if !now.Before(e.expires) {
			c.removeLocked(key)
		}
	}
}

// Clear drops only disposable acquisition data, not durable tasks or secrets.
// It does not cancel admitted work or start a parallel replacement request.
func (c *ArtifactCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	c.entries = map[string]*artifactEntry{}
	c.blobs = map[string]*artifactBlob{}
	c.used = 0
}
func (c *ArtifactCache) Close() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.epoch++
	c.entries = map[string]*artifactEntry{}
	c.blobs = map[string]*artifactBlob{}
	c.used = 0
	c.mu.Unlock()
	c.wg.Wait()
}

// ValidateArtifact checks original torrent bytes or a bounded magnet result.
func ValidateArtifact(result Artifact) (Artifact, error) {
	if len(result.TorrentBytes) > 0 && result.Magnet == "" {
		if result.infoHash == "" {
			parsed, err := torrentmeta.Parse(result.TorrentBytes)
			if err != nil {
				return Artifact{}, ErrReference
			}
			result.infoHash = parsed.InfoHash
		}
	} else if len(result.TorrentBytes) == 0 && result.Magnet != "" {
		magnet, err := torrentmeta.ParseMagnet(result.Magnet)
		if err != nil {
			return Artifact{}, ErrReference
		}
		result.Magnet, result.infoHash = magnet.URI, magnet.InfoHash
	} else {
		return Artifact{}, ErrReference
	}
	return result, nil
}
