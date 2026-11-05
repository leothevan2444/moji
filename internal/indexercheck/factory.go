package indexercheck

import (
	"context"

	"github.com/leothevan2444/moji/internal/indexer"
)

// AdapterFactory shares guarded network pacing with normal sourcing while each
// check owns fresh account sessions and bypasses application search caches.
type AdapterFactory struct {
	Network *indexer.Network
	Bundles indexer.BundleProvider
}
type adapterExecution struct {
	instance     indexer.InstanceSnapshot
	adapter      indexer.Adapter
	authenticate func(context.Context) error
	connect      func(context.Context) error
	close        func()
}

func (f AdapterFactory) Open(_ context.Context, instance indexer.InstanceSnapshot) (Execution, error) {
	if f.Network == nil {
		return nil, indexer.ErrNotConfigured
	}
	if instance.Kind == indexer.Builtin {
		sessions, err := indexer.NewTransientSessions(f.Network)
		if err != nil {
			return nil, err
		}
		adapter, err := indexer.NewBuiltinAdapter(f.Bundles, sessions)
		if err != nil {
			sessions.Close()
			return nil, err
		}
		return &adapterExecution{instance: instance, adapter: adapter, connect: func(ctx context.Context) error {
			lease, err := f.Bundles.Acquire(ctx, instance.BundleRevision)
			if err != nil {
				return err
			}
			defer lease.Release()
			plan := lease.Bundle.Plans[instance.DefinitionID]
			if plan == nil {
				return indexer.ErrUnsupported
			}
			target := instance
			if target.BaseURL == "" {
				target.BaseURL = plan.Definition.Links[0]
			}
			return f.Network.CheckConnection(ctx, target, plan.Definition.Certificates)
		}, authenticate: func(ctx context.Context) error { return adapter.Authenticate(ctx, instance) }, close: sessions.Close}, nil
	}
	if instance.Kind == indexer.Jackett {
		adapter, err := indexer.NewJackettAdapter(f.Network)
		if err != nil {
			return nil, err
		}
		return &adapterExecution{instance: instance, adapter: adapter, connect: func(ctx context.Context) error { return f.Network.CheckConnection(ctx, instance, nil) }, close: adapter.Close}, nil
	}
	return nil, indexer.ErrUnsupported
}
func (e *adapterExecution) Authenticate(ctx context.Context) error {
	if e.authenticate == nil {
		return indexer.ErrUnsupported
	}
	return e.authenticate(ctx)
}
func (e *adapterExecution) Search(ctx context.Context, q indexer.Query) (indexer.Batch, error) {
	return e.adapter.Search(ctx, e.instance, q)
}
func (e *adapterExecution) Fetch(ctx context.Context, ref indexer.ResourceRef) (indexer.Artifact, error) {
	return e.adapter.Fetch(ctx, e.instance, ref)
}
func (e *adapterExecution) Close() { e.close() }

func (e *adapterExecution) Connect(ctx context.Context) error { return e.connect(ctx) }
