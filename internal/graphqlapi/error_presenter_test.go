package graphqlapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/leothevan2444/moji/internal/taskruntime"
)

func TestErrorPresenterContract(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{fmt.Errorf("create task: %w", taskruntime.ErrDuplicateCodeTask), ErrorDuplicateCodeTask},
		{taskruntime.ErrDuplicateTorrentTask, ErrorDuplicateTorrentTask},
		{taskruntime.ErrDuplicateLibraryCode, ErrorDuplicateLibraryCode},
		{taskruntime.ErrTaskCodeRequired, ErrorTaskCodeRequired},
		{fmt.Errorf("tracker is not configured"), ErrorTrackerNotConfigured},
		{fmt.Errorf("stash client is not configured"), ErrorStashNotConfigured},
		{fmt.Errorf("resolve qB relative path failed: outside root"), ErrorTransferPathFailed},
		{fmt.Errorf("trigger stash scan: unavailable"), ErrorStashScanFailed},
		{fmt.Errorf("no downloadable torrent candidate found"), ErrorNoTorrentCandidate},
		{fmt.Errorf("secret internal detail"), ErrorInternal},
	}
	for _, tt := range tests {
		got := ErrorPresenter(context.Background(), tt.err)
		if got.Message != "request failed" { t.Fatalf("message leaked: %q", got.Message) }
		if got.Extensions["code"] != tt.code { t.Fatalf("code = %v, want %s", got.Extensions["code"], tt.code) }
		if got.Extensions["correlationId"] == "" { t.Fatal("missing correlation id") }
		if _, ok := got.Extensions["params"].(map[string]any); !ok { t.Fatal("missing params object") }
	}
}
