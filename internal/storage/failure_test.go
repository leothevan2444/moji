package storage

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestSQLiteFullRollsBackDurableTransaction(t *testing.T) {
	d := testDatabase(t)
	ctx := context.Background()
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`CREATE TABLE important(id TEXT PRIMARY KEY,value TEXT); INSERT INTO important VALUES('kept','original'); CREATE TABLE large_data(data BLOB)`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		var pages int
		if err := tx.Get(&pages, "PRAGMA page_count"); err != nil {
			return err
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA max_page_count=%d", pages+2)); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE important SET value='uncommitted'"); err != nil {
			return err
		}
		_, err := tx.Exec("INSERT INTO large_data VALUES(zeroblob(1048576))")
		return err
	})
	var sqliteErr interface{ Code() int }
	if !errors.As(err, &sqliteErr) || sqliteErr.Code()&255 != 13 {
		t.Fatalf("expected actual SQLITE_FULL, got %v", err)
	}
	var value string
	if err := d.Reader.Get(&value, "SELECT value FROM important WHERE id='kept'"); err != nil || value != "original" {
		t.Fatal("full disk partially committed business state", value, err)
	}
}

func TestStorageCrashHelper(t *testing.T) {
	root := os.Getenv("MOJI_STORAGE_CRASH_ROOT")
	if root == "" {
		t.Skip("subprocess helper")
	}
	ctx := context.Background()
	m, err := OpenManager(ctx, DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec(`INSERT INTO tasks(id,source,code,stage,stage_status,created_at,updated_at) VALUES('kept','MANUAL','KEEP-123','DOWNLOADING','RUNNING','2026-10-27','2026-10-27');
 INSERT INTO task_downloads(task_id,content_path) VALUES('kept','/downloads/keep.mkv');
 INSERT INTO task_file_plans(task_id,confirmed,keep_subtitles,pause_requested,applied) VALUES('kept',1,0,0,1);
 INSERT INTO task_files(task_id,file_index,position,path,size,kind,selected) VALUES('kept',7,0,'keep.mkv',100,'MAIN',1);
 INSERT INTO external_commands(id,task_id,target_hash,kind,state,payload_json,created_at,updated_at) VALUES('intent','kept','hash','TASK_ADD','SENT','{}','2026-10-27','2026-10-27')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// Exit only through the parent's SIGKILL, with both committed and uncommitted
	// WAL work present and the OS instance lock still held.
	_ = m.Core.WithWrite(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`UPDATE tasks SET code='UNCOMMITTED' WHERE id='kept'; INSERT INTO task_attachments(task_id,torrent_file_data) VALUES('kept',zeroblob(4194304))`); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(os.Stdout, "READY")
		select {}
	})
}
func TestCrashRecoveryPreservesChoicesAndRollsBackIncompleteWrites(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestStorageCrashHelper$", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "MOJI_STORAGE_CRASH_ROOT="+root)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := false
	defer func() {
		if !done {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(pipe).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "READY\n" {
			t.Fatalf("helper failed before crash point: %q %s", line, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("helper did not reach crash point")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	done = true
	m, err := OpenManager(context.Background(), DefaultManagerOptions(root))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	var code, path, integrity string
	var selected, count int
	if err := m.Core.Reader.Get(&code, "SELECT code FROM tasks WHERE id='kept'"); err != nil || code != "KEEP-123" {
		t.Fatal("crash lost committed task or retained incomplete update", code, err)
	}
	if err := m.Core.Reader.Get(&path, "SELECT content_path FROM task_downloads WHERE task_id='kept'"); err != nil || path != "/downloads/keep.mkv" {
		t.Fatal("crash lost ingest input", err)
	}
	if err := m.Core.Reader.Get(&selected, "SELECT selected FROM task_files WHERE task_id='kept' AND file_index=7"); err != nil || selected != 1 {
		t.Fatal("crash lost selected file", err)
	}
	if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM external_commands"); err != nil || count != 1 {
		t.Fatal("crash lost operation intention", err)
	}
	if err := m.Core.Reader.Get(&count, "SELECT count(*) FROM task_attachments"); err != nil || count != 0 {
		t.Fatal("incomplete attachment survived crash", err)
	}
	if err := m.Core.Reader.Get(&integrity, "PRAGMA integrity_check"); err != nil || integrity != "ok" {
		t.Fatal("crash recovery left corrupt data", err)
	}
}
