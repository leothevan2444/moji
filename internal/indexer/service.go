package indexer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/indexerquery"
)

type Service struct {
	provider      RegistryProvider
	adapters      map[Kind]Adapter
	bundles       BundleProvider
	resources     ResourceRegistry
	events        EventPublisher
	health        HealthRecorder
	healthError   func(error)
	searchCache   *SearchCache
	artifactCache *ArtifactCache
	cachePolicy   func() SearchCachePolicy
}

func (s *Service) SetSearchCache(cache *SearchCache, policy func() SearchCachePolicy) {
	s.searchCache = cache
	s.cachePolicy = policy
}

func (s *Service) searchObserved(ctx context.Context, instance InstanceSnapshot, query Query, adapter Adapter) (Batch, error) {
	started := time.Now()
	batch, err := adapter.Search(ctx, instance, query)
	if s.searchCache != nil && (errors.Is(err, ErrAuthentication) || errors.Is(err, ErrChallenge)) {
		s.searchCache.InvalidateInstance(instance.ID)
	}
	batch.ObservationCorrelationID = uuid.NewString()
	batch.InstanceID = instance.ID
	if len(batch.Releases) > query.Limit {
		batch.Releases = batch.Releases[:query.Limit]
		batch.Truncated = true
	}
	for _, release := range batch.Releases {
		if release.InstanceID != instance.ID || release.Resource.InstanceID != instance.ID || release.Resource.Version != 1 || release.Resource.Kind != instance.Kind || release.Resource.ConfigRevision != instance.ConfigRevision {
			err = ErrReference
			batch.Releases = nil
			break
		}
	}
	result := err
	if result == nil && batch.Partial {
		result = ErrIncomplete
	}
	if !batch.CacheHit {
		s.observe(ctx, instance, HealthSearch, started, result, batch.ObservationCorrelationID)
	}
	return batch, err
}

type HealthRecorder interface {
	Observe(context.Context, InstanceSnapshot, HealthOperation, time.Time, error, string) (bool, error)
}

func (s *Service) SetHealthRecorder(health HealthRecorder, onError func(error)) {
	s.health = health
	s.healthError = onError
}
func (s *Service) observe(ctx context.Context, instance InstanceSnapshot, op HealthOperation, started time.Time, result error, correlation string) {
	if s.health == nil {
		return
	}
	observationCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := s.health.Observe(observationCtx, instance, op, started, result, correlation); err != nil && s.healthError != nil {
		s.healthError(err)
	}
}

// SetEventPublisher is an assembly-time dependency, installed before workers run.
func (s *Service) SetEventPublisher(events EventPublisher) { s.events = events }

func NewService(provider RegistryProvider, resources ResourceRegistry, builtin, external Adapter) (*Service, error) {
	if provider == nil || resources == nil {
		return nil, ErrNotConfigured
	}
	adapters := map[Kind]Adapter{}
	if builtin != nil {
		adapters[Builtin] = builtin
	}
	if external != nil {
		adapters[Jackett] = external
	}
	service := &Service{provider: provider, adapters: adapters, resources: resources}
	if adapter, ok := builtin.(*BuiltinAdapter); ok {
		service.bundles = adapter.bundles
	}
	return service, nil
}

func normalizeQuery(query Query) (Query, error) {
	if query.Mode == "" {
		query.Mode = "search"
	}
	switch query.Mode {
	case "search", "movie", "tvsearch", "music", "book":
	default:
		return Query{}, ErrInvalidQuery
	}
	if len(query.Keywords) > 4096 || len(query.Categories) > 128 || len(query.Parameters) > 64 || len(query.InstanceIDs) > 256 || query.Limit < 0 || query.Limit > 1000 || query.Offset < 0 || query.Offset > 1000000 {
		return Query{}, ErrInvalidQuery
	}
	if query.Limit == 0 {
		query.Limit = 100
	}
	if query.CachePolicy == "" {
		query.CachePolicy = CacheDefault
	}
	if query.CachePolicy != CacheDefault && query.CachePolicy != CacheRefresh {
		return Query{}, ErrInvalidQuery
	}
	parameters, err := indexerquery.NormalizeParameters(query.Parameters)
	if err != nil {
		return Query{}, ErrInvalidQuery
	}
	// Take ownership before freezing an operation; callers can edit their own
	// configuration/query slices without changing in-flight work.
	query.Categories = append([]int(nil), query.Categories...)
	query.InstanceIDs = append([]string(nil), query.InstanceIDs...)
	query.Parameters = parameters
	return query, nil
}

func orderedInstances(registry Registry, scope []string) ([]InstanceSnapshot, error) {
	if len(registry.Instances) > 1024 {
		return nil, ErrBusy
	}
	byID := map[string]InstanceSnapshot{}
	for _, instance := range registry.Instances {
		if instance.ID == "" || byID[instance.ID].ID != "" {
			return nil, ErrInvalidSettings
		}
		byID[instance.ID] = instance
	}
	selected := map[string]bool{}
	for _, id := range scope {
		instance, ok := byID[id]
		if !ok || !instance.Enabled {
			return nil, ErrInvalidScope
		}
		selected[id] = true
	}
	seen := map[string]bool{}
	ordered := []InstanceSnapshot{}
	add := func(id string) {
		instance, ok := byID[id]
		if !ok || !instance.Enabled || seen[id] || len(selected) > 0 && !selected[id] {
			return
		}
		seen[id] = true
		ordered = append(ordered, cloneInstance(instance))
	}
	for _, id := range registry.Order {
		add(id)
	}
	for _, instance := range registry.Instances {
		add(instance.ID)
	}
	return ordered, nil
}

func cloneInstance(instance InstanceSnapshot) InstanceSnapshot {
	instance.TrustedOrigins = append([]string(nil), instance.TrustedOrigins...)
	settings := map[string]SettingValue{}
	for key, value := range instance.Settings {
		copy := SettingValue{}
		if value.String != nil {
			v := *value.String
			copy.String = &v
		}
		if value.Boolean != nil {
			v := *value.Boolean
			copy.Boolean = &v
		}
		if value.Number != nil {
			v := *value.Number
			copy.Number = &v
		}
		settings[key] = copy
	}
	instance.Settings = settings
	return instance
}

func (s *Service) SearchSequential(ctx context.Context, query Query, accept func(context.Context, Batch) (bool, error)) (SearchReport, error) {
	return s.SearchWithProgress(ctx, query, nil, accept)
}
func (s *Service) SearchWithProgress(ctx context.Context, query Query, progress func(context.Context, SearchProgress) error, accept func(context.Context, Batch) (bool, error)) (SearchReport, error) {
	ctx, cancelOperation := context.WithTimeout(ctx, 10*time.Minute)
	defer cancelOperation()
	if accept == nil {
		return SearchReport{}, ErrInvalidQuery
	}
	query, err := normalizeQuery(query)
	if err != nil {
		return SearchReport{}, err
	}
	if err := ctx.Err(); err != nil {
		return SearchReport{}, err
	}
	registry, err := s.provider.Snapshot(ctx)
	if err != nil {
		return SearchReport{}, err
	}
	instances, err := orderedInstances(registry, query.InstanceIDs)
	if err != nil {
		return SearchReport{}, err
	}
	var bundleErr error
	if s.bundles != nil {
		for _, instance := range instances {
			if instance.Kind != Builtin {
				continue
			}
			lease, err := s.bundles.Acquire(ctx, "")
			bundleErr = err
			if err == nil {
				defer lease.Release()
				for i := range instances {
					if instances[i].Kind == Builtin {
						instances[i].BundleRevision = lease.Bundle.Manifest.Revision
					}
				}
			}
			break
		}
	}
	report := SearchReport{Sites: append([]SiteReport(nil), registry.Diagnostics...)}
	notify := func(event SearchProgress) error {
		if progress == nil {
			return nil
		}
		return progress(ctx, event)
	}
	sources := make([]SearchSource, 0, len(instances))
	for _, instance := range instances {
		sources = append(sources, SearchSource{ID: instance.ID, Name: instance.Name, Kind: instance.Kind})
	}
	if err := notify(SearchProgress{Phase: "READY", Sources: sources, Diagnostics: registry.Diagnostics}); err != nil {
		return report, err
	}

	for _, diagnostic := range registry.Diagnostics {
		if diagnostic.Partial || diagnostic.Code != "" {
			report.Partial = true
		}
	}
	if len(instances) == 0 {
		if report.Partial {
			return report, ErrIncomplete
		}
		return report, ErrNotConfigured
	}
	for _, instance := range instances {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if err := notify(SearchProgress{Phase: "STARTED", InstanceID: instance.ID}); err != nil {
			return report, err
		}
		started := time.Now()
		adapter := s.adapters[instance.Kind]
		siteTimeout := instance.SiteTimeout
		if siteTimeout <= 0 {
			siteTimeout = 120 * time.Second
		}
		if siteTimeout > 10*time.Minute {
			siteTimeout = 10 * time.Minute
		}
		siteCtx, cancelSite := context.WithTimeout(ctx, siteTimeout)
		defer cancelSite()
		var batch Batch
		var searchErr error
		attempted := false
		if instance.Kind == Builtin && bundleErr != nil {
			searchErr = bundleErr
		} else if adapter == nil {
			searchErr = ErrUnsupported
		} else {
			if s.searchCache != nil && s.cachePolicy != nil {
				batch, searchErr = s.searchCache.Search(siteCtx, instance, query, s.cachePolicy(), func(worker context.Context) (Batch, error) { return s.searchObserved(worker, instance, query, adapter) })
			} else {
				attempted = true
				batch, searchErr = adapter.Search(siteCtx, instance, query)
			}
		}
		if batch.release != nil {
			defer batch.release()
		}
		if err := ctx.Err(); err != nil {
			return report, err
		}
		batch.InstanceID = instance.ID
		if len(batch.Releases) > query.Limit {
			batch.Releases = batch.Releases[:query.Limit]
			batch.Truncated = true
		}
		for _, release := range batch.Releases {
			if release.InstanceID != instance.ID || release.Resource.InstanceID != instance.ID || release.Resource.Version != 1 || release.Resource.Kind != instance.Kind || release.Resource.ConfigRevision != instance.ConfigRevision {
				searchErr = ErrReference
				batch.Releases = nil
				break
			}
		}
		correlation := uuid.NewString()
		if batch.ObservationCorrelationID != "" {
			correlation = batch.ObservationCorrelationID
		}
		if attempted && !batch.CacheHit {
			healthResult := searchErr
			if healthResult == nil && batch.Partial {
				healthResult = ErrIncomplete
			}
			s.observe(ctx, instance, HealthSearch, started, healthResult, correlation)
		}
		if len(batch.Releases) > 0 {
			refs := make([]ResourceRef, len(batch.Releases))
			for i := range batch.Releases {
				refs[i] = batch.Releases[i].Resource
			}
			if err := s.resources.SaveBatch(siteCtx, instance, refs); err != nil {
				searchErr = err
				batch.Releases = nil
			}
		}
		site := SiteReport{InstanceID: instance.ID, Results: len(batch.Releases), Elapsed: time.Since(started), CacheHit: batch.CacheHit, Truncated: batch.Truncated, Partial: batch.Partial || searchErr != nil, CorrelationID: correlation}
		if searchErr != nil {
			site.Code = ErrorCode(searchErr)
			var cooling *CoolingDownError
			if errors.As(searchErr, &cooling) {
				until := cooling.Until
				site.RetryAt = &until
			}
		}
		if site.Partial {
			report.Partial = true
			batch.Partial = true
		}
		if site.Truncated {
			report.Truncated = true
		}
		report.Sites = append(report.Sites, site)
		if s.events != nil {
			s.events.Publish(EventSite)
		}
		finished := func() error {
			return notify(SearchProgress{Phase: "FINISHED", InstanceID: instance.ID, Report: report.Sites[len(report.Sites)-1], FetchedAt: batch.FetchedAt})
		}
		if len(batch.Releases) == 0 {
			cancelSite()
			if err := finished(); err != nil {
				return report, err
			}
			continue
		}
		accepted, err := accept(siteCtx, batch)
		siteErr := siteCtx.Err()
		cancelSite()
		report.Sites[len(report.Sites)-1].Elapsed = time.Since(started)
		if err != nil {
			report.Partial = true
			report.Sites[len(report.Sites)-1].Partial = true
			report.Sites[len(report.Sites)-1].Code = ErrorCode(err)
			if persistErr := finished(); persistErr != nil {
				return report, persistErr
			}
			return report, err
		}
		if accepted {
			if err := finished(); err != nil {
				return report, err
			}
			report.AcceptedInstanceID = instance.ID
			return report, nil
		}
		if siteErr != nil {
			report.Partial = true
			report.Sites[len(report.Sites)-1].Partial = true
			report.Sites[len(report.Sites)-1].Code = ErrorCode(siteErr)
		}
		if err := finished(); err != nil {
			return report, err
		}
	}
	if report.Partial {
		return report, ErrIncomplete
	}
	return report, nil
}

func (s *Service) Search(ctx context.Context, query Query) (SearchResult, error) {
	var result SearchResult
	truncated := false
	report, err := s.SearchSequential(ctx, query, func(_ context.Context, batch Batch) (bool, error) {
		remaining := 1000 - len(result.Releases)
		if remaining <= 0 {
			if len(batch.Releases) > 0 {
				truncated = true
			}
			return false, nil
		}
		if len(batch.Releases) > remaining {
			truncated = true
			batch.Releases = batch.Releases[:remaining]
		}
		result.Releases = append(result.Releases, batch.Releases...)
		return false, nil
	})
	result.Report = report
	result.Report.Truncated = result.Report.Truncated || truncated
	return result, err
}

// Fetch receives a server-resolved configuration/reference pair, never browser
// locator data. Older already accepted work can intentionally use its snapshot.
// SetArtifactCache installs the application-owned acquisition coordinator.
func (s *Service) SetArtifactCache(cache *ArtifactCache) { s.artifactCache = cache }
func (s *Service) Fetch(ctx context.Context, instance InstanceSnapshot, ref ResourceRef) (Artifact, error) {
	if ref.Version != 1 || ref.ID == "" || ref.InstanceID != instance.ID || ref.Kind != instance.Kind || ref.ConfigRevision != instance.ConfigRevision {
		return Artifact{}, ErrReference
	}
	instance = cloneInstance(instance)
	ref = ResourceRecord{Ref: ref}.Clone().Ref
	if s.artifactCache != nil {
		return s.artifactCache.Fetch(ctx, instance, ref, func(worker context.Context) (Artifact, error) { return s.fetchObserved(worker, instance, ref) })
	}
	return s.fetchObserved(ctx, instance, ref)
}
func (s *Service) fetchObserved(ctx context.Context, instance InstanceSnapshot, ref ResourceRef) (result Artifact, resultErr error) {
	if ref.Version != 1 || ref.ID == "" || ref.InstanceID != instance.ID || ref.Kind != instance.Kind || ref.ConfigRevision != instance.ConfigRevision || strings.TrimSpace(ref.ID) == "" {
		return Artifact{}, ErrReference
	}
	adapter := s.adapters[instance.Kind]
	if adapter == nil {
		return Artifact{}, ErrUnsupported
	}
	started := time.Now().UTC()
	defer func() {
		if s.searchCache != nil && (errors.Is(resultErr, ErrAuthentication) || errors.Is(resultErr, ErrChallenge)) {
			s.searchCache.InvalidateInstance(instance.ID)
		}
		s.observe(ctx, instance, HealthFetch, started, resultErr, uuid.NewString())
		if s.events != nil {
			s.events.Publish(EventSite)
		}
	}()
	artifact, err := adapter.Fetch(ctx, cloneInstance(instance), ref)
	if err != nil {
		return Artifact{}, err
	}
	if len(artifact.TorrentBytes) == 0 && artifact.Magnet == "" || len(artifact.TorrentBytes) > 0 && artifact.Magnet != "" || len(artifact.TorrentBytes) > 8<<20 {
		return Artifact{}, ErrReference
	}
	artifact, err = ValidateArtifact(artifact)
	if err != nil {
		return Artifact{}, err
	}
	artifact.Resource = ref
	if artifact.AcquiredAt.IsZero() {
		artifact.AcquiredAt = time.Now().UTC()
	}
	return artifact, nil
}

func (s *Service) FetchReference(ctx context.Context, id string) (Artifact, error) {
	lease, err := s.resources.Resolve(ctx, id)
	if err != nil {
		return Artifact{}, err
	}
	defer lease.Release()
	return s.Fetch(ctx, lease.Record.Instance, lease.Record.Ref)
}

func (s *Service) ResolveReference(ctx context.Context, id string) (*ResourceLease, error) {
	if s.resources == nil {
		return nil, ErrNotConfigured
	}
	lease, err := s.resources.Resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = lease.Record.Validate(); err != nil {
		lease.Release()
		return nil, err
	}
	if err = s.validateCurrentResource(ctx, lease.Record); err != nil {
		lease.Release()
		return nil, err
	}
	return lease, nil
}
func (s *Service) FetchRecord(ctx context.Context, record ResourceRecord) (Artifact, error) {
	if err := record.Validate(); err != nil {
		return Artifact{}, err
	}
	return s.Fetch(ctx, record.Instance, record.Ref)
}

// ValidateQuery returns an owned, normalized query for durable admission.
func ValidateQuery(query Query) (Query, error) { return normalizeQuery(query) }
