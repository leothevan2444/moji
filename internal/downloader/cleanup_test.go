package downloader

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func TestCapacityNoGainBudgetPausesAfterThreeConfirmedRemovals(t *testing.T) {
	s, q, _ := capacityFixture(t)
	q.released = 0
	clock := time.Now()
	s.now = func() time.Time { return clock }
	for i := 0; i < cleanupNoGainLimit; i++ {
		if err := s.Cycle(context.Background()); err != nil {
			t.Fatal(err)
		}
		if q.deleted != i+1 {
			t.Fatal("wrong removal count", i, q.deleted)
		}
		snap, _ := s.Snapshot(context.Background())
		if snap.Cleanup.Phase != "RUNNING" {
			t.Fatal(snap.Cleanup)
		}
		clock = clock.Add(cleanupSettleTime)
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if q.deleted != 3 || snap.Cleanup.Phase != "WARNING" || snap.Cleanup.Reason != "NO_SPACE_GAIN" {
		t.Fatal(q.deleted, snap.Cleanup)
	}
	clock = clock.Add(time.Hour)
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ = s.Snapshot(context.Background())
	if snap.Cleanup.Phase != "WARNING" {
		t.Fatal("unbounded busy state", snap.Cleanup)
	}
	*q.space = 41e9
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	var c cleanupControl
	if err := s.read(context.Background(), "cleanup", &c); err != nil || c.NoGain != 0 || c.Step != "" {
		t.Fatal(c, err)
	}
}
func TestPendingRemovalStopsAnimatingAndNeverDeletesAnotherHash(t *testing.T) {
	s, q, _ := capacityFixture(t)
	clock := time.Now()
	s.now = func() time.Time { return clock }
	// qB accepts the command but continues listing the torrent.
	wrapper := &acceptedPendingClient{capacityClient: q}
	s.client = wrapper
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if snap.Cleanup.Phase != "RUNNING" {
		t.Fatal(snap.Cleanup)
	}
	clock = clock.Add(cleanupConfirmTime)
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ = s.Snapshot(context.Background())
	if wrapper.calls != 1 || snap.Cleanup.Phase != "WARNING" || snap.Cleanup.Hash != strings.Repeat("a", 40) {
		t.Fatal(wrapper.calls, snap.Cleanup)
	}
}

type acceptedPendingClient struct {
	*capacityClient
	calls int
}

func (c *acceptedPendingClient) DeleteTorrents(context.Context, []string, bool) error {
	c.calls++
	return nil
}

func TestDefiniteRejectionIsErrorButTransportFailureIsUncertain(t *testing.T) {
	for _, tc := range []struct {
		name          string
		err           error
		phase, reason string
	}{{"rejected", &qbittorrent.CommandRejectedError{StatusCode: 403}, "ERROR", "DELETE_REJECTED"}, {"timeout", errors.New("response lost"), "WARNING", "CONFIRMING"}} {
		t.Run(tc.name, func(t *testing.T) {
			s, q, _ := capacityFixture(t)
			q.errorDelete = tc.err
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			snap, _ := s.Snapshot(context.Background())
			if snap.Cleanup.Phase != tc.phase || snap.Cleanup.Reason != tc.reason {
				t.Fatal(snap.Cleanup)
			}
			if err := s.Cycle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if q.deleted != 1 {
				t.Fatal("request replayed without observation/cooldown", q.deleted)
			}
		})
	}
}
func TestImmediateCleanupPublishesSameActiveStateBeforeRequest(t *testing.T) {
	s, q, _ := capacityFixture(t)
	yes, no := true, false
	if _, err := s.PatchPolicy(context.Background(), nil, &yes, nil, &no, nil, nil); err != nil {
		t.Fatal(err)
	}
	observed := &inspectCleanupClient{capacityClient: q, inspect: func() {
		snapshot, _ := s.Snapshot(context.Background())
		if !snapshot.Cleanup.Enabled || snapshot.Cleanup.Phase != "RUNNING" || snapshot.Cleanup.Hash == "" {
			t.Fatal("active operation was not published", snapshot.Cleanup)
		}
	}}
	s.client = observed
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := s.Snapshot(context.Background())
	if q.deleted != 3 || snapshot.Cleanup.Phase != "IDLE" || snapshot.Cleanup.UpdatedAt == "" {
		t.Fatal(q.deleted, snapshot.Cleanup)
	}
}

type inspectCleanupClient struct {
	*capacityClient
	inspect func()
}

func (c *inspectCleanupClient) DeleteTorrents(ctx context.Context, hashes []string, files bool) error {
	c.inspect()
	return c.capacityClient.DeleteTorrents(ctx, hashes, files)
}
func TestDisabledCleanupDoesNotStartNewDeletesWhilePending(t *testing.T) {
	s, q, _ := capacityFixture(t)
	pending := &acceptedPendingClient{capacityClient: q}
	s.client = pending
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	disabled := false
	if _, err := s.PatchPolicy(context.Background(), &disabled, nil, nil, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if pending.calls != 1 {
		t.Fatal(pending.calls)
	}
	pending.torrents = pending.torrents[1:]
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	snap, _ := s.Snapshot(context.Background())
	if snap.Cleanup.Enabled || snap.Cleanup.Phase != "IDLE" {
		t.Fatal(snap.Cleanup)
	}
}

func TestRecoveredCapacityStartsANewNoGainBudget(t *testing.T) {
	s, q, _ := capacityFixture(t)
	*q.space = 60e9
	if err := s.write(context.Background(), "cleanup", cleanupControl{NoGain: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.Cycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	var control cleanupControl
	if err := s.read(context.Background(), "cleanup", &control); err != nil || control.NoGain != 0 || q.deleted != 0 {
		t.Fatal(control, q.deleted, err)
	}
}
