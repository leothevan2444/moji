package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TransferPreferences is restricted to qB's queueing and seeding settings.
// Pointers preserve unsupported fields and make updates genuinely sparse.
type TransferPreferences struct {
	QueueingEnabled               *bool    `json:"queueing_enabled,omitempty"`
	MaxActiveDownloads            *int     `json:"max_active_downloads,omitempty"`
	MaxActiveUploads              *int     `json:"max_active_uploads,omitempty"`
	MaxActiveTorrents             *int     `json:"max_active_torrents,omitempty"`
	DontCountSlowTorrents         *bool    `json:"dont_count_slow_torrents,omitempty"`
	SlowTorrentDLRateThreshold    *int     `json:"slow_torrent_dl_rate_threshold,omitempty"`
	SlowTorrentULRateThreshold    *int     `json:"slow_torrent_ul_rate_threshold,omitempty"`
	SlowTorrentInactiveTimer      *int     `json:"slow_torrent_inactive_timer,omitempty"`
	MaxRatioEnabled               *bool    `json:"max_ratio_enabled,omitempty"`
	MaxRatio                      *float64 `json:"max_ratio,omitempty"`
	MaxRatioAction                *int     `json:"max_ratio_act,omitempty"`
	MaxSeedingTimeEnabled         *bool    `json:"max_seeding_time_enabled,omitempty"`
	MaxSeedingTime                *int     `json:"max_seeding_time,omitempty"`
	MaxInactiveSeedingTimeEnabled *bool    `json:"max_inactive_seeding_time_enabled,omitempty"`
	MaxInactiveSeedingTime        *int     `json:"max_inactive_seeding_time,omitempty"`
	ShareLimitMode                *string  `json:"share_limits_mode,omitempty"`
}

type TorrentPreferences struct {
	TransferPreferences
	SavePath string `json:"save_path"`
}

func (c *Client) GetTorrentPreferences(ctx context.Context) (*TorrentPreferences, error) {
	base, httpClient := c.resolve()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v2/app/preferences", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doAuthenticated(req, httpClient)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qB preferences: HTTP %d", resp.StatusCode)
	}
	var out TorrentPreferences
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PatchTransferPreferences(ctx context.Context, patch TransferPreferences) error {
	data, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	_, err = c.postDownloadCommand(ctx, "app/setPreferences", url.Values{"json": {string(data)}})
	return err
}
