package stashsync

import (
	"context"
	"errors"
	"testing"
	"time"

	graphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

type diagnosticClient struct {
	fakeIngestClient
	logs   []*graphql.IngestLogs_Logs
	logErr error
}

func (c *diagnosticClient) IngestLogs(context.Context) ([]*graphql.IngestLogs_Logs, error) {
	return c.logs, c.logErr
}

func TestIdentifyFailureRequiresExactSceneAndJobInterval(t *testing.T) {
	now := time.Now().UTC()
	start, end := now.Add(-time.Minute), now
	message := "Error encountered identifying /library/DEMO-123/movie.mp4: error modifying scene: error getting studio: studio with name 'Existing' already exists"
	for _, tc := range []struct {
		name, message string
		time          time.Time
		logErr        error
		want          bool
	}{
		{"conflict", message, start.Add(time.Second), nil, true},
		{"old job", message, start.Add(-time.Minute), nil, false},
		{"later job", message, end.Add(time.Minute), nil, false},
		{"other scene", "Error encountered identifying /library/DEMO-1234/movie.mp4: error modifying scene: error getting studio: studio with name 'Existing' already exists", now, nil, false},
		{"other error", "Error encountered identifying /library/DEMO-123/movie.mp4: network unavailable", now, nil, false},
		{"unavailable", message, now, errors.New("unavailable"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &diagnosticClient{logs: []*graphql.IngestLogs_Logs{{Time: tc.time, Level: graphql.LogLevelError, Message: tc.message}}, logErr: tc.logErr}
			client.job = &graphql.FindJob_FindJob{ID: "job", StartTime: &start, EndTime: &end}
			service, _ := NewService(client, nil)
			err := service.IdentifyFailure(context.Background(), "job", []IngestScene{{Paths: []string{"/library/DEMO-123/movie.mp4"}}})
			if errors.Is(err, ErrStudioConflict) != tc.want {
				t.Fatalf("got %v, want conflict %t", err, tc.want)
			}
		})
	}
}
