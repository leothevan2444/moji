package indexer

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestHealthIsolationLateResponsesCancellationAndRestart(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if manager != nil {
			manager.Close()
		}
	}()
	health, err := NewHealthStore(manager)
	if err != nil {
		t.Fatal(err)
	}
	instance := InstanceSnapshot{ID: "source", ConfigRevision: "account-a", BundleRevision: "rule-a"}
	base := time.Now().Add(-time.Minute)
	correlation := uuid.NewString()
	if changed, err := health.Observe(ctx, instance, HealthSearch, base.Add(10*time.Second), fmt.Errorf("%w: synthetic-secret-url", ErrAuthentication), correlation); err != nil || !changed {
		t.Fatal("failure not recorded", err)
	}
	if changed, err := health.Observe(ctx, instance, HealthSearch, base, nil, uuid.NewString()); err != nil || changed {
		t.Fatal("older success replaced newer failure", err)
	}
	if _, err = health.Observe(ctx, instance, HealthFetch, base, ErrTransport, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if changed, err := health.Observe(canceled, instance, HealthSearch, base.Add(20*time.Second), context.Canceled, uuid.NewString()); err != nil || changed {
		t.Fatal("caller cancellation recorded as site failure", err)
	}
	otherAccount := instance
	otherAccount.ConfigRevision = "account-b"
	otherRules := instance
	otherRules.BundleRevision = "rule-b"
	states, err := health.Snapshot(ctx, []HealthKey{HealthScope(instance), HealthScope(otherAccount), HealthScope(otherRules)})
	if err != nil {
		t.Fatal(err)
	}
	state := states[HealthScope(instance)]
	if len(states) != 1 || state.Search == nil || state.Search.ErrorCode != ErrAuthentication.Error() || state.Search.CorrelationID != correlation || state.Fetch == nil || state.Fetch.ErrorCode != ErrTransport.Error() {
		t.Fatal("observations crossed stages, accounts or rule versions")
	}
	if err = manager.Close(); err != nil {
		t.Fatal(err)
	}
	manager = nil
	manager, err = storage.OpenManager(ctx, storage.DefaultManagerOptions(directory))
	if err != nil {
		t.Fatal(err)
	}
	health, err = NewHealthStore(manager)
	if err != nil {
		t.Fatal(err)
	}
	states, err = health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	if err != nil || states[HealthScope(instance)].Search.ErrorCode != ErrAuthentication.Error() {
		t.Fatal("restart discarded health observation", err)
	}
	if _, err = health.Observe(ctx, instance, HealthSearch, base.Add(30*time.Second), nil, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	states, err = health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	if err != nil || states[HealthScope(instance)].Search.ErrorCode != "" || states[HealthScope(instance)].Fetch.ErrorCode != ErrTransport.Error() {
		t.Fatal("search success erased independent acquisition failure", err)
	}
}

func TestServiceObservesActualSearchButDoesNotRefreshHealthFromCachedResults(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	health, err := NewHealthStore(manager)
	if err != nil {
		t.Fatal(err)
	}
	instance := InstanceSnapshot{ID: "one", Kind: Builtin, Enabled: true, ConfigRevision: "account", BundleRevision: "rules"}
	cached := true
	adapter := fakeAdapter{search: func(context.Context, InstanceSnapshot, Query) (Batch, error) {
		if cached {
			batch := candidateBatch(instance)
			batch.CacheHit = true
			return batch, nil
		}
		return Batch{}, ErrAuthentication
	}}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, adapter, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.SetHealthRecorder(health, func(err error) { t.Error(err) })
	if _, err = service.Search(ctx, Query{Keywords: "ABCD-123"}); err != nil {
		t.Fatal(err)
	}
	states, err := health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	if err != nil || len(states) != 0 {
		t.Fatal("cache result presented as a new upstream verification", err)
	}
	cached = false
	result, err := service.Search(ctx, Query{Keywords: "ABCD-123"})
	if err == nil || len(result.Report.Sites) != 1 {
		t.Fatal("authentication failure not reported")
	}
	states, err = health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	if err != nil || states[HealthScope(instance)].Search == nil || states[HealthScope(instance)].Search.CorrelationID != result.Report.Sites[0].CorrelationID {
		t.Fatal("persisted diagnostic cannot be correlated with actual search report", err)
	}
}

func TestHealthMigrationPreservesRuntimeV3AndBoundsHistoricalNamespaces(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	path := filepath.Join(directory, "runtime.db")
	db, err := storage.Open(ctx, storage.Options{Path: path, Durability: storage.Rebuildable})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Migrate(ctx, storage.RuntimeMigrations()[:3]); err != nil {
		t.Fatal(err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO indexer_resources(id,instance_id,payload_version,payload_json,expires_at,created_at) VALUES('preserved','source',1,'{}','2099-01-01T00:00:00Z','2026-01-01T00:00:00Z')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = storage.Open(ctx, storage.Options{Path: path, Durability: storage.Rebuildable})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(ctx, storage.RuntimeMigrations()); err != nil {
		t.Fatal(err)
	}
	var preserved int
	if err = db.Reader.GetContext(ctx, &preserved, `SELECT count(*) FROM indexer_resources WHERE id='preserved'`); err != nil || preserved != 1 {
		t.Fatal("runtime migration discarded existing references", err)
	}
	if err = db.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, `WITH RECURSIVE seq(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM seq WHERE x<8193) INSERT INTO indexer_health(instance_id,config_revision,bundle_revision,operation,started_at,finished_at,error_code,correlation_id) SELECT 'old-'||x,'account','rules','SEARCH','2020-01-01T00:00:00.000000000Z','2020-01-01T00:00:00.000000000Z','','id' FROM seq`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	health := &HealthStore{db: db}
	instance := InstanceSnapshot{ID: "current", ConfigRevision: "account", BundleRevision: "rules"}
	if _, err = health.Observe(ctx, instance, HealthSearch, time.Now().Add(-time.Second), &CoolingDownError{Until: time.Now().Add(time.Minute)}, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.Reader.GetContext(ctx, &count, `SELECT count(*) FROM indexer_health`); err != nil || count != 8192 {
		t.Fatal("observation row budget not enforced", count, err)
	}
	states, err := health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	if err != nil || states[HealthScope(instance)].Search == nil || states[HealthScope(instance)].Search.RetryAt == nil {
		t.Fatal("current cooling observation lost", err)
	}
}

func TestManagedSearchHealthAndReportUseJackettGateDeadline(t *testing.T) {
	ctx := context.Background()
	manager, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	key := cooldownKey("http://jackett.invalid:80|direct")
	if err = manager.Runtime.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`INSERT INTO indexer_network_cooldowns VALUES(?,?,?,?)`, key, 1, time.Now().Add(-time.Second).UnixMilli(), time.Now().Add(time.Hour).UnixMilli())
		return e
	}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	network := persistentTestNetwork(t, manager, func(*http.Request) (*http.Response, error) {
		calls++
		return httpResponse(429, http.Header{"Retry-After": []string{"5"}}, "limited"), nil
	}, nil)
	defer network.Close()
	adapter, err := NewJackettAdapter(network)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()
	instance := InstanceSnapshot{ID: "jackett/one", Kind: Jackett, Enabled: true, RemoteID: "one", ConfigRevision: "synthetic-account", BaseURL: "http://jackett.invalid", JackettAPIKey: "synthetic"}
	service, err := NewService(registryFunc(func(context.Context) (Registry, error) { return Registry{Instances: []InstanceSnapshot{instance}}, nil }), testResources{}, nil, adapter)
	if err != nil {
		t.Fatal(err)
	}
	health, err := NewHealthStore(manager)
	if err != nil {
		t.Fatal(err)
	}
	service.SetHealthRecorder(health, func(e error) { t.Error(e) })
	result, err := service.Search(ctx, Query{Keywords: "ABCD-123"})
	if err == nil || len(result.Report.Sites) != 1 || result.Report.Sites[0].Code != ErrCoolingDown.Error() || result.Report.Sites[0].RetryAt == nil || time.Until(*result.Report.Sites[0].RetryAt) < 59*time.Second {
		t.Fatal("managed search report lost gate deadline", err)
	}
	states, err := health.Snapshot(ctx, []HealthKey{HealthScope(instance)})
	observation := states[HealthScope(instance)].Search
	if err != nil || observation == nil || observation.RetryAt == nil || !observation.RetryAt.Equal(*result.Report.Sites[0].RetryAt) || observation.CorrelationID != result.Report.Sites[0].CorrelationID {
		t.Fatal("durable health differs from search report", err)
	}
	first := *observation.RetryAt
	result, err = service.Search(ctx, Query{Keywords: "EFGH-456"})
	if err == nil || calls != 1 || len(result.Report.Sites) != 1 || result.Report.Sites[0].RetryAt == nil || !result.Report.Sites[0].RetryAt.Equal(first) {
		t.Fatal("cooldown admission fabricated new upstream result or deadline", err, calls)
	}
}
