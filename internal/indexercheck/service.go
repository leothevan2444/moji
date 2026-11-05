// Package indexercheck executes explicit, bounded site diagnostics without
// registering search resources, populating query caches or creating tasks.
package indexercheck

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/indexer"
)

type Input struct {
	InstanceID, ExpectedRevision, Keywords string
	Draft                                  *config.BuiltinInstancePatch
	Search, Download, ConnectionOnly       bool
}
type Prepared struct {
	Instance                     indexer.InstanceSnapshot
	ConfigRevision, RuleRevision string
	AuthenticationRequired       bool
	Release                      func()
}
type Source interface {
	PrepareCheck(context.Context, Input) (Prepared, error)
}
type Execution interface {
	Connect(context.Context) error
	Authenticate(context.Context) error
	Search(context.Context, indexer.Query) (indexer.Batch, error)
	Fetch(context.Context, indexer.ResourceRef) (indexer.Artifact, error)
	Close()
}
type Factory interface {
	Open(context.Context, indexer.InstanceSnapshot) (Execution, error)
}
type Stage struct {
	Name, State, ErrorCode string
	Elapsed                time.Duration
}
type Report struct {
	ID, InstanceID, ConfigRevision, RuleRevision string
	Kind                                         indexer.Kind
	StartedAt, FinishedAt                        time.Time
	Results                                      int
	Partial, Truncated                           bool
	Stages                                       []Stage
}

type Service struct {
	owner   context.Context
	cancel  context.CancelFunc
	source  Source
	factory Factory
	mu      sync.Mutex
	wg      sync.WaitGroup
	active  map[string]bool
	closed  bool
}

func New(owner context.Context, source Source, factory Factory) (*Service, error) {
	if owner == nil || source == nil || factory == nil {
		return nil, indexer.ErrNotConfigured
	}
	ctx, cancel := context.WithCancel(owner)
	return &Service{owner: ctx, cancel: cancel, source: source, factory: factory, active: map[string]bool{}}, nil
}
func (s *Service) Check(ctx context.Context, input Input) (report Report, resultErr error) {
	if len(input.InstanceID) > 128 || len(input.ExpectedRevision) > 128 {
		return Report{}, indexer.ErrInvalidSettings
	}
	var query indexer.Query
	var err error
	if input.ConnectionOnly && (input.Search || input.Download) {
		return Report{}, indexer.ErrInvalidQuery
	}
	if input.Download && !input.Search {
		return Report{}, indexer.ErrInvalidQuery
	}
	if input.Search {
		query, err = indexer.ValidateQuery(indexer.Query{Mode: "search", Keywords: input.Keywords, Limit: 20, CachePolicy: indexer.CacheRefresh})
		if err != nil {
			return Report{}, err
		}
	}
	if err = ctx.Err(); err != nil {
		return Report{}, err
	}
	key := input.InstanceID
	if key == "" {
		key = "draft"
	}
	s.mu.Lock()
	if s.closed || s.owner.Err() != nil {
		s.mu.Unlock()
		return Report{}, context.Canceled
	}
	if len(s.active) >= 4 || s.active[key] {
		s.mu.Unlock()
		return Report{}, indexer.ErrBusy
	}
	s.active[key] = true
	s.wg.Add(1)
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.active, key); s.mu.Unlock(); s.wg.Done() }()
	work, cancel := context.WithTimeout(ctx, 3*time.Minute)
	stop := context.AfterFunc(s.owner, cancel)
	defer stop()
	defer cancel()
	report = Report{ID: uuid.NewString(), InstanceID: input.InstanceID, StartedAt: time.Now().UTC(), Stages: []Stage{{Name: "CONFIGURATION", State: "SKIPPED"}, {Name: "AUTHENTICATION", State: "SKIPPED"}, {Name: "SEARCH", State: "SKIPPED"}, {Name: "DOWNLOAD", State: "SKIPPED"}}}
	if !input.Search {
		report.Stages[2].State = "NOT_REQUESTED"
	}
	if !input.Download {
		report.Stages[3].State = "NOT_REQUESTED"
	}
	defer func() { report.FinishedAt = time.Now().UTC() }()
	run := func(i int, operation func() error) bool {
		started := time.Now()
		failure := operation()
		report.Stages[i].Elapsed = time.Since(started)
		report.Stages[i].State = "PASSED"
		if failure != nil {
			report.Stages[i].State = "FAILED"
			report.Stages[i].ErrorCode = errorCode(failure)
			return false
		}
		return true
	}
	var prepared Prepared
	var execution Execution
	if !run(0, func() error {
		var err error
		prepared, err = s.source.PrepareCheck(work, input)
		if err != nil {
			return err
		}
		report.InstanceID = prepared.Instance.ID
		report.Kind = prepared.Instance.Kind
		report.ConfigRevision = prepared.ConfigRevision
		report.RuleRevision = prepared.RuleRevision
		execution, err = s.factory.Open(work, prepared.Instance)
		if err == nil && execution == nil {
			return indexer.ErrNotConfigured
		}
		return err
	}) {
		if execution != nil {
			execution.Close()
		}
		if prepared.Release != nil {
			prepared.Release()
		}
		return report, nil
	}
	if prepared.Release != nil {
		defer prepared.Release()
	}
	defer execution.Close()
	if input.ConnectionOnly {
		report.Stages[1].State = "NOT_REQUESTED"
		report.Stages = append(report.Stages, Stage{Name: "CONNECTION", State: "SKIPPED"})
		run(4, func() error { return execution.Connect(work) })
		return report, nil
	}
	if prepared.Instance.Kind == indexer.Jackett {
		report.Stages[1].State = "NOT_OBSERVED"
	} else if !prepared.AuthenticationRequired {
		report.Stages[1].State = "NOT_REQUIRED"
	} else if !run(1, func() error { return execution.Authenticate(work) }) {
		return report, nil
	}
	if !input.Search {
		return report, nil
	}
	var batch indexer.Batch
	if !run(2, func() error { var err error; batch, err = execution.Search(work, query); return err }) {
		batch.Close()
		return report, nil
	}
	defer batch.Close()
	if len(batch.Releases) > query.Limit {
		batch.Releases = batch.Releases[:query.Limit]
		batch.Truncated = true
	}
	report.Results = len(batch.Releases)
	report.Partial = batch.Partial
	report.Truncated = batch.Truncated
	if batch.Partial {
		report.Stages[2].State = "PARTIAL"
		report.Stages[2].ErrorCode = indexer.ErrIncomplete.Error()
	} else if len(batch.Releases) == 0 {
		report.Stages[2].State = "EMPTY"
	}
	if !input.Download {
		return report, nil
	}
	if len(batch.Releases) == 0 {
		report.Stages[3].State = "NO_CANDIDATE"
		return report, nil
	}
	run(3, func() error {
		ref := batch.Releases[0].Resource
		i := prepared.Instance
		if ref.Version != 1 || ref.InstanceID != i.ID || ref.Kind != i.Kind || ref.ConfigRevision != i.ConfigRevision || ref.Kind == indexer.Builtin && (ref.BundleRevision != i.BundleRevision || ref.DefinitionID != i.DefinitionID) {
			return indexer.ErrReference
		}
		artifact, err := execution.Fetch(work, ref)
		if err != nil {
			return err
		}
		_, err = indexer.ValidateArtifact(artifact)
		return err
	})
	return report, nil
}
func errorCode(err error) string {
	if errors.Is(err, config.ErrConfigRevision) {
		return config.ErrConfigRevision.Error()
	}
	if errors.Is(err, config.ErrInvalidSettings) {
		return indexer.ErrInvalidSettings.Error()
	}
	return indexer.ErrorCode(err)
}
func (s *Service) Close() { s.mu.Lock(); s.closed = true; s.cancel(); s.mu.Unlock(); s.wg.Wait() }
