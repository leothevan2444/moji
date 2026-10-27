package storage

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

const PerformanceInterval = 5 * time.Second

type DatabasePoolMetrics struct {
	Open             int
	InUse            int
	Idle             int
	Limit            int
	WaitCount        int64
	WaitMilliseconds float64
}
type DatabasePerformance struct {
	Role                         string
	DatabaseBytes                int64
	WALBytes                     int64
	FreeBytes                    int64
	WriteAttempts                int64
	WriteErrors                  int64
	BusyRetries                  int64
	QueueRejected                int64
	QueueCanceled                int64
	Queued                       int
	QueueCapacity                int
	WritesPerSecond              float64
	AverageWaitMilliseconds      float64
	AverageExecutionMilliseconds float64
	TotalWaitMilliseconds        float64
	TotalExecutionMilliseconds   float64
	Reader                       *DatabasePoolMetrics
	Writer                       *DatabasePoolMetrics
}
type DatabasePerformanceSnapshot struct {
	StartedAt       string
	SampledAt       string
	IntervalSeconds float64
	LastError       string
	Databases       []*DatabasePerformance
}
type performanceSubscriber struct {
	channel chan *DatabasePerformanceSnapshot
	done    chan struct{}
}

type performanceMonitor struct {
	subscribers map[uint64]performanceSubscriber
	nextID      uint64
	closed      bool
	sampleMu    sync.Mutex
	mu          sync.RWMutex
	snapshot    DatabasePerformanceSnapshot
	previous    map[string]Metrics
	previousAt  time.Time
}

func poolMetrics(s sql.DBStats) *DatabasePoolMetrics {
	return &DatabasePoolMetrics{Open: s.OpenConnections, InUse: s.InUse, Idle: s.Idle, Limit: s.MaxOpenConnections, WaitCount: s.WaitCount, WaitMilliseconds: float64(s.WaitDuration) / float64(time.Millisecond)}
}

// PerformanceSnapshot only reads bounded in-memory state from the live owner.
// HTTP requests never open another manager, acquire its lock or query SQLite.
func (m *Manager) PerformanceSnapshot() DatabasePerformanceSnapshot {
	m.performance.mu.RLock()
	defer m.performance.mu.RUnlock()
	return clonePerformanceSnapshot(m.performance.snapshot)
}

func clonePerformanceSnapshot(snapshot DatabasePerformanceSnapshot) DatabasePerformanceSnapshot {
	s := snapshot
	s.Databases = make([]*DatabasePerformance, 0, len(snapshot.Databases))
	for _, row := range snapshot.Databases {
		copy := *row
		reader, writer := *row.Reader, *row.Writer
		copy.Reader = &reader
		copy.Writer = &writer
		s.Databases = append(s.Databases, &copy)
	}
	return s
}

// SamplePerformance measures only metadata and counters; it performs no writes,
// checkpoints or backups. A failed sample retains the previous timestamp/data.
func (m *Manager) SamplePerformance(ctx context.Context) error {
	m.performance.sampleMu.Lock()
	defer m.performance.sampleMu.Unlock()
	status, err := m.Status(ctx)
	now := time.Now()
	m.performance.mu.Lock()
	defer func() { m.performance.publishLocked(); m.performance.mu.Unlock() }()
	if err != nil {
		m.performance.snapshot.LastError = err.Error()
		return err
	}
	elapsed := now.Sub(m.performance.previousAt).Seconds()
	var rows []*DatabasePerformance
	for _, role := range []string{"core", "runtime", "cache"} {
		s := status[role]
		v := s.Metrics
		previous := m.performance.previous[role]
		count := v.Writes - previous.Writes
		r := &DatabasePerformance{Role: role, DatabaseBytes: s.FileBytes, WALBytes: s.WALBytes, FreeBytes: s.FreePages * s.PageBytes, WriteAttempts: int64(v.Writes), WriteErrors: int64(v.WriteErrors), BusyRetries: int64(v.BusyRetries), QueueRejected: int64(v.QueueRejected), QueueCanceled: int64(v.QueueCanceled), Queued: v.Queued, QueueCapacity: v.QueueCapacity, TotalWaitMilliseconds: float64(v.Wait) / float64(time.Millisecond), TotalExecutionMilliseconds: float64(v.Execution) / float64(time.Millisecond), Reader: poolMetrics(v.Reader), Writer: poolMetrics(v.Writer)}
		if !m.performance.previousAt.IsZero() && elapsed > 0 {
			r.WritesPerSecond = float64(count) / elapsed
			if count > 0 {
				r.AverageWaitMilliseconds = float64(v.Wait-previous.Wait) / float64(time.Millisecond) / float64(count)
				r.AverageExecutionMilliseconds = float64(v.Execution-previous.Execution) / float64(time.Millisecond) / float64(count)
			}
		}
		rows = append(rows, r)
	}
	m.performance.snapshot.SampledAt = now.UTC().Format(time.RFC3339Nano)
	m.performance.snapshot.IntervalSeconds = 0
	if !m.performance.previousAt.IsZero() {
		m.performance.snapshot.IntervalSeconds = elapsed
	}
	m.performance.snapshot.LastError = ""
	m.performance.snapshot.Databases = rows
	m.performance.previousAt = now
	m.performance.previous = make(map[string]Metrics, len(status))
	for role, s := range status {
		m.performance.previous[role] = s.Metrics
	}
	return nil
}

// MonitorPerformance is owned and joined by the same runtime as maintenance.
func (m *Manager) MonitorPerformance(ctx context.Context) {
	defer m.ClosePerformanceSubscriptions()
	tick := time.NewTicker(PerformanceInterval)
	defer tick.Stop()
	for {
		work, cancel := context.WithTimeout(ctx, 2*time.Second)
		_ = m.SamplePerformance(work)
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// SubscribePerformance atomically registers and queues the current snapshot.
// Each subscriber keeps only its latest snapshot, so a slow WebSocket can never
// block database sampling or accumulate an unbounded event backlog.
func (m *Manager) SubscribePerformance(ctx context.Context) <-chan *DatabasePerformanceSnapshot {
	channel := make(chan *DatabasePerformanceSnapshot, 1)
	m.performance.mu.Lock()
	if m.performance.closed || ctx.Err() != nil {
		close(channel)
		m.performance.mu.Unlock()
		return channel
	}
	if m.performance.subscribers == nil {
		m.performance.subscribers = make(map[uint64]performanceSubscriber)
	}
	m.performance.nextID++
	id := m.performance.nextID
	done := make(chan struct{})
	m.performance.subscribers[id] = performanceSubscriber{channel: channel, done: done}
	initial := clonePerformanceSnapshot(m.performance.snapshot)
	channel <- &initial
	m.performance.mu.Unlock()
	go func() {
		select {
		case <-ctx.Done():
			m.performance.mu.Lock()
			m.performance.unsubscribeLocked(id)
			m.performance.mu.Unlock()
		case <-done:
		}
	}()
	return channel
}
func (p *performanceMonitor) publishLocked() {
	for _, subscriber := range p.subscribers {
		snapshot := clonePerformanceSnapshot(p.snapshot)
		select {
		case subscriber.channel <- &snapshot:
		default:
			select {
			case <-subscriber.channel:
			default:
			}
			select {
			case subscriber.channel <- &snapshot:
			default:
			}
		}
	}
}
func (p *performanceMonitor) unsubscribeLocked(id uint64) {
	if subscriber, ok := p.subscribers[id]; ok {
		delete(p.subscribers, id)
		close(subscriber.done)
		close(subscriber.channel)
	}
}
func (m *Manager) ClosePerformanceSubscriptions() {
	m.performance.mu.Lock()
	defer m.performance.mu.Unlock()
	m.performance.closed = true
	for id := range m.performance.subscribers {
		m.performance.unsubscribeLocked(id)
	}
}
