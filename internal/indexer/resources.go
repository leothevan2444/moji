package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/indexerdefs"
	"github.com/leothevan2444/moji/internal/storage"
)

type RevisionRetainer interface {
	Retain(context.Context, string) (func(), error)
}
type ResourceRecord struct {
	Version  int
	Instance InstanceSnapshot
	Ref      ResourceRef
}

func (r ResourceRecord) String() string { return r.Ref.String() }

type ResourceLease struct {
	Record  ResourceRecord
	once    sync.Once
	release func()
}

func (l *ResourceLease) Release() {
	if l != nil {
		l.once.Do(func() {
			if l.release != nil {
				l.release()
			}
		})
	}
}

type ResourceRegistry interface {
	SaveBatch(context.Context, InstanceSnapshot, []ResourceRef) error
	Resolve(context.Context, string) (*ResourceLease, error)
}

// Resources owns short-lived, server-only locator/configuration snapshots. A
// selected task must copy its record into core before this lease is released;
// these runtime rows can never be its durable recovery source.
type Resources struct {
	db          *storage.Database
	definitions RevisionRetainer
	mu          sync.Mutex
	pins        map[string]func()
	closed      bool
}

func NewResources(ctx context.Context, manager *storage.Manager, definitions RevisionRetainer) (*Resources, error) {
	if manager == nil || manager.Runtime == nil || definitions == nil {
		return nil, ErrNotConfigured
	}
	r := &Resources{db: manager.Runtime, definitions: definitions, pins: map[string]func(){}}
	if err := r.Prune(ctx); err != nil {
		r.Close()
		return nil, err
	}
	return r, nil
}

func (r *Resources) SaveBatch(ctx context.Context, instance InstanceSnapshot, refs []ResourceRef) error {
	if len(refs) > 1000 {
		return ErrResponseBudget
	}
	if len(refs) == 0 {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrExpired
	}
	var data []string
	var bytes int64
	revisions := map[string]bool{}
	seen := map[string]bool{}
	for _, ref := range refs {
		if ref.Version != 1 || ref.ID == "" || len(ref.ID) > 128 || seen[ref.ID] || ref.InstanceID != instance.ID || ref.Kind != instance.Kind || ref.ConfigRevision != instance.ConfigRevision {
			return ErrReference
		}
		seen[ref.ID] = true
		if ref.Kind == Builtin {
			if ref.BundleRevision == "" || ref.BundleRevision != instance.BundleRevision {
				return ErrReference
			}
			revisions[ref.BundleRevision] = true
		}
		payload, err := json.Marshal(ResourceRecord{Version: 1, Instance: cloneInstance(instance), Ref: ref})
		if err != nil {
			return ErrReference
		}
		if len(payload) > 256<<10 {
			return ErrResponseBudget
		}
		bytes += int64(len(payload))
		data = append(data, string(payload))
	}
	newPins := map[string]func(){}
	for revision := range revisions {
		if r.pins[revision] != nil {
			continue
		}
		release, err := r.definitions.Retain(ctx, revision)
		if err != nil {
			for _, undo := range newPins {
				undo()
			}
			return err
		}
		newPins[revision] = release
	}
	now := time.Now()
	err := r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM indexer_resources WHERE expires_at<=?`, instant(now)); err != nil {
			return err
		}
		var count int
		var used int64
		if err := tx.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(length(CAST(payload_json AS BLOB))),0) FROM indexer_resources`).Scan(&count, &used); err != nil {
			return err
		}
		if count+len(refs) > 5000 || used+bytes > 128<<20 {
			return ErrBusy
		}
		for i, ref := range refs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO indexer_resources(id,instance_id,bundle_revision,payload_version,payload_json,expires_at,created_at) VALUES(?,?,?,1,?,?,?)`, ref.ID, instance.ID, ref.BundleRevision, data[i], instant(now.Add(30*time.Minute)), instant(now)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		for _, release := range newPins {
			release()
		}
		return err
	}
	for revision, release := range newPins {
		r.pins[revision] = release
	}
	return nil
}

func (r *Resources) Resolve(ctx context.Context, id string) (*ResourceLease, error) {
	if id == "" || len(id) > 128 {
		return nil, ErrReference
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrExpired
	}
	var payload, instanceID, revision string
	err := r.db.Reader.QueryRowContext(ctx, `SELECT payload_json,instance_id,bundle_revision FROM indexer_resources WHERE id=? AND expires_at>?`, id, instant(time.Now())).Scan(&payload, &instanceID, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrExpired
	}
	if err != nil {
		return nil, err
	}
	if len(payload) > 256<<10 {
		return nil, ErrReference
	}
	var record ResourceRecord
	if err := json.Unmarshal([]byte(payload), &record); err != nil || record.Version != 1 || record.Ref.Version != 1 || record.Ref.ID != id || record.Ref.InstanceID != instanceID || record.Instance.ID != instanceID || record.Instance.ConfigRevision != record.Ref.ConfigRevision || record.Instance.Kind != record.Ref.Kind || record.Ref.BundleRevision != revision {
		return nil, ErrReference
	}
	lease := &ResourceLease{Record: record}
	if record.Ref.Kind == Builtin {
		release, err := r.definitions.Retain(ctx, revision)
		if err != nil {
			return nil, ErrExpired
		}
		lease.release = release
	}
	return lease, nil
}

func (r *Resources) Prune(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrExpired
	}
	if err := r.db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM indexer_resources WHERE id IN (SELECT id FROM indexer_resources WHERE expires_at<=? ORDER BY expires_at,id LIMIT 512)`, instant(time.Now()))
		return err
	}); err != nil {
		return err
	}
	var revisions []string
	if err := r.db.Reader.SelectContext(ctx, &revisions, `SELECT DISTINCT bundle_revision FROM indexer_resources WHERE bundle_revision<>'' AND expires_at>?`, instant(time.Now())); err != nil {
		return err
	}
	wanted := map[string]bool{}
	for _, revision := range revisions {
		wanted[revision] = true
		if r.pins[revision] != nil {
			continue
		}
		release, err := r.definitions.Retain(ctx, revision)
		if err != nil {
			continue
		}
		r.pins[revision] = release
	}
	for revision, release := range r.pins {
		if !wanted[revision] {
			release()
			delete(r.pins, revision)
		}
	}
	return nil
}

// Stop/join the definition update worker before closing this registry, so no
// shutdown-time GC can race the release of its restart-restorable references.
func (r *Resources) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	for revision, release := range r.pins {
		release()
		delete(r.pins, revision)
	}
}

// Maintain is run and joined by the application owner; it creates no detached
// worker. Expired candidates stop retaining old definition versions.
func (r *Resources) Maintain(ctx context.Context, report func(error)) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Prune(ctx); err != nil && ctx.Err() == nil && report != nil {
				report(err)
			}
		}
	}
}

// Validate checks a server-owned snapshot before it becomes durable task state.
// It intentionally does not rebind it to today's account or active definition.
func (r ResourceRecord) Validate() error {
	ref, instance := r.Ref, r.Instance
	if r.Version != 1 || ref.Version != 1 || ref.ID == "" || len(ref.ID) > 128 || ref.InstanceID == "" || ref.InstanceID != instance.ID || ref.Kind != instance.Kind || ref.ConfigRevision == "" || ref.ConfigRevision != instance.ConfigRevision {
		return ErrReference
	}
	switch ref.Kind {
	case Builtin:
		if !indexerdefs.ValidRevision(ref.BundleRevision) || ref.BundleRevision != instance.BundleRevision || ref.DefinitionID == "" || ref.DefinitionID != instance.DefinitionID || len(ref.DefinitionDigest) != 64 {
			return ErrReference
		}
	case Jackett:
		if ref.BundleRevision != "" {
			return ErrReference
		}
	default:
		return ErrReference
	}
	return nil
}
func (r ResourceRecord) Clone() ResourceRecord {
	out := r
	out.Instance = cloneInstance(r.Instance)
	if r.Ref.Locator.Fields != nil {
		out.Ref.Locator.Fields = map[string]string{}
		for key, value := range r.Ref.Locator.Fields {
			out.Ref.Locator.Fields[key] = value
		}
	}
	return out
}
