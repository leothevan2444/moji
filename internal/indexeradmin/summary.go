package indexeradmin

import (
	"context"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdefs"
)

// Summary is a bounded, local projection. Reading Home never tests a site or
// loads the remote Jackett directory. It exposes neither settings nor locators.
type DefinitionJobReader interface {
	CurrentJob(context.Context) (*indexerdefs.Job, error)
}

func (s *Service) SetDefinitionJobReader(reader DefinitionJobReader) { s.definitionJobs = reader }

type Summary struct {
	DefinitionJob                                *indexerdefs.Job
	DefinitionErrorCode                          string
	Revision                                     string
	Total                                        *int
	BuiltinEnabled, BuiltinUnverified, Attention int
	BuiltinSearchSucceeded                       int
	HealthErrorCode                              string
	Issues                                       []SummaryIssue
	Jackett                                      *JackettSummary
}

type SummaryIssue struct {
	ID, Name, ErrorCode string
	Operation           string
	Observation         *indexer.Observation
}
type JackettSummary struct {
	Sites     *int
	Stale     bool
	UpdatedAt *time.Time
	ErrorCode string
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	page, err := s.Instances(ctx)
	if err != nil {
		return Summary{}, err
	}
	result := summarize(page)
	if s.definitionJobs != nil {
		job, err := s.definitionJobs.CurrentJob(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return Summary{}, ctx.Err()
			}
			result.DefinitionErrorCode = indexerdefs.ErrUnavailable.Error()
			result.Attention++
		} else if job != nil && (job.State == indexerdefs.Failed || job.State == indexerdefs.Interrupted) {
			result.DefinitionJob = job
			result.Attention++
		}
	}
	return result, nil
}

func summarize(page InstancePage) Summary {
	result := Summary{Revision: page.Revision, HealthErrorCode: page.HealthErrorCode, Issues: []SummaryIssue{}}
	remote := 0
	for _, site := range page.Items {
		if !site.Enabled {
			continue
		}
		if site.Kind == indexer.Jackett {
			remote++
		} else {
			result.BuiltinEnabled++
			if site.ErrorCode == "" && page.HealthErrorCode == "" {
				if site.Health.Search == nil {
					result.BuiltinUnverified++
				} else if site.Health.Search.ErrorCode == "" {
					result.BuiltinSearchSucceeded++
				}
			}
		}
		code := site.ErrorCode
		operation := ""
		var observation *indexer.Observation
		if code == "" {
			if site.Health.Search != nil && site.Health.Search.ErrorCode != "" {
				observation = site.Health.Search
				operation = string(indexer.HealthSearch)
			}
			if site.Health.Fetch != nil && site.Health.Fetch.ErrorCode != "" && (observation == nil || site.Health.Fetch.StartedAt.After(observation.StartedAt)) {
				observation = site.Health.Fetch
				operation = string(indexer.HealthFetch)
			}
			if observation != nil {
				code = observation.ErrorCode
			}
		}
		if code == "" {
			continue
		}
		result.Attention++
		if len(result.Issues) < 5 {
			result.Issues = append(result.Issues, SummaryIssue{ID: site.ID, Name: site.Name, ErrorCode: code, Operation: operation, Observation: observation})
		}
	}
	total := result.BuiltinEnabled
	result.Total = &total
	if page.JackettEnabled && page.JackettConfigured {
		result.Jackett = &JackettSummary{Stale: page.JackettCatalog.Stale, UpdatedAt: page.JackettCatalog.UpdatedAt, ErrorCode: page.CatalogErrorCode}
		if page.JackettCatalog.Known {
			result.Jackett.Sites = &remote
			total += remote
		} else {
			result.Total = nil
		}
		if page.CatalogErrorCode != "" {
			result.Attention++
		}
	}
	return result
}
