// Package storage owns SQLite connections, lifecycle, migrations and recovery.
// Business repositories receive these handles; they do not create competing pools.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Durability string

const (
	Durable     Durability = "FULL"
	Rebuildable Durability = "NORMAL"
)

var (
	ErrClosed         = errors.New("storage: database is closed")
	ErrWriteQueueFull = errors.New("storage: write admission capacity exhausted")
)

type Options struct {
	Path        string
	Durability  Durability
	Readers     int
	WriteQueue  int
	BusyTimeout time.Duration
}

type Database struct {
	Reader         *sqlx.DB
	writer         *sqlx.DB
	Path           string
	admission      chan struct{}
	writerGate     chan struct{}
	closeMu        sync.Mutex
	closed         atomic.Bool
	writes         atomic.Uint64
	writeErrors    atomic.Uint64
	waitNanos      atomic.Uint64
	executionNanos atomic.Uint64
	busyTimeout    time.Duration
	busyRetries    atomic.Uint64
	queueRejected  atomic.Uint64
	queueCanceled  atomic.Uint64
}

type Metrics struct {
	QueueRejected uint64
	QueueCanceled uint64
	QueueCapacity int
	Writes        uint64
	BusyRetries   uint64
	WriteErrors   uint64
	Queued        int
	Wait          time.Duration
	Execution     time.Duration
	Reader        sql.DBStats
	Writer        sql.DBStats
}

func Open(ctx context.Context, options Options) (*Database, error) {
	if options.Path == "" {
		return nil, errors.New("storage: database path required")
	}
	if options.Durability == "" {
		options.Durability = Durable
	}
	if options.Durability != Durable && options.Durability != Rebuildable {
		return nil, errors.New("storage: invalid durability")
	}
	if options.Readers == 0 {
		options.Readers = 4
	}
	if options.Readers < 1 || options.Readers > 16 {
		return nil, errors.New("storage: reader count must be between 1 and 16")
	}
	if options.WriteQueue == 0 {
		options.WriteQueue = 64
	}
	if options.WriteQueue < 1 {
		return nil, errors.New("storage: write capacity must be positive")
	}
	if options.BusyTimeout < 0 {
		return nil, errors.New("storage: busy timeout must not be negative")
	}
	if options.BusyTimeout == 0 {
		options.BusyTimeout = 5 * time.Second
	}
	path, err := filepath.Abs(options.Path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	d := &Database{Path: path, admission: make(chan struct{}, options.WriteQueue), writerGate: make(chan struct{}, 1), busyTimeout: options.BusyTimeout}
	open := func(readOnly bool) (*sqlx.DB, error) {
		u := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
		q := url.Values{}
		// SQLite's native busy handler can delay Go context cancellation for its
		// entire timeout. Keep each attempt short; the writer retries in Go where
		// both request deadlines and the configured overall budget are observable.
		slice := options.BusyTimeout
		if slice > 50*time.Millisecond {
			slice = 50 * time.Millisecond
		}
		milliseconds := slice.Milliseconds()
		if milliseconds < 1 {
			milliseconds = 1
		}
		q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", milliseconds))
		q.Add("_pragma", "foreign_keys(1)")
		q.Add("_pragma", "synchronous("+string(options.Durability)+")")
		if readOnly {
			q.Set("mode", "ro")
			q.Add("_pragma", "query_only(1)")
		} else {
			q.Set("_txlock", "immediate")
			q.Add("_pragma", "journal_size_limit(67108864)")
			q.Add("_pragma", "wal_autocheckpoint(1000)")
		}
		u.RawQuery = q.Encode()
		db, err := sqlx.Open("sqlite", u.String())
		if err != nil {
			return nil, err
		}
		count := 1
		if readOnly {
			count = options.Readers
		}
		db.SetMaxOpenConns(count)
		db.SetMaxIdleConns(count)
		if err = db.PingContext(ctx); err != nil {
			db.Close()
			return nil, err
		}
		return db, nil
	}
	d.writer, err = open(false)
	if err != nil {
		return nil, err
	}
	var journal string
	if err = d.writer.GetContext(ctx, &journal, "PRAGMA journal_mode=WAL"); err != nil || journal != "wal" {
		d.writer.Close()
		return nil, fmt.Errorf("storage: enable WAL: mode=%s error=%v", journal, err)
	}
	d.Reader, err = open(true)
	if err != nil {
		d.writer.Close()
		return nil, err
	}
	return d, nil
}

// WithWrite reserves a bounded admission slot before obtaining the sole writer.
// The callback must perform only database work, never network or media operations.
func (d *Database) WithWrite(ctx context.Context, work func(*sqlx.Tx) error) error {
	if d.closed.Load() {
		return ErrClosed
	}
	select {
	case d.admission <- struct{}{}:
	default:
		d.queueRejected.Add(1)
		return ErrWriteQueueFull
	}
	defer func() { <-d.admission }()
	start := time.Now()
	select {
	case d.writerGate <- struct{}{}:
	case <-ctx.Done():
		d.queueCanceled.Add(1)
		return ctx.Err()
	}
	defer func() { <-d.writerGate }()
	wait := time.Since(start)
	if d.closed.Load() {
		return ErrClosed
	}
	begin := time.Now()
	err := func() error {
		tx, err := d.beginWrite(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if err = work(tx); err != nil {
			return err
		}
		return tx.Commit()
	}()
	d.waitNanos.Add(uint64(wait))
	d.executionNanos.Add(uint64(time.Since(begin)))
	d.writes.Add(1)
	if err != nil {
		d.writeErrors.Add(1)
	}
	return err
}

func (d *Database) Metrics() Metrics {
	return Metrics{QueueRejected: d.queueRejected.Load(), QueueCanceled: d.queueCanceled.Load(), QueueCapacity: cap(d.admission), Writes: d.writes.Load(), BusyRetries: d.busyRetries.Load(), WriteErrors: d.writeErrors.Load(), Queued: len(d.admission), Wait: time.Duration(d.waitNanos.Load()), Execution: time.Duration(d.executionNanos.Load()), Reader: d.Reader.Stats(), Writer: d.writer.Stats()}
}

// Close refuses new writes, waits for the current write and closes both pools.
// Owners cancel and join workers before closing; unfinished callers receive errors.
func (d *Database) Close() error {
	d.closeMu.Lock()
	defer d.closeMu.Unlock()
	if d.closed.Swap(true) {
		return nil
	}
	d.writerGate <- struct{}{}
	defer func() { <-d.writerGate }()
	return errors.Join(d.Reader.Close(), d.writer.Close())
}

type Checkpoint struct{ Busy, LogFrames, CheckpointedFrames int }

func (d *Database) Checkpoint(ctx context.Context) (Checkpoint, error) {
	var result Checkpoint
	if d.closed.Load() {
		return result, ErrClosed
	}
	select {
	case d.writerGate <- struct{}{}:
	case <-ctx.Done():
		return result, ctx.Err()
	}
	defer func() { <-d.writerGate }()
	if d.closed.Load() {
		return result, ErrClosed
	}
	err := d.writer.QueryRowContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)").Scan(&result.Busy, &result.LogFrames, &result.CheckpointedFrames)
	return result, err
}

func (d *Database) beginWrite(ctx context.Context) (*sqlx.Tx, error) {
	deadline := time.Now().Add(d.busyTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tx, err := d.writer.BeginTxx(ctx, nil)
		if err == nil {
			return tx, nil
		}
		var coded interface{ Code() int }
		if !errors.As(err, &coded) || (coded.Code()&255 != 5 && coded.Code()&255 != 6) {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, err
		}
		d.busyRetries.Add(1)
		pause := 10 * time.Millisecond
		if pause > remaining {
			pause = remaining
		}
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
