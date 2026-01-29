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

// All Search API methods are under "search", e.g.: /api/v2/search/methodName.

func (c *Client) StartSearch(ctx context.Context, pattern, plugins, category string) (int, error) {
	params := url.Values{}
	params.Set("pattern", pattern)
	params.Set("plugins", plugins)
	params.Set("category", category)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/start",
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to start search: status code %d %s", resp.StatusCode, resp.Status)
	}

	var respObj struct {
		ID int `json:"id"` // ID of the search job
	}
	if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
		return 0, err
	}

	return respObj.ID, nil
}

func (c *Client) StopSearch(ctx context.Context, searchID int) error {
	params := url.Values{}
	params.Set("id", strconv.Itoa(searchID))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/stop",
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

type SearchStatus struct {
	ID     int    `json:"id"`     // ID of the search job
	Status string `json:"status"` // Current status of the search job (either Running or Stopped)
	Total  int    `json:"total"`  // Total number of results. If the status is Running this number may contineu to increase
}

func (c *Client) GetSearchStatus(ctx context.Context, searchID *int) ([]SearchStatus, error) {
	params := url.Values{}
	if searchID != nil {
		params.Set("id", strconv.Itoa(*searchID))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/search/status?"+params.Encode(),
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

	var status []SearchStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return status, nil
}

type SearchResult struct {
	DescrLink  string `json:"descrLink"`  // URL of the torrent's description page
	FileName   string `json:"fileName"`   // Name of the file
	FileSize   int64  `json:"fileSize"`   // Size of the file in Bytes
	FileURL    string `json:"fileUrl"`    // Torrent download link (usually either .torrent file or magnet link)
	NBLeechers int    `json:"nbLeechers"` // Number of leechers
	NBSeeders  int    `json:"nbSeeders"`  // Number of seeders
	SiteURL    string `json:"siteUrl"`    // URL of the torrent site
}

func (c *Client) GetSearchResults(ctx context.Context, searchID int) (string, []SearchResult, error) {
	params := url.Values{}
	params.Set("id", strconv.Itoa(searchID))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/search/results?"+params.Encode(),
		nil,
	)
	if err != nil {
		return "", nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// not found
	if resp.StatusCode == http.StatusNotFound {
		return "", nil, nil
	}

	var respObj struct {
		Results []SearchResult `json:"results"` // Array of result objects- see table below
		Status  string         `json:"status"`  // Current status of the search job (either Running or Stopped)
		Total   int            `json:"total"`   // Total number of results. If the status is Running this number may continue to increase
	}
	if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
		return "", nil, err
	}

	return respObj.Status, respObj.Results, nil
}

func (c *Client) DeleteSearch(ctx context.Context, searchID int) error {
	params := url.Values{}
	params.Set("id", strconv.Itoa(searchID))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/delete",
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

type SearchPluginCategory struct {
	ID   string `json:"id"`   // Category ID
	Name string `json:"name"` // Category name
}

type SearchPlugin struct {
	Enabled             bool                   `json:"enabled"`             // Whether the plugin is enabled
	FullName            string                 `json:"fullName"`            // Full name of the plugin
	Name                string                 `json:"name"`                // Short name of the plugin
	SupportedCategories []SearchPluginCategory `json:"supportedCategories"` // List of category objects
	URL                 string                 `json:"url"`                 // URL of the torrent site
	Version             string                 `json:"version"`             // Installed version of the plugin
}

func (c *Client) GetSearchPlugins(ctx context.Context) ([]SearchPlugin, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2/search/plugins",
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

	var plugins []SearchPlugin
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, err
	}

	return plugins, nil
}

func (c *Client) InstallSearchPlugin(ctx context.Context, sources []string) error {
	params := url.Values{}
	params.Set("sources", strings.Join(sources, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/installPlugin",
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

func (c *Client) UninstallSearchPlugin(ctx context.Context, pluginNames []string) error {
	params := url.Values{}
	params.Set("names", strings.Join(pluginNames, "|"))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/uninstallPlugin?",
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

func (c *Client) ToggleSearchPlugin(ctx context.Context, pluginNames []string, enable bool) error {
	params := url.Values{}
	params.Set("names", strings.Join(pluginNames, "|"))
	params.Set("enable", strconv.FormatBool(enable))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/enablePlugin",
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

func (c *Client) UpdateSearchPlugins(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2/search/updatePlugins",
		nil,
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
