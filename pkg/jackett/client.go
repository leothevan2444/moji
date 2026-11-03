/* jackett package provides a client for interacting with the Jackett torznab API. */
package jackett

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// IndexerStatus captures the per-indexer response Jackett returns alongside a
// search. We expose this so the stats collector can surface indexer latency
// and per-indexer errors without re-issuing a search request.
type IndexerStatus struct {
	ID          string
	Name        string
	Status      int
	Results     int
	Error       string
	ElapsedTime int // milliseconds
}

// IndexerStatusHook is an optional global callback fired once per Search call.
// The stats collector registers this during startup; production code that does
// not care about indexer telemetry leaves it nil.
var (
	indexerStatusMu sync.RWMutex
	indexerStatusFn func(statuses []IndexerStatus, query string)
)

// SetIndexerStatusHook installs (or clears with nil) the global callback fired
// by SearchWithIndexerStatus. Only one hook is supported; the most recent call
// wins.
func SetIndexerStatusHook(fn func(statuses []IndexerStatus, query string)) {
	indexerStatusMu.Lock()
	defer indexerStatusMu.Unlock()
	indexerStatusFn = fn
}

func fireIndexerStatusHook(statuses []IndexerStatus, query string) {
	indexerStatusMu.RLock()
	fn := indexerStatusFn
	indexerStatusMu.RUnlock()
	if fn != nil {
		fn(statuses, query)
	}
}

type Client struct {
	baseURL  string
	apiKey   string
	password string

	httpClient *http.Client
}

// NewAPIClient does not open a management session or perform network I/O.
// The caller owns the supplied HTTP transport and request lifecycle.
func NewAPIClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	target, err := url.Parse(baseURL)
	if err != nil || target.Hostname() == "" || target.User != nil || target.Fragment != "" || target.RawQuery != "" || target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("invalid Jackett API URL")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil}}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, httpClient: httpClient}, nil
}

func NewClient(baseURL string, apiKey string, password string) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create cookie jar: %v", err))
	}
	client := &Client{
		baseURL:  baseURL,
		apiKey:   apiKey,
		password: password,
		httpClient: &http.Client{
			Jar:       jar,
			Timeout:   30 * time.Second,
			Transport: &http.Transport{Proxy: nil},
		},
	}
	if password != "" {
		if err := client.updateCookies(); err != nil {
			panic(fmt.Sprintf("failed to update cookies: %v", err))
		}
	}
	return client
}

func (c *Client) updateCookies() error {
	u, err := url.Parse(fmt.Sprintf("%s/UI/Dashboard", c.baseURL))
	if err != nil {
		return err
	}

	// fill password form data
	formData := url.Values{}
	formData.Set("password", c.password)

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "moji")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jackett login error: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

type SearchRequest struct {
	Query      string
	Trackers   []string
	Categories []int
}

type HTTPError struct {
	Status     int
	RetryAfter string
}

var ErrResponseTooLarge = errors.New("Jackett response exceeds byte budget")

func (e *HTTPError) Error() string { return fmt.Sprintf("Jackett API HTTP %d", e.Status) }

type SearchResult struct {
	FirstSeen            string   `json:"FirstSeen"`
	Tracker              string   `json:"Tracker"`
	TrackerID            string   `json:"TrackerId"`
	TrackerType          string   `json:"TrackerType"`
	CategoryDesc         string   `json:"CategoryDesc"`
	BlackholeLink        string   `json:"BlackholeLink"`
	Title                string   `json:"Title"`
	GUID                 string   `json:"Guid"`
	Link                 string   `json:"Link"`
	Details              string   `json:"Details"`
	PublishDate          string   `json:"PublishDate"`
	Category             []int    `json:"Category"`
	Size                 int64    `json:"Size"`
	Files                []string `json:"Files"`
	Grabs                int      `json:"Grabs"`
	Description          string   `json:"Description"`
	RageID               string   `json:"RageID"`
	TVDBID               string   `json:"TVDBId"`
	IMDB                 string   `json:"Imdb"`
	TMDb                 string   `json:"TMDb"`
	TVMazeID             string   `json:"TVMazeId"`
	TraktID              string   `json:"TraktId"`
	DoubanID             string   `json:"DoubanId"`
	Genres               []string `json:"Genres"`
	Languages            []string `json:"Languages"`
	Subs                 []string `json:"Subs"`
	Year                 int      `json:"Year"`
	Author               string   `json:"Author"`
	BookTitle            string   `json:"BookTitle"`
	Publisher            string   `json:"Publisher"`
	Artist               string   `json:"Artist"`
	Album                string   `json:"Album"`
	Label                string   `json:"Label"`
	Track                string   `json:"Track"`
	Seeders              int      `json:"Seeders"`
	Peers                int      `json:"Peers"`
	Poster               string   `json:"Poster"`
	InfoHash             string   `json:"InfoHash"`
	MagnetURI            string   `json:"MagnetUri"`
	MinimumRatio         float64  `json:"MinimumRatio"`
	MinimumSeedTime      int      `json:"MinimumSeedTime"`
	DownloadVolumeFactor float64  `json:"DownloadVolumeFactor"`
	UploadVolumeFactor   float64  `json:"UploadVolumeFactor"`
	Gain                 float64  `json:"Gain"`
	PeersKnown           bool     `json:"-"`
	SizeKnown            bool     `json:"-"`
}

// UnmarshalJSON preserves missing/null seeder counts as -1. Zero is a real
// observation and must not be used to represent an unknown count.
func (r *SearchResult) UnmarshalJSON(data []byte) error {
	type plain SearchResult
	decoded := struct {
		*plain
		Seeders *int   `json:"Seeders"`
		Peers   *int   `json:"Peers"`
		Size    *int64 `json:"Size"`
	}{plain: (*plain)(r)}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	r.Seeders = -1
	if decoded.Seeders != nil && *decoded.Seeders >= 0 {
		r.Seeders = *decoded.Seeders
	}
	r.PeersKnown = decoded.Peers != nil && *decoded.Peers >= 0
	if decoded.Peers != nil {
		r.Peers = *decoded.Peers
	}
	r.SizeKnown = decoded.Size != nil && *decoded.Size >= 0
	if decoded.Size != nil {
		r.Size = *decoded.Size
	}
	return nil
}

// SearchResultEnvelope is the parsed shape of a Jackett search response.
// Returning the raw Indexers array alongside the Results lets callers expose
// per-indexer latency and errors to the stats collector without a second HTTP
// call.
type SearchResultEnvelope struct {
	Results  []SearchResult
	Indexers []IndexerStatus
}

func (c *Client) Search(req SearchRequest) ([]SearchResult, error) {
	results, _, err := c.searchInternal(req)
	return results, err
}

// SearchWithIndexerStatus behaves like Search but also returns the
// per-indexer status slice (latency + per-indexer error). It also fires the
// global indexer-status hook if one is registered, so the stats collector can
// surface indexer telemetry even when callers only care about Results.
func (c *Client) SearchWithIndexerStatus(req SearchRequest) ([]SearchResult, []IndexerStatus, error) {
	results, indexers, err := c.searchInternal(req)
	if err == nil {
		fireIndexerStatusHook(indexers, req.Query)
	}
	return results, indexers, err
}

func (c *Client) searchInternal(req SearchRequest) ([]SearchResult, []IndexerStatus, error) {
	return c.searchContext(context.Background(), req)
}
func (c *Client) SearchContext(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	results, statuses, err := c.searchContext(ctx, req)
	if err == nil {
		fireIndexerStatusHook(statuses, req.Query)
	}
	return results, err
}

// SearchWithIndexerStatusContext returns observations to its caller without a
// global hook; independent indexer services can own their own status updates.
func (c *Client) SearchWithIndexerStatusContext(ctx context.Context, req SearchRequest) ([]SearchResult, []IndexerStatus, error) {
	return c.searchContext(ctx, req)
}
func (c *Client) searchContext(ctx context.Context, req SearchRequest) ([]SearchResult, []IndexerStatus, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v2.0/indexers/all/results", c.baseURL))
	if err != nil {
		return nil, nil, err
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	q.Set("Query", req.Query)

	if len(req.Trackers) > 0 {
		for _, tracker := range req.Trackers {
			q.Add("Tracker[]", tracker)
		}
	}

	if len(req.Categories) > 0 {
		for _, category := range req.Categories {
			q.Add("Category[]", fmt.Sprintf("%d", category))
		}
	}

	u.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, &HTTPError{Status: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After")}
	}

	var searchResults struct {
		Results  []SearchResult `json:"Results"`
		Indexers []struct {
			ID          string `json:"ID"`
			Name        string `json:"Name"`
			Status      int    `json:"Status"`
			Results     int    `json:"Results"`
			Error       string `json:"Error"`
			ElapsedTime int    `json:"ElapsedTime"`
		} `json:"Indexers"`
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (16<<20)+1))
	if err != nil {
		return nil, nil, err
	}
	if len(data) > 16<<20 {
		return nil, nil, ErrResponseTooLarge
	}
	if err := json.Unmarshal(data, &searchResults); err != nil {
		return nil, nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	indexers := make([]IndexerStatus, len(searchResults.Indexers))
	for i, in := range searchResults.Indexers {
		indexers[i] = IndexerStatus{
			ID:          in.ID,
			Name:        in.Name,
			Status:      in.Status,
			Results:     in.Results,
			Error:       in.Error,
			ElapsedTime: in.ElapsedTime,
		}
	}
	return searchResults.Results, indexers, nil
}

type Indexer struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Type                 string   `json:"type"`
	Configured           bool     `json:"configured"`
	SiteLink             string   `json:"site_link"`
	AlternativeSiteLinks []string `json:"alternativesitelinks"`
	Language             string   `json:"language"`
	Tags                 []string `json:"tags"`
	LastError            string   `json:"last_error"`
	PotatoEnabled        bool     `json:"potatoenabled"`
	Caps                 []struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	} `json:"caps"`
}

func (c *Client) GetIndexers() ([]Indexer, error) { return c.GetIndexersContext(context.Background()) }
func (c *Client) GetIndexersContext(ctx context.Context) ([]Indexer, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v2.0/indexers", c.baseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	u.RawQuery = q.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jackett API error: %s, body: %s", resp.Status, string(body))
	}

	var indexers []Indexer
	if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return indexers, nil
}

// CloseIdleConnections releases idle connections held by a short-lived client.
func (c *Client) CloseIdleConnections() { c.httpClient.CloseIdleConnections() }

// GetAPIIndexersContext checks the same API-key authentication used by searches.
// t=indexers reads local definitions; it does not send searches to tracker sites.
// The dashboard /indexers endpoint uses a separate management login session.
func (c *Client) GetAPIIndexersContext(ctx context.Context) ([]Indexer, error) {
	u, err := url.Parse(c.baseURL + "/api/v2.0/indexers/all/results/torznab/api")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("apikey", c.apiKey)
	q.Set("t", "indexers")
	q.Set("configured", "true")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jackett API status: %d", resp.StatusCode)
	}
	var result struct {
		XMLName xml.Name `xml:"indexers"`
		Items   []struct {
			ID          string `xml:"id,attr"`
			Configured  bool   `xml:"configured,attr"`
			Name        string `xml:"title"`
			Description string `xml:"description"`
			SiteLink    string `xml:"link"`
			Language    string `xml:"language"`
			Type        string `xml:"type"`
		} `xml:"indexer"`
	}
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if readErr != nil {
		return nil, readErr
	}
	if len(data) > 2<<20 {
		return nil, ErrResponseTooLarge
	}
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid Jackett indexers response: %w", err)
	}
	sites := make([]Indexer, 0, len(result.Items))
	for _, item := range result.Items {
		if item.ID == "" || item.Name == "" {
			return nil, fmt.Errorf("invalid Jackett indexer")
		}
		sites = append(sites, Indexer{ID: item.ID, Configured: item.Configured, Name: item.Name, Description: item.Description, SiteLink: item.SiteLink, Language: item.Language, Type: item.Type})
	}
	return sites, nil
}
