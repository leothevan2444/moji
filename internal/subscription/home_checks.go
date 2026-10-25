package subscription

import (
	"context"
	"time"
)

const homeCheckRunLimit = 3
const homeCheckPerformerLimit = 3

// HomeCheckRun is a bounded projection; full actor results belong to the run detail.
type HomeCheckRun struct {
	ID, Trigger, Scope, Status                                            string
	StartedAt                                                             time.Time
	FinishedAt                                                            *time.Time
	TotalCount, CompletedCount, SucceededCount, FailedCount, SkippedCount int
	CurrentPerformers                                                     []CheckItem
}

type HomeChecks struct {
	Enabled                bool
	NextRunAt              *time.Time
	ActiveCount            int
	LatestScheduled        *HomeCheckRun
	ActiveRuns, RecentRuns []HomeCheckRun
}

// HomeChecks reads local run state only, without fetching upstream catalogues or full run history.
func (s *Service) HomeChecks(context.Context) (HomeChecks, error) {
	t := s.checks
	t.mu.Lock()
	defer t.mu.Unlock()
	out := HomeChecks{Enabled: t.enabled, NextRunAt: copyTime(t.nextRunAt), ActiveRuns: []HomeCheckRun{}, RecentRuns: []HomeCheckRun{}}
	for _, run := range t.runs {
		if run.Status == "RUNNING" {
			out.ActiveCount++
			if len(out.ActiveRuns) < homeCheckRunLimit {
				out.ActiveRuns = append(out.ActiveRuns, summarizeHomeCheck(run))
			}
		} else if len(out.RecentRuns) < homeCheckRunLimit {
			out.RecentRuns = append(out.RecentRuns, summarizeHomeCheck(run))
		}
		if out.LatestScheduled == nil && run.Trigger == "SCHEDULED" {
			value := summarizeHomeCheck(run)
			out.LatestScheduled = &value
		}
	}
	return out, nil
}

func summarizeHomeCheck(run CheckRun) HomeCheckRun {
	out := HomeCheckRun{ID: run.ID, Trigger: run.Trigger, Scope: run.Scope, Status: run.Status, StartedAt: run.StartedAt, FinishedAt: copyTime(run.FinishedAt), TotalCount: len(run.Items), CurrentPerformers: []CheckItem{}}
	for _, item := range run.Items {
		switch item.Status {
		case "SUCCEEDED":
			out.SucceededCount++
			out.CompletedCount++
		case "SKIPPED":
			out.SkippedCount++
			out.CompletedCount++
		case "FAILED", "INTERRUPTED":
			out.FailedCount++
			out.CompletedCount++
		case "RUNNING":
			if len(out.CurrentPerformers) < homeCheckPerformerLimit {
				out.CurrentPerformers = append(out.CurrentPerformers, CheckItem{PerformerID: item.PerformerID, Name: item.Name, Status: item.Status})
			}
		}
	}
	return out
}
