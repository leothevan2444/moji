// Package flaresolverr implements the optional v1 protocol. Network policy,
// pacing and endpoint ownership belong to the caller, not this client.
package flaresolverr

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidRequest = errors.New("FLARESOLVERR_INVALID_REQUEST")
	ErrProtocol       = errors.New("FLARESOLVERR_INVALID_RESPONSE")
	ErrRejected       = errors.New("FLARESOLVERR_FAILED")
	ErrBudget         = errors.New("FLARESOLVERR_RESPONSE_TOO_LARGE")
)

const MaxResponseBytes = 32 << 20

// Caller performs one JSON POST to the configured /v1 endpoint. It must honor
// context cancellation and apply a response bound before returning the body.
type Caller interface {
	Call(context.Context, []byte) (int, []byte, error)
}
type CallFunc func(context.Context, []byte) (int, []byte, error)

func (f CallFunc) Call(ctx context.Context, body []byte) (int, []byte, error) { return f(ctx, body) }

type Client struct {
	endpoint Caller
	gate     chan struct{}
}

func New(endpoint Caller) (*Client, error) {
	if endpoint == nil {
		return nil, ErrInvalidRequest
	}
	return &Client{endpoint: endpoint, gate: make(chan struct{}, 1)}, nil
}

type Proxy struct {
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain,omitempty"`
	Path     string  `json:"path,omitempty"`
	Expires  float64 `json:"expires,omitempty"`
	HTTPOnly bool    `json:"httpOnly,omitempty"`
	Secure   bool    `json:"secure,omitempty"`
	SameSite string  `json:"sameSite,omitempty"`
}
type Request struct {
	URL, Method, PostData string
	Cookies               []Cookie
	Proxy                 *Proxy
	Timeout               time.Duration
}
type Solution struct {
	URL       string            `json:"url"`
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Response  string            `json:"response"`
	Cookies   []Cookie          `json:"cookies"`
	UserAgent string            `json:"userAgent"`
}
type command struct {
	Cmd        string   `json:"cmd"`
	URL        string   `json:"url,omitempty"`
	MaxTimeout int64    `json:"maxTimeout,omitempty"`
	PostData   *string  `json:"postData,omitempty"`
	Cookies    []Cookie `json:"cookies,omitempty"`
	Proxy      *Proxy   `json:"proxy,omitempty"`
}
type reply struct {
	Status   string    `json:"status"`
	Solution *Solution `json:"solution"`
}

func (c *Client) call(ctx context.Context, cmd command) (reply, error) {
	if err := ctx.Err(); err != nil {
		return reply{}, err
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return reply{}, ErrInvalidRequest
	}
	status, body, err := c.endpoint.Call(ctx, data)
	if ctx.Err() != nil {
		return reply{}, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, ErrBudget) {
			return reply{}, ErrBudget
		}
		if ctx.Err() != nil {
			return reply{}, ctx.Err()
		}
		return reply{}, ErrProtocol
	}
	if len(body) > MaxResponseBytes {
		return reply{}, ErrBudget
	}
	if status != 200 && status != 500 {
		return reply{}, ErrProtocol
	}
	var result reply
	if err = json.Unmarshal(body, &result); err != nil {
		return reply{}, ErrProtocol
	}
	if result.Status != "ok" {
		return reply{}, ErrRejected
	}
	if status != 200 {
		return reply{}, ErrProtocol
	}
	return result, nil
}
func targetURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(raw, "\r\n")
}

// Solve uses the server's temporary browser. No remote session is created or
// retained by Moji. Cancellation never triggers another solve or POST replay.
func (c *Client) Solve(ctx context.Context, input Request) (solution Solution, resultErr error) {
	if c == nil || c.endpoint == nil {
		return Solution{}, ErrInvalidRequest
	}
	method := input.Method
	if method == "" {
		method = "GET"
	}
	if !targetURL(input.URL) || (method != "GET" && method != "POST") || (method == "GET" && input.PostData != "") || len(input.PostData) > 1<<20 || len(input.Cookies) > 512 || input.Timeout <= 0 || input.Timeout > 300*time.Second {
		return Solution{}, ErrInvalidRequest
	}
	if input.Proxy != nil {
		p, err := url.Parse(input.Proxy.URL)
		if err != nil || p.Hostname() == "" || p.User != nil || p.Fragment != "" || p.RawQuery != "" || (p.Path != "" && p.Path != "/") || (p.Scheme != "http" && p.Scheme != "https" && p.Scheme != "socks4" && p.Scheme != "socks5") {
			return Solution{}, ErrInvalidRequest
		}
	}
	work, cancel := context.WithTimeout(ctx, input.Timeout)
	defer cancel()
	select {
	case c.gate <- struct{}{}:
	case <-work.Done():
		return Solution{}, work.Err()
	}
	defer func() { <-c.gate }()
	cmd := command{Cmd: "request.get", URL: input.URL, Cookies: input.Cookies, Proxy: input.Proxy}
	if method == "POST" {
		cmd.Cmd = "request.post"
		cmd.PostData = &input.PostData
	}
	deadline, _ := work.Deadline()
	remaining := time.Until(deadline).Milliseconds()
	if remaining < 1 {
		if work.Err() != nil {
			return Solution{}, work.Err()
		}
		return Solution{}, context.DeadlineExceeded
	}
	cmd.MaxTimeout = remaining
	result, err := c.call(work, cmd)
	if err != nil {
		return Solution{}, err
	}
	if result.Solution == nil {
		return Solution{}, ErrProtocol
	}
	out := *result.Solution
	if !targetURL(out.URL) || out.Status < 100 || out.Status > 599 || len(out.UserAgent) == 0 || len(out.UserAgent) > 4096 || strings.ContainsAny(out.UserAgent, "\r\n") || len(out.Cookies) > 512 {
		return Solution{}, ErrProtocol
	}
	if len(out.Response) > 16<<20 {
		return Solution{}, ErrBudget
	}
	if err := work.Err(); err != nil {
		return Solution{}, err
	}
	return out, nil
}

func (Request) String() string  { return "FlareSolverr request" }
func (Solution) String() string { return "FlareSolverr solution" }
func (Proxy) String() string    { return "FlareSolverr proxy" }
func (Cookie) String() string   { return "FlareSolverr cookie" }

// FlareSolverr's proxy extension checks for both authentication keys. Preserve
// an empty password (or username) when its counterpart is configured.
func (p Proxy) MarshalJSON() ([]byte, error) {
	if p.Username != "" || p.Password != "" {
		return json.Marshal(struct {
			URL      string `json:"url"`
			Username string `json:"username"`
			Password string `json:"password"`
		}{p.URL, p.Username, p.Password})
	}
	return json.Marshal(struct {
		URL string `json:"url"`
	}{p.URL})
}
