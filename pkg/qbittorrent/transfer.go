package qbittorrent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// All Transfer info API methods are under "transfer", e.g.: /api/v2/transfer/methodName.

type GlobalTransferInfo struct {
	DLInfoSpeed       int64  `json:"dl_info_speed"`        // Global download rate (bytes/s)
	DLInfoData        int64  `json:"dl_info_data"`         // Data downloaded this session (bytes)
	UPInfoSpeed       int64  `json:"up_info_speed"`        // Global upload rate (bytes/s)
	UPInfoData        int64  `json:"up_info_data"`         // Data uploaded this session (bytes)
	DLRateLimit       int64  `json:"dl_rate_limit"`        // Download rate limit (bytes/s)
	UPRateLimit       int64  `json:"up_rate_limit"`        // Upload rate limit (bytes/s)
	DHTNodes          int64  `json:"dht_nodes"`            // DHT nodes connected to
	ConnectionStatus  string `json:"connection_status"`    // Connection status. Possible values: "connected", "firewalled", "disconnected"
	Queueing          bool   `json:"queueing"`             // True if torrent queueing is enabled
	UseAltSpeedLimits bool   `json:"use_alt_speed_limits"` // True if alternative speed limits are enabled
	RefreshInterval   int64  `json:"refresh_interval"`     // Transfer list refresh interval (milliseconds)
}

func (c *Client) GetGlobalTransferInfo(ctx context.Context) (*GlobalTransferInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/transfer/info", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info GlobalTransferInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

func (c *Client) GetAlternativeSpeedLimitsState(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/transfer/speedLimitsMode", nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// The response is 1 if alternative speed limits are enabled, 0 otherwise.
	var enabled int
	if err := json.NewDecoder(resp.Body).Decode(&enabled); err != nil {
		return false, err
	}

	return enabled == 1, nil
}

func (c *Client) ToggleAlternativeSpeedLimits(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/transfer/toggleSpeedLimitsMode", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// The response is the value of current global download speed limit in bytes/second; this value will be zero if no limit is applied.
func (c *Client) GetGlobalDownloadLimit(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/transfer/downloadLimit", nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var limit int64
	if err := json.NewDecoder(resp.Body).Decode(&limit); err != nil {
		return 0, err
	}

	return limit, nil
}

// Set the global download speed limit in bytes/second
func (c *Client) SetGlobalDownloadLimit(ctx context.Context, limit int) error {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/transfer/setDownloadLimit",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// The response is the value of current global upload speed limit in bytes/second; this value will be zero if no limit is applied.
func (c *Client) GetGlobalUploadLimit(ctx context.Context) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/transfer/uploadLimit", nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var limit int64
	if err := json.NewDecoder(resp.Body).Decode(&limit); err != nil {
		return 0, err
	}

	return limit, nil
}

// Set the global upload speed limit in bytes/second
func (c *Client) SetGlobalUploadLimit(ctx context.Context, limit int) error {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/transfer/setUploadLimit",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) BanPeers(ctx context.Context, peers []string) error {
	params := url.Values{}
	params.Set("peers", strings.Join(peers, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/transfer/banPeers",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
