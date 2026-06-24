package stash

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Yamashou/gqlgenc/clientv2"
	"github.com/leothevan2444/moji/pkg/stash/graphql"

	"golang.org/x/time/rate"
)

// DefaultMaxRequestsPerMinute is the default maximum number of requests per minute.
const DefaultMaxRequestsPerMinute = 240

// Config captures the runtime Stash connection fields. Mirroring
// config.Config.Stash as a typed struct lets callers hand a provider to
// NewClient instead of capturing values at startup.
type Config struct {
	URL    string
	APIKey string
}

// ConfigProvider supplies the latest Stash Config at the moment of each
// GraphQL request. Reading the config lazily means Web UI edits to
// stash.url / api_key take effect on the next call without restarting Moji.
type ConfigProvider func() Config

// Client represents the client interface to access the Stash server instance
type Client struct {
	configProvider ConfigProvider

	// mu guards graphql + httpClient. The pair is rebuilt only when the
	// provider's identity (URL or API key) changes.
	mu         sync.RWMutex
	graphql    *graphql.Client
	httpClient *http.Client

	maxRequestsPerMinute int
	lastConfig          Config
}

type ClientOption func(*Client)

func MaxRequestsPerMinute(n int) ClientOption {
	return func(c *Client) {
		if n > 0 {
			c.maxRequestsPerMinute = n
		}
	}
}

func setApiKeyHeader(apiKey string) clientv2.RequestInterceptor {
	return func(ctx context.Context, req *http.Request, gqlInfo *clientv2.GQLRequestInfo, res interface{}, next clientv2.RequestInterceptorFunc) error {
		req.Header.Set("ApiKey", apiKey)
		return next(ctx, req, gqlInfo, res)
	}
}

func rateLimit(n int) clientv2.RequestInterceptor {
	perSec := float64(n) / 60
	limiter := rate.NewLimiter(rate.Limit(perSec), 1)

	return func(ctx context.Context, req *http.Request, gqlInfo *clientv2.GQLRequestInfo, res interface{}, next clientv2.RequestInterceptorFunc) error {
		if err := limiter.Wait(ctx); err != nil {
			// should only happen if the context is canceled
			return err
		}

		return next(ctx, req, gqlInfo, res)
	}
}

// NewClient creates a new Stash client backed by a config provider. The
// provider is consulted on every GraphQL call so Web UI edits take effect
// without restarting the process.
func NewClient(configProvider ConfigProvider) *Client {
	ret := &Client{
		configProvider:       configProvider,
		maxRequestsPerMinute: DefaultMaxRequestsPerMinute,
	}
	if configProvider != nil {
		cfg := configProvider()
		ret.lastConfig = cfg
		ret.rebuildLocked(cfg)
	} else {
		ret.httpClient = &http.Client{
			Transport: &http.Transport{Proxy: nil},
		}
	}
	return ret
}

// resolve returns the graphql client matching the latest provider config,
// rebuilding it only when URL or APIKey change. Safe for concurrent use.
func (c *Client) resolve() *graphql.Client {
	if c.configProvider == nil {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.graphql
	}
	cfg := c.configProvider()
	c.mu.RLock()
	if cfg == c.lastConfig && c.graphql != nil {
		g := c.graphql
		c.mu.RUnlock()
		return g
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if cfg == c.lastConfig && c.graphql != nil {
		return c.graphql
	}
	c.lastConfig = cfg
	c.rebuildLocked(cfg)
	return c.graphql
}

// rebuildLocked rebuilds the http and graphql clients for cfg. Caller must
// hold c.mu for writing.
func (c *Client) rebuildLocked(cfg Config) {
	c.httpClient = &http.Client{
		Transport: &http.Transport{Proxy: nil},
	}
	authHeader := setApiKeyHeader(cfg.APIKey)
	limitRequests := rateLimit(c.maxRequestsPerMinute)
	gql := graphql.Client{
		Client: clientv2.NewClient(c.httpClient, cfg.URL, nil, authHeader, limitRequests),
	}
	c.graphql = &gql
}

func (c *Client) GetVersion(ctx context.Context) (*graphql.GetVersion_Version, error) {
	resp, err := c.resolve().GetVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &resp.Version, nil
}

// GetSceneCount returns the total number of scenes in the Stash library.
// It is used by the stats collector for the home-page service card.
func (c *Client) GetSceneCount(ctx context.Context) (int, error) {
	resp, err := c.resolve().FindSceneCount(ctx)
	if err != nil {
		return 0, err
	}
	inner := resp.GetFindScenes()
	if inner == nil {
		return 0, nil
	}
	return inner.GetCount(), nil
}

// StashBoxEndpoint describes a Stash-Box instance configured in the Stash server.
type StashBoxEndpoint struct {
	Name                 string
	Endpoint             string
	APIKey               string
	MaxRequestsPerMinute int
}

type StashLibrary struct {
	Path string
}

// GetStashBoxes queries the Stash server for the list of configured Stash-Box
// endpoints. The API key is never logged or returned to callers outside this
// package.
func (c *Client) GetStashBoxes(ctx context.Context) ([]StashBoxEndpoint, error) {
	resp, err := c.resolve().Configuration(ctx)
	if err != nil {
		return nil, fmt.Errorf("stash: query configuration: %w", err)
	}

	cfg := resp.GetConfiguration()
	if cfg == nil {
		return nil, nil
	}
	general := cfg.GetGeneral()
	if general == nil {
		return nil, nil
	}

	boxes := general.GetStashBoxes()
	out := make([]StashBoxEndpoint, 0, len(boxes))
	for _, box := range boxes {
		if box == nil {
			continue
		}
		endpoint := box.GetEndpoint()
		if endpoint == "" {
			continue
		}
		out = append(out, StashBoxEndpoint{
			Name:                 box.GetName(),
			Endpoint:             endpoint,
			APIKey:               box.GetAPIKey(),
			MaxRequestsPerMinute: box.GetMaxRequestsPerMinute(),
		})
	}
	return out, nil
}

func (c *Client) GetStashLibraries(ctx context.Context) ([]StashLibrary, error) {
	resp, err := c.resolve().Configuration(ctx)
	if err != nil {
		return nil, fmt.Errorf("stash: query configuration: %w", err)
	}

	cfg := resp.GetConfiguration()
	if cfg == nil {
		return nil, nil
	}
	general := cfg.GetGeneral()
	if general == nil {
		return nil, nil
	}

	stashes := general.GetStashes()
	out := make([]StashLibrary, 0, len(stashes))
	for _, stashConfig := range stashes {
		if stashConfig == nil || strings.TrimSpace(stashConfig.Path) == "" {
			continue
		}
		out = append(out, StashLibrary{Path: stashConfig.Path})
	}
	return out, nil
}
