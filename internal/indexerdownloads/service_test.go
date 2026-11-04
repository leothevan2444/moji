package indexerdownloads

import (
	"context"
	"errors"
	"fmt"
	"github.com/leothevan2444/moji/internal/indexer"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fetchFunc func(context.Context, string) (indexer.Artifact, error)

func (f fetchFunc) FetchCurrentReference(ctx context.Context, id string) (indexer.Artifact, error) {
	return f(ctx, id)
}
func testTorrent() []byte { return []byte("d4:infod6:lengthi1e4:name12:ABCD-123.mp4ee") }
func fixture(t *testing.T, f fetchFunc) *Service {
	t.Helper()
	s, err := New(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}
func TestHandlesOwnArtifactsExpireAbsolutelyAndRetainActiveReaderBudget(t *testing.T) {
	data := testTorrent()
	s := fixture(t, func(context.Context, string) (indexer.Artifact, error) {
		return indexer.Artifact{TorrentBytes: data}, nil
	})
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	r, err := s.Prepare(context.Background(), "private-ref")
	if err != nil || r.Kind != "TORRENT" || len(r.Handle) != 64 {
		t.Fatal(r, err)
	}
	data[0] = 'x'
	lease, err := s.Open(r.Handle)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(lifetime)
	if _, err = s.Open(r.Handle); !errors.Is(err, indexer.ErrExpired) {
		t.Fatal("expired handle was reopened", err)
	}
	if s.used == 0 || len(s.entries) != 1 {
		t.Fatal("active read lost its retained budget")
	}
	got, err := io.ReadAll(lease.Reader)
	if err != nil || string(got) != string(testTorrent()) || lease.Filename != "resource.torrent" {
		t.Fatal("artifact changed during download", err)
	}
	lease.Release()
	lease.Release()
	if s.used != 0 || s.readers != 0 || len(s.entries) != 0 {
		t.Fatal("release leaked or double released")
	}
}
func TestInvalidAcquisitionDoesNotCreateHandlesOrRetry(t *testing.T) {
	cases := []indexer.Artifact{{TorrentBytes: []byte("<html>login</html>")}, {Magnet: "https://site.example/grab?secret=synthetic"}, {Magnet: "magnet:?dn=missing-identity"}, {Magnet: "magnet:?xt=urn:btih:synthetic\r\nHeader:bad"}, {Magnet: "magnet:?xt=urn:btih:x&tr=%GG"}, {Magnet: "magnet:?xt=urn:btih:x#fragment"}, {TorrentBytes: testTorrent(), Magnet: "magnet:?xt=x"}}
	for _, a := range cases {
		t.Run(a.Magnet, func(t *testing.T) {
			calls := 0
			s := fixture(t, func(context.Context, string) (indexer.Artifact, error) { calls++; return a, nil })
			if _, err := s.Prepare(context.Background(), "ref"); !errors.Is(err, indexer.ErrReference) || calls != 1 || len(s.entries) != 0 || s.preparing != 0 || s.used != 0 {
				t.Fatal("invalid artifact was retained/retried", err)
			}
		})
	}
}
func TestMagnetHandleDoesNotRedirectOrExposeSourceConfiguration(t *testing.T) {
	magnet := "magnet:?xt=urn:btih:synthetic&tr=https%3A%2F%2Ftracker.example%2Fsynthetic-passkey"
	s := fixture(t, func(context.Context, string) (indexer.Artifact, error) {
		return indexer.Artifact{Magnet: magnet, Resource: indexer.ResourceRef{ID: "private-ref"}}, nil
	})
	r, err := s.Prepare(context.Background(), "ref")
	if err != nil || r.Kind != "MAGNET" || strings.Contains(r.Handle, "synthetic") {
		t.Fatal(r, err)
	}
	l, err := s.Open(r.Handle)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	body, _ := io.ReadAll(l.Reader)
	if string(body) != magnet || l.ContentType != "text/uri-list; charset=utf-8" || l.Filename != "resource.magnet" {
		t.Fatal("magnet attachment changed")
	}
	for _, id := range []string{"", "../private", strings.Repeat("z", 64), r.Handle + "?url=https://site.example"} {
		if _, err = s.Open(id); !errors.Is(err, indexer.ErrExpired) {
			t.Fatal("unrecognized locator accepted", err)
		}
	}
}
func TestAdmissionAndShutdownCancelAndJoinBoundedAcquisition(t *testing.T) {
	entered := make(chan struct{}, maxPreparing)
	var calls atomic.Int32
	s := fixture(t, func(ctx context.Context, _ string) (indexer.Artifact, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-ctx.Done()
		return indexer.Artifact{}, ctx.Err()
	})
	var wg sync.WaitGroup
	for i := 0; i < maxPreparing; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Prepare(context.Background(), "ref"); !errors.Is(err, context.Canceled) {
				t.Error(err)
			}
		}()
	}
	for i := 0; i < maxPreparing; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("acquisition never entered")
		}
	}
	if _, err := s.Prepare(context.Background(), "overflow"); !errors.Is(err, indexer.ErrBusy) || calls.Load() != maxPreparing {
		t.Fatal("overflow reached upstream", err)
	}
	s.Close()
	wg.Wait()
	if len(s.entries) != 0 || s.preparing != 0 {
		t.Fatal("shutdown returned before work exited")
	}
	if _, err := s.Prepare(context.Background(), "after-close"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestReaderAndEntryLimitsRejectWithoutDisplacingLiveHandles(t *testing.T) {
	s := fixture(t, func(context.Context, string) (indexer.Artifact, error) {
		return indexer.Artifact{TorrentBytes: testTorrent()}, nil
	})
	first, err := s.Prepare(context.Background(), "ref")
	if err != nil {
		t.Fatal(err)
	}
	var readers []*Lease
	for i := 0; i < maxReaders; i++ {
		l, err := s.Open(first.Handle)
		if err != nil {
			t.Fatal(err)
		}
		readers = append(readers, l)
	}
	if _, err = s.Open(first.Handle); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("unbounded readers", err)
	}
	for _, l := range readers {
		l.Release()
	}
	for i := 1; i < maxEntries; i++ {
		if _, err = s.Prepare(context.Background(), "ref"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.Prepare(context.Background(), "overflow"); !errors.Is(err, indexer.ErrBusy) {
		t.Fatal("entry cap ignored", err)
	}
	l, err := s.Open(first.Handle)
	if err != nil {
		t.Fatal("live handle displaced by later requests", err)
	}
	l.Release()
}

func TestByteBudgetAdmitsBeforeFetchAndReturnsAfterAbsoluteExpiry(t *testing.T) {
	padding := strings.Repeat("p", 7<<20)
	data := []byte(fmt.Sprintf("d7:comment%d:%s4:infod6:lengthi1e4:name12:ABCD-123.mp4ee", len(padding), padding))
	calls := 0
	s := fixture(t, func(context.Context, string) (indexer.Artifact, error) {
		calls++
		return indexer.Artifact{TorrentBytes: data}, nil
	})
	now := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	count := 0
	for {
		_, err := s.Prepare(context.Background(), "ref")
		if errors.Is(err, indexer.ErrBusy) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
		if count > 16 {
			t.Fatal("byte cap ignored")
		}
	}
	if count < 2 || calls != count || s.used > maxBytes {
		t.Fatal("capacity checked after upstream acquisition", count, calls, s.used)
	}
	now = now.Add(lifetime)
	if _, err := s.Prepare(context.Background(), "ref"); err != nil || len(s.entries) != 1 {
		t.Fatal("expired bytes did not return to admission budget", err)
	}
}
func TestCloseJoinsLateSuccessWithoutPublishingHandle(t *testing.T) {
	entered, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	s := fixture(t, func(ctx context.Context, _ string) (indexer.Artifact, error) {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		<-release
		return indexer.Artifact{TorrentBytes: testTorrent()}, nil
	})
	result := make(chan error, 1)
	go func() { _, err := s.Prepare(context.Background(), "ref"); result <- err }()
	<-entered
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	<-cancelled
	select {
	case <-closed:
		t.Fatal("close returned while acquisition was still running")
	default:
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close did not join acquisition")
	}
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal("late success was acknowledged", err)
	}
	if len(s.entries) != 0 || s.used != 0 {
		t.Fatal("late success filled closed handle store")
	}
}
