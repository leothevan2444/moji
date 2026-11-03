package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/internal/indexercategory"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type BundleProvider interface {
	Acquire(context.Context, string) (*indexerdefs.Lease, error)
}

type BuiltinAdapter struct {
	bundles  BundleProvider
	sessions *Sessions
}

func NewBuiltinAdapter(bundles BundleProvider, sessions *Sessions) (*BuiltinAdapter, error) {
	if bundles == nil || sessions == nil {
		return nil, ErrNotConfigured
	}
	return &BuiltinAdapter{bundles: bundles, sessions: sessions}, nil
}

// Authenticate verifies the declared login independently of a search. Public
// sites require no login; callers should report that distinction explicitly.
func (a *BuiltinAdapter) Authenticate(ctx context.Context, instance InstanceSnapshot) error {
	lease, err := a.bundles.Acquire(ctx, instance.BundleRevision)
	if err != nil {
		return err
	}
	defer lease.Release()
	plan := lease.Bundle.Plans[instance.DefinitionID]
	if plan == nil {
		return ErrUnsupported
	}
	return engineError(a.sessions.With(ctx, instance, plan, false, func(*cardigann.Execution) error { return nil }))
}

func (a *BuiltinAdapter) Search(ctx context.Context, instance InstanceSnapshot, query Query) (Batch, error) {
	lease, err := a.bundles.Acquire(ctx, instance.BundleRevision)
	if err != nil {
		return Batch{}, err
	}
	retained := false
	defer func() {
		if !retained {
			lease.Release()
		}
	}()
	plan := lease.Bundle.Plans[instance.DefinitionID]
	if plan == nil {
		return Batch{}, ErrUnsupported
	}
	for _, category := range query.Categories {
		if _, known := indexercategory.Name(category); !known {
			return Batch{}, ErrUnsupported
		}
	}
	var result cardigann.SearchResult
	err = a.sessions.With(ctx, instance, plan, true, func(execution *cardigann.Execution) error {
		var searchErr error
		result, searchErr = execution.Search(ctx, cardigann.Query{Mode: query.Mode, Keywords: query.Keywords, Categories: query.Categories, Parameters: query.Parameters, Limit: &query.Limit, Offset: &query.Offset})
		return searchErr
	})
	batch := Batch{InstanceID: instance.ID, Partial: len(result.Issues) > 0, IssueCount: len(result.Issues), Truncated: result.Truncated, ReportedCount: result.ReportedCount}
	for _, source := range result.Results {
		ref := ResourceRef{Version: 1, ID: uuid.NewString(), InstanceID: instance.ID, Kind: Builtin, BundleRevision: lease.Bundle.Manifest.Revision, DefinitionID: instance.DefinitionID, DefinitionDigest: plan.Digest, ConfigRevision: instance.ConfigRevision, Locator: Locator{DownloadURL: source.DownloadURL, MagnetURI: source.MagnetURI, InfoHash: source.InfoHash, Title: source.Title, Fields: source.Fields}}
		release := Release{Title: source.Title, GUID: source.Fields["guid"], InstanceID: instance.ID, InstanceName: instance.Name, Kind: Builtin, DetailsURL: source.DetailsURL, PublishedAt: source.PublishedAt, Categories: categoriesFromNames(source.Categories), Seeders: source.Seeders, InfoHash: source.InfoHash, Resource: ref}
		if source.Fields["size"] != "" {
			value := source.Size
			release.Size = &value
		}
		if source.Seeders != nil && source.Leechers != nil {
			value := *source.Seeders + *source.Leechers
			release.Peers = &value
		}
		batch.Releases = append(batch.Releases, release)
	}
	if len(batch.Releases) > 0 {
		retained = true
		batch.release = lease.Release
	}
	return batch, engineError(err)
}

func (a *BuiltinAdapter) Fetch(ctx context.Context, instance InstanceSnapshot, ref ResourceRef) (Artifact, error) {
	if ref.Version != 1 || ref.Kind != Builtin || ref.InstanceID != instance.ID || ref.ConfigRevision != instance.ConfigRevision || ref.DefinitionID != instance.DefinitionID || instance.BundleRevision != "" && ref.BundleRevision != instance.BundleRevision {
		return Artifact{}, ErrReference
	}
	lease, err := a.bundles.Acquire(ctx, ref.BundleRevision)
	if err != nil {
		return Artifact{}, err
	}
	defer lease.Release()
	plan := lease.Bundle.Plans[ref.DefinitionID]
	if plan == nil || plan.Digest != ref.DefinitionDigest {
		return Artifact{}, ErrReference
	}
	var artifact cardigann.Artifact
	err = a.sessions.With(ctx, instance, plan, false, func(execution *cardigann.Execution) error {
		var fetchErr error
		artifact, fetchErr = execution.Fetch(ctx, cardigann.Result{Title: ref.Locator.Title, DownloadURL: ref.Locator.DownloadURL, MagnetURI: ref.Locator.MagnetURI, InfoHash: ref.Locator.InfoHash, Fields: ref.Locator.Fields})
		return fetchErr
	})
	if err != nil {
		return Artifact{}, engineError(err)
	}
	return Artifact{TorrentBytes: artifact.TorrentBytes, Magnet: artifact.Magnet, Resource: ref, AcquiredAt: time.Now().UTC()}, nil
}

func engineError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, cardigann.ErrAuthentication) {
		return ErrAuthentication
	}
	if errors.Is(err, cardigann.ErrInvalidQuery) {
		return ErrInvalidQuery
	}
	return err
}
