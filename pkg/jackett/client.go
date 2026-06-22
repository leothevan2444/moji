/* jackett package provides a client for interacting with the Jackett torznab API. */
package jackett

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
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

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("jackett API error: %s, body: %s", resp.Status, string(body))
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
	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
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

func (c *Client) GetIndexers() ([]Indexer, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v2.0/indexers", c.baseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("apikey", c.apiKey)
	u.RawQuery = q.Encode()

	resp, err := c.httpClient.Get(u.String())
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
