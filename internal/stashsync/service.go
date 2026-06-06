package stashsync

import (
	"context"
	"errors"
	"strings"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type Client interface {
	MetadataScan(ctx context.Context, input stashgraphql.ScanMetadataInput) (string, error)
	FindJob(ctx context.Context, id string) (*stashgraphql.FindJob_FindJob, error)
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
	client       Client
	defaultPaths []string
}

func NewService(client Client, defaultPaths []string) (*Service, error) {
	if client == nil {
		return nil, errors.New("stashsync: client is required")
	}

	return &Service{
		client:       client,
		defaultPaths: cleanPaths(defaultPaths),
	}, nil
}

func (s *Service) MetadataScan(ctx context.Context, req ScanRequest) (string, error) {
	paths := cleanPaths(req.Paths)
	if len(paths) == 0 {
		paths = s.defaultPaths
	}
	if len(paths) == 0 {
		return "", errors.New("stashsync: at least one scan path is required")
	}

	return s.client.MetadataScan(ctx, stashgraphql.ScanMetadataInput{
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
}

func (s *Service) FindJob(ctx context.Context, id string) (*Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("stashsync: job id is required")
	}

	job, err := s.client.FindJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
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
