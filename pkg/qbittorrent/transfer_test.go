package qbittorrent

import (
	"context"
	"testing"
)

func TestGetGlobalTransferInfo(t *testing.T) {
	info, err := client.GetGlobalTransferInfo(context.Background())
	if err != nil {
		t.Fatalf("GetGlobalTransferInfo failed: %v", err)
	}

	t.Logf("Global Transfer Info: %+v", info)
}

func TestGetAlternativeSpeedLimitsState(t *testing.T) {
	state, err := client.GetAlternativeSpeedLimitsState(context.Background())
	if err != nil {
		t.Fatalf("GetAlternativeSpeedLimitsState failed: %v", err)
	}

	t.Logf("Alternative Speed Limits State: %v", state)
}

func TestToggleAlternativeSpeedLimits(t *testing.T) {
	err := client.ToggleAlternativeSpeedLimits(context.Background())
	if err != nil {
		t.Fatalf("ToggleAlternativeSpeedLimits failed: %v", err)
	}

	t.Log("Toggled Alternative Speed Limits successfully")
}

func TestGetGlobalDownloadLimit(t *testing.T) {
	limit, err := client.GetGlobalDownloadLimit(context.Background())
	if err != nil {
		t.Fatalf("GetGlobalDownloadLimit failed: %v", err)
	}

	t.Logf("Global Download Limit: %d", limit)
}

func TestSetGlobalDownloadLimit(t *testing.T) {
	limit := 10240
	err := client.SetGlobalDownloadLimit(context.Background(), limit)
	if err != nil {
		t.Fatalf("SetGlobalDownloadLimit failed: %v", err)
	}

	t.Logf("Set Global Download Limit to %d successfully", limit)
}

func TestGetGlobalUploadLimit(t *testing.T) {
	limit, err := client.GetGlobalUploadLimit(context.Background())
	if err != nil {
		t.Fatalf("GetGlobalUploadLimit failed: %v", err)
	}

	t.Logf("Global Upload Limit: %d", limit)
}

func TestSetGlobalUploadLimit(t *testing.T) {
	limit := 5120
	err := client.SetGlobalUploadLimit(context.Background(), limit)
	if err != nil {
		t.Fatalf("SetGlobalUploadLimit failed: %v", err)
	}

	t.Logf("Set Global Upload Limit to %d successfully", limit)
}
