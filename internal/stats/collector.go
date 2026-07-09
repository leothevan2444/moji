// Package stats periodically polls external services (Stash, Jackett,
// qBittorrent) and exposes a snapshot suitable for direct mapping onto the
// Moji GraphQL service-stats types.
//
// The collector runs a single goroutine with two ticker intervals:
//   - intervalFast (default 10s) — qBittorrent transfer speed + local scan-queue
//     aggregation. Cheap and users expect liveness here.
//   - intervalSlow (default 120s) — Stash library count + version, Jackett
//     indexer enumeration. Costlier; we don't need minute-level freshness.
//
// On any error, the previously-good snapshot is preserved and `lastError` is
// set on the affected stat. This avoids data flapping when an upstream
// connection hiccups for a few seconds.
//
// Concurrency: Snapshot() follows the same RWMutex + clone-on-read pattern as
// [internal/config/store.go]. Snapshot values are immutable after publish;
// callers must not mutate returned structs.
package stats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/downloader"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
	"github.com/leothevan2444/moji/pkg/stash"
)

// TaskLister is the subset of *downloader.Service we depend on, kept narrow so
// tests can pass a fake without spinning up a SQLite store.
type TaskLister interface {
	ListTasks(ctx context.Context) ([]*downloader.Task, error)
}

// StashStats is the snapshot value exposed to the GraphQL layer. All fields
// except lastError and okAt are zero-valued until first successful refresh.
type StashStats struct {
	Version              string
	SceneCount           *int
	PendingMojiScanCount int
	LastError            string
	OKAt                 time.Time
}

// JackettStats mirrors the GraphQL JackettStats shape.
type JackettStats struct {
	IndexerCount           int
	ConfiguredIndexerCount int
	LastIndexerLatencyMs   int
	LastIndexerError       string
	LastIndexerSearchAt    *time.Time
	LastError              string
	OKAt                   time.Time
}

// QBittorrentStats mirrors the GraphQL QBittorrentStats shape.
type QBittorrentStats struct {
	DownloadSpeed        int64
	UploadSpeed          int64
	ActiveTorrentCount   int
	ConnectionStatus     string
	AltSpeedLimitEnabled bool
	LastError            string
	OKAt                 time.Time
}

// Snapshot is the immutable, race-safe view returned by Collector.Snapshot.
type Snapshot struct {
	mu         sync.RWMutex
	Stash      StashStats
	Jackett    JackettStats
	QBitt      QBittorrentStats
	lastSearch []jackett.IndexerStatus // latest hook payload, copied on read
	searchAt   *time.Time
}

// Clone returns a deep-enough copy safe to hand to GraphQL serialisation.
func (s *Snapshot) Clone() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Snapshot{
		Stash:   s.Stash,
		Jackett: s.Jackett,
		QBitt:   s.QBitt,
	}
	if s.searchAt != nil {
		t := *s.searchAt
		out.searchAt = &t
	}
	if len(s.lastSearch) > 0 {
		out.lastSearch = append([]jackett.IndexerStatus(nil), s.lastSearch...)
	}
	return out
}

// Collector periodically refreshes a Snapshot. A nil client means "skip that
// service" — useful in tests or partial configurations.
type Collector struct {
	Snapshot *Snapshot

	stashClient   *stash.Client
	jackettClient *jackett.Client
	qbittClient   *qbittorrent.Client
	taskLister    TaskLister
	intervalFast  time.Duration
	intervalSlow  time.Duration
	logger        *slog.Logger

	// searchHookInstalled guards against double-install.
	hookOnce sync.Once
}

// NewCollector wires a collector with the given clients. Any client may be nil
// (the collector skips refresh for that service).
func NewCollector(
	stashClient *stash.Client,
	jackettClient *jackett.Client,
	qbittClient *qbittorrent.Client,
	taskLister TaskLister,
	logger *slog.Logger,
) *Collector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Collector{
		Snapshot:      &Snapshot{},
		stashClient:   stashClient,
		jackettClient: jackettClient,
		qbittClient:   qbittClient,
		taskLister:    taskLister,
		intervalFast:  10 * time.Second,
		intervalSlow:  120 * time.Second,
		logger:        logger,
	}
}

// SetIntervals overrides the default poll intervals. Primarily for tests.
func (c *Collector) SetIntervals(fast, slow time.Duration) {
	if fast > 0 {
		c.intervalFast = fast
	}
	if slow > 0 {
		c.intervalSlow = slow
	}
}

// Run blocks until ctx is canceled. The first refresh is performed
// synchronously so the GraphQL layer has data immediately on startup.
func (c *Collector) Run(ctx context.Context) {
	c.installHookOnce()

	if err := c.refreshFast(ctx); err != nil {
		c.logger.Warn("stats: initial fast refresh failed", slog.Any("error", err))
	}
	if err := c.refreshSlow(ctx); err != nil {
		c.logger.Warn("stats: initial slow refresh failed", slog.Any("error", err))
	}

	fastTicker := time.NewTicker(c.intervalFast)
	slowTicker := time.NewTicker(c.intervalSlow)
	defer fastTicker.Stop()
	defer slowTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fastTicker.C:
			if err := c.refreshFast(ctx); err != nil {
				c.logger.Warn("stats: fast refresh failed", slog.Any("error", err))
			}
		case <-slowTicker.C:
			if err := c.refreshSlow(ctx); err != nil {
				c.logger.Warn("stats: slow refresh failed", slog.Any("error", err))
			}
		}
	}
}

func (c *Collector) installHookOnce() {
	c.hookOnce.Do(func() {
		if c.jackettClient == nil {
			return
		}
		jackett.SetIndexerStatusHook(func(statuses []jackett.IndexerStatus, _ string) {
			snap := c.Snapshot
			snap.mu.Lock()
			defer snap.mu.Unlock()
			snap.lastSearch = statuses
			now := time.Now()
			snap.searchAt = &now
		})
	})
}

// Snapshot returns a clone of the current snapshot. Callers may read it freely;
// it never mutates.
func (c *Collector) SnapshotView() Snapshot {
	return c.Snapshot.Clone()
}

func (c *Collector) refreshFast(ctx context.Context) error {
	if c.qbittClient != nil {
		c.refreshQBittorrent(ctx)
	}
	c.refreshMojiScanCount(ctx)
	c.refreshJackettSearchLatency()
	return nil
}

func (c *Collector) refreshSlow(ctx context.Context) error {
	if c.stashClient != nil {
		c.refreshStash(ctx)
	}
	if c.jackettClient != nil {
		c.refreshJackettIndexers(ctx)
	}
	return nil
}

func (c *Collector) refreshStash(ctx context.Context) {
	snap := c.Snapshot
	snap.mu.Lock()
	defer snap.mu.Unlock()

	now := time.Now()
	if v, err := c.stashClient.GetVersion(ctx); err != nil {
		snap.Stash.LastError = fmt.Sprintf("version: %v", err)
	} else if v != nil {
		if version := v.GetVersion(); version != nil {
			snap.Stash.Version = *version
		}
	}

	if n, err := c.stashClient.GetSceneCount(ctx); err != nil {
		snap.Stash.LastError = appendErr(snap.Stash.LastError, fmt.Sprintf("sceneCount: %v", err))
	} else {
		count := n
		snap.Stash.SceneCount = &count
		snap.Stash.LastError = ""
		snap.Stash.OKAt = now
	}
}

func (c *Collector) refreshQBittorrent(ctx context.Context) {
	snap := c.Snapshot
	snap.mu.Lock()
	defer snap.mu.Unlock()

	now := time.Now()
	if info, err := c.qbittClient.GetGlobalTransferInfo(ctx); err != nil {
		snap.QBitt.LastError = fmt.Sprintf("transfer: %v", err)
	} else if info != nil {
		snap.QBitt.DownloadSpeed = info.DLInfoSpeed
		snap.QBitt.UploadSpeed = info.UPInfoSpeed
		snap.QBitt.ConnectionStatus = info.ConnectionStatus
		snap.QBitt.AltSpeedLimitEnabled = info.UseAltSpeedLimits
		snap.QBitt.LastError = ""
		snap.QBitt.OKAt = now
	}

	if torrents, err := c.qbittClient.GetTorrentList(ctx, &qbittorrent.TorrentListOptions{Filter: "active"}); err != nil {
		snap.QBitt.LastError = appendErr(snap.QBitt.LastError, fmt.Sprintf("torrents: %v", err))
	} else {
		snap.QBitt.ActiveTorrentCount = len(torrents)
	}
}

func (c *Collector) refreshMojiScanCount(ctx context.Context) {
	if c.taskLister == nil {
		return
	}
	snap := c.Snapshot
	snap.mu.Lock()
	defer snap.mu.Unlock()

	tasks, err := c.taskLister.ListTasks(ctx)
	if err != nil {
		snap.Stash.LastError = appendErr(snap.Stash.LastError, fmt.Sprintf("tasks: %v", err))
		return
	}
	count := 0
	for _, t := range tasks {
		if t == nil {
			continue
		}
		if t.Stage == downloader.TaskStagePendingIngest ||
			t.Stage == downloader.TaskStageTransferring ||
			t.Stage == downloader.TaskStageScanning {
			count++
		}
	}
	snap.Stash.PendingMojiScanCount = count
}

func (c *Collector) refreshJackettIndexers(ctx context.Context) {
	snap := c.Snapshot
	snap.mu.Lock()
	defer snap.mu.Unlock()

	now := time.Now()
	indexers, err := c.jackettClient.GetIndexers()
	if err != nil {
		snap.Jackett.LastError = fmt.Sprintf("indexers: %v", err)
		return
	}
	total := len(indexers)
	configured := 0
	for _, idx := range indexers {
		if idx.Configured {
			configured++
		}
	}
	snap.Jackett.IndexerCount = total
	snap.Jackett.ConfiguredIndexerCount = configured
	snap.Jackett.LastError = ""
	snap.Jackett.OKAt = now
}

// refreshJackettSearchLatency rolls the latest search-result telemetry into
// the JackettStats fields. Called from the fast ticker because it's cheap.
func (c *Collector) refreshJackettSearchLatency() {
	snap := c.Snapshot
	snap.mu.Lock()
	defer snap.mu.Unlock()

	if snap.searchAt == nil {
		return
	}
	// Copy in the latest statuses. We avoid holding onto the raw hook buffer
	// across mutations by reading lastSearch (already a slice header; the
	// hook itself writes under the same mutex so this is safe).
	var worstMs int
	var lastErr string
	for _, s := range snap.lastSearch {
		if s.Error != "" {
			lastErr = s.Error
		}
		if s.ElapsedTime > worstMs {
			worstMs = s.ElapsedTime
		}
	}
	snap.Jackett.LastIndexerLatencyMs = worstMs
	snap.Jackett.LastIndexerError = lastErr
	at := *snap.searchAt
	snap.Jackett.LastIndexerSearchAt = &at
}

// appendErr joins two error strings with a separator. Returns empty if both
// inputs are empty.
func appendErr(existing, next string) string {
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	return existing + "; " + next
}
