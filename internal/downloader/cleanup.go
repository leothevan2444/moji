package downloader

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

const (
	cleanupSettleTime  = 6 * time.Second
	cleanupConfirmTime = 15 * time.Second
	cleanupRetryTime   = 30 * time.Second
	cleanupNoGainLimit = 3
)

var ErrCleanupPersistence = errors.New("cleanup state could not be saved")

type CleanupStatus struct {
	Enabled   bool
	Phase     string
	Reason    string
	Hash      string
	Name      string
	UpdatedAt string
}

// Control decisions survive a process restart and loss of runtime.db.
// Observed free space is evidence, not a measurement of bytes released by us.
type cleanupControl struct {
	Step      string
	Reason    string
	Mode      string
	Hash      string
	Name      string
	At        time.Time
	Until     time.Time
	Before    *int64
	NoGain    int
	PauseAt   *int64
	LastAt    string
	Uncertain bool
}

func (s *Service) saveCleanup(ctx context.Context, c cleanupControl) error {
	if err := s.write(ctx, "cleanup", c); err != nil {
		return errors.Join(ErrCleanupPersistence, err)
	}
	return nil
}
func cleanupView(p Policy, c cleanupControl) CleanupStatus {
	v := CleanupStatus{Enabled: p.Enabled, Phase: "IDLE", Reason: c.Reason, Hash: c.Hash, Name: c.Name, UpdatedAt: c.LastAt}
	switch c.Step {
	case "PREPARING", "CONFIRMING", "SETTLING":
		v.Phase = "RUNNING"
	case "PAUSED":
		v.Phase = "WARNING"
	case "FAILED":
		v.Phase = "ERROR"
	}
	if c.Step == "CONFIRMING" && c.Uncertain {
		v.Phase = "WARNING"
	}
	return v
}
func markCleanup(v *Snapshot, phase, reason string) {
	v.Cleanup.Phase = phase
	v.Cleanup.Reason = reason
}
func belowThreshold(v Snapshot, p Policy) bool {
	return v.FreeSpace != nil && float64(*v.FreeSpace) <= p.MinFreeGB*1e9
}
func resetOperation(c *cleanupControl) {
	c.Uncertain = false
	c.Step = ""
	c.Reason = ""
	c.Hash = ""
	c.Name = ""
	c.Before = nil
	c.Until = time.Time{}
	c.At = time.Time{}
}
func findRaw(all []qbittorrent.Torrent, hash string) (qbittorrent.Torrent, bool) {
	for _, t := range all {
		if strings.EqualFold(t.Hash, hash) {
			return t, true
		}
	}
	return qbittorrent.Torrent{}, false
}
func confirmRemoved(v *Snapshot, hash, now string) {
	for i := range v.Torrents {
		t := &v.Torrents[i]
		if t.Hash == hash {
			t.Present = false
			t.State = "removed"
			if !strings.HasPrefix(t.RemovalReason, "MANUAL") {
				t.RemovalReason = "POLICY"
			} else {
				t.RemovalReason = "MANUAL"
			}
			t.RemovedAt = now
			t.DownloadSpeed = 0
			t.UploadSpeed = 0
			t.CleanupError = false
		}
	}
}

func (s *Service) advanceCleanup(ctx context.Context, v *Snapshot, p Policy, execute bool, all []qbittorrent.Torrent, stateUnavailable bool, isolation *contentIsolation) error {
	c := cleanupControl{}
	if err := s.read(ctx, "cleanup", &c); err != nil {
		v.Cleanup = CleanupStatus{Enabled: p.Enabled, Phase: "WARNING", Reason: "STATE_UNAVAILABLE"}
		return err
	}
	now := s.now().UTC()
	v.Cleanup = cleanupView(p, c)
	persist := func() error {
		if err := s.saveCleanup(ctx, c); err != nil {
			markCleanup(v, "ERROR", "STATE_SAVE_FAILED")
			return err
		}
		v.Cleanup = cleanupView(p, c)
		return nil
	}
	if c.Step == "" && s.manager != nil {
		var pending struct {
			Hash string `db:"target_hash"`
			At   string `db:"created_at"`
		}
		err := s.manager.Core.Reader.GetContext(ctx, &pending, `SELECT target_hash,created_at FROM external_commands WHERE kind='REMOVE' AND state IN ('PENDING','SENT','UNCERTAIN') AND json_extract(payload_json,'$.reason')='POLICY' ORDER BY created_at,id LIMIT 1`)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			markCleanup(v, "WARNING", "STATE_UNAVAILABLE")
			return err
		}
		if err == nil {
			c.Step = "CONFIRMING"
			c.Reason = "CONFIRMING"
			c.Hash = pending.Hash
			c.At, _ = time.Parse(time.RFC3339Nano, pending.At)
			if c.At.IsZero() {
				c.At = now
			}
			c.LastAt = pending.At
			c.Mode = "AFTER_INGEST"
			if p.CapacityEnabled {
				c.Mode = "CAPACITY"
			}
			if raw, ok := findRaw(all, c.Hash); ok {
				c.Name = raw.Name
			}
			if err := persist(); err != nil {
				return err
			}
		}
	}

	if c.Step == "PREPARING" {
		for i := range v.Torrents {
			if v.Torrents[i].Hash == c.Hash && v.Torrents[i].RemovalReason == "POLICY_PENDING" {
				v.Torrents[i].RemovalReason = ""
			}
		}
		c.Step = "FAILED"
		c.Reason = "STATE_SAVE_FAILED"
		c.Until = s.now().UTC().Add(cleanupRetryTime)
		if err := persist(); err != nil {
			return err
		}
	}
	// A pending delete always remains observable, even after the policy is disabled.
	if c.Step == "CONFIRMING" {
		if _, present := findRaw(all, c.Hash); present {
			if now.Sub(c.At) >= cleanupConfirmTime {
				markCleanup(v, "WARNING", "CONFIRMING")
			}
			return nil
		}
		confirmRemoved(v, c.Hash, now.Format(time.RFC3339))
		c.LastAt = now.Format(time.RFC3339)
		if c.Mode == "CAPACITY" {
			c.Step = "SETTLING"
			c.Reason = "CONFIRMING"
			c.Until = s.now().UTC().Add(cleanupSettleTime)
		} else {
			resetOperation(&c)
		}
		if err := persist(); err != nil {
			return err
		}
	}
	if !p.Enabled {
		c = cleanupControl{LastAt: c.LastAt}
		if err := persist(); err != nil {
			return err
		}
		return nil
	}
	if stateUnavailable {
		markCleanup(v, "WARNING", "STATE_UNAVAILABLE")
		return nil
	}
	if c.Step == "FAILED" && now.Before(c.Until) {
		return nil
	}
	if c.Step == "FAILED" {
		resetOperation(&c)
		if err := persist(); err != nil {
			return err
		}
	}
	if c.Step == "SETTLING" && !p.CapacityEnabled {
		c.NoGain = 0
		resetOperation(&c)
		if err := persist(); err != nil {
			return err
		}
	}
	if c.Step == "SETTLING" {
		if v.FreeSpace == nil {
			markCleanup(v, "WARNING", "SPACE_UNKNOWN")
			return nil
		}
		if !belowThreshold(*v, p) {
			c.NoGain = 0
			resetOperation(&c)
			if err := persist(); err != nil {
				return err
			}
		} else {
			if now.Before(c.Until) {
				return nil
			}
			if c.Before != nil && *v.FreeSpace <= *c.Before {
				c.NoGain++
			} else {
				c.NoGain = 0
			}
			resetOperation(&c)
			if c.NoGain >= cleanupNoGainLimit {
				c.Step = "PAUSED"
				c.Reason = "NO_SPACE_GAIN"
				n := *v.FreeSpace
				c.PauseAt = &n
			}
			if err := persist(); err != nil {
				return err
			}
		}
	}
	if c.Step == "PAUSED" {
		if !p.CapacityEnabled || (v.FreeSpace != nil && (c.PauseAt == nil || *v.FreeSpace > *c.PauseAt || !belowThreshold(*v, p))) {
			c.NoGain = 0
			c.PauseAt = nil
			resetOperation(&c)
			if err := persist(); err != nil {
				return err
			}
		} else {
			return nil
		}
	}
	// Idle and in-flight are not inferred from the browser's request state.
	v.Cleanup = cleanupView(p, c)
	if !p.CapacityEnabled && c.NoGain != 0 {
		c.NoGain = 0
		c.PauseAt = nil
		if err := persist(); err != nil {
			return err
		}
	}
	root := ""
	if p.CapacityEnabled {
		if v.FreeSpace == nil {
			markCleanup(v, "WARNING", "SPACE_UNKNOWN")
			return nil
		}
		if !belowThreshold(*v, p) {
			if c.NoGain != 0 {
				c.NoGain = 0
				c.PauseAt = nil
				if err := persist(); err != nil {
					return err
				}
			}
			return nil
		}
		prefs, err := s.TorrentPreferences(ctx)
		if err != nil || prefs == nil || prefs.SavePath == "" {
			markCleanup(v, "WARNING", "PATH_UNKNOWN")
			return nil
		}
		root = prefs.SavePath
	}
	candidates := append([]qbittorrent.Torrent(nil), all...)
	if p.CapacityEnabled {
		sort.SliceStable(candidates, func(i, j int) bool {
			a, b := candidates[i], candidates[j]
			switch p.CapacityOrder {
			case "RATIO":
				if a.Ratio != b.Ratio {
					return a.Ratio > b.Ratio
				}
			case "SEED_TIME":
				if a.SeedingTime != b.SeedingTime {
					return a.SeedingTime > b.SeedingTime
				}
			default:
				if a.Completed != b.Completed {
					return a.Completed > b.Completed
				}
			}
			return strings.ToLower(a.Hash) < strings.ToLower(b.Hash)
		})
	}
	observed := map[string]*Torrent{}
	for i := range v.Torrents {
		observed[v.Torrents[i].Hash] = &v.Torrents[i]
	}
	reason := ""

	for _, raw := range candidates {
		t := observed[strings.ToLower(raw.Hash)]
		if t == nil || !t.Present || !t.AllCompleted {
			continue
		}
		if !removableState(t.State) && !(p.AfterIngest && raw.State == qbittorrent.TorrentStateMissingFiles) {
			continue
		}
		if p.CapacityEnabled && (raw.Progress < 1 || raw.SavePath == "" || filepath.Clean(raw.SavePath) != filepath.Clean(root)) {
			continue
		}
		if p.DeleteFiles && !t.CanDeleteFiles {
			if reason == "" {
				if r := isolation.check(raw); r != contentIsolated {
					reason = string(r)
				} else {
					reason = "LIBRARY_COPY_UNVERIFIED"
				}
			}
			continue
		}
		if !execute {
			markCleanup(v, "IDLE", "")
			return nil
		}
		groups, err := s.taskGroups(ctx)
		verified := Torrent{}
		attach(&verified, groups[t.Hash])
		if err != nil || !verified.AllCompleted {
			markCleanup(v, "WARNING", "STATE_UNAVAILABLE")
			return nil
		}
		if p.DeleteFiles && !verified.CanDeleteFiles {
			reason = "LIBRARY_COPY_UNVERIFIED"
			continue
		}
		if p.DeleteFiles {
			if r := s.recheckFileRemoval(ctx, raw, p.AfterIngest); r != contentIsolated {
				t.CanDeleteFiles = false
				reason = string(r)
				if r != contentShared {
					break
				}
				continue
			}
		}
		c.Step = "PREPARING"
		c.Reason = "REMOVING"
		c.Hash = t.Hash
		c.Name = t.Name
		c.At = s.now().UTC()
		c.LastAt = c.At.Format(time.RFC3339)
		c.Mode = "AFTER_INGEST"
		c.Before = nil
		if p.CapacityEnabled {
			c.Mode = "CAPACITY"
			n := *v.FreeSpace
			c.Before = &n
		}
		if err := persist(); err != nil {
			return err
		}
		abortPreparation := func(cause error) error {
			t.RemovalReason = ""
			c.Step = "FAILED"
			c.Reason = "STATE_SAVE_FAILED"
			c.Until = s.now().UTC().Add(cleanupRetryTime)
			saveErr := persist()
			markCleanup(v, "ERROR", "STATE_SAVE_FAILED")
			return errors.Join(cause, saveErr)
		}
		t.RemovalReason = "POLICY_PENDING"
		t.CleanupError = false
		// Publish the active operation before the blocking qB request.
		if err := s.write(ctx, "snapshot", *v); err != nil {
			return abortPreparation(err)
		}
		err = s.removeTorrent(ctx, t.Hash, p.DeleteFiles, "POLICY")
		c.Step = "CONFIRMING"
		c.Reason = "CONFIRMING"
		if err != nil {
			t.CleanupError = true
			var rejected *qbittorrent.CommandRejectedError
			if errors.Is(err, ErrCleanupPersistence) {
				return abortPreparation(err)
			}
			if errors.As(err, &rejected) {
				t.RemovalReason = ""
				c.Step = "FAILED"
				c.Reason = "DELETE_REJECTED"
				c.Until = s.now().UTC().Add(cleanupRetryTime)
				if e := persist(); e != nil {
					return e
				}
				return nil
			}
			// Timeout/5xx can follow successful execution. Keep this hash reserved.
			c.Uncertain = true
			if e := persist(); e != nil {
				return e
			}
			return nil
		}
		observedRaw, observeErr := s.client.GetTorrentList(ctx, &qbittorrent.TorrentListOptions{Hashes: []string{t.Hash}})
		if observeErr != nil {
			return nil
		}
		if _, present := findRaw(observedRaw, t.Hash); present {
			return nil
		}
		confirmRemoved(v, t.Hash, s.now().UTC().Format(time.RFC3339))
		if p.CapacityEnabled {
			c.Step = "SETTLING"
			c.Until = s.now().UTC().Add(cleanupSettleTime)
			state, e := s.client.GetDownloadServerState(ctx)
			if e == nil && state != nil && state.FreeSpace != nil && *state.FreeSpace >= 0 {
				v.FreeSpace = state.FreeSpace
			}
			if v.FreeSpace != nil && !belowThreshold(*v, p) {
				c.NoGain = 0
				resetOperation(&c)
			}
			return persist()
		}
		resetOperation(&c)
		if err := persist(); err != nil {
			return err
		}
	}
	if reason != "" {
		markCleanup(v, "WARNING", reason)
	} else if p.CapacityEnabled {
		markCleanup(v, "WARNING", "NO_CANDIDATES")
	}
	return nil
}
