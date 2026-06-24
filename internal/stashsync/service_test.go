package stashsync

import (
	"context"
	"errors"
	"testing"
	"time"

	stashgraphql "github.com/leothevan2444/moji/pkg/stash/graphql"
)

func TestNewServiceRequiresClient(t *testing.T) {
	service, err := NewService(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
}

func TestMetadataScanUsesRequestPaths(t *testing.T) {
	client := &fakeClient{metadataScanID: "job-1"}
	service, err := NewService(client, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	rescan := true
	id, err := service.MetadataScan(context.Background(), ScanRequest{
		Paths:  []string{"  /custom-a  ", "", "/custom-b"},
		Rescan: &rescan,
	})
	if err != nil {
		t.Fatalf("metadata scan: %v", err)
	}
	if id != "job-1" {
		t.Fatalf("expected job id %q, got %q", "job-1", id)
	}

	wantPaths := []string{"/custom-a", "/custom-b"}
	if len(client.metadataScanInput.Paths) != len(wantPaths) {
		t.Fatalf("expected %d paths, got %#v", len(wantPaths), client.metadataScanInput.Paths)
	}
	for i := range wantPaths {
		if client.metadataScanInput.Paths[i] != wantPaths[i] {
			t.Fatalf("unexpected paths: %#v", client.metadataScanInput.Paths)
		}
	}
	if client.metadataScanInput.Rescan != &rescan {
		t.Fatal("expected rescan pointer to be forwarded")
	}
}

func TestMetadataScanRequiresAnyPath(t *testing.T) {
	client := &fakeClient{}
	service, err := NewService(client, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.MetadataScan(context.Background(), ScanRequest{Paths: []string{" ", ""}})
	if err == nil {
		t.Fatal("expected error when no scan path is available")
	}
	if got := err.Error(); got != "stashsync: at least one scan path is required" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestFindJobRequiresID(t *testing.T) {
	service, err := NewService(&fakeClient{}, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	job, err := service.FindJob(context.Background(), "  ")
	if err == nil {
		t.Fatal("expected error for empty id")
	}
	if job != nil {
		t.Fatalf("expected nil job, got %#v", job)
	}
}

func TestFindJobReturnsNilWhenClientReturnsNil(t *testing.T) {
	service, err := NewService(&fakeClient{}, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	job, err := service.FindJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if job != nil {
		t.Fatalf("expected nil job, got %#v", job)
	}
}

func TestFindJobMapsClientResponse(t *testing.T) {
	progress := 0.75
	startTime := time.Unix(100, 0).UTC()
	endTime := time.Unix(120, 0).UTC()
	addTime := time.Unix(90, 0).UTC()
	errMsg := "scan failed"

	service, err := NewService(&fakeClient{
		job: &stashgraphql.FindJob_FindJob{
			ID:          "job-123",
			Status:      stashgraphql.JobStatusRunning,
			Description: "metadata scan",
			Progress:    &progress,
			StartTime:   &startTime,
			EndTime:     &endTime,
			AddTime:     addTime,
			Error:       &errMsg,
			SubTasks:    []string{"scan", "sprites"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	job, err := service.FindJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if job == nil {
		t.Fatal("expected job")
	}
	if job.ID != "job-123" || job.Status != "RUNNING" || job.Description != "metadata scan" {
		t.Fatalf("unexpected job mapping: %#v", job)
	}
	if job.Progress != &progress || job.StartTime != &startTime || job.EndTime != &endTime {
		t.Fatalf("expected pointer fields to be preserved: %#v", job)
	}
	if job.AddTime != addTime || job.Error != &errMsg {
		t.Fatalf("unexpected mapped times or error: %#v", job)
	}
	if len(job.SubTasks) != 2 || job.SubTasks[1] != "sprites" {
		t.Fatalf("unexpected subtasks: %#v", job.SubTasks)
	}
}

func TestFindJobPropagatesClientError(t *testing.T) {
	service, err := NewService(&fakeClient{findJobErr: errors.New("boom")}, nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.FindJob(context.Background(), "job-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "boom" {
		t.Fatalf("unexpected error: %q", got)
	}
}

type fakeClient struct {
	metadataScanID    string
	metadataScanInput stashgraphql.ScanMetadataInput
	job               *stashgraphql.FindJob_FindJob
	findJobErr        error
}

func (f *fakeClient) MetadataScan(_ context.Context, input stashgraphql.ScanMetadataInput) (string, error) {
	f.metadataScanInput = input
	return f.metadataScanID, nil
}

func (f *fakeClient) FindJob(_ context.Context, _ string) (*stashgraphql.FindJob_FindJob, error) {
	if f.findJobErr != nil {
		return nil, f.findJobErr
	}
	return f.job, nil
}
