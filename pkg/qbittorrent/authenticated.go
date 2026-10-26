package qbittorrent

import (
	"fmt"
	"net/http"
	"strings"
)

// Only retry after an explicit authentication rejection, never after a timeout:
// a timed-out mutation may already have been applied by qB.
func (c *Client) doAuthenticated(req *http.Request, client *http.Client) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusForbidden || c.configProvider == nil {
		return resp, err
	}
	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}
	resp.Body.Close()
	c.authMu.Lock()
	defer c.authMu.Unlock()
	replay := func() (*http.Response, error) {
		base, current := c.resolve()
		if !strings.HasPrefix(req.URL.String(), strings.TrimRight(base, "/")+"/api/v2/") {
			return nil, fmt.Errorf("qB configuration changed during request")
		}
		next := req.Clone(req.Context())
		if req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, bodyErr
			}
			next.Body = body
		}
		return current.Do(next)
	}
	// Another request may have renewed the shared cookie while this one waited.
	resp, err = replay()
	if err != nil || resp.StatusCode != http.StatusForbidden {
		return resp, err
	}
	resp.Body.Close()
	cfg := c.configProvider()
	if err = c.Login(req.Context(), cfg.Username, cfg.Password); err != nil {
		return nil, err
	}
	return replay()
}
