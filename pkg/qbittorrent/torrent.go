package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// All Torrent management API methods are under "torrents", e.g.: /api/v2/torrents/methodName.

type TorrentState string

const (
	TorrentStateError        TorrentState = "error"        // Some error occurred, applies to paused torrents
	TorrentStateMissingFiles TorrentState = "missingFiles" // Torrent data files is missing

	TorrentStateUploading  TorrentState = "uploading"  // Torrent is being seeded and data is being transferred
	TorrentStatePausedUP   TorrentState = "pausedUP"   // Torrent is paused and has finished downloading
	TorrentStateQueuedUP   TorrentState = "queuedUP"   // Queuing is enabled and torrent is queued for upload
	TorrentStateStalledUP  TorrentState = "stalledUP"  // Torrent is being seeded, but no connection were made
	TorrentStateCheckingUP TorrentState = "checkingUP" // Torrent has finished downloading and is being checked
	TorrentStateForcedUP   TorrentState = "forcedUP"   // Torrent is forced to uploading and ignore queue limit

	TorrentStateAllocating  TorrentState = "allocating"  // Torrent is allocating disk space for download
	TorrentStateDownloading TorrentState = "downloading" // Torrent is being downloaded and data is being transferred
	TorrentStateMetaDL      TorrentState = "metaDL"      // Torrent has just started downloading and is fetching metadata
	TorrentStatePausedDL    TorrentState = "pausedDL"    // Torrent is paused and has NOT finished downloading
	TorrentStateQueuedDL    TorrentState = "queuedDL"    // Queuing is enabled and torrent is queued for download
	TorrentStateStalledDL   TorrentState = "stalledDL"   // Torrent is being downloaded, but no connection were made
	TorrentStateCheckingDL  TorrentState = "checkingDL"  // Same as checkingUP, but torrent has NOT finished downloading
	TorrentStateForcedDL    TorrentState = "forcedDL"    // Torrent is forced to downloading to ignore queue limit

	TorrentStateCheckingResumeData TorrentState = "checkingResumeData" // Checking resume data on qBt startup
	TorrentStateMoving             TorrentState = "moving"             // Torrent is moving to another location
	TorrentStateUnknown            TorrentState = "unknown"            // Unknown status
)

type Torrent struct {
	AddedOn            int64        `json:"added_on"`           // Time (Unix Epoch) when the torrent was added to the client
	AmountLeft         int64        `json:"amount_left"`        // Amount of data left to download (bytes)
	AutoTMM            bool         `json:"auto_tmm"`           // Whether this torrent is managed by Automatic Torrent Management
	Availability       float64      `json:"availability"`       // Percentage of file pieces currently available
	Category           string       `json:"category"`           // Category of the torrent
	Completed          int64        `json:"completed"`          // Amount of transfer data completed (bytes)
	CompletionOn       int64        `json:"completion_on"`      // Time (Unix Epoch) when the torrent completed
	ContentPath        string       `json:"content_path"`       // Absolute path of torrent content (root path for multifile torrents, absolute file path for singlefile torrents)
	DLLimit            int64        `json:"dl_limit"`           // Torrent download speed limit (bytes/s). -1 if unlimited.
	DLSpeed            int64        `json:"dlspeed"`            // Torrent download speed (bytes/s)
	Downloaded         int64        `json:"downloaded"`         // Amount of data downloaded
	DownloadedSession  int64        `json:"downloaded_session"` // Amount of data downloaded this session
	ETA                int64        `json:"eta"`                // Torrent ETA (seconds)
	FirstLastPiecePrio bool         `json:"f_l_piece_prio"`     // True if first last piece are prioritized
	ForceStart         bool         `json:"force_start"`        // True if force start is enabled for this torrent
	Hash               string       `json:"hash"`               // Torrent hash
	IsPrivate          bool         `json:"isPrivate"`          // True if torrent is from a private tracker (added in 5.0.0)
	LastActivity       int64        `json:"last_activity"`      // Last time (Unix Epoch) when a chunk was downloaded/uploaded
	MagnetURI          string       `json:"magnet_uri"`         // Magnet URI corresponding to this torrent
	MaxRatio           float64      `json:"max_ratio"`          // Maximum share ratio until torrent is stopped from seeding/uploading
	MaxSeedingTime     int64        `json:"max_seeding_time"`   // Maximum seeding time (seconds) until torrent is stopped from seeding
	Name               string       `json:"name"`               // Torrent name
	NumComplete        int64        `json:"num_complete"`       // Number of seeds in the swarm
	NumIncomplete      int64        `json:"num_incomplete"`     // Number of leechers in the swarm
	NumLeechs          int64        `json:"num_leechs"`         // Number of leechers connected to
	NumSeeds           int64        `json:"num_seeds"`          // Number of seeds connected to
	Priority           int64        `json:"priority"`           // Torrent priority. Returns -1 if queuing is disabled or torrent is in seed mode
	Progress           float64      `json:"progress"`           // Torrent progress (percentage/100)
	Ratio              float64      `json:"ratio"`              // Torrent share ratio. Max ratio value: 9999.
	RatioLimit         float64      `json:"ratio_limit"`        // TODO (what is different from max_ratio?)
	Reannounce         int64        `json:"reannounce"`         // Time until the next tracker reannounce
	SavePath           string       `json:"save_path"`          // Path where this torrent's data is stored
	SeedingTime        int64        `json:"seeding_time"`       // Torrent elapsed time while complete (seconds)
	SeedingTimeLimit   int64        `json:"seeding_time_limit"` // TODO (what is different from max_seeding_time?) seeding_time_limit is a per torrent setting, when Automatic Torrent Management is disabled, furthermore then max_seeding_time is set to seeding_time_limit for this torrent. If Automatic Torrent Management is enabled, the value is -2. And if max_seeding_time is unset it have a default value -1.
	SeenComplete       int64        `json:"seen_complete"`      // Time (Unix Epoch) when this torrent was last seen complete
	SequentialDownload bool         `json:"seq_dl"`             // True if sequential download is enabled
	Size               int64        `json:"size"`               // Total size (bytes) of files selected for download
	State              TorrentState `json:"state"`              // Torrent state. See table here below for the possible values
	SuperSeeding       bool         `json:"super_seeding"`      // True if super seeding is enabled
	Tags               string       `json:"tags"`               // Comma-concatenated tag list of the torrent
	TimeActive         int64        `json:"time_active"`        // Total active time (seconds)
	TotalSize          int64        `json:"total_size"`         // Total size (bytes) of all file in this torrent (including unselected ones)
	Tracker            string       `json:"tracker"`            // The first tracker with working status. Returns empty string if no tracker is working.
	ULLimit            int64        `json:"up_limit"`           // Torrent upload speed limit (bytes/s). -1 if unlimited.
	Uploaded           int64        `json:"uploaded"`           // Amount of data uploaded
	UploadedSession    int64        `json:"uploaded_session"`   // Amount of data uploaded this session
	UPSpeed            int64        `json:"upspeed"`            // Torrent upload speed (bytes/s)
}

type TorrentListOptions struct {
	Filter   string   // Filter torrent list by state. Allowed state filters: all, downloading, seeding, completed, stopped, active, inactive, running, stalled, stalled_uploading, stalled_downloading, errored
	Category string   // Get torrents with the given category (empty string means "without category"; no "category" parameter means "any category"). Remember to URL-encode the category name. For example, My category becomes My%20category
	Tag      string   // Get torrents with the given tag (empty string means "without tag"; no "tag" parameter means "any tag". Remember to URL-encode the category name. For example, My tag becomes My%20tag
	Sort     string   // Sort torrents by given key. They can be sorted using any field of the response's JSON array (which are documented below) as the sort key.
	Reverse  *bool    // Enable reverse sorting. Defaults to false
	Limit    *int     // Limit the number of torrents returned
	Offset   *int     // Set offset (if less than 0, offset from end)
	Hashes   []string // Filter by hashes. Can contain multiple hashes separated by |
}

func buildTorrentListParams(opts *TorrentListOptions) url.Values {
	params := url.Values{}

	if opts == nil {
		return params
	}

	if opts.Filter != "" {
		params.Set("filter", opts.Filter)
	}

	if opts.Category != "" {
		params.Set("category", opts.Category)
	}

	if opts.Tag != "" {
		params.Set("tag", opts.Tag)
	}

	if opts.Sort != "" {
		params.Set("sort", opts.Sort)
	}

	if opts.Reverse != nil {
		params.Set("reverse", strconv.FormatBool(*opts.Reverse))
	}

	if opts.Limit != nil {
		params.Set("limit", strconv.Itoa(*opts.Limit))
	}

	if opts.Offset != nil {
		params.Set("offset", strconv.Itoa(*opts.Offset))
	}

	if len(opts.Hashes) > 0 {
		params.Set("hashes", strings.Join(opts.Hashes, "|"))
	}

	return params
}

func (c *Client) GetTorrentList(ctx context.Context, options *TorrentListOptions) ([]Torrent, error) {
	params := buildTorrentListParams(options)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/info?"+params.Encode(),
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

	var torrents []Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, err
	}

	return torrents, nil
}

type TorrentGenericProperties struct {
	SavePath               string  `json:"save_path"`                // Torrent save path
	CreationDate           int64   `json:"creation_date"`            // Torrent creation date (Unix timestamp)
	PieceSize              int64   `json:"piece_size"`               // Torrent piece size (bytes)
	Comment                string  `json:"comment"`                  // Torrent comment
	TotalWasted            int64   `json:"total_wasted"`             // Total data wasted for torrent (bytes)
	TotalUploaded          int64   `json:"total_uploaded"`           // Total data uploaded for torrent (bytes)
	TotalUploadedSession   int64   `json:"total_uploaded_session"`   // Total data uploaded this session (bytes)
	TotalDownloaded        int64   `json:"total_downloaded"`         // Total data downloaded for torrent (bytes)
	TotalDownloadedSession int64   `json:"total_downloaded_session"` // Total data downloaded this session (bytes)
	UPLimit                int64   `json:"up_limit"`                 // Torrent upload limit (bytes/s)
	DLLimit                int64   `json:"dl_limit"`                 // Torrent download limit (bytes/s)
	TimeElapsed            int64   `json:"time_elapsed"`             // Torrent elapsed time (seconds)
	SeedingTime            int64   `json:"seeding_time"`             // Torrent elapsed time while complete (seconds)
	NbConnections          int64   `json:"nb_connections"`           // Torrent connection count
	NbConnectionsLimit     int64   `json:"nb_connections_limit"`     // Torrent connection count limit
	ShareRatio             float64 `json:"share_ratio"`              // Torrent share ratio
	AdditionDate           int64   `json:"addition_date"`            // When this torrent was added (unix timestamp)
	CompletionDate         int64   `json:"completion_date"`          // Torrent completion date (unix timestamp)
	CreatedBy              string  `json:"created_by"`               // Torrent creator
	DLSpeedAvg             int64   `json:"dl_speed_avg"`             // Torrent average download speed (bytes/second)
	DLSpeed                int64   `json:"dl_speed"`                 // Torrent download speed (bytes/second)
	ETA                    int64   `json:"eta"`                      // Torrent ETA (seconds)
	LastSeen               int64   `json:"last_seen"`                // Last seen complete date (unix timestamp)
	Peers                  int64   `json:"peers"`                    // Number of peers connected to
	PeersTotal             int64   `json:"peers_total"`              // Number of peers in the swarm
	PiecesHave             int64   `json:"pieces_have"`              // Number of pieces owned
	PiecesNum              int64   `json:"pieces_num"`               // Number of pieces of the torrent
	Reannounce             int64   `json:"reannounce"`               // Number of seconds until the next announce
	Seeds                  int64   `json:"seeds"`                    // Number of seeds connected to
	SeedsTotal             int64   `json:"seeds_total"`              // Number of seeds in the swarm
	TotalSize              int64   `json:"total_size"`               // Torrent total size (bytes)
	UPSpeedAvg             int64   `json:"up_speed_avg"`             // Torrent average upload speed (bytes/second)
	UPSpeed                int64   `json:"up_speed"`                 // Torrent upload speed (bytes/second)
	IsPrivate              bool    `json:"isPrivate"`                // True if torrent is from a private tracker
}

func (c *Client) GetTorrentGenericProperties(ctx context.Context, hash string) (*TorrentGenericProperties, error) {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/properties?"+params.Encode(),
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

	var properties TorrentGenericProperties
	if err := json.NewDecoder(resp.Body).Decode(&properties); err != nil {
		return nil, err
	}

	return &properties, nil
}

type TorrentTrackerStatus int

const (
	TorrentTrackerStatusDisabled     TorrentTrackerStatus = 0 // Tracker is disabled (used for DHT, PeX, and LSD)
	TorrentTrackerStatusNotContacted TorrentTrackerStatus = 1 // Tracker has not been contacted yet
	TorrentTrackerStatusWorking      TorrentTrackerStatus = 2 // Tracker has been contacted and is working
	TorrentTrackerStatusUpdating     TorrentTrackerStatus = 3 // Tracker is updating
	TorrentTrackerStatusNotWorking   TorrentTrackerStatus = 4 // Tracker has been contacted, but it is not working (or doesn't send proper replies)
)

type TorrentTracker struct {
	URL           string               `json:"url"`            // Tracker url
	Status        TorrentTrackerStatus `json:"status"`         // Tracker status. See the table below for possible values
	Tier          int                  `json:"tier"`           // Tracker priority tier. Lower tier trackers are tried before higher tiers. Tier numbers are valid when >= 0, < 0 is used as placeholder when tier does not exist for special entries (such as DHT).
	NumPeers      int                  `json:"num_peers"`      // Number of peers for current torrent, as reported by the tracker
	NumSeeds      int                  `json:"num_seeds"`      // Number of seeds for current torrent, as reported by the tracker
	NumLeeches    int                  `json:"num_leeches"`    // Number of leeches for current torrent, as reported by the tracker
	NumDownloaded int                  `json:"num_downloaded"` // Number of completed downloads for current torrent, as reported by the tracker
	Msg           string               `json:"msg"`            // Tracker message (there is no way of knowing what this message is - it's up to tracker admins)
}

func (c *Client) GetTorrentTrackers(ctx context.Context, hash string) ([]TorrentTracker, error) {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/trackers?"+params.Encode(),
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

	var trackers []TorrentTracker
	if err := json.NewDecoder(resp.Body).Decode(&trackers); err != nil {
		return nil, err
	}

	return trackers, nil
}

func (c *Client) GetTorrentWebSeeds(ctx context.Context, hash string) ([]string, error) {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/webseeds?"+params.Encode(),
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

	var webSeeds []string
	if err := json.NewDecoder(resp.Body).Decode(&webSeeds); err != nil {
		return nil, err
	}

	return webSeeds, nil
}

type TorrentFilePriority int

const (
	TorrentFilePriorityDontDownload TorrentFilePriority = 0 // Do not download
	TorrentFilePriorityNormal       TorrentFilePriority = 1 // Normal priority
	TorrentFilePriorityHigh         TorrentFilePriority = 6 // High priority
	TorrentFilePriorityMax          TorrentFilePriority = 7 // Maximum priority
)

type TorrentContentFile struct {
	Index        int     `json:"index"`        // File index (since 2.8.2)
	Name         string  `json:"name"`         // File name (including relative path)
	Size         int64   `json:"size"`         // File size (bytes)
	Progress     float64 `json:"progress"`     // File progress (percentage/100)
	Priority     int     `json:"priority"`     // File priority. See possible values here below
	IsSeed       bool    `json:"is_seed"`      // True if file is seeding/complete
	PieceRange   []int   `json:"piece_range"`  // The first number is the starting piece index and the second number is the ending piece index (inclusive)
	Availability float64 `json:"availability"` // Percentage of file pieces currently available (percentage/100)
}

func (c *Client) GetTorrentContents(ctx context.Context, hash string, indexes []string) ([]TorrentContentFile, error) {
	params := url.Values{}
	params.Set("hash", hash)
	if len(indexes) > 0 {
		params.Set("indexes", strings.Join(indexes, "|"))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/files?"+params.Encode(),
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

	var files []TorrentContentFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, err
	}

	return files, nil
}

type PieceState int

const (
	PieceStatNotDownloaded   PieceState = 0 // Not downloaded yet
	PieceStateNowDownloading PieceState = 1 // Now downloading
	PieceStateDownloaded     PieceState = 2 // Already downloaded
)

func (c *Client) GetTorrentPiecesStates(ctx context.Context, hash string) ([]int, error) {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/pieceStates?"+params.Encode(),
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

	var pieceStates []int
	if err := json.NewDecoder(resp.Body).Decode(&pieceStates); err != nil {
		return nil, err
	}

	return pieceStates, nil
}

func (c *Client) GetTorrentPiecesHashes(ctx context.Context, hash string) ([]string, error) {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/pieceHashes?"+params.Encode(),
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

	var pieceHashes []string
	if err := json.NewDecoder(resp.Body).Decode(&pieceHashes); err != nil {
		return nil, err
	}

	return pieceHashes, nil
}

func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/stop",
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

func (c *Client) ResumeTorrents(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/start",
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

func (c *Client) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	if deleteFiles {
		params.Set("deleteFiles", "true")
	} else {
		params.Set("deleteFiles", "false")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/delete",
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

func (c *Client) RecheckTorrents(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/recheck",
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

func (c *Client) ReannounceTorrents(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/reannounce",
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

type TorrentFile struct {
	Filename string
	Data     []byte
}

type AddTorrentOptions struct {
	URLs               []string      `json:"-"`                            // URLs separated with newlines
	Torrents           []TorrentFile `json:"-"`                            // Raw data of torrent file. torrents can be presented multiple times.
	SavePath           *string       `json:"savepath,omitempty"`           // Download folder
	Category           *string       `json:"category,omitempty"`           // Category for the torrent
	Tags               *string       `json:"tags,omitempty"`               // Tags for the torrent, split by ','
	SkipChecking       *bool         `json:"skip_checking,omitempty"`      // Skip hash checking. Possible values are true, false (default)
	Paused             *bool         `json:"paused,omitempty"`             // Add torrents in the paused state. Possible values are true, false (default)
	RootFolder         *string       `json:"root_folder,omitempty"`        // Create the root folder. Possible values are true, false, unset (default)
	Rename             *string       `json:"rename,omitempty"`             // Rename torrent
	UPLimit            *int64        `json:"upLimit,omitempty"`            // Set torrent upload speed limit. Unit in bytes/second
	DLLimit            *int64        `json:"dlLimit,omitempty"`            // Set torrent download speed limit. Unit in bytes/second
	RatioLimit         *float64      `json:"ratioLimit,omitempty"`         // Set torrent share ratio limit (since 2.8.1)
	SeedingTimeLimit   *int          `json:"seedingTimeLimit,omitempty"`   // Set torrent seeding time limit. Unit in minutes (since 2.8.1)
	AutoTMM            *bool         `json:"autoTMM,omitempty"`            // Whether Automatic Torrent Management should be used
	SequentialDownload *bool         `json:"sequentialDownload,omitempty"` // Enable sequential download. Possible values are true, false (default)
	FirstLastPiecePrio *bool         `json:"firstLastPiecePrio,omitempty"` // Prioritize download first last piece. Possible values are true, false (default)
}

func (c *Client) AddNewTorrent(ctx context.Context, opts AddTorrentOptions) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("urls", strings.Join(opts.URLs, "\n"))

	// Optional fields
	if opts.SavePath != nil {
		writer.WriteField("savepath", *opts.SavePath)
	}
	if opts.Category != nil {
		writer.WriteField("category", *opts.Category)
	}
	if opts.Tags != nil {
		writer.WriteField("tags", *opts.Tags)
	}
	if opts.SkipChecking != nil {
		writer.WriteField("skip_checking", strconv.FormatBool(*opts.SkipChecking))
	}
	if opts.Paused != nil {
		writer.WriteField("paused", strconv.FormatBool(*opts.Paused))
	}
	if opts.RootFolder != nil {
		writer.WriteField("root_folder", *opts.RootFolder) // "true", "false", "unset"
	}
	if opts.Rename != nil {
		writer.WriteField("rename", *opts.Rename)
	}
	if opts.UPLimit != nil {
		writer.WriteField("upLimit", strconv.FormatInt(*opts.UPLimit, 10))
	}
	if opts.DLLimit != nil {
		writer.WriteField("dlLimit", strconv.FormatInt(*opts.DLLimit, 10))
	}
	if opts.AutoTMM != nil {
		writer.WriteField("autoTMM", strconv.FormatBool(*opts.AutoTMM))
	}
	if opts.SequentialDownload != nil {
		writer.WriteField("sequentialDownload", strconv.FormatBool(*opts.SequentialDownload))
	}
	if opts.FirstLastPiecePrio != nil {
		writer.WriteField("firstLastPiecePrio", strconv.FormatBool(*opts.FirstLastPiecePrio))
	}

	for _, torrent := range opts.Torrents {
		part, err := writer.CreateFormFile("torrents", torrent.Filename)
		if err != nil {
			return err
		}
		if _, err := part.Write(torrent.Data); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/add",
		body,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add torrent failed with status: %s", resp.Status)
	}

	return nil
}

func (c *Client) AddTrackers(ctx context.Context, hash string, urls []string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("urls", strings.Join(urls, "\n"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/addTrackers",
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

func (c *Client) EditTrackers(ctx context.Context, hash string, oldURL string, newURL string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("oldUrl", oldURL)
	params.Set("newUrl", newURL)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/editTracker",
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

func (c *Client) RemoveTrackers(ctx context.Context, hash string, urls []string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("urls", strings.Join(urls, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/removeTrackers",
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

func (c *Client) AddPeers(ctx context.Context, hash string, peers []string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("peers", strings.Join(peers, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/addPeers",
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

func (c *Client) IncreaseTorrentPriority(ctx context.Context, hash string) error {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/increasePrio",
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

func (c *Client) DecreaseTorrentPriority(ctx context.Context, hash string) error {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/decreasePrio",
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

func (c *Client) MaximalTorrentPriority(ctx context.Context, hash string) error {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/topPrio",
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

func (c *Client) MinimalTorrentPriority(ctx context.Context, hash string) error {
	params := url.Values{}
	params.Set("hash", hash)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/bottomPrio",
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

func (c *Client) SetTorrentFilePriority(ctx context.Context, hash string, fileIndexs []int, priority TorrentFilePriority) error {
	params := url.Values{}
	params.Set("hash", hash)
	strIndexes := make([]string, 0, len(fileIndexs))
	for _, index := range fileIndexs {
		strIndexes = append(strIndexes, strconv.Itoa(index))
	}
	params.Set("indexes", strings.Join(strIndexes, "|"))
	params.Set("priority", strconv.Itoa(int(priority)))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/filePrio",
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

type TorrentDownloadLimit map[string]int64 // key: torrent hash | value: Download limit (bytes/s). zero if unlimited.

func (c *Client) GetTorrentDownloadLimit(ctx context.Context, hashes []string) (*TorrentDownloadLimit, error) {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/downloadLimit?"+params.Encode(),
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

	limitMap := make(TorrentDownloadLimit)
	if err := json.NewDecoder(resp.Body).Decode(&limitMap); err != nil {
		return nil, err
	}

	return &limitMap, nil
}

func (c *Client) SetTorrentDownloadLimit(ctx context.Context, hashes []string, limit int64) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("limit", strconv.FormatInt(limit, 10))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setDownloadLimit",
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

type TorrentShareLimit struct {
	ratio               float64 // The maximum seeding ratio for the torrent. -2 means the global limit should be used, -1 means no limit.
	seedingTime         int64   // The maximum seeding time (minutes) for the torrent. -2 means the global limit should be used, -1 means no limit.
	inactiveSeedingTime int64   // The maximum amount of time (minutes) the torrent is allowed to seed while being inactive. -2 means the global limit should be used, -1 means no limit.
}

func (c *Client) SetTorrentShareLimits(ctx context.Context, hashes []string, limit TorrentShareLimit) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("ratioLimit", strconv.FormatFloat(limit.ratio, 'f', -1, 64))
	params.Set("seedingTimeLimit", strconv.FormatInt(limit.seedingTime, 10))
	params.Set("inactiveSeedingTimeLimit", strconv.FormatInt(limit.inactiveSeedingTime, 10))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setShareLimits",
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

type TorrentUploadLimit map[string]int64 // key: torrent hash | value: Upload limit (bytes/s). zero if unlimited.

func (c *Client) GetTorrentUploadLimit(ctx context.Context, hashes []string) (*TorrentUploadLimit, error) {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/uploadLimit?"+params.Encode(),
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

	limitMap := make(TorrentUploadLimit)
	if err := json.NewDecoder(resp.Body).Decode(&limitMap); err != nil {
		return nil, err
	}

	return &limitMap, nil
}

func (c *Client) SetTorrentUploadLimit(ctx context.Context, hashes []string, limit int64) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("limit", strconv.FormatInt(limit, 10))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setUploadLimit",
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

func (c *Client) SetTorrentLocation(ctx context.Context, hashes []string, location string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("location", location)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setLocation",
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

func (c *Client) SetTorrentName(ctx context.Context, hash string, newName string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("name", newName)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/rename",
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

func (c *Client) SetTorrentCategory(ctx context.Context, hashes []string, categoryName string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("category", categoryName)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setCategory",
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

type Category struct {
	Name     string `json:"name"`     // Category name
	SavePath string `json:"savePath"` // Category save path
}

func (c *Client) GetAllCategories(ctx context.Context) (map[string]Category, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/categories",
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

	categories := make(map[string]Category)
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, err
	}

	return categories, nil
}

func (c *Client) AddNewCategory(ctx context.Context, category Category) error {
	params := url.Values{}
	params.Set("category", category.Name)
	params.Set("savePath", category.SavePath)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/addCategory",
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

func (c *Client) EditCategory(ctx context.Context, category Category) error {
	params := url.Values{}
	params.Set("category", category.Name)
	params.Set("savePath", category.SavePath)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/editCategory",
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

func (c *Client) RemoveCategory(ctx context.Context, categoryName string) error {
	params := url.Values{}
	params.Set("category", categoryName)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/removeCategory",
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

func (c *Client) AddTorrentTags(ctx context.Context, hashes []string, tags []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/addTags",
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

func (c *Client) RemoveTorrentTags(ctx context.Context, hashes []string, tags []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/removeTags?",
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

type TorrentTag string

func (c *Client) GetAllTags(ctx context.Context) ([]TorrentTag, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/torrents/tags",
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

	var tags []TorrentTag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	return tags, nil
}

func (c *Client) CreateTag(ctx context.Context, tags []TorrentTag) error {
	params := url.Values{}
	params.Set("tags", strings.Join(func() []string {
		strs := make([]string, 0, len(tags))
		for _, tag := range tags {
			strs = append(strs, string(tag))
		}
		return strs
	}(), ","))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/createTags",
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

func (c *Client) DeleteTags(ctx context.Context, tags []TorrentTag) error {
	params := url.Values{}
	params.Set("tags", strings.Join(func() []string {
		strs := make([]string, 0, len(tags))
		for _, tag := range tags {
			strs = append(strs, string(tag))
		}
		return strs
	}(), ","))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/deleteTags?",
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

func (c *Client) SetAutomaticTorrentManagement(ctx context.Context, hashes []string, enable bool) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("enable", strconv.FormatBool(enable))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setAutoManagement",
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

func (c *Client) ToggleSequentialDownload(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/toggleSequentialDownload",
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

func (c *Client) SetFirstLastPiecePriority(ctx context.Context, hashes []string) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/toggleFirstLastPiecePrio",
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

func (c *Client) SetForceStart(ctx context.Context, hashes []string, enable bool) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("value", strconv.FormatBool(enable))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setForceStart",
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

func (c *Client) SetSuperSeeding(ctx context.Context, hashes []string, enable bool) error {
	params := url.Values{}
	params.Set("hashes", strings.Join(hashes, "|"))
	params.Set("value", strconv.FormatBool(enable))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/setSuperSeeding",
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

func (c *Client) RenameTorrentFile(ctx context.Context, hash string, oldPath string, newPath string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("oldPath", oldPath)
	params.Set("newPath", newPath)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/renameFile",
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

func (c *Client) RenameFolder(ctx context.Context, hash string, oldPath string, newPath string) error {
	params := url.Values{}
	params.Set("hash", hash)
	params.Set("oldPath", oldPath)
	params.Set("newPath", newPath)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/torrents/renameFolder",
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
