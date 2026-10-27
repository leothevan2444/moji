package datamigration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestEventArchiveRoundTripAndTamperDetection(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE task_events(id INTEGER PRIMARY KEY,task_id TEXT,message TEXT,payload BLOB); INSERT INTO task_events VALUES(1,'task','中文 newline',x'000102'),(2,NULL,'second',NULL)`); err != nil {
		t.Fatal(err)
	}
	archive, err := archiveEvents(context.Background(), db, filepath.Join(t.TempDir(), "history.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if archive.Rows != 2 || len(archive.Columns) != 4 {
		t.Fatalf("wrong archive: %+v", archive)
	}
	if err := VerifyArchive(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	archive.Rows++
	if err := VerifyArchive(context.Background(), archive); err == nil {
		t.Fatal("wrong row count accepted")
	}
}
