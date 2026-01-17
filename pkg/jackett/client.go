/* jackett package provides a client for interacting with the Jackett torznab API. */
package jackett

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL    string
	apiKey     string
	password   string
	httpClient *http.Client
	cookie     http.Cookie
}

func NewClient(baseURL string, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Transport: &http.Transport{Proxy: nil}},
	}
}

func (c *Client) SetPassword(password string) {
	c.password = password
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

func (c *Client) Search(req SearchRequest) ([]SearchResult, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v2.0/indexers/all/results", c.baseURL))
	if err != nil {
		return nil, err
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
	println("Jackett Search URL:", u.String())

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jackett API error: %s, body: %s", resp.Status, string(body))
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
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return searchResults.Results, nil
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
	if c.cookie.String() == "" {
		if err := c.fetchCookie(); err != nil {
			return nil, fmt.Errorf("failed to fetch cookie: %w", err)
		}
	}

	u, err := url.Parse(fmt.Sprintf("%s/api/v2.0/indexers", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("apikey", c.apiKey)

	u.RawQuery = q.Encode()

	req := &http.Request{}
	req.Method = http.MethodPost
	req.URL = u
	req.Header.Set("Cookie", c.cookie.String())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cookie := resp.Header.Get("Set-Cookie")
		if cookie != "" {
			c.apiKey = cookie
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jackett API error: %s, body: %s", resp.Status, string(body))
	}

	var indexers []Indexer
	if err := json.NewDecoder(resp.Body).Decode(&indexers); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return indexers, nil
}

/*
async function fetchJackettCookie(widget, loginURL) {
  const url = new URL(formatApiCall(loginURL, widget));
  const loginData = `password=${encodeURIComponent(widget.password)}`;
  const [status, , , , params] = await httpProxy(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: loginData,
  });

  if (!(status === 200) || !params?.headers?.Cookie) {
    logger.error("Failed to fetch Jackett cookie, status: %d", status);
    return null;
  }
  return params.headers.Cookie;
}
*/

func (c *Client) fetchCookie() error {
	u, err := url.Parse(fmt.Sprintf("%s/UI/Dashboard", c.baseURL))
	if err != nil {
		return err
	}

	req := &http.Request{}
	req.Method = http.MethodPost
	req.URL = u
	req.Header = http.Header{}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(fmt.Sprintf("password=%s", url.QueryEscape(c.password))))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch cookie: %s", resp.Status)
	}

	c.cookie.Unparsed = resp.Header["Set-Cookie"]
	println("Jackett cookie:", c.cookie.String())
	return nil
}
