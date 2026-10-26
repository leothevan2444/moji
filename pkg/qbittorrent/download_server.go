package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// DownloadServerState describes the qB host, rather than the Moji filesystem.
// FreeSpace is nullable because some qB versions/backends omit it.
type DownloadServerState struct {
	FreeSpace        *int64 `json:"free_space_on_disk"`
	DownloadSpeed    int64  `json:"dl_info_speed"`
	UploadSpeed      int64  `json:"up_info_speed"`
	DownloadLimit    int64  `json:"dl_rate_limit"`
	UploadLimit      int64  `json:"up_rate_limit"`
	ConnectionStatus string `json:"connection_status"`
	DHTNodes         int64  `json:"dht_nodes"`
}

func (c *Client) GetDownloadServerState(ctx context.Context) (*DownloadServerState, error) {
	base, httpClient := c.resolve()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v2/sync/maindata?rid=0", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.doAuthenticated(req, httpClient)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qB server state: HTTP %d", resp.StatusCode)
	}
	var data struct {
		ServerState *DownloadServerState `json:"server_state"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.ServerState == nil {
		return nil, fmt.Errorf("qB server state missing")
	}
	return data.ServerState, nil
}
func (c *Client) postDownloadCommand(ctx context.Context, path string, values url.Values) (int, error) {
	base, httpClient := c.resolve()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v2/"+path, strings.NewReader(values.Encode()))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.doAuthenticated(req, httpClient)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("qB command: HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}
func (c *Client) changeRunning(ctx context.Context, hashes []string, start bool) error {
	endpoint, legacy := "stop", "pause"
	if start {
		endpoint, legacy = "start", "resume"
	}
	values := url.Values{"hashes": {strings.Join(hashes, "|")}}
	status, err := c.postDownloadCommand(ctx, "torrents/"+endpoint, values)
	if status == http.StatusNotFound {
		_, err = c.postDownloadCommand(ctx, "torrents/"+legacy, values)
	}
	return err
}
func (c *Client) SetDownloadSpeedLimit(ctx context.Context, upload bool, bytesPerSecond int64) error {
	if bytesPerSecond < 0 {
		return fmt.Errorf("negative speed limit")
	}
	endpoint := "transfer/setDownloadLimit"
	if upload {
		endpoint = "transfer/setUploadLimit"
	}
	_, err := c.postDownloadCommand(ctx, endpoint, url.Values{"limit": {strconv.FormatInt(bytesPerSecond, 10)}})
	return err
}
