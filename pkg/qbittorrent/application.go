package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// All Application API methods are under "app", e.g.: /api/v2/app/methodName.

func (c *Client) GetApplicationVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/version", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get application version: %s, body: %s", resp.Status, string(body))
	}

	return string(body), nil
}

func (c *Client) GetAPIVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/webapiVersion", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get API version: %s, body: %s", resp.Status, string(body))
	}

	return string(body), nil
}

type ApplicationBuildInfo struct {
	Qt         string `json:"qt"`
	LibTorrent string `json:"libtorrent"`
	Boost      string `json:"boost"`
	OpenSSL    string `json:"openssl"`
	Bitness    int    `json:"bitness"`
}

func (c *Client) GetBuildInfo(ctx context.Context) (*ApplicationBuildInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/buildInfo", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get build info: %s, body: %s", resp.Status, string(body))
	}

	var buildInfo ApplicationBuildInfo

	if err := json.NewDecoder(resp.Body).Decode(&buildInfo); err != nil {
		return nil, err
	}

	return &buildInfo, nil
}

type ScanDirMode int

const (
	ScanDirToMonitoredFolder ScanDirMode = 0 // Download to the monitored folder
	ScanDirToDefaultSavePath ScanDirMode = 1 // Download to the default save path
)

type ScanDirTarget struct {
	Mode ScanDirMode
	Path string // Download to this path (if not empty)
}

func (s ScanDirTarget) MarshalJSON() ([]byte, error) {
	if s.Path != "" {
		return json.Marshal(s.Path)
	}
	return json.Marshal(int(s.Mode))
}

type SchedulerDays int

const (
	ScheduleEveryDay     SchedulerDays = iota // Every day
	ScheduleEveryWeekday                      // Every weekday
	ScheduleEveryWeekend                      // Every weekend
	ScheduleMonday                            // Every Monday
	ScheduleTuesday                           // Every Tuesday
	ScheduleWednesday                         // Every Wednesday
	ScheduleThursday                          // Every Thursday
	ScheduleFriday                            // Every Friday
	ScheduleSaturday                          // Every Saturday
	ScheduleSunday                            // Every Sunday
)

type EncryptionMode int

const (
	EncryptionPrefer   EncryptionMode = iota // Prefer encryption
	EncryptionForceOn                        // Force encryption on
	EncryptionForceOff                       // Force encryption off
)

type ProxyType int

const (
	ProxyDisabled ProxyType = -1 // Proxy is disabled

	ProxyHTTPNoAuth   ProxyType = 1 // HTTP Proxy without authentication
	ProxySOCKS5NoAuth ProxyType = 2 // SOCKS5 Proxy without authentication
	ProxyHTTPAuth     ProxyType = 3 // HTTP Proxy with authentication
	ProxySOCKS5Auth   ProxyType = 4 // SOCKS5 Proxy with authentication
	ProxySOCKS4NoAuth ProxyType = 5 // SOCKS4 Proxy without authentication
)

type ProxyTypeValue struct {
	Value ProxyType
}

func (p *ProxyTypeValue) UnmarshalJSON(data []byte) error {
	// 尝试 int
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		p.Value = ProxyType(i)
		return nil
	}

	// 尝试 string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch strings.ToUpper(s) {
		case "HTTP":
			p.Value = ProxyHTTPNoAuth
		case "SOCKS5":
			p.Value = ProxySOCKS5NoAuth
		case "SOCKS4":
			p.Value = ProxySOCKS4NoAuth
		case "NONE":
			p.Value = ProxyDisabled
		default:
			return fmt.Errorf("unknown proxy_type: %s", s)
		}
		return nil
	}

	return fmt.Errorf("invalid proxy_type: %s", string(data))
}

type DynDNSService int

const (
	DynDNS DynDNSService = iota // Use DynDNS
	NoIP                        // Use No-IP
)

type MaxRatioAction int

const (
	MaxRatioPause  MaxRatioAction = iota // Pause torrent
	MaxRatioRemove                       // Remove torrent
)

type BitTorrentProtocol int

const (
	ProtocolTCPAndUTP BitTorrentProtocol = iota //TCP and μTP
	ProtocolTCPOnly                             // TCP
	ProtocolUTPOnly                             // μTP
)

type UploadChokingAlgorithm int

const (
	ChokingRoundRobin    UploadChokingAlgorithm = iota // Round-robin
	ChokingFastestUpload                               // Fastest upload
	ChokingAntiLeech                                   // Anti-leech
)

type UploadSlotsBehavior int

const (
	SlotsFixed           UploadSlotsBehavior = iota // Fixed slots
	SlotsUploadRateBased                            // Upload rate based
)

type UtpTcpMixedMode int

const (
	PreferTCP        UtpTcpMixedMode = iota // Prefer TCP
	PeerProportional                        // Peer proportional
)

type ApplicationPreferences struct {
	// -------- Downloads --------
	SavePath        string                   `json:"save_path"`
	TempPathEnabled bool                     `json:"temp_path_enabled"`
	TempPath        string                   `json:"temp_path"`
	ScanDirs        map[string]ScanDirTarget `json:"scan_dirs"`
	ExportDir       string                   `json:"export_dir"`
	ExportDirFin    string                   `json:"export_dir_fin"`

	// -------- Torrent Handling --------
	PreallocateAll     bool `json:"preallocate_all"`
	IncompleteFilesExt bool `json:"incomplete_files_ext"`
	AutoDeleteMode     int  `json:"auto_delete_mode"`

	// -------- Disk IO --------
	DiskCache      int `json:"disk_cache"`
	DiskCacheTTL   int `json:"disk_cache_ttl"`
	AsyncIOThreads int `json:"async_io_threads"`

	// -------- Queueing --------
	QueueingEnabled            bool `json:"queueing_enabled"`
	MaxActiveDownloads         int  `json:"max_active_downloads"`
	MaxActiveUploads           int  `json:"max_active_uploads"`
	MaxActiveTorrents          int  `json:"max_active_torrents"`
	DontCountSlowTorrents      bool `json:"dont_count_slow_torrents"`
	SlowTorrentDLRateThreshold int  `json:"slow_torrent_dl_rate_threshold"`
	SlowTorrentULRateThreshold int  `json:"slow_torrent_ul_rate_threshold"`
	SlowTorrentInactiveTimer   int  `json:"slow_torrent_inactive_timer"`

	// -------- Share Limits --------
	MaxRatioEnabled               bool           `json:"max_ratio_enabled"`
	MaxRatio                      float64        `json:"max_ratio"`
	MaxRatioAction                MaxRatioAction `json:"max_ratio_act"`
	MaxSeedingTimeEnabled         bool           `json:"max_seeding_time_enabled"`
	MaxSeedingTime                int            `json:"max_seeding_time"`
	MaxInactiveSeedingTimeEnabled bool           `json:"max_inactive_seeding_time_enabled"`
	MaxInactiveSeedingTime        int            `json:"max_inactive_seeding_time"`

	// -------- BitTorrent --------
	DHT           bool           `json:"dht"`
	PEX           bool           `json:"pex"`
	LSD           bool           `json:"lsd"`
	Encryption    EncryptionMode `json:"encryption"`
	AnonymousMode bool           `json:"anonymous_mode"`

	// -------- Connection --------
	ListenPort                   int                    `json:"listen_port"`
	UPnP                         bool                   `json:"upnp"`
	RandomPort                   bool                   `json:"random_port"`
	ReannounceWhenAddressChanged bool                   `json:"reannounce_when_address_changed"`
	BitTorrentProtocol           BitTorrentProtocol     `json:"bittorrent_protocol"`
	UploadChokingAlgorithm       UploadChokingAlgorithm `json:"upload_choking_algorithm"`
	UploadSlotsBehavior          UploadSlotsBehavior    `json:"upload_slots_behavior"`
	UtpTcpMixedMode              UtpTcpMixedMode        `json:"utp_tcp_mixed_mode"`

	// -------- Speed Limits --------
	DLLimit          int           `json:"dl_limit"`
	ULLimit          int           `json:"up_limit"`
	AltDLLimit       int           `json:"alt_dl_limit"`
	AltULLimit       int           `json:"alt_up_limit"`
	SchedulerEnabled bool          `json:"scheduler_enabled"`
	ScheduleFromHour int           `json:"schedule_from_hour"`
	ScheduleFromMin  int           `json:"schedule_from_min"`
	ScheduleToHour   int           `json:"schedule_to_hour"`
	ScheduleToMin    int           `json:"schedule_to_min"`
	SchedulerDays    SchedulerDays `json:"scheduler_days"`

	// -------- Proxy --------
	ProxyType            ProxyTypeValue `json:"proxy_type"`
	ProxyIP              string         `json:"proxy_ip"`
	ProxyPort            int            `json:"proxy_port"`
	ProxyPeerConnections bool           `json:"proxy_peer_connections"`
	ProxyAuthEnabled     bool           `json:"proxy_auth_enabled"`
	ProxyUsername        string         `json:"proxy_username"`
	ProxyPassword        string         `json:"proxy_password"`

	// -------- Dynamic DNS --------
	DynDNSEnabled  bool          `json:"dyndns_enabled"`
	DynDNSService  DynDNSService `json:"dyndns_service"`
	DynDNSUsername string        `json:"dyndns_username"`
	DynDNSPassword string        `json:"dyndns_password"`
	DynDNSDomain   string        `json:"dyndns_domain"`

	// -------- RSS --------
	RSSAutoDownloadingEnabled bool `json:"rss_auto_downloading_enabled"`
	RSSRefreshInterval        int  `json:"rss_refresh_interval"`
	RSSProcessingEnabled      bool `json:"rss_processing_enabled"`
	RSSMaxArticlesPerFeed     int  `json:"rss_max_articles_per_feed"`

	// -------- WebUI --------
	WebUIAddress               string `json:"web_ui_address"`
	WebUIPort                  int    `json:"web_ui_port"`
	WebUIUsername              string `json:"web_ui_username"`
	WebUIPassword              string `json:"web_ui_password"` // Note: This field will always be empty when retrieving preferences
	WebUICSRFProtectionEnabled bool   `json:"web_ui_csrf_protection_enabled"`
	WebUIHostHeaderValidation  bool   `json:"web_ui_host_header_validation_enabled"`
	WebUIUseHTTPS              bool   `json:"web_ui_use_https"`
	WebUICertificate           string `json:"web_ui_certificate"`
	WebUIKey                   string `json:"web_ui_key"`

	// -------- Advanced --------
	BypassLocalAuth           bool   `json:"bypass_local_auth"`
	BypassAuthSubnetWhitelist string `json:"bypass_auth_subnet_whitelist"`
	AltSpeedEnabled           bool   `json:"alt_speed_enabled"`
}

func (c *Client) GetApplicationPreferences(ctx context.Context) (*ApplicationPreferences, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/preferences", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get application preferences: %s, body: %s", resp.Status, string(body))
	}

	var prefs ApplicationPreferences

	if err := json.NewDecoder(resp.Body).Decode(&prefs); err != nil {
		return nil, err
	}

	return &prefs, nil
}

func (c *Client) SetApplicationPreferences(ctx context.Context, prefs *ApplicationPreferences) error {
	data, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/app/setPreferences",
		strings.NewReader(string(data)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set application preferences: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) GetDefaultSavePath(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/defaultSavePath", nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get default save path: %s, body: %s", resp.Status, string(body))
	}

	return string(body), nil
}

type Cookie struct {
	Name           string `json:"name"`           // Cookie name
	Value          string `json:"value"`          // Cookie value
	Domain         string `json:"domain"`         // Cookie domain
	Path           string `json:"path"`           // Cookie path
	ExpirationDate int64  `json:"expirationDate"` // Seconds since epoch
}

func (c *Client) GetCookies(ctx context.Context) ([]Cookie, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/app/cookies", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get cookies: %s, body: %s", resp.Status, string(body))
	}

	var cookies []Cookie

	if err := json.NewDecoder(resp.Body).Decode(&cookies); err != nil {
		return nil, err
	}

	return cookies, nil
}

func (c *Client) SetCookies(ctx context.Context, cookies []Cookie) error {
	data, err := json.Marshal(cookies)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/app/setCookies", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set cookies: %s, body: %s", resp.Status, string(body))
	}

	return nil
}
