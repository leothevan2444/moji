package taskruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/stashsync"
)

const (
	StashIngestScan              = "SCAN"
	StashIngestGenerate          = "GENERATE"
	StashIngestIdentify          = "IDENTIFY"
	StashIngestVerify            = "VERIFY"
	TaskStageErrorSceneMissing   = "STASH_SCENE_MISSING"
	TaskStageErrorSceneChanged   = "STASH_SCENE_CHANGED"
	TaskStageErrorSourcesMissing = "STASH_IDENTIFY_SOURCES_MISSING"
	TaskStageErrorGenerate       = "STASH_GENERATE_FAILED"
	TaskStageErrorIdentify       = "STASH_IDENTIFY_FAILED"
	TaskStageErrorIngestVerify   = "STASH_INGEST_INCOMPLETE"
)

var (
	errStashSceneMissing = errors.New("no Stash scenes exist under the delivered path; scan the path in Stash and retry")
	errStashSceneChanged = errors.New("Stash scenes changed repeatedly; review the library and retry")
)

// Stash owns metadata creation. Moji only schedules jobs and reads their results.
type StashIngestor interface {
	StashScanner
	FindIngestScenes(context.Context, string, []string) ([]stashsync.IngestScene, error)
	MetadataGenerate(context.Context, []string) (string, error)
	MetadataIdentify(context.Context, []string) (string, error)
}

var _ StashIngestor = (*stashsync.Service)(nil)

func (t *Task) StashSceneIDs() []string {
	ids := []string{}
	_ = json.Unmarshal([]byte(t.StashSceneIDsJSON), &ids)
	if ids == nil {
		return []string{}
	}
	return ids
}

func stashJobID(task *Task) string {
	switch task.StashIngestPhase {
	case StashIngestGenerate:
		return task.StashGenerateJobID
	case StashIngestIdentify:
		return task.StashIdentifyJobID
	default:
		return task.StashScanJobID
	}
}

func setStashJobID(task *Task, id string) {
	switch task.StashIngestPhase {
	case StashIngestGenerate:
		task.StashGenerateJobID = id
	case StashIngestIdentify:
		task.StashIdentifyJobID = id
	default:
		task.StashScanJobID = id
	}
}

func ingestErrorCode(task *Task) string {
	switch task.StashIngestPhase {
	case StashIngestGenerate:
		return TaskStageErrorGenerate
	case StashIngestIdentify:
		return TaskStageErrorIdentify
	case StashIngestVerify:
		return TaskStageErrorIngestVerify
	default:
		return TaskStageErrorScanTrigger
	}
}

func (s *Service) failStashIngest(task *Task, err error) (*Task, error) {
	task.StashScanError = err.Error()
	logging.Errorf("taskruntime: Stash ingest blocked task=%s phase=%s: %v", task.ID, task.StashIngestPhase, err)
	code := ingestErrorCode(task)
	switch {
	case errors.Is(err, errStashSceneMissing):
		code = TaskStageErrorSceneMissing
	case errors.Is(err, errStashSceneChanged):
		code = TaskStageErrorSceneChanged
	case errors.Is(err, stashsync.ErrIdentifySourcesMissing):
		code = TaskStageErrorSourcesMissing
	}
	blockTask(task, code, err.Error(), s.now().UTC())
	return task, err
}

func (s *Service) syncStashScanJob(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	if task == nil {
		return task, nil
	}
	if task.StashIngestPhase == "" {
		task.StashIngestPhase = StashIngestScan
	}
	if task.StashIngestPhase == StashIngestVerify || stashJobID(task) == "" {
		return s.startStashIngestStep(ctx, task, scanner)
	}
	job, err := scanner.FindJob(ctx, stashJobID(task))
	if err != nil {
		return s.failStashIngest(task, err)
	}
	if job == nil {
		id := stashJobID(task)
		setStashJobID(task, "")
		return s.failStashIngest(task, fmt.Errorf("Stash job %s is unavailable; check Stash task history and retry", id))
	}
	task.StashScanError = ""
	clearTaskStageError(task)
	switch strings.ToUpper(strings.TrimSpace(job.Status)) {
	case "FINISHED":
		switch task.StashIngestPhase {
		case StashIngestScan:
			task.StashIngestPhase = StashIngestGenerate
		case StashIngestGenerate:
			task.StashIngestPhase = StashIngestIdentify
		case StashIngestIdentify:
			task.StashIngestPhase = StashIngestVerify
		default:
			return s.failStashIngest(task, errors.New("invalid Stash ingest phase"))
		}
		// Persist the completed step before submitting the next remote job.
		if err := s.store.Update(ctx, task); err != nil {
			return task, err
		}
		return s.startStashIngestStep(ctx, task, scanner)
	case "FAILED", "CANCELLED":
		message := firstNonEmpty([]string{stringValue(job.Error), "Stash ingest job failed or was cancelled"})
		setStashJobID(task, "") // retry this step, never transfer again
		if strings.ToUpper(strings.TrimSpace(job.Status)) == "FAILED" && task.StashIngestPhase != StashIngestScan && isMissingStashSceneError(errors.New(message)) {
			return s.recoverStashScenes(ctx, task, scanner)
		}
		return s.failStashIngest(task, errors.New(message))
	default:
		setTaskStage(task, TaskStageScanning, TaskStageStatusRunning)
		task.UpdatedAt = s.now().UTC()
		return task, nil
	}
}

func (s *Service) startStashIngestStep(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	if task.StashIngestPhase == "" {
		task.StashIngestPhase = StashIngestScan
	}
	setTaskStage(task, TaskStageScanning, TaskStageStatusRunning)
	clearTaskStageError(task)
	task.StashScanError = ""
	task.UpdatedAt = s.now().UTC()
	var id string
	var err error
	if task.StashIngestPhase == StashIngestScan {
		if strings.TrimSpace(task.StashScanPath) == "" {
			plan := planDelivery(scanner.CurrentConfig(), resolveSource(task))
			if plan.ValidationError != nil {
				return s.failStashIngest(task, plan.ValidationError)
			}
			task.StashScanPath = plan.ResolvedScanPath
		}
		id, err = scanner.MetadataScan(ctx, stashsync.ScanRequest{Paths: []string{task.StashScanPath}})
		now := s.now().UTC()
		task.StashScanStartedAt = &now
	} else {
		ingestor, ok := scanner.(StashIngestor)
		if !ok {
			return s.failStashIngest(task, errors.New("Stash service does not support the complete ingest workflow"))
		}
		scenes, findErr := s.resolveStashScenes(ctx, task, ingestor, false)
		if findErr != nil {
			return s.failStashIngest(task, findErr)
		}
		ids := task.StashSceneIDs()

		switch task.StashIngestPhase {
		case StashIngestGenerate:
			id, err = ingestor.MetadataGenerate(ctx, ids)
		case StashIngestIdentify:
			id, err = ingestor.MetadataIdentify(ctx, ids)
		case StashIngestVerify:
			if verifyErr := verifyIngestScenes(task, ids, scenes); verifyErr != nil {
				// A FINISHED Identify job may have skipped or failed individual matches.
				task.StashIngestPhase = StashIngestIdentify
				task.StashIdentifyJobID = ""
				task.StashScanError = verifyErr.Error()
				blockTask(task, TaskStageErrorIngestVerify, verifyErr.Error(), s.now().UTC())
				return task, verifyErr
			}
			setTaskStage(task, TaskStageCompleted, TaskStageStatusDone)
			return task, nil
		default:
			err = errors.New("invalid Stash ingest phase")
		}
	}
	if err != nil {
		if task.StashIngestPhase != StashIngestScan && isMissingStashSceneError(err) {
			return s.recoverStashScenes(ctx, task, scanner)
		}
		return s.failStashIngest(task, err)
	}
	if strings.TrimSpace(id) == "" {
		return s.failStashIngest(task, errors.New("Stash returned an empty job ID"))
	}
	setStashJobID(task, id)
	logging.Infof("taskruntime: Stash ingest started task=%s phase=%s job=%s", task.ID, task.StashIngestPhase, id)
	return task, nil
}

func verifyIngestScenes(task *Task, ids []string, scenes []stashsync.IngestScene) error {
	byID := make(map[string]stashsync.IngestScene, len(scenes))
	for _, scene := range scenes {
		byID[scene.ID] = scene
	}
	for _, id := range ids {
		scene, ok := byID[id]
		if !ok {
			return fmt.Errorf("Stash scene %s is missing", id)
		}
		if len(scene.Paths) == 0 || strings.TrimSpace(scene.Title) == "" || scene.StudioID == "" || scene.PerformerCount == 0 || scene.Date == "" {
			return fmt.Errorf("Stash scene %s has incomplete metadata (title, date, studio, performers, or media file); configure Stash Identify sources and retry", id)
		}
		if task.Code != "" && extractCode(scene.Code) != normalizeCode(task.Code) {
			return fmt.Errorf("Stash scene %s code does not match task %s; review the identification in Stash and retry", id, task.Code)
		}
	}
	return nil
}

// Query by path rather than by cached IDs: Stash rejects a whole ID query if
// even one record has disappeared. Revalidate the file association as well.
func (s *Service) resolveStashScenes(ctx context.Context, task *Task, ingestor StashIngestor, force bool) ([]stashsync.IngestScene, error) {
	if strings.TrimSpace(task.StashScanPath) == "" {
		return nil, errors.New("Stash scan path is required")
	}
	scenes, err := ingestor.FindIngestScenes(ctx, task.StashScanPath, nil)
	if err != nil {
		return nil, err
	}
	root := filepath.Clean(task.StashScanPath)
	scoped := []stashsync.IngestScene{}
	ids := []string{}
	seen := map[string]bool{}
	for _, scene := range scenes {
		belongs := false
		for _, path := range scene.Paths {
			clean := filepath.Clean(path)
			if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
				belongs = true
				break
			}
		}
		if belongs && scene.ID != "" && !seen[scene.ID] {
			scoped = append(scoped, scene)
			ids = append(ids, scene.ID)
			seen[scene.ID] = true
		}
	}
	if len(ids) == 0 {
		return nil, errStashSceneMissing
	}
	sort.Strings(ids)
	previous := task.StashSceneIDs()
	sort.Strings(previous)
	changed := !slices.Equal(previous, ids)
	if force || (len(previous) > 0 && changed) {
		if task.StashSceneRecoveryCount >= 1 {
			return nil, errStashSceneChanged
		}
		task.StashSceneRecoveryCount++
		task.StashGenerateJobID, task.StashIdentifyJobID = "", ""
		task.StashIngestPhase = StashIngestGenerate
		setTaskStage(task, TaskStageScanning, TaskStageStatusRunning)
		logging.Infof("taskruntime: recovered Stash scenes task=%s old=%v new=%v", task.ID, previous, ids)
	}
	if changed || force {
		encoded, _ := json.Marshal(ids)
		task.StashSceneIDsJSON = string(encoded)
		task.UpdatedAt = s.now().UTC()
		if err := s.store.Update(ctx, task); err != nil {
			return nil, err
		}
	}
	return scoped, nil
}

func isMissingStashSceneError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "scene with id ") && strings.Contains(message, "not found")
}

func (s *Service) recoverStashScenes(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	ingestor, ok := scanner.(StashIngestor)
	if !ok {
		return s.failStashIngest(task, errors.New("Stash service does not support scene recovery"))
	}
	if _, err := s.resolveStashScenes(ctx, task, ingestor, true); err != nil {
		return s.failStashIngest(task, err)
	}
	return s.startStashIngestStep(ctx, task, scanner)
}
