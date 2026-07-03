package stashsync

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/logging"
	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type Client interface {
	MetadataScan(ctx context.Context, input stashgraphql.ScanMetadataInput) (string, error)
	FindJob(ctx context.Context, id string) (*stashgraphql.FindJob_FindJob, error)
}

type DeliveryMode string

const (
	DeliveryModePathMap  DeliveryMode = "PATH_MAP"
	DeliveryModeTransfer DeliveryMode = "TRANSFER"
)

type TransferAction string

const (
	TransferActionCopy    TransferAction = "COPY"
	TransferActionMove    TransferAction = "MOVE"
	TransferActionSymlink TransferAction = "SYMLINK"
)

type IntegrationConfig struct {
	DeliveryMode DeliveryMode
	Downloads    DownloadsPathConfig
	Library      LibraryPathConfig
	Transfer     TransferConfig
}

type DownloadsPathConfig struct {
	QBRoot   string
	MojiRoot string
}

type LibraryPathConfig struct {
	MojiRoot  string
	StashRoot string
}

type TransferConfig struct {
	Action TransferAction
}

type ScanRequest struct {
	Paths                     []string
	Rescan                    *bool
	ScanGenerateCovers        *bool
	ScanGeneratePreviews      *bool
	ScanGenerateImagePreviews *bool
	ScanGenerateSprites       *bool
	ScanGeneratePhashes       *bool
	ScanGenerateThumbnails    *bool
	ScanGenerateClipPreviews  *bool
}

type Job struct {
	ID          string
	Status      string
	Description string
	Progress    *float64
	StartTime   *time.Time
	EndTime     *time.Time
	AddTime     time.Time
	Error       *string
	SubTasks    []string
}

type Service struct {
	client         Client
	configProvider func() IntegrationConfig
}

func NewService(client Client, configProvider func() IntegrationConfig) (*Service, error) {
	if client == nil {
		return nil, errors.New("stashsync: client is required")
	}

	return &Service{
		client:         client,
		configProvider: configProvider,
	}, nil
}

func (s *Service) MetadataScan(ctx context.Context, req ScanRequest) (string, error) {
	paths := cleanPaths(req.Paths)
	if len(paths) == 0 {
		return "", errors.New("stashsync: at least one scan path is required")
	}
	logging.Infof("stashsync: metadata scan requested for %d paths", len(paths))

	jobID, err := s.client.MetadataScan(ctx, stashgraphql.ScanMetadataInput{
		Paths:                     paths,
		Rescan:                    req.Rescan,
		ScanGenerateCovers:        req.ScanGenerateCovers,
		ScanGeneratePreviews:      req.ScanGeneratePreviews,
		ScanGenerateImagePreviews: req.ScanGenerateImagePreviews,
		ScanGenerateSprites:       req.ScanGenerateSprites,
		ScanGeneratePhashes:       req.ScanGeneratePhashes,
		ScanGenerateThumbnails:    req.ScanGenerateThumbnails,
		ScanGenerateClipPreviews:  req.ScanGenerateClipPreviews,
	})
	if err != nil {
		logging.Errorf("stashsync: metadata scan request failed for paths %v: %v", paths, err)
		return "", err
	}
	logging.Infof("stashsync: metadata scan started with job %s for paths %v", jobID, paths)
	return jobID, nil
}

func (s *Service) CurrentConfig() IntegrationConfig {
	if s == nil || s.configProvider == nil {
		return IntegrationConfig{}
	}
	cfg := s.configProvider()
	cfg.Downloads.QBRoot = strings.TrimSpace(cfg.Downloads.QBRoot)
	cfg.Downloads.MojiRoot = strings.TrimSpace(cfg.Downloads.MojiRoot)
	cfg.Library.MojiRoot = strings.TrimSpace(cfg.Library.MojiRoot)
	cfg.Library.StashRoot = strings.TrimSpace(cfg.Library.StashRoot)
	return cfg
}

func (s *Service) FindJob(ctx context.Context, id string) (*Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("stashsync: job id is required")
	}

	job, err := s.client.FindJob(ctx, id)
	if err != nil {
		logging.Errorf("stashsync: find job %s failed: %v", id, err)
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	if job.Error != nil && strings.TrimSpace(*job.Error) != "" {
		logging.Errorf("stashsync: job %s status=%s error=%s", id, job.Status, *job.Error)
	}

	return &Job{
		ID:          job.ID,
		Status:      string(job.Status),
		Description: job.Description,
		Progress:    job.Progress,
		StartTime:   job.StartTime,
		EndTime:     job.EndTime,
		AddTime:     job.AddTime,
		Error:       job.Error,
		SubTasks:    job.SubTasks,
	}, nil
}

func cleanPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			out = append(out, path)
		}
	}
	return out
}
