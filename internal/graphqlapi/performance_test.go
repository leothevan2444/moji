package graphqlapi

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/storage"
	"sync/atomic"
	"time"
)

type performanceSourceFunc func(context.Context) <-chan *storage.DatabasePerformanceSnapshot

func (f performanceSourceFunc) SubscribePerformance(ctx context.Context) <-chan *storage.DatabasePerformanceSnapshot {
	return f(ctx)
}

type performanceSettings struct {
	SettingsEditor
	enabled atomic.Bool
}

func (s *performanceSettings) Snapshot() *SettingsSnapshot {
	return &SettingsSnapshot{System: SystemSettingsSnapshot{DeveloperMode: s.enabled.Load()}}
}

func TestDatabasePerformanceSubscriptionRequiresDeveloperModeAndStopsWhenDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	settings := &performanceSettings{}
	resolver := &subscriptionResolver{Resolver: &Resolver{SettingsEditor: settings, DatabasePerformanceSource: m}}
	if _, err = resolver.DatabasePerformance(ctx); err == nil {
		t.Fatal("subscription exposed while developer mode is disabled")
	}
	settings.enabled.Store(true)
	channel, err := resolver.DatabasePerformance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	first := <-channel
	if first.SampledAt == "" || len(first.Databases) != 3 {
		t.Fatal("current snapshot missing")
	}
	settings.enabled.Store(false)
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-channel:
		if ok {
			t.Fatal("metrics still published after mode disabled")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription did not stop")
	}
}

func TestDatabasePerformanceOverGraphQLWebSocket(t *testing.T) {
	ctx := context.Background()
	m, err := storage.OpenManager(ctx, storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	canceled := make(chan struct{})
	source := performanceSourceFunc(func(ctx context.Context) <-chan *storage.DatabasePerformanceSnapshot {
		go func() { <-ctx.Done(); close(canceled) }()
		return m.SubscribePerformance(ctx)
	})
	client := newGraphQLWebSocketTestClient(t, &Resolver{RuntimeSettings: &SettingsSnapshot{System: SystemSettingsSnapshot{DeveloperMode: true}}, DatabasePerformanceSource: source})
	client.subscribe(t, "performance", `subscription {databasePerformance{sampledAt lastError databases{role writeAttempts}}}`)
	var first, second struct {
		Data struct {
			Metrics struct {
				SampledAt string
				LastError string
				Databases []struct {
					Role          string
					WriteAttempts int64
				}
			} `json:"databasePerformance"`
		}
	}
	client.readNext(t, "performance", &first)
	if len(first.Data.Metrics.Databases) != 3 {
		t.Fatal("initial metrics missing")
	}
	if err = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, e := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('subscription','MANUAL','','DOWNLOADING','RUNNING','now','now')`)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	if err = m.SamplePerformance(ctx); err != nil {
		t.Fatal(err)
	}
	client.readNext(t, "performance", &second)
	if second.Data.Metrics.SampledAt == first.Data.Metrics.SampledAt || second.Data.Metrics.Databases[0].WriteAttempts != first.Data.Metrics.Databases[0].WriteAttempts+1 {
		t.Fatal("new metrics were not pushed", first, second)
	}
	if err = client.connection.WriteJSON(map[string]any{"id": "performance", "type": "complete"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("unsubscribe did not cancel the owner subscription")
	}
}

func TestDatabasePerformanceOwnerCloseDoesNotWaitForSlowConsumer(t *testing.T) {
	m, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	resolver := &subscriptionResolver{Resolver: &Resolver{RuntimeSettings: &SettingsSnapshot{System: SystemSettingsSnapshot{DeveloperMode: true}}, DatabasePerformanceSource: m}}
	out, err := resolver.DatabasePerformance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("slow consumer kept the subscription alive after owner close")
		}
	}
}

func TestSystemMutationUpdatesAndPreservesDeveloperMode(t *testing.T) {
	editor := &fakeSettingsEditor{snapshot: &SettingsSnapshot{System: SystemSettingsSnapshot{DeveloperMode: true, ImageCache: ImageCacheSettingsSnapshot{Enabled: true, MaxSizeMB: 1024, RetentionDays: 30}, StashBoxDataCache: StashBoxDataCacheSettingsSnapshot{TTLHours: 24}}}}
	resolver := NewResolver(nil, nil, nil, nil, "test")
	resolver.SettingsEditor = editor
	for _, test := range []struct {
		field string
		want  bool
	}{{"", true}, {"developerMode:false", false}, {"developerMode:true", true}} {
		response := executeGraphQL(t, resolver, `mutation{updateSystemSettings(input:{taskDeletePolicy:KEEP_ONLY `+test.field+`}){system{developerMode}}}`)
		if len(response.Errors) > 0 {
			t.Fatal(response.Errors)
		}
		if editor.systemInput.DeveloperMode != test.want {
			t.Fatalf("developer mode %q = %v", test.field, editor.systemInput.DeveloperMode)
		}
	}
}
