// https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-5.0)

package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create cookie jar: %v", err))
	}
	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Jar:       jar,
			Transport: &http.Transport{Proxy: nil},
		},
	}

	return client
}

// All Authentication API methods are under "auth", e.g.: /api/v2/auth/methodName.
// qBittorrent uses cookie-based authentication.

func (c *Client) Login(ctx context.Context, username, password string) error {
	params := url.Values{}
	params.Set("username", username)
	params.Set("password", password)

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/auth/login",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %s", resp.Status)
	}

	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/auth/logout",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create logout request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logout request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed with status: %s", resp.Status)
	}

	return nil
}

// All Log API methods are under "log", e.g.: /api/v2/log/methodName.

type LogType int

const (
	LogTypeNormal   LogType = 1 << 0                                                         // 1 - Normal messages
	LogTypeInfo     LogType = 1 << 1                                                         // 2 - Info messages
	LogTypeWarning  LogType = 1 << 2                                                         // 4 - Warning messages
	LogTypeCritical LogType = 1 << 3                                                         // 8 - Critical messages
	LogTypeAll      LogType = LogTypeNormal | LogTypeInfo | LogTypeWarning | LogTypeCritical // All levels
)

type LastKnownID int // Messages with "message id" less than or equal to this value will be ignored.

type LogEntry struct {
	ID        int     `json:"id"`        // ID of the message
	Type      LogType `json:"type"`      // Text of the message
	Message   string  `json:"message"`   // Type of the message: Log::NORMAL: 1, Log::INFO: 2, Log::WARNING: 4, Log::CRITICAL: 8
	TimeStamp int     `json:"timestamp"` // Seconds since epoch (Note: switched from milliseconds to seconds in v4.5.0)
}

func (c *Client) GetLog(ctx context.Context, types LogType, lastKnownID *LastKnownID) ([]LogEntry, error) {
	params := url.Values{}
	if types&LogTypeNormal != 0 {
		params.Set("normal", "true")
	}
	if types&LogTypeInfo != 0 {
		params.Set("info", "true")
	}
	if types&LogTypeWarning != 0 {
		params.Set("warning", "true")
	}
	if types&LogTypeCritical != 0 {
		params.Set("critical", "true")
	}
	if lastKnownID != nil {
		params.Set("last_known_id", strconv.Itoa(int(*lastKnownID)))
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/log/main?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get log failed with status: %s", resp.Status)
	}

	var logEntries []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logEntries); err != nil {
		return nil, err
	}

	return logEntries, nil
}

type PeerLogEntry struct {
	ID        int    `json:"id"`        // ID of the peer
	IP        string `json:"ip"`        // IP of the peer
	Timestamp int64  `json:"timestamp"` // Seconds since epoch
	Blocked   bool   `json:"blocked"`   // Whether or not the peer was blocked
	Reason    string `json:"reason"`    // Reason of the block
}

func (c *Client) GetPeerLog(ctx context.Context, lastKnownID *LastKnownID) ([]PeerLogEntry, error) {
	params := url.Values{}
	if lastKnownID != nil {
		params.Set("last_known_id", strconv.Itoa(int(*lastKnownID)))
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/log/peers?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get peer log failed with status: %s", resp.Status)
	}

	var peerLogEntries []PeerLogEntry
	if err := json.NewDecoder(resp.Body).Decode(&peerLogEntries); err != nil {
		return nil, err
	}

	return peerLogEntries, nil
}

// Sync API implements requests for obtaining changes since the last request. All Sync API methods are under "sync", e.g.: /api/v2/sync/methodName.

type SyncRID int // Response ID. If the given rid is different from the one of last server reply, full_update will be true (see the server reply details for more info)

type MainData struct {
	RID               SyncRID              `json:"rid"`                // Response ID
	FullUpdate        bool                 `json:"full_update"`        // Whether the response contains all the data or partial data
	Torrents          map[string]*Torrent  `json:"torrents"`           // Property: torrent hash, value: same as torrent list
	TorrentsRemoved   []string             `json:"torrents_removed"`   // List of hashes of torrents removed since last request
	Categories        map[string]*Category `json:"categories"`         // Info for categories added since last request
	CategoriesRemoved []string             `json:"categories_removed"` // List of categories removed since last request
	Tags              []string             `json:"tags"`               // List of tags added since last request
	TagsRemoved       []string             `json:"tags_removed"`       // List of tags removed since last request
	ServerState       *GlobalTransferInfo  `json:"server_state"`       // Global transfer info
}

func (c *Client) GetMainData(ctx context.Context, rid *SyncRID) (*MainData, error) {
	params := url.Values{}
	if rid != nil {
		params.Set("rid", strconv.Itoa(int(*rid)))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/sync/maindata?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get main data failed with status: %s", resp.Status)
	}

	var mainData MainData
	if err := json.NewDecoder(resp.Body).Decode(&mainData); err != nil {
		return nil, err
	}

	return &mainData, nil
}
