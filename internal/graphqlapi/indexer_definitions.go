package graphqlapi

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

type IndexerDefinitionsService interface {
	Status(context.Context) (indexerdefs.Status, error)
	Job(context.Context, string) (*indexerdefs.Job, error)
	CheckThrottled(context.Context) (indexerdefs.State, error)
	Begin(context.Context, string, string, string) (*indexerdefs.Job, error)
	Cancel(context.Context, string) error
	Subscribe() (<-chan struct{}, func(), error)
}

func indexerTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func definitionJobToModel(job *indexerdefs.Job) *model.IndexerDefinitionJob {
	if job == nil {
		return nil
	}
	result := &model.IndexerDefinitionJob{ID: job.ID, Kind: model.IndexerDefinitionJobKind(job.Kind), State: model.IndexerDefinitionJobState(job.State), TargetRevision: job.Target, ExpectedActiveRevision: job.ExpectedActive, DownloadedBytes: job.DownloadedBytes, ErrorCode: job.ErrorCode, CorrelationID: job.CorrelationID, StartedAt: indexerTime(job.StartedAt), Revision: job.Revision}
	if job.FinishedAt != nil {
		value := indexerTime(*job.FinishedAt)
		result.FinishedAt = &value
	}
	return result
}
func definitionStatusToModel(status indexerdefs.Status) *model.IndexerDefinitionStatus {
	result := &model.IndexerDefinitionStatus{StreamID: status.StreamID, ActiveRevision: status.State.Active, PreviousRevision: status.State.Previous, AvailableRevision: status.State.Available, Revision: status.State.Revision, Available: status.Available, CheckErrorCode: status.State.CheckError, ErrorCode: status.ErrorCode, Job: definitionJobToModel(status.Job), Installed: make([]*model.IndexerDefinitionManifest, 0, len(status.Installed))}
	if status.State.LastCheckedAt != nil {
		value := indexerTime(*status.State.LastCheckedAt)
		result.LastCheckedAt = &value
	}
	for _, manifest := range status.Installed {
		result.Installed = append(result.Installed, &model.IndexerDefinitionManifest{Revision: manifest.Revision, SchemaVersion: manifest.Schema, EngineVersion: manifest.Engine, DefinitionCount: manifest.DefinitionCount, InstalledAt: indexerTime(manifest.InstalledAt)})
	}
	return result
}
func (r *Resolver) definitionStatus(ctx context.Context) (*model.IndexerDefinitionStatus, error) {
	if r.DefinitionLibrary == nil {
		return nil, indexerdefs.ErrUnavailable
	}
	status, err := r.DefinitionLibrary.Status(ctx)
	if err != nil {
		return nil, err
	}
	return definitionStatusToModel(status), nil
}
func (r *Resolver) beginDefinitionChange(ctx context.Context, kind string, input model.IndexerDefinitionChangeInput) (*model.IndexerDefinitionJob, error) {
	if r.DefinitionLibrary == nil {
		return nil, indexerdefs.ErrUnavailable
	}
	if input.ExpectedActiveRevision != "" && !indexerdefs.ValidRevision(input.ExpectedActiveRevision) {
		return nil, indexerdefs.ErrConflict
	}
	target := ""
	if input.TargetRevision != nil {
		target = *input.TargetRevision
		if !indexerdefs.ValidRevision(target) {
			return nil, indexerdefs.ErrConflict
		}
	}
	job, err := r.DefinitionLibrary.Begin(ctx, kind, target, input.ExpectedActiveRevision)
	if err != nil {
		return nil, err
	}
	return definitionJobToModel(job), nil
}
func (r *Resolver) definitionEvents(ctx context.Context) (<-chan *model.IndexerDefinitionStatus, error) {
	if r.DefinitionLibrary == nil {
		return nil, indexerdefs.ErrUnavailable
	}
	events, unsubscribe, err := r.DefinitionLibrary.Subscribe()
	if err != nil {
		return nil, err
	}
	out := make(chan *model.IndexerDefinitionStatus, 1)
	go func() {
		defer close(out)
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-events:
				if !ok {
					return
				}
				snapshot, err := r.definitionStatus(ctx)
				if err != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- snapshot:
				}
			}
		}
	}()
	return out, nil
}
