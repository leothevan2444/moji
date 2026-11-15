package subscription

import (
	"context"
	"time"
)

// CheckDetails records observations from this run, independently of durable
// release decisions. Old runs have no details; absence must not imply zeros.
type CheckDetails struct {
	Version                                                                int
	Phase, CatalogStatus, CatalogErrorCode                                 string
	CatalogSource, CatalogMode, CatalogStopReason                          string
	CatalogPages, CatalogPagesWithResults, CatalogScenes, CatalogReleases  int
	NewCount, KnownCount, UpcomingCount, InLibraryCount                    int
	QueuedCount, ReviewCount, SkippedCount, CreationFailedCount            int
	RetryTotal, RetryAttempted, RetrySucceeded, RetryFailed, RetryDeferred int
	OmittedResults                                                         int
	Results                                                                []CheckReleaseResult
}

type CheckReleaseResult struct {
	Key, Code, Title, Phase, Outcome, ReasonCode, ErrorCode, Error, TaskID string
	NextRetryAt                                                            *time.Time
}

func (d *CheckDetails) Clone() *CheckDetails {
	if d == nil {
		return nil
	}
	v := *d
	v.Results = append([]CheckReleaseResult(nil), d.Results...)
	for i := range v.Results {
		v.Results[i].NextRetryAt = copyTime(v.Results[i].NextRetryAt)
	}
	return &v
}

func (d *CheckDetails) add(record ReleaseReview, phase, outcome string) {
	if len(d.Results) >= 320 {
		d.OmittedResults++
		return
	}
	d.Results = append(d.Results, CheckReleaseResult{Key: record.Key, Code: record.Code, Title: record.Title, Phase: phase, Outcome: outcome, ReasonCode: record.DecisionReason, ErrorCode: record.FailureCode, Error: record.Error, TaskID: record.TaskID, NextRetryAt: copyTime(record.NextRetryAt)})
}

type checkDetailsContextKey struct{}

func detailsForCheck(ctx context.Context) *CheckDetails {
	v, _ := ctx.Value(checkDetailsContextKey{}).(*CheckDetails)
	return v
}

func (s *Service) saveCheckDetails(ctx context.Context, id string, details *CheckDetails) error {
	return s.updateCheck(ctx, func(run *CheckRun) {
		for i := range run.Items {
			if run.Items[i].PerformerID == id {
				run.Items[i].Details = details.Clone()
			}
		}
	})
}
