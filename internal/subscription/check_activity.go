package subscription

import (
	"context"
	"fmt"
	"strings"
)

const maxCheckActivityPerformers = 256

type CheckActivityPerformer struct{ PerformerID, RunID, Status string }
type CheckActivity struct {
	ActiveCount int
	AllRunID    string
	Performers  []CheckActivityPerformer
}

// CheckActivity returns only the reservations needed by the visible and selected actors.
func (s *Service) CheckActivity(_ context.Context, ids []string) (CheckActivity, error) {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}
	if len(wanted) > maxCheckActivityPerformers {
		return CheckActivity{}, fmt.Errorf("subscription check activity supports at most %d performers", maxCheckActivityPerformers)
	}
	t := s.checks
	t.mu.Lock()
	defer t.mu.Unlock()
	out := CheckActivity{Performers: []CheckActivityPerformer{}}
	for _, run := range t.runs {
		if run.Status != "RUNNING" {
			continue
		}
		out.ActiveCount++
		if run.Scope == "ALL" {
			out.AllRunID = run.ID
		}
		for _, item := range run.Items {
			if wanted[item.PerformerID] {
				out.Performers = append(out.Performers, CheckActivityPerformer{PerformerID: item.PerformerID, RunID: run.ID, Status: item.Status})
			}
		}
	}
	return out, nil
}
