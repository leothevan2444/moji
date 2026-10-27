package taskruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/stashsync"
	"github.com/leothevan2444/moji/internal/tracker"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type TorrentAdder interface {
	AddNewTorrent(ctx context.Context, opts qbittorrent.AddTorrentOptions) error
}

type TorrentLister interface {
	GetTorrentList(ctx context.Context, options *qbittorrent.TorrentListOptions) ([]qbittorrent.Torrent, error)
}

type TorrentRemover interface {
	DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error
}

type TorrentClient interface {
	TorrentAdder
	TorrentLister
	TorrentRemover
}

type LibraryCodeChecker interface {
	HasCode(ctx context.Context, code string) (bool, error)
}

type TaskStore interface {
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Find(ctx context.Context, id string) (*Task, error)
	FindByCode(ctx context.Context, code string) (*Task, error)
	FindByTorrentIdentity(ctx context.Context, infoHash string, magnetURI string) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
	Delete(ctx context.Context, id string) (*Task, error)
}

type TaskSource string

const (
	TaskSourceManual       TaskSource = "MANUAL"
	TaskSourceSearch       TaskSource = "SEARCH"
	TaskSourceSubscription TaskSource = "SUBSCRIPTION"
)

type Task struct {
	Revision                int64
	Generation              int64
	RestartStateJSON        string
	TorrentFileData         []byte
	SourcingLastCheckedAt   *time.Time
	SourcingNextCheckAt     *time.Time
	FilePlanJSON            string
	ID                      string
	Source                  TaskSource
	Code                    string
	Stage                   TaskStage
	StageStatus             TaskStageStatus
	StageLabel              string
	StageStatusLabel        string
	StageErrorCode          string
	StageErrorMessage       string
	Candidate               Candidate
	TorrentURL              string
	SavePath                string
	Category                string
	Tags                    string
	TorrentIdentityHash     string
	TorrentIdentityMagnet   string
	TorrentHash             string
	TorrentName             string
	Progress                float64
	QBittorrentState        string
	ContentPath             string
	SourcingCompletedAt     *time.Time
	DownloadCompletedAt     *time.Time
	IngestCompletedAt       *time.Time
	MojiSourcePath          string
	TransferAction          string
	MojiTransferPath        string
	TransferError           string
	StashScanJobID          string
	StashIngestPhase        string
	StashGenerateJobID      string
	StashIdentifyJobID      string
	StashSceneIDsJSON       string
	StashSceneRecoveryCount int
	StashScanPath           string
	StashScanError          string
	StashScanHint           string
	StashScanStartedAt      *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	LastProgressAt          *time.Time
	SourcingRequestJSON     string
}

type Candidate struct {
	Title     string
	Tracker   string
	InfoHash  string
	Link      string
	MagnetURI string
	Size      int64
	Seeders   int
	Peers     int
}

type DownloadRequest struct {
	Source     TaskSource
	Code       string
	Trackers   []string
	Categories []int
	Limit      int
	SavePath   string
	Category   string
	Tags       string
	Paused     *bool
}

type AddTorrentRequest struct {
	Source   TaskSource
	URL      string
	SavePath string
	Category string
	Tags     string
	Paused   *bool
}

type ResolveBlockedSourcingRequest struct {
	TorrentData []byte
	URL         string
	Title       string
	Tracker     string
	InfoHash    string
	Size        int64
	Seeders     int
	Peers       int
	Paused      *bool
}

var (
	ErrInvalidTorrentInput  = errors.New("provide exactly one torrent URL or a non-empty torrent file up to 8 MiB")
	ErrSourcingUnavailable  = errors.New("task is no longer waiting for source selection")
	ErrDuplicateTorrentTask = errors.New("duplicate torrent task")
	ErrDuplicateCodeTask    = errors.New("duplicate code task")
	ErrDuplicateLibraryCode = errors.New("duplicate library code")
	ErrTaskCodeRequired     = errors.New("task code is required")
)

type Service struct {
	sourcingRetryInterval func() time.Duration
	sourcingWake          chan struct{}
	sourcingRunMu         sync.Mutex
	mediaConfig           func() stashsync.IntegrationConfig
	mediaPreparer         mediafiles.Preparer
	mediaContext          context.Context
	mediaMu               sync.Mutex
	mediaWG               sync.WaitGroup
	mediaJobs             map[string]context.CancelFunc
	mediaSlots            chan struct{}

	tracker            tracker.Tracker
	qbt                TorrentClient
	store              TaskStore
	httpClient         *http.Client
	libraryCodeChecker LibraryCodeChecker
	selector           CandidateSelector
	fileOps            FileOperator
	candidateSelection func() config.CandidateSelectionConfig
	taskDeletePolicy   func() config.TaskDeletePolicy
	now                func() time.Time
	newID              func() string
	metadataHostsMu    sync.Mutex
	metadataHosts      map[string]*torrentMetadataHost
	inspectionCacheMu  sync.Mutex
	inspectionCache    map[string]cachedTorrentInspection
	taskLocksMu        sync.Mutex
	taskLocks          map[string]*taskOperationLock
}

type Option func(*Service)

type FileOperator interface {
	Transfer(ctx context.Context, sourcePath string, action stashsync.TransferAction, targetPath string) error
}

type CandidateSelector interface {
	Select(ctx context.Context, query string, results []jackett.SearchResult, cfg config.CandidateSelectionConfig) (jackett.SearchResult, error)
	Preview(ctx context.Context, query string, results []jackett.SearchResult, cfg config.CandidateSelectionConfig, applyFastRules bool, applyFileRules bool) (CandidateSelectionPreview, error)
}

type CandidateSelectionPreview struct {
	Results []jackett.SearchResult
	Meta    CandidateSelectionPreviewMeta
}

type CandidateSelectionPreviewMeta struct {
	AppliedFastRules bool
	AppliedFileRules bool
	InspectedCount   int
	InspectableCount int
}

type PreviewJackettSelectionRequest struct {
	Query                    string
	Results                  []jackett.SearchResult
	ApplyFastRules           bool
	ApplyFileRules           bool
	InspectionCandidateLimit int
}

type cachedTorrentInspection struct {
	inspection torrentInspection
	expiresAt  time.Time
}

const torrentInspectionCacheTTL = 20 * time.Minute

func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

func WithIDGenerator(newID func() string) Option {
	return func(s *Service) {
		if newID != nil {
			s.newID = newID
		}
	}
}

func WithLibraryCodeChecker(checker LibraryCodeChecker) Option {
	return func(s *Service) {
		s.libraryCodeChecker = checker
	}
}

func NewService(tr tracker.Tracker, qbt TorrentClient, store TaskStore, options ...Option) (*Service, error) {
	if tr == nil {
		return nil, errors.New("taskruntime: tracker is required")
	}
	if qbt == nil {
		return nil, errors.New("taskruntime: qBittorrent client is required")
	}
	if store == nil {
		store = NewMemoryTaskStore()
	}

	s := &Service{
		sourcingRetryInterval: func() time.Duration { return 6 * time.Hour },
		sourcingWake:          make(chan struct{}, 1),
		tracker:               tr,
		qbt:                   qbt,
		store:                 store,
		httpClient:            &http.Client{Timeout: 15 * time.Second},
		fileOps:               osFileOperator{},
		candidateSelection:    config.DefaultCandidateSelectionConfig,
		taskDeletePolicy: func() config.TaskDeletePolicy {
			return config.TaskDeletePolicyKeepOnly
		},
		now:             time.Now,
		newID:           newTaskID,
		inspectionCache: make(map[string]cachedTorrentInspection),
		taskLocks:       make(map[string]*taskOperationLock),
	}
	for _, option := range options {
		option(s)
	}
	return s, nil
}

func WithCandidateSelectionProvider(provider func() config.CandidateSelectionConfig) Option {
	return func(s *Service) {
		if provider != nil {
			s.candidateSelection = provider
		}
	}
}

func WithCandidateSelector(selector CandidateSelector) Option {
	return func(s *Service) {
		if selector != nil {
			s.selector = selector
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(s *Service) {
		if client != nil {
			s.httpClient = client
		}
	}
}

func WithFileOperator(fileOps FileOperator) Option {
	return func(s *Service) {
		if fileOps != nil {
			s.fileOps = fileOps
		}
	}
}

func WithTaskDeletePolicyProvider(provider func() config.TaskDeletePolicy) Option {
	return func(s *Service) {
		if provider != nil {
			s.taskDeletePolicy = provider
		}
	}
}

func (s *Service) DownloadMedia(code string) (*Task, error) {
	return s.DownloadMediaContext(context.Background(), DownloadRequest{Code: code})
}

func (s *Service) PreviewJackettSelectionContext(ctx context.Context, req PreviewJackettSelectionRequest) (*CandidateSelectionPreview, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, errors.New("taskruntime: query is required")
	}
	if !req.ApplyFastRules && !req.ApplyFileRules {
		return &CandidateSelectionPreview{
			Results: append([]jackett.SearchResult(nil), req.Results...),
		}, nil
	}

	selectionConfig := config.DefaultCandidateSelectionConfig()
	if s.candidateSelection != nil {
		selectionConfig = s.candidateSelection().Effective()
	}
	if req.InspectionCandidateLimit > 0 {
		selectionConfig.InspectionCandidateLimit = req.InspectionCandidateLimit
	}

	selector := s.selector
	if selector == nil {
		selector = defaultCandidateSelector{inspectTorrent: s.inspectSearchResultTorrent}
	}
	preview, err := selector.Preview(ctx, query, req.Results, selectionConfig, req.ApplyFastRules, req.ApplyFileRules)
	if err != nil {
		return nil, err
	}
	return &preview, nil
}

func (s *Service) FindTask(ctx context.Context, id string) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("taskruntime: task id is required")
	}
	return s.store.Find(ctx, id)
}

func (s *Service) ListTasks(ctx context.Context) ([]*Task, error) {
	return s.store.List(ctx)
}

func (s *Service) DeleteTask(ctx context.Context, id string) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("taskruntime: task id is required")
	}
	unlock := s.lockTask(id)
	defer unlock()

	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	policy := config.TaskDeletePolicyKeepOnly
	if task.RestartStateJSON != "" {
		return task, ErrRestartCleanup
	}
	if s.taskDeletePolicy != nil {
		policy = config.NormalizeTaskDeletePolicy(string(s.taskDeletePolicy()))
	}

	if hash := strings.TrimSpace(task.TorrentHash); hash != "" && policy != config.TaskDeletePolicyKeepOnly {
		deleteFiles := policy == config.TaskDeletePolicyRemoveTorrentAndFiles
		if err := s.runTaskCommand(ctx, task, "TASK_DELETE", hash, map[string]any{"delete_files": deleteFiles}, func() error { return s.qbt.DeleteTorrents(ctx, []string{hash}, deleteFiles) }); err != nil {
			return nil, fmt.Errorf("delete qBittorrent torrent %q with policy %s: %w", hash, policy, err)
		}
	}

	s.mediaMu.Lock()
	if cancel := s.mediaJobs[id]; cancel != nil {
		cancel()
	}
	s.mediaMu.Unlock()
	return s.store.Delete(ctx, id)
}

func (s *Service) RetryTask(ctx context.Context, id string, scanner StashScanner) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("taskruntime: task id is required")
	}
	unlock := s.lockTask(id)
	defer unlock()

	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	if task.RestartStateJSON != "" {
		return s.restartTaskSourcingLocked(ctx, task, scanner)
	}
	if task.StageErrorCode == TaskStageErrorManualSource {
		return task, ErrSourcingUnavailable
	}
	if task.Stage == TaskStageCompleted {
		return nil, fmt.Errorf("taskruntime: task %q is already completed", id)
	}
	if task.StageStatus != TaskStageStatusBlocked && !waitingForResource(task) {
		return nil, fmt.Errorf("taskruntime: task %q is not blocked", id)
	}

	switch task.Stage {
	case TaskStageSourcing:
		return s.retrySourcingTask(ctx, task)
	case TaskStageDownloading:
		return s.retryDownloadingTask(ctx, task)
	case TaskStagePendingIngest, TaskStageTransferring, TaskStageScanning:
		if scanner == nil {
			return nil, errors.New("taskruntime: stash scanner is required")
		}
		return s.retryIngestTask(ctx, task, scanner)
	default:
		return nil, fmt.Errorf("taskruntime: task %q has unsupported retry stage %s", id, task.Stage)
	}
}

func (s *Service) ResolveBlockedSourcingTask(ctx context.Context, id string, req ResolveBlockedSourcingRequest) (*Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("taskruntime: task id is required")
	}
	unlock := s.lockTask(id)
	defer unlock()
	task, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	if task.RestartStateJSON != "" || task.Stage != TaskStageSourcing || (task.StageStatus != TaskStageStatusBlocked && !waitingForResource(task)) {
		return nil, ErrSourcingUnavailable
	}

	torrentURL := strings.TrimSpace(req.URL)
	if (torrentURL == "") == (req.TorrentData == nil) || (req.TorrentData != nil && (len(req.TorrentData) == 0 || len(req.TorrentData) > MaxTorrentBytes)) {
		return task, ErrInvalidTorrentInput
	}
	var candidate Candidate
	var metadata *downloadedTorrentMetadata
	if req.TorrentData != nil {
		parsed, parseErr := parseTorrentMetadata(req.TorrentData)
		if parseErr != nil {
			return task, candidateError(TaskStageErrorTorrentMetadata, "invalid uploaded torrent metadata")
		}
		metadata = &parsed
		candidate = Candidate{Title: parsed.Name, InfoHash: parsed.InfoHash}
		torrentURL = "magnet:?xt=urn:btih:" + parsed.InfoHash + "&dn=" + url.QueryEscape(parsed.Name)
		candidate.MagnetURI = torrentURL
	} else {
		candidate, _, _, metadata, err = s.resolveManualTorrent(ctx, torrentURL)
	}
	if err != nil {
		if waitingForResource(task) {
			return task, candidateError(TaskStageErrorTorrentMetadata, "unable to read selected torrent metadata")
		}
		return task, s.blockSourcingValidation(ctx, task, candidateError(TaskStageErrorTorrentMetadata, "unable to read selected torrent metadata"))
	}
	if value := strings.TrimSpace(req.Title); value != "" {
		candidate.Title = value
	}
	if value := strings.TrimSpace(req.Tracker); value != "" {
		candidate.Tracker = value
	}
	if value := normalizeInfoHash(req.InfoHash); value != "" {
		candidate.InfoHash = value
	}
	if req.Size > 0 {
		candidate.Size = req.Size
	}
	candidate.Seeders = req.Seeders
	candidate.Peers = req.Peers
	var prepared preparedTorrent
	if req.TorrentData != nil {
		prepared, err = prepareTorrentFile(task.Code, candidate, *metadata)
	} else {
		prepared, err = s.prepareTorrentWithMetadata(ctx, task.Code, candidate, torrentURL, metadata)
	}
	if err != nil {
		if waitingForResource(task) {
			return task, err
		}
		return task, s.blockSourcingValidation(ctx, task, err)
	}
	candidate = prepared.candidate
	identity := torrentIdentityFromCandidate(candidate, torrentURL)
	if err := s.ensureTaskIdentityAvailable(ctx, task.ID, identity); err != nil {
		return task, err
	}

	next := cloneTask(task)
	next.TorrentFileData = append([]byte(nil), req.TorrentData...)
	if err := ensureSourcingRequest(next); err != nil {
		return task, err
	}
	next.SourcingNextCheckAt = nil
	setTaskStage(next, TaskStageSourcing, TaskStageStatusRunning)
	next.Candidate = candidate
	next.TorrentURL = torrentURL
	next.TorrentIdentityHash = identity.InfoHash
	next.TorrentIdentityMagnet = identity.MagnetURI
	if err := s.submitPreparedTaskTorrent(ctx, next, torrentURL, prepared, next.SavePath, next.Category, next.Tags, req.Paused); err != nil {
		return next, err
	}
	return next, nil
}

func (s *Service) SyncProgress(ctx context.Context) ([]*Task, error) {
	tasks, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	torrents, err := s.qbt.GetTorrentList(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list torrents: %w", err)
	}

	updated := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		next, syncErr := s.syncTaskProgress(ctx, task, torrents)
		if syncErr != nil {
			return updated, syncErr
		}
		updated = append(updated, next)
	}

	return updated, nil
}

func (s *Service) syncTaskProgress(ctx context.Context, snapshot *Task, torrents []qbittorrent.Torrent) (*Task, error) {
	if snapshot == nil {
		return nil, nil
	}
	unlock := s.lockTask(snapshot.ID)
	defer unlock()
	task, err := s.store.Find(ctx, snapshot.ID)
	if err != nil {
		return snapshot, nil
	}
	if task.RestartStateJSON != "" || task.Stage == TaskStageSourcing || taskHashForMonitoring(task) == "" && task.Stage != TaskStageDownloading {
		return task, nil
	}
	torrent, ok := matchTaskTorrent(task, torrents)
	if task.Stage != TaskStageDownloading {
		state := "missing"
		if ok {
			state = string(torrent.State)
		}
		contentPath := task.ContentPath
		if ok && torrent.ContentPath != "" {
			contentPath = torrent.ContentPath
		}
		if task.QBittorrentState == state && contentPath == task.ContentPath {
			return task, nil
		}
		next := cloneTask(task)
		next.QBittorrentState = state
		next.ContentPath = contentPath
		next.UpdatedAt = s.now().UTC()
		if err := s.store.Update(ctx, next); err != nil {
			return task, err
		}
		return next, nil
	}
	// Recover older submissions without ever attaching an unrelated fuzzy candidate.
	if task.TorrentHash == "" && task.Progress == 0 {
		if _, err := validateCodeEvidence(task.Code, task.Candidate.Title, magnetDisplayName(task.TorrentURL)); err != nil {
			next := cloneTask(task)
			if blockErr := s.blockSourcingValidation(ctx, next, err); !errors.Is(blockErr, err) {
				return task, blockErr
			}
			return next, nil
		}
	}
	problem := ""
	if !ok {
		if taskHashForMonitoring(task) == "" {
			return task, nil
		}
		problem = "TORRENT_MISSING"
	} else if torrent.State == qbittorrent.TorrentStateError || torrent.State == qbittorrent.TorrentStateMissingFiles {
		problem = "TORRENT_ERROR"
	}
	if problem != "" {
		state := "missing"
		if ok {
			state = string(torrent.State)
		}
		if task.QBittorrentState == state && task.StageErrorCode == problem {
			return task, nil
		}
		next := cloneTask(task)
		next.QBittorrentState = state
		// Preserve file-review and other independent blockers.
		if task.StageStatus != TaskStageStatusBlocked || task.StageErrorCode == "TORRENT_MISSING" || task.StageErrorCode == "TORRENT_ERROR" {
			blockTask(next, problem, problem, s.now().UTC())
		}
		if err := s.store.Update(ctx, next); err != nil {
			return task, err
		}
		return next, nil
	}

	next := cloneTask(task)
	prevStage := next.Stage
	prevStageStatus := next.StageStatus
	prevProgress := next.Progress
	if next.StageErrorCode == "TORRENT_MISSING" || next.StageErrorCode == "TORRENT_ERROR" {
		clearTaskStageError(next)
	}
	applyTorrentProgress(next, torrent, s.now().UTC())
	if s.mediaConfig != nil {
		if task.StageStatus == TaskStageStatusBlocked && task.StageErrorCode == TaskStageErrorFileReview {
			return task, nil
		}
		complete, fileErr := s.syncFileSelection(ctx, next)
		switch torrent.State {
		case qbittorrent.TorrentStateMoving, qbittorrent.TorrentStateCheckingDL, qbittorrent.TorrentStateCheckingUP, qbittorrent.TorrentStateCheckingResumeData:
			complete = false
		}
		if fileErr != nil {
			setTaskStage(next, TaskStageDownloading, TaskStageStatusRunning)
			next.DownloadCompletedAt = nil
			code := TaskStageErrorFileSelection
			if errors.Is(fileErr, mediafiles.ErrReview) {
				code = TaskStageErrorFileReview
			}
			blockTask(next, code, fileErr.Error(), s.now().UTC())
		} else if !complete {
			setTaskStage(next, TaskStageDownloading, TaskStageStatusRunning)
			next.DownloadCompletedAt = nil
		} else {
			setTaskStage(next, TaskStagePendingIngest, TaskStageStatusPending)
			clearTaskStageError(next)
			if next.DownloadCompletedAt == nil {
				now := s.now().UTC()
				next.DownloadCompletedAt = &now
			}
		}
	}
	if !taskChangedBeyondProgress(task, next) && task.Progress == next.Progress {
		return task, nil
	}
	if err := s.store.Update(ctx, next); err != nil {
		return task, fmt.Errorf("update task %q: %w", next.ID, err)
	}
	if next.Stage != prevStage || next.StageStatus != prevStageStatus {
		logging.Infof(
			"taskruntime: task %s stage %s/%s -> %s/%s (%s %.1f%%)",
			next.ID,
			prevStage,
			prevStageStatus,
			next.Stage,
			next.StageStatus,
			next.Candidate.Title,
			next.Progress*100,
		)
	} else if next.Stage == TaskStagePendingIngest && next.Progress != prevProgress {
		logging.Infof("taskruntime: task %s download completed with content path %s", next.ID, next.ContentPath)
	}
	return next, nil
}

func (s *Service) retrySourcingTask(ctx context.Context, task *Task) (*Task, error) {
	next := cloneTask(task)
	next.SourcingNextCheckAt = nil
	if next.SourcingRequestJSON != "" {
		setTaskStage(next, TaskStageSourcing, TaskStageStatusPending)
		clearTaskStageError(next)
		next.UpdatedAt = s.now().UTC()
		if err := s.store.Update(ctx, next); err != nil {
			return nil, err
		}
		select {
		case s.sourcingWake <- struct{}{}:
		default:
		}
		return next, nil
	}
	setTaskStage(next, TaskStageSourcing, TaskStageStatusRunning)
	clearTaskStageError(next)
	next.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, next); err != nil {
		return nil, fmt.Errorf("update task %q: %w", next.ID, err)
	}
	return s.runSourcingFlow(ctx, next, DownloadRequest{
		Source:   next.Source,
		Code:     next.Code,
		SavePath: next.SavePath,
		Category: next.Category,
		Tags:     next.Tags,
	})
}

func (s *Service) retryDownloadingTask(ctx context.Context, task *Task) (*Task, error) {
	if s.mediaConfig != nil && task.TorrentHash != "" && task.StageErrorCode != TaskStageErrorTorrentSubmit {
		return s.retryFileDownload(ctx, task)
	}
	next := cloneTask(task)
	if err := s.submitTaskTorrent(ctx, next, next.TorrentURL, next.SavePath, next.Category, next.Tags, nil); err != nil {
		return next, err
	}
	return next, nil
}

func (s *Service) retryIngestTask(ctx context.Context, task *Task, scanner StashScanner) (*Task, error) {
	next := cloneTask(task)
	next.StashSceneRecoveryCount = 0
	setTaskStage(next, next.Stage, TaskStageStatusPending)
	clearTaskStageError(next)
	next.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, next); err != nil {
		return nil, fmt.Errorf("update task %q: %w", next.ID, err)
	}

	var retried *Task
	var execErr error
	if next.Stage == TaskStageScanning {
		if next.StashIngestPhase == "" {
			next.StashIngestPhase = StashIngestScan
		}
		if stashJobID(next) != "" && next.StashIngestPhase != StashIngestVerify {
			retried, execErr = s.syncStashScanJob(ctx, next, scanner)
		} else {
			retried, execErr = s.startStashIngestStep(ctx, next, scanner)
		}
	} else {
		retried, execErr = s.executeTaskStashIntegration(ctx, next, scanner)
	}
	if persistErr := s.store.Update(ctx, retried); persistErr != nil {
		return nil, fmt.Errorf("update task %q: %w", retried.ID, persistErr)
	}
	if execErr != nil {
		return retried, execErr
	}
	return retried, nil
}

func (s *Service) DownloadMediaContext(ctx context.Context, req DownloadRequest) (*Task, error) {
	code := extractCode(req.Code)
	if code == "" {
		return nil, ErrTaskCodeRequired
	}
	if err := s.ensureTaskCodeCanBeCreated(ctx, code); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	source := req.Source
	if source == "" {
		source = TaskSourceManual
	}
	task := &Task{
		ID:        s.newID(),
		Source:    source,
		Code:      code,
		SavePath:  req.SavePath,
		Category:  req.Category,
		Tags:      req.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusRunning)

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("taskruntime: create task failed for code %q: %v", code, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof("taskruntime: created %s task %s for code %q", strings.ToLower(string(source)), task.ID, code)

	return s.runSourcingFlow(ctx, task, req)
}

func (s *Service) runSourcingFlow(ctx context.Context, task *Task, req DownloadRequest) (*Task, error) {
	code := strings.TrimSpace(task.Code)
	searchOptions := []tracker.SearchOption{
		tracker.WithTrackers(req.Trackers),
		tracker.WithCategories(req.Categories),
	}
	if req.Limit > 0 {
		searchOptions = append(searchOptions, tracker.WithLimit(req.Limit))
	}

	results, err := s.tracker.Search(code, searchOptions...)
	if ctx.Err() != nil {
		return task, ctx.Err()
	}
	if err != nil {
		blockTask(task, TaskStageErrorSearch, err.Error(), s.now().UTC())
		_ = s.store.Update(ctx, task)
		logging.Errorf("taskruntime: search failed for code %q: %v", code, err)
		return task, fmt.Errorf("search torrents: %w", err)
	}
	logging.Infof("taskruntime: search returned %d results for code %q", len(results), code)
	if len(results) == 0 {
		err = errors.New("no candidate found for the current code")
		if persistErr := s.recordSourcingFailure(ctx, task, TaskStageErrorNoCandidate, err.Error()); persistErr != nil {
			return task, persistErr
		}
		return task, err
	}

	selectionConfig := config.DefaultCandidateSelectionConfig()
	if s.candidateSelection != nil {
		selectionConfig = s.candidateSelection().Effective()
	}
	selector := s.selector
	if selector == nil {
		selector = defaultCandidateSelector{inspectTorrent: s.inspectSearchResultTorrent}
	}
	result, prepared, err := s.selectVerifiedCandidate(ctx, selector, code, results, selectionConfig)
	if err != nil {
		var validation *candidateValidationError
		if errors.As(err, &validation) {
			return task, s.blockSourcingValidation(ctx, task, err)
		}
		errorCode := TaskStageErrorSearch
		if strings.Contains(err.Error(), "no downloadable torrent candidate found") {
			errorCode = TaskStageErrorNoDownloadCandidate
		}
		if persistErr := s.recordSourcingFailure(ctx, task, errorCode, err.Error()); persistErr != nil {
			return task, persistErr
		}
		if resourceUnavailable(errorCode) {
			logging.Infof("taskruntime: no verified resource yet for code %q", code)
		} else {
			logging.Errorf("taskruntime: select candidate failed for code %q: %v", code, err)
		}
		return task, err
	}

	candidate := prepared.candidate
	torrentURL := preferredTorrentURL(result)
	identity := torrentIdentityFromCandidate(candidate, torrentURL)
	if err := s.ensureTaskIdentityAvailable(ctx, task.ID, identity); err != nil {
		blockTask(task, TaskStageErrorDuplicateTorrent, err.Error(), s.now().UTC())
		_ = s.store.Update(ctx, task)
		return task, err
	}
	task.Candidate = candidate
	task.TorrentFileData = nil
	task.TorrentURL = torrentURL
	task.TorrentIdentityHash = identity.InfoHash
	task.TorrentIdentityMagnet = identity.MagnetURI

	if err := s.submitPreparedTaskTorrent(ctx, task, torrentURL, prepared, req.SavePath, req.Category, req.Tags, req.Paused); err != nil {
		return task, err
	}
	logging.Infof("taskruntime: %s task %s added to qBittorrent for code %q", strings.ToLower(string(task.Source)), task.ID, code)

	return task, nil
}

func (s *Service) AddTorrentContext(ctx context.Context, req AddTorrentRequest) (*Task, error) {
	torrentURL := strings.TrimSpace(req.URL)
	if torrentURL == "" {
		return nil, errors.New("taskruntime: torrent url is required")
	}

	candidate, code, identity, metadata, err := s.resolveManualTorrent(ctx, torrentURL)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTaskCanBeCreated(ctx, identity, code); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	source := req.Source
	if source == "" {
		source = TaskSourceManual
	}
	task := &Task{
		ID:                    s.newID(),
		Source:                source,
		Code:                  code,
		Stage:                 TaskStageSourcing,
		StageStatus:           TaskStageStatusRunning,
		Candidate:             candidate,
		TorrentURL:            torrentURL,
		SavePath:              req.SavePath,
		Category:              req.Category,
		Tags:                  req.Tags,
		TorrentIdentityHash:   identity.InfoHash,
		TorrentIdentityMagnet: identity.MagnetURI,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	refreshTaskStageFields(task)

	if err := s.store.Create(ctx, task); err != nil {
		logging.Errorf("taskruntime: create manual task failed for url %q: %v", torrentURL, err)
		return nil, fmt.Errorf("create task: %w", err)
	}
	logging.Infof("taskruntime: created %s task %s for url %q", strings.ToLower(string(source)), task.ID, torrentURL)

	if err := s.submitTaskTorrentWithMetadata(ctx, task, torrentURL, metadata, req.SavePath, req.Category, req.Tags, req.Paused); err != nil {
		return task, err
	}
	logging.Infof("taskruntime: %s task %s added to qBittorrent", strings.ToLower(string(source)), task.ID)

	return task, nil
}

func (s *Service) submitTaskTorrent(ctx context.Context, task *Task, torrentURL string, savePath string, category string, tags string, paused *bool) error {
	return s.submitTaskTorrentWithMetadata(ctx, task, torrentURL, nil, savePath, category, tags, paused)
}

func (s *Service) submitTaskTorrentWithMetadata(ctx context.Context, task *Task, torrentURL string, metadata *downloadedTorrentMetadata, savePath string, category string, tags string, paused *bool) error {
	var prepared preparedTorrent
	var err error
	if len(task.TorrentFileData) > 0 {
		parsed, parseErr := parseTorrentMetadata(task.TorrentFileData)
		if parseErr != nil {
			return s.blockSourcingValidation(ctx, task, candidateError(TaskStageErrorTorrentMetadata, "invalid persisted torrent metadata"))
		}
		prepared, err = prepareTorrentFile(task.Code, task.Candidate, parsed)
	} else {
		prepared, err = s.prepareTorrentWithMetadata(ctx, task.Code, task.Candidate, torrentURL, metadata)
	}
	if err != nil {
		return s.blockSourcingValidation(ctx, task, err)
	}
	task.Candidate = prepared.candidate
	identity := torrentIdentityFromCandidate(task.Candidate, torrentURL)
	if err := s.ensureTaskIdentityAvailable(ctx, task.ID, identity); err != nil {
		return s.blockSourcingValidation(ctx, task, candidateError(TaskStageErrorDuplicateTorrent, err.Error()))
	}
	task.TorrentIdentityHash = identity.InfoHash
	task.TorrentIdentityMagnet = identity.MagnetURI
	return s.submitPreparedTaskTorrent(ctx, task, torrentURL, prepared, savePath, category, tags, paused)
}

func (s *Service) submitPreparedTaskTorrent(ctx context.Context, task *Task, torrentURL string, prepared preparedTorrent, savePath string, category string, tags string, paused *bool) error {
	addOptions := prepared.options
	if s.mediaConfig != nil {
		stopCondition := "MetadataReceived"
		addOptions.StopCondition = &stopCondition
		p := task.FilePlan()
		p.KeepSubtitles = s.mediaConfig().KeepSubtitles
		p.PauseRequested = paused != nil && *paused
		task.setFilePlan(p)
		// .torrent metadata is available while stopped. Magnets need a short metadata
		// acquisition phase before qB exposes their file list.
		if len(addOptions.Torrents) > 0 || !strings.HasPrefix(strings.ToLower(torrentURL), "magnet:") {
			stopped := true
			paused = &stopped
		}
	}
	if savePath != "" {
		addOptions.SavePath = &savePath
	}
	if category != "" {
		addOptions.Category = &category
	}
	if tags != "" {
		addOptions.Tags = &tags
	}
	if paused != nil {
		addOptions.Paused = paused
	}
	if task.SourcingRequestJSON == "" {
		setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
	}
	clearTaskStageError(task)
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		return fmt.Errorf("persist validated torrent before submission: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.runTaskCommand(ctx, task, "TASK_ADD", task.TorrentIdentityHash, map[string]any{"torrent_url": torrentURL}, func() error { return s.qbt.AddNewTorrent(ctx, addOptions) }); err != nil && !s.adoptExistingTorrent(ctx, task) {
		setTaskStage(task, TaskStageDownloading, TaskStageStatusBlocked)
		blockTask(task, TaskStageErrorTorrentSubmit, err.Error(), s.now().UTC())
		_ = s.store.Update(ctx, task)
		logging.Errorf("taskruntime: add torrent failed for task %s: %v", task.ID, err)
		return fmt.Errorf("add torrent: %w", err)
	}

	setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
	clearTaskStageError(task)
	task.UpdatedAt = s.now().UTC()
	if task.SourcingCompletedAt == nil {
		completedAt := task.UpdatedAt
		task.SourcingCompletedAt = &completedAt
	}
	if err := s.store.Update(ctx, task); err != nil {
		logging.Errorf("taskruntime: persist task %s failed after qB submit: %v", task.ID, err)
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

// A rejected add can be a retry after qB already accepted the same torrent.
// Reconcile only an exact validated info hash; never adopt a fuzzy title match.
func (s *Service) adoptExistingTorrent(ctx context.Context, task *Task) bool {
	hash := normalizeInfoHash(task.TorrentIdentityHash)
	if hash == "" || ctx.Err() != nil {
		return false
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	torrents, err := s.qbt.GetTorrentList(lookupCtx, &qbittorrent.TorrentListOptions{Hashes: []string{hash}})
	if err != nil {
		return false
	}
	for _, torrent := range torrents {
		if normalizeInfoHash(torrent.Hash) != hash {
			continue
		}
		task.TorrentHash = strings.ToLower(torrent.Hash)
		task.TorrentName = torrent.Name
		task.Progress = torrent.Progress
		task.QBittorrentState = string(torrent.State)
		task.ContentPath = torrent.ContentPath
		task.SavePath = torrent.SavePath
		logging.Infof("taskruntime: reconciled existing qB torrent for task %s", task.ID)
		return true
	}
	return false
}

func preferredTorrentURL(result jackett.SearchResult) string {
	if result.MagnetURI != "" {
		return result.MagnetURI
	}
	return result.Link
}

func candidateFromSearchResult(result jackett.SearchResult) Candidate {
	return Candidate{
		Title:     result.Title,
		Tracker:   result.Tracker,
		InfoHash:  result.InfoHash,
		Link:      result.Link,
		MagnetURI: result.MagnetURI,
		Size:      result.Size,
		Seeders:   result.Seeders,
		Peers:     result.Peers,
	}
}

func candidateFromTorrentURL(torrentURL string) Candidate {
	return Candidate{
		Title:     torrentURL,
		InfoHash:  infoHashFromMagnet(torrentURL),
		Link:      torrentURL,
		MagnetURI: magnetURI(torrentURL),
	}
}

func magnetURI(torrentURL string) string {
	if strings.HasPrefix(strings.ToLower(torrentURL), "magnet:") {
		return torrentURL
	}
	return ""
}

func infoHashFromMagnet(torrentURL string) string {
	if !strings.HasPrefix(strings.ToLower(torrentURL), "magnet:") {
		return ""
	}

	parsed, err := url.Parse(torrentURL)
	if err != nil {
		return ""
	}
	for _, xt := range parsed.Query()["xt"] {
		parts := strings.Split(xt, ":")
		if len(parts) >= 3 && strings.EqualFold(parts[0], "urn") && strings.EqualFold(parts[1], "btih") {
			return parts[2]
		}
	}
	return ""
}

func matchTaskTorrent(task *Task, torrents []qbittorrent.Torrent) (qbittorrent.Torrent, bool) {
	if task.TorrentHash != "" {
		for _, torrent := range torrents {
			if strings.EqualFold(torrent.Hash, task.TorrentHash) {
				return torrent, true
			}
		}
		return qbittorrent.Torrent{}, false
	}
	if task.TorrentIdentityHash != "" {
		for _, torrent := range torrents {
			if strings.EqualFold(torrent.Hash, task.TorrentIdentityHash) {
				return torrent, true
			}
		}
	}

	needle := normalizeMagnetURI(task.TorrentURL)
	for _, torrent := range torrents {
		if needle != "" && normalizeMagnetURI(torrent.MagnetURI) == needle {
			return torrent, true
		}
		if task.Candidate.Title != "" && torrent.Name == task.Candidate.Title {
			return torrent, true
		}
		if task.Candidate.InfoHash != "" && strings.EqualFold(torrent.Hash, task.Candidate.InfoHash) {
			return torrent, true
		}
	}

	return qbittorrent.Torrent{}, false
}

func applyTorrentProgress(task *Task, torrent qbittorrent.Torrent, now time.Time) {
	task.TorrentHash = torrent.Hash
	task.TorrentIdentityHash = normalizeInfoHash(torrent.Hash)
	task.TorrentName = torrent.Name
	task.Progress = torrent.Progress
	task.QBittorrentState = string(torrent.State)
	task.ContentPath = torrent.ContentPath
	task.SavePath = torrent.SavePath
	task.UpdatedAt = now

	if torrent.Progress >= 1 || torrent.CompletionOn > 0 || isCompletedTorrentState(torrent.State) {
		setTaskStage(task, TaskStagePendingIngest, TaskStageStatusPending)
		clearTaskStageError(task)
		completedAt := now
		if torrent.CompletionOn > 0 {
			completedAt = time.Unix(torrent.CompletionOn, 0).UTC()
		}
		task.DownloadCompletedAt = &completedAt
		return
	}

	if torrent.Progress > 0 || isDownloadingTorrentState(torrent.State) {
		setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
		return
	}

	setTaskStage(task, TaskStageDownloading, TaskStageStatusRunning)
}

func taskHashForMonitoring(task *Task) string {
	return firstNonEmpty([]string{task.TorrentHash, task.TorrentIdentityHash})
}

func isCompletedTorrentState(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.TorrentState("stoppedUP"), qbittorrent.TorrentStateUploading,
		qbittorrent.TorrentStatePausedUP,
		qbittorrent.TorrentStateQueuedUP,
		qbittorrent.TorrentStateStalledUP,
		qbittorrent.TorrentStateCheckingUP,
		qbittorrent.TorrentStateForcedUP:
		return true
	default:
		return false
	}
}

func isDownloadingTorrentState(state qbittorrent.TorrentState) bool {
	switch state {
	case qbittorrent.TorrentStateAllocating,
		qbittorrent.TorrentStateDownloading,
		qbittorrent.TorrentStateMetaDL,
		qbittorrent.TorrentStatePausedDL,
		qbittorrent.TorrentStateQueuedDL,
		qbittorrent.TorrentStateStalledDL,
		qbittorrent.TorrentStateCheckingDL,
		qbittorrent.TorrentStateForcedDL,
		qbittorrent.TorrentStateCheckingResumeData,
		qbittorrent.TorrentStateMoving:
		return true
	default:
		return false
	}
}

func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return "task-" + hex.EncodeToString(b[:])
}

type MemoryTaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{tasks: make(map[string]*Task)}
}

func (s *MemoryTaskStore) Create(_ context.Context, task *Task) error {
	if task == nil {
		return errors.New("taskruntime: task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("taskruntime: task %q already exists", task.ID)
	}
	trackTaskProgress(nil, task)
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryTaskStore) Update(_ context.Context, task *Task) error {
	if task == nil {
		return errors.New("taskruntime: task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("taskruntime: task %q not found", task.ID)
	}
	trackTaskProgress(s.tasks[task.ID], task)
	s.tasks[task.ID] = cloneTask(task)
	return nil
}

func (s *MemoryTaskStore) Find(_ context.Context, id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	return cloneTask(task), nil
}

func (s *MemoryTaskStore) FindByCode(_ context.Context, code string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	for _, task := range s.tasks {
		if strings.EqualFold(strings.TrimSpace(task.Code), code) {
			return cloneTask(task), nil
		}
	}
	return nil, nil
}

func (s *MemoryTaskStore) FindByTorrentIdentity(_ context.Context, infoHash string, magnetURI string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infoHash = normalizeInfoHash(infoHash)
	magnetURI = normalizeMagnetURI(magnetURI)
	if infoHash == "" && magnetURI == "" {
		return nil, nil
	}
	for _, task := range s.tasks {
		if infoHash != "" && normalizeInfoHash(task.TorrentHash) == infoHash {
			return cloneTask(task), nil
		}
		if infoHash != "" && normalizeInfoHash(task.TorrentIdentityHash) == infoHash {
			return cloneTask(task), nil
		}
		if infoHash != "" && normalizeInfoHash(task.Candidate.InfoHash) == infoHash {
			return cloneTask(task), nil
		}
		if magnetURI != "" {
			if normalizeMagnetURI(task.TorrentIdentityMagnet) == magnetURI ||
				normalizeMagnetURI(task.TorrentURL) == magnetURI ||
				normalizeMagnetURI(task.Candidate.MagnetURI) == magnetURI {
				return cloneTask(task), nil
			}
		}
	}
	return nil, nil
}

func (s *MemoryTaskStore) List(_ context.Context) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	sortTasks(tasks)
	return tasks, nil
}

func (s *MemoryTaskStore) Delete(_ context.Context, id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("taskruntime: task %q not found", id)
	}
	delete(s.tasks, id)
	return cloneTask(task), nil
}

func sortTasks(tasks []*Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
}

func cloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cp := *task
	cp.TorrentFileData = append([]byte(nil), task.TorrentFileData...)
	cp.SourcingLastCheckedAt = cloneTime(task.SourcingLastCheckedAt)
	cp.SourcingNextCheckAt = cloneTime(task.SourcingNextCheckAt)
	cp.LastProgressAt = cloneTime(task.LastProgressAt)
	cp.SourcingCompletedAt = cloneTime(task.SourcingCompletedAt)
	cp.DownloadCompletedAt = cloneTime(task.DownloadCompletedAt)
	cp.IngestCompletedAt = cloneTime(task.IngestCompletedAt)
	cp.StashScanStartedAt = cloneTime(task.StashScanStartedAt)
	refreshTaskStageFields(&cp)
	return &cp
}

func blockTask(task *Task, code string, message string, now time.Time) {
	if task == nil {
		return
	}
	setTaskStage(task, task.Stage, TaskStageStatusBlocked)
	setTaskStageError(task, code, message)
	task.UpdatedAt = now
	refreshTaskStageFields(task)
}

type osFileOperator struct{}

func (osFileOperator) Transfer(ctx context.Context, sourcePath string, action stashsync.TransferAction, targetPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(sourcePath) == "" {
		return errors.New("taskruntime: source path is required for file transfer")
	}
	if strings.TrimSpace(targetPath) == "" {
		return errors.New("taskruntime: target path is required for file transfer")
	}
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("taskruntime: transfer target already exists: %s", targetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("taskruntime: stat transfer target %q: %w", targetPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("taskruntime: create transfer target dir for %q: %w", targetPath, err)
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("taskruntime: stat transfer source %q: %w", sourcePath, err)
	}

	switch action {
	case stashsync.TransferActionCopy:
		if sourceInfo.IsDir() {
			return copyDir(ctx, sourcePath, targetPath)
		}
		return copyFile(ctx, sourcePath, targetPath)
	case stashsync.TransferActionMove:
		if err := os.Rename(sourcePath, targetPath); err == nil {
			return nil
		} else if !isCrossDeviceError(err) {
			return fmt.Errorf("taskruntime: move %q -> %q: %w", sourcePath, targetPath, err)
		}
		if sourceInfo.IsDir() {
			if err := copyDir(ctx, sourcePath, targetPath); err != nil {
				return err
			}
			if err := os.RemoveAll(sourcePath); err != nil {
				return fmt.Errorf("taskruntime: remove transferred source %q: %w", sourcePath, err)
			}
			return nil
		}
		if err := copyFile(ctx, sourcePath, targetPath); err != nil {
			return err
		}
		if err := os.Remove(sourcePath); err != nil {
			return fmt.Errorf("taskruntime: remove transferred source %q: %w", sourcePath, err)
		}
		return nil
	default:
		return fmt.Errorf("taskruntime: unsupported transfer action %q", action)
	}
}

func copyDir(ctx context.Context, sourcePath string, targetPath string) error {
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return fmt.Errorf("taskruntime: create transfer target dir %q: %w", targetPath, err)
	}
	return filepath.Walk(sourcePath, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(sourcePath, current)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		destination := filepath.Join(targetPath, relative)
		if info.IsDir() {
			if err := os.MkdirAll(destination, info.Mode()); err != nil {
				return fmt.Errorf("taskruntime: create transfer target dir %q: %w", destination, err)
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("taskruntime: create transfer target dir for %q: %w", destination, err)
		}
		return copyFile(ctx, current, destination)
	})
}

func copyFile(ctx context.Context, sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("taskruntime: open transfer source %q: %w", sourcePath, err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("taskruntime: create transfer target %q: %w", targetPath, err)
	}
	defer target.Close()

	if _, err := io.Copy(target, &contextReader{ctx: ctx, reader: source}); err != nil {
		return fmt.Errorf("taskruntime: copy %q -> %q: %w", sourcePath, targetPath, err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("taskruntime: finalize transfer target %q: %w", targetPath, err)
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func isCrossDeviceError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "cross-device link")
}

// WaitBackground is called after HTTP admission and queue workers have stopped.
func (s *Service) WaitBackground() {
	s.mediaMu.Lock()
	for _, cancel := range s.mediaJobs {
		cancel()
	}
	s.mediaMu.Unlock()
	s.mediaWG.Wait()
}
