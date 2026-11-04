package tracker

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/indexer"
)

// IndexerService adapts existing consumers while their public search operations
// migrate to the neutral service. All source ordering remains owned by indexer.
type IndexerService struct {
	service  *indexer.Service
	registry indexer.RegistryProvider
}

func NewIndexerService(service *indexer.Service, providers ...indexer.RegistryProvider) *IndexerService {
	bridge := &IndexerService{service: service}
	if len(providers) > 0 {
		bridge.registry = providers[0]
	}
	return bridge
}

func (s *IndexerService) ListIndexerInstances(ctx context.Context) ([]indexer.InstanceSnapshot, error) {
	if s.registry == nil {
		return nil, indexer.ErrNotConfigured
	}
	snapshot, err := s.registry.Snapshot(ctx)
	return snapshot.Instances, err
}
func (s *IndexerService) Search(query string, options ...SearchOption) ([]indexer.SelectionCandidate, error) {
	opts := &SearchOptions{}
	for _, option := range options {
		option(opts)
	}
	if opts.Context == nil {
		return nil, indexer.ErrInvalidQuery
	}
	var rows []indexer.SelectionCandidate
	err := s.SearchSequential(opts.Context, query, func(batch []indexer.SelectionCandidate) (bool, error) {
		rows = append(rows, batch...)
		return false, nil
	}, options...)
	return rows, err
}
func (s *IndexerService) SearchSequential(ctx context.Context, query string, accept func([]indexer.SelectionCandidate) (bool, error), options ...SearchOption) error {
	opts := &SearchOptions{}
	for _, option := range options {
		option(opts)
	}
	_, err := s.service.SearchSequential(ctx, indexer.Query{Keywords: query, InstanceIDs: InstanceIDsFromLegacyTrackers(opts.Trackers), Categories: opts.Categories, Limit: opts.Limit}, func(ctx context.Context, batch indexer.Batch) (bool, error) {
		rows := make([]indexer.SelectionCandidate, 0, len(batch.Releases))
		for _, release := range batch.Releases {
			rows = append(rows, indexer.SelectionFromRelease(release))
		}
		return accept(rows)
	})
	return err
}

// InstanceIDsFromLegacyTrackers bridges existing API inputs. Persisted task
// requests use their own versioned decoder: legacy UUID-shaped IDs must not be
// inferred as builtin instances. Neutral Query accepts explicit instance IDs.
func InstanceIDsFromLegacyTrackers(ids []string) []string {
	qualified := append([]string(nil), ids...)
	for i, id := range qualified {
		if strings.Contains(id, "/") {
			continue
		}
		if _, err := uuid.Parse(id); err == nil {
			continue
		}
		qualified[i] = "jackett/" + id
	}
	return qualified
}
