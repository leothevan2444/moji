package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/leothevan2444/moji/internal/indexer"
)

type IndexedResourceAccess interface {
	ResolveReference(context.Context, string) (*indexer.ResourceLease, error)
	FetchRecord(context.Context, indexer.ResourceRecord) (indexer.Artifact, error)
}
type QueueIndexedResourceRequest struct {
	Code, ResourceID, SavePath, Category, Tags string
	Paused                                     *bool
}

// QueueIndexedResourceContext accepts only the opaque server reference. The
// caller cannot replace its locator, account, title, hash or source metadata.
func (s *Service) QueueIndexedResourceContext(ctx context.Context, req QueueIndexedResourceRequest) (*Task, error) {
	code := extractCode(req.Code)
	if code == "" {
		return nil, ErrTaskCodeRequired
	}
	creator, ok := s.store.(interface {
		CreateIndexedTask(context.Context, *Task) error
	})
	if !ok || s.resourceAccess == nil {
		return nil, indexer.ErrUnsupported
	}
	unlock := s.lockTask("sourcing-code:" + code)
	defer unlock()
	existing, err := s.store.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: task %q already uses code %s", ErrDuplicateCodeTask, existing.ID, code)
	}
	lease, err := s.resourceAccess.ResolveReference(ctx, req.ResourceID)
	if err != nil {
		return nil, err
	}
	defer lease.Release()
	record := lease.Record.Clone()
	if err = record.Validate(); err != nil {
		return nil, err
	}
	if _, err = validateCodeEvidence(code, record.Ref.Locator.Title); err != nil {
		return nil, err
	}
	release := indexer.Release{Title: record.Ref.Locator.Title, InstanceID: record.Instance.ID, InstanceName: record.Instance.Name, Kind: record.Instance.Kind, InfoHash: record.Ref.Locator.InfoHash, Resource: record.Ref}
	candidate := candidateFromSearchResult(indexer.SelectionFromRelease(release))
	request, err := json.Marshal(DownloadRequest{Source: TaskSourceSearch, Code: code, SavePath: req.SavePath, Category: req.Category, Tags: req.Tags, Paused: req.Paused})
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	task := &Task{ID: s.newID(), Code: code, Source: TaskSourceSearch, Candidate: candidate, TorrentURL: "indexer-resource:" + record.Ref.ID, SelectedResourceID: record.Ref.ID, IndexerResource: &record, SavePath: req.SavePath, Category: req.Category, Tags: req.Tags, SourcingRequestJSON: string(request), CreatedAt: now, UpdatedAt: now}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusPending)
	if err = creator.CreateIndexedTask(ctx, task); err != nil {
		return nil, err
	}
	select {
	case s.sourcingWake <- struct{}{}:
	default:
	}
	return cloneTask(task), nil
}

// The acquisition checkpoint replaces secret configuration/locators with raw
// bytes or the exact magnet before validation or any qB side effect. A later
// retry must reuse that acquisition, rather than consume another grab quota.
func (s *Service) acquireSelectedResource(ctx context.Context, task *Task) error {
	if s.resourceAccess == nil {
		return candidateError(TaskStageErrorTorrentMetadata, "indexed resource acquisition unavailable")
	}
	artifact, err := s.resourceAccess.FetchRecord(ctx, task.IndexerResource.Clone())
	if err != nil {
		return candidateError(TaskStageErrorTorrentMetadata, "unable to acquire selected indexer resource")
	}
	if len(artifact.TorrentBytes) > 0 && len(artifact.TorrentBytes) <= MaxTorrentBytes && artifact.Magnet == "" {
		if _, err = parseTorrentMetadata(artifact.TorrentBytes); err != nil {
			return candidateError(TaskStageErrorTorrentMetadata, "invalid selected torrent metadata")
		}
		task.TorrentFileData = append([]byte(nil), artifact.TorrentBytes...)
	} else if len(artifact.TorrentBytes) == 0 && normalizeMagnetURI(artifact.Magnet) != "" {
		task.TorrentURL = artifact.Magnet
		task.Candidate.MagnetURI = artifact.Magnet
	} else {
		return candidateError(TaskStageErrorTorrentMetadata, "invalid selected resource artifact")
	}
	task.IndexerResource = nil
	task.UpdatedAt = s.now().UTC()
	if err = s.store.Update(ctx, task); err != nil {
		return fmt.Errorf("persist acquired resource: %w", err)
	}
	return nil
}

// ResolveIndexedResourceContext persists the chosen source on an existing
// eligible task. The application worker performs acquisition after admission.
var ErrSubmissionUncertain = errors.New("task submission has not been reconciled")

type ResolveIndexedResourceRequest struct {
	ResourceID       string
	ExpectedRevision int64
	Paused           *bool
}

func eligibleIndexedChoice(task *Task) bool {
	return task != nil && task.RestartStateJSON == "" && task.Stage == TaskStageSourcing && (task.StageStatus == TaskStageStatusBlocked || waitingForResource(task)) && task.TorrentHash == "" && task.ContentPath == ""
}
func (s *Service) ResolveIndexedResourceContext(ctx context.Context, id string, req ResolveIndexedResourceRequest) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 128 || req.ResourceID == "" || len(req.ResourceID) > 128 || req.ExpectedRevision < 1 {
		return nil, ErrInvalidTorrentInput
	}
	updater, ok := s.store.(interface {
		UpdateIndexedTask(context.Context, *Task) error
	})
	if !ok || s.resourceAccess == nil {
		return nil, indexer.ErrUnsupported
	}
	unlock := s.lockTask(id)
	defer unlock()
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrSourcingUnavailable
	}
	// An acknowledged identical choice is safe to reread after a lost response.
	// This never changes options or executes acquisition/submission again.
	if task.SelectedResourceID == req.ResourceID && task.Revision > req.ExpectedRevision {
		return cloneTask(task), nil
	}
	if task.Revision != req.ExpectedRevision {
		return nil, ErrTaskRevisionConflict
	}
	if !eligibleIndexedChoice(task) {
		return nil, ErrSourcingUnavailable
	}
	code := extractCode(task.Code)
	if code == "" || code != task.Code {
		return nil, ErrTaskCodeRequired
	}
	lease, err := s.resourceAccess.ResolveReference(ctx, req.ResourceID)
	if err != nil {
		return nil, err
	}
	defer lease.Release()
	record := lease.Record.Clone()
	if err = record.Validate(); err != nil {
		return nil, err
	}
	if _, err = validateCodeEvidence(code, record.Ref.Locator.Title); err != nil {
		return nil, err
	}
	next := cloneTask(task)
	request := DownloadRequest{Source: task.Source, Code: code, SavePath: task.SavePath, Category: task.Category, Tags: task.Tags}
	if task.SourcingRequestJSON != "" {
		if err = json.Unmarshal([]byte(task.SourcingRequestJSON), &request); err != nil {
			return nil, candidateError(TaskStageErrorSourcingRequest, "unable to read queued task request")
		}
	}
	request.Source = task.Source
	request.Code = code
	request.SavePath = task.SavePath
	request.Category = task.Category
	request.Tags = task.Tags
	if req.Paused != nil {
		request.Paused = req.Paused
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	release := indexer.Release{Title: record.Ref.Locator.Title, InstanceID: record.Instance.ID, InstanceName: record.Instance.Name, Kind: record.Instance.Kind, InfoHash: record.Ref.Locator.InfoHash, Resource: record.Ref}
	next.Candidate = candidateFromSearchResult(indexer.SelectionFromRelease(release))
	next.SelectedResourceID = record.Ref.ID
	next.IndexerResource = &record
	next.TorrentURL = "indexer-resource:" + record.Ref.ID
	next.TorrentFileData = nil
	next.TorrentIdentityHash = ""
	next.TorrentIdentityMagnet = ""
	next.FilePlanJSON = ""
	next.SourcingNextCheckAt = nil
	next.SourcingRequestJSON = string(encoded)
	setTaskStage(next, TaskStageSourcing, TaskStageStatusPending)
	clearTaskStageError(next)
	next.UpdatedAt = s.now().UTC()
	if err = updater.UpdateIndexedTask(ctx, next); err != nil {
		return nil, err
	}
	select {
	case s.sourcingWake <- struct{}{}:
	default:
	}
	return cloneTask(next), nil
}
