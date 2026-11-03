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

	"github.com/google/uuid"
)

type SearchCachePolicy struct {
	Enabled       bool
	TTL, EmptyTTL time.Duration
	MaxBytes      int64
}
type SearchCacheStats struct {
	Entries, ActiveRequests, ActiveWaiters int
	UsedBytes                              int64
}

func (c *SearchCache) Status(policy SearchCachePolicy) (SearchCacheStats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return SearchCacheStats{}, ErrNotConfigured
	}
	if !validCachePolicy(policy) {
		return SearchCacheStats{}, ErrInvalidSettings
	}
	if c.policy != policy {
		c.clearLocked()
		c.policy = policy
	}
	c.pruneLocked(c.now())
	result := SearchCacheStats{UsedBytes: c.used, ActiveRequests: c.active, ActiveWaiters: c.waiters}
	for _, slot := range c.slots {
		if slot.entry != nil {
			result.Entries++
		}
	}
	return result, nil
}

type searchEntry struct {
	data              []byte
	expires, lastUsed time.Time
	instance          string
}
type searchFlight struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	epoch   uint64
	data    []byte
	err     error
}
type searchSlot struct {
	instance string
	epoch    uint64
	active   int
	entry    *searchEntry
	flights  map[CachePolicy]*searchFlight
}

// SearchCache is an application-owned coordinator for exact per-site queries.
// It caches normalized candidates, never acceptance/selection decisions. Raw
// locators stay server-only; even lookup keys use a process-private MAC.
type SearchCache struct {
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	wg      sync.WaitGroup
	secret  [32]byte
	slots   map[string]*searchSlot
	policy  SearchCachePolicy
	used    int64
	active  int
	waiters int
	closed  bool
	now     func() time.Time
	events  EventPublisher
}

// SetEventPublisher is installed before application workers start.
func (c *SearchCache) SetEventPublisher(events EventPublisher) { c.events = events }

func NewSearchCache(owner context.Context) (*SearchCache, error) {
	if owner == nil {
		return nil, ErrNotConfigured
	}
	ctx, cancel := context.WithCancel(owner)
	c := &SearchCache{ctx: ctx, cancel: cancel, slots: map[string]*searchSlot{}, now: time.Now}
	if _, err := rand.Read(c.secret[:]); err != nil {
		cancel()
		return nil, err
	}
	return c, nil
}
func (c *SearchCache) Close() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.slots = map[string]*searchSlot{}
	c.used = 0
	c.mu.Unlock()
	c.wg.Wait()
}
func (c *SearchCache) key(instance InstanceSnapshot, query Query) (string, error) {
	// The complete frozen source also covers network identity and credentials;
	// the digest is never exported. No keyword/case/category normalization here.
	query.InstanceIDs = nil
	query.CachePolicy = ""
	data, err := json.Marshal(struct {
		Instance InstanceSnapshot
		Query    Query
		Day      string
	}{instance, query, c.now().UTC().Format("2006-01-02")})
	if err != nil {
		return "", ErrInvalidSettings
	}
	mac := hmac.New(sha256.New, c.secret[:])
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
func (c *SearchCache) clearLocked() {
	for key, slot := range c.slots {
		slot.epoch++
		slot.entry = nil
		if slot.active == 0 {
			delete(c.slots, key)
		}
	}
	c.used = 0
}

// Clear does not cancel already admitted searches or invalidate durable tasks.
// Epoch fencing prevents older flights from silently repopulating the cache.
func (c *SearchCache) Clear() { c.mu.Lock(); defer c.mu.Unlock(); c.clearLocked() }

func (c *SearchCache) InvalidateInstance(instance string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, slot := range c.slots {
		if slot.instance != instance {
			continue
		}
		slot.epoch++
		if slot.entry != nil {
			c.used -= int64(len(slot.entry.data))
			slot.entry = nil
		}
		if slot.active == 0 {
			delete(c.slots, key)
		}
	}
}

func validCachePolicy(policy SearchCachePolicy) bool {
	return policy.TTL > 0 && policy.TTL <= time.Hour && policy.EmptyTTL > 0 && policy.EmptyTTL <= 5*time.Minute && policy.MaxBytes >= 1<<20 && policy.MaxBytes <= 256<<20
}

// Search merges only identical source/query/policy requests. Cancellation
// detaches one waiter; the shared request stops only when all waiters leave or
// the application closes. The worker owns a bounded site context.
func (c *SearchCache) Search(ctx context.Context, instance InstanceSnapshot, query Query, policy SearchCachePolicy, run func(context.Context) (Batch, error)) (Batch, error) {
	if run == nil || !validCachePolicy(policy) {
		return Batch{}, ErrInvalidSettings
	}
	if err := ctx.Err(); err != nil {
		return Batch{}, err
	}
	key, err := c.key(instance, query)
	if err != nil {
		return Batch{}, err
	}
	mode := query.CachePolicy
	if mode == "" {
		mode = CacheDefault
	}
	if mode != CacheDefault && mode != CacheRefresh {
		return Batch{}, ErrInvalidQuery
	}
	c.mu.Lock()
	if c.closed || c.ctx.Err() != nil {
		c.mu.Unlock()
		return Batch{}, context.Canceled
	}
	if policy != c.policy {
		c.clearLocked()
		c.policy = policy
	}
	c.pruneLocked(c.now())
	slot := c.slots[key]
	if slot == nil {
		slot = &searchSlot{instance: instance.ID, flights: map[CachePolicy]*searchFlight{}}
		c.slots[key] = slot
	}
	if mode == CacheDefault && policy.Enabled && instance.Kind == Builtin && slot.entry != nil {
		entry := slot.entry
		entry.lastUsed = c.now()
		data := entry.data
		c.mu.Unlock()
		return decodeSearchBatch(data, true)
	}
	flight := slot.flights[mode]
	if flight == nil || flight.epoch != slot.epoch {
		if c.active >= 64 {
			if slot.active == 0 && slot.entry == nil {
				delete(c.slots, key)
			}
			c.mu.Unlock()
			return Batch{}, ErrBusy
		}
		if mode == CacheRefresh {
			slot.epoch++
			if slot.entry != nil {
				c.used -= int64(len(slot.entry.data))
				slot.entry = nil
			}
		}
		timeout := instance.SiteTimeout
		if timeout <= 0 || timeout > 10*time.Minute {
			timeout = 2 * time.Minute
		}
		worker, cancel := context.WithTimeout(c.ctx, timeout)
		flight = &searchFlight{done: make(chan struct{}), cancel: cancel, epoch: slot.epoch}
		slot.flights[mode] = flight
		slot.active++
		c.active++
		c.wg.Add(1)
		go c.execute(worker, key, slot, mode, flight, instance, policy, run)
	}
	flight.waiters++
	c.waiters++
	c.mu.Unlock()
	if c.events != nil {
		c.events.Publish(EventCache)
	}
	defer func() {
		c.mu.Lock()
		c.waiters--
		c.mu.Unlock()
		if c.events != nil {
			c.events.Publish(EventCache)
		}
	}()
	select {
	case <-ctx.Done():
		c.detach(key, slot, mode, flight)
		return Batch{}, ctx.Err()
	case <-flight.done:
		if err := ctx.Err(); err != nil {
			return Batch{}, err
		}
		batch, err := decodeSearchBatch(flight.data, false)
		if err != nil {
			return Batch{}, err
		}
		return batch, flight.err
	}
}
func (c *SearchCache) detach(key string, slot *searchSlot, mode CachePolicy, flight *searchFlight) {
	c.mu.Lock()
	defer c.mu.Unlock()
	flight.waiters--
	if flight.waiters == 0 {
		flight.cancel()
		if slot.flights[mode] == flight {
			delete(slot.flights, mode)
		}
	}
}
func (c *SearchCache) execute(ctx context.Context, key string, slot *searchSlot, mode CachePolicy, flight *searchFlight, instance InstanceSnapshot, policy SearchCachePolicy, run func(context.Context) (Batch, error)) {
	defer c.wg.Done()
	defer flight.cancel()
	batch, err := run(ctx)
	if batch.FetchedAt.IsZero() && instance.Kind == Builtin {
		batch.FetchedAt = c.now().UTC()
	}
	if batch.release != nil {
		defer batch.release()
	}
	data, encodeErr := json.Marshal(batch)
	if encodeErr != nil || len(data) > 8<<20 {
		err = ErrResponseBudget
		data = []byte(`{}`)
	}
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		if c.events != nil {
			c.events.Publish(EventCache)
		}
	}()
	flight.data, flight.err = data, err
	slot.active--
	c.active--
	if slot.flights[mode] == flight {
		delete(slot.flights, mode)
	}
	if !c.closed && ctx.Err() == nil && flight.waiters > 0 && slot.epoch == flight.epoch && policy == c.policy && policy.Enabled && instance.Kind == Builtin && err == nil && !batch.Partial && !batch.Truncated && batch.IssueCount == 0 && int64(len(data)) <= policy.MaxBytes {
		ttl := policy.TTL
		if len(batch.Releases) == 0 {
			ttl = policy.EmptyTTL
		}
		if slot.entry != nil {
			c.used -= int64(len(slot.entry.data))
		}
		slot.entry = &searchEntry{data: data, expires: c.now().Add(ttl), lastUsed: c.now(), instance: instance.ID}
		c.used += int64(len(data))
		c.trimLocked(instance.ID, policy.MaxBytes)
	}
	if slot.active == 0 && slot.entry == nil && c.slots[key] == slot {
		delete(c.slots, key)
	}
	close(flight.done)
}
func (c *SearchCache) pruneLocked(now time.Time) {
	for key, slot := range c.slots {
		if slot.entry != nil && !now.Before(slot.entry.expires) {
			c.used -= int64(len(slot.entry.data))
			slot.entry = nil
		}
		if slot.active == 0 && slot.entry == nil {
			delete(c.slots, key)
		}
	}
}
func (c *SearchCache) trimLocked(instance string, maxBytes int64) {
	for {
		count, total := 0, 0
		for _, slot := range c.slots {
			if slot.entry != nil {
				total++
				if slot.entry.instance == instance {
					count++
				}
			}
		}
		if c.used <= maxBytes && count <= 100 && total <= 2048 {
			return
		}
		oldest := ""
		for key, slot := range c.slots {
			if slot.entry != nil && (count <= 100 || slot.entry.instance == instance) && (oldest == "" || slot.entry.lastUsed.Before(c.slots[oldest].entry.lastUsed)) {
				oldest = key
			}
		}
		if oldest == "" {
			return
		}
		slot := c.slots[oldest]
		c.used -= int64(len(slot.entry.data))
		slot.entry = nil
		if slot.active == 0 {
			delete(c.slots, oldest)
		}
	}
}
func decodeSearchBatch(data []byte, hit bool) (Batch, error) {
	var batch Batch
	if err := json.Unmarshal(data, &batch); err != nil {
		return Batch{}, ErrReference
	}
	batch.CacheHit = hit
	// Each acceptance path registers its own short-lived resource reference.
	// Candidate identity/locator are retained; opaque IDs are never reused.
	for i := range batch.Releases {
		batch.Releases[i].Resource.ID = uuid.NewString()
	}
	return batch, nil
}
