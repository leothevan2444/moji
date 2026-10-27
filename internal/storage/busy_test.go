package storage

import (
	"context"
	"database/sql"
	"net/url"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestBusyWriterRespectsRequestDeadlineAndRecovers(t *testing.T) {
	d := testDatabase(t)
	seedDatabase(t, d)
	ctx := context.Background()
	uri := url.URL{Scheme: "file", Path: d.Path}
	other, err := sql.Open("sqlite", uri.String())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	connection, err := other.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer connection.ExecContext(context.Background(), "ROLLBACK")
	request, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	called := false
	err = d.WithWrite(request, func(*sqlx.Tx) error { called = true; return nil })
	if err == nil || called {
		t.Fatal("entered transaction despite external writer lock", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("busy wait ignored request deadline: %s", time.Since(start))
	}
	if _, err := connection.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatal(err)
	}
	if err := d.WithWrite(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.Exec("INSERT INTO parent(id,value) VALUES(123,'after-busy')")
		return err
	}); err != nil {
		t.Fatal("writer did not recover", err)
	}
}
