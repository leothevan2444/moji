package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leothevan2444/moji/internal/storage"
)

func TestHTTPRuntimeProcessHelper(t *testing.T) {
	root := os.Getenv("MOJI_HTTP_LIFECYCLE_ROOT")
	if root == "" {
		t.Skip("subprocess helper")
	}
	flag.CommandLine = flag.NewFlagSet("moji-lifecycle", flag.ExitOnError)
	os.Args = []string{"moji", "-config", filepath.Join(root, "config.yaml"), "-addr", "127.0.0.1:0"}
	main()
}
func TestHTTPRuntimeStopsWithOpenWebsocketAndReleasesStorage(t *testing.T) {
	t.Run("jackett", func(t *testing.T) {
		assertHTTPRuntimeLifecycle(t, []byte("connection:\n  jackett:\n    url: http://127.0.0.1:9\n    api_key: local-test\nautomation:\n  task_processing_enabled: false\n  subscription_poll_interval_hours: -1\nsystem:\n  developer_mode: true\n"))
	})
	t.Run("no-indexers", func(t *testing.T) {
		assertHTTPRuntimeLifecycle(t, []byte("config_version: 3\nsystem: {developer_mode: true}\n"))
	})
}
func assertHTTPRuntimeLifecycle(t *testing.T, raw []byte) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "config.yaml"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHTTPRuntimeProcessHelper$", "-test.timeout=30s")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "MOJI_HTTP_LIFECYCLE_ROOT="+root)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	terminal := false
	defer func() {
		if !terminal {
			cmd.Process.Kill()
			<-exited
		}
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	address := ""
	listenPattern := regexp.MustCompile(`moji listening on (127\.0\.0\.1:\d+)`)
	for address == "" {
		select {
		case err := <-exited:
			terminal = true
			t.Fatalf("runtime exited before startup: %v %s", err, output.String())
		case <-deadline.C:
			t.Fatal("runtime did not start")
		case <-tick.C:
			if data, err := os.ReadFile(filepath.Join(root, "logs", "moji.log")); err == nil {
				if match := listenPattern.FindSubmatch(data); len(match) > 1 {
					address = string(match[1])
				}
			}
		}
	}
	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}, HandshakeTimeout: 3 * time.Second, NetDialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext}
	connection, _, err := dialer.Dial("ws://"+address+"/graphql", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(map[string]any{"type": "connection_init"}); err != nil {
		t.Fatal(err)
	}
	connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ack struct{ Type string }
	if err := connection.ReadJSON(&ack); err != nil || ack.Type != "connection_ack" {
		t.Fatal("websocket initialization failed", err)
	}
	// Keep a real performance subscription open during SIGTERM. Its owner must
	// complete it before waiting for HTTP/WebSocket requests and closing SQLite.
	if err := connection.WriteJSON(map[string]any{"id": "performance", "type": "subscribe", "payload": map[string]any{"query": "subscription {databasePerformance{startedAt}}"}}); err != nil {
		t.Fatal(err)
	}
	var event struct{ Type string }
	if err := connection.ReadJSON(&event); err != nil || event.Type != "next" {
		t.Fatal("performance subscription did not start", err, event)
	}
	if err := connection.WriteJSON(map[string]any{"id": "definitions", "type": "subscribe", "payload": map[string]any{"query": "subscription {indexerDefinitionEvents{activeRevision errorCode}}"}}); err != nil {
		t.Fatal(err)
	}
	if err := connection.ReadJSON(&event); err != nil || event.Type != "next" {
		t.Fatal("definitions subscription did not start", err, event)
	}
	if err := connection.WriteJSON(map[string]any{"id": "indexers", "type": "subscribe", "payload": map[string]any{"query": "subscription {indexerEvents{streamId sequence kind}}"}}); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for {
		var notification struct {
			ID, Type string
			Payload  struct {
				Data struct {
					IndexerEvents *struct {
						StreamID string `json:"streamId"`
						Kind     string
					} `json:"indexerEvents"`
				}
				Errors []struct{ Message string }
			}
		}
		if err := connection.ReadJSON(&notification); err != nil {
			t.Fatal("indexer subscription did not start", err)
		}
		if notification.ID == "indexers" {
			if notification.Type != "next" || len(notification.Payload.Errors) > 0 || notification.Payload.Data.IndexerEvents == nil || notification.Payload.Data.IndexerEvents.StreamID == "" {
				t.Fatal("indexer subscription failed", notification)
			}
			break
		}
	}
	if duplicate, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(filepath.Join(root, storage.DefaultDirectory))); !errors.Is(err, storage.ErrDirectoryInUse) {
		if duplicate != nil {
			duplicate.Close()
		}
		t.Fatal("running process did not own directory lock", err)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-exited:
		terminal = true
		if err != nil {
			t.Fatalf("graceful shutdown failed: %v %s", err, output.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("open websocket prevented database shutdown")
	}
	reopened, err := storage.OpenManager(context.Background(), storage.DefaultManagerOptions(filepath.Join(root, storage.DefaultDirectory)))
	if err != nil {
		t.Fatal("storage was not cleanly released", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}
