package stashsync

import (
	"context"
	"errors"
	"strings"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

var ErrStudioConflict = errors.New("Stash could not link an existing studio; resolve the studio association in Stash and retry")

// Identify jobs can finish even when an individual scene update rolled back.
// Logs are supplementary evidence: only accept a known error for an exact file
// within this job's execution interval. Missing/rotated logs prove nothing.
func (s *Service) IdentifyFailure(ctx context.Context, jobID string, scenes []IngestScene) error {
	client, ok := s.client.(interface {
		IngestLogs(context.Context) ([]*stashgraphql.IngestLogs_Logs, error)
	})
	if !ok || jobID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	job, err := s.FindJob(ctx, jobID)
	if err != nil || job == nil || job.StartTime == nil || job.EndTime == nil {
		return nil
	}
	logs, err := client.IngestLogs(ctx)
	if err != nil {
		return nil
	}
	for _, entry := range logs {
		if entry == nil || entry.Level != stashgraphql.LogLevelError || entry.Time.Before(job.StartTime.Add(-time.Second)) || entry.Time.After(job.EndTime.Add(time.Second)) {
			continue
		}
		for _, scene := range scenes {
			for _, path := range scene.Paths {
				prefix := "Error encountered identifying " + path + ": error modifying scene: error getting studio: studio with name '"
				if strings.HasPrefix(entry.Message, prefix) && strings.HasSuffix(entry.Message, "' already exists") {
					return ErrStudioConflict
				}
			}
		}
	}
	return nil
}
