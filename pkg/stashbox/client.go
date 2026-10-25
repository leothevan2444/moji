// Package stashbox provides a generic Stash-Box GraphQL client. It is not
// specific to JAVStash; the endpoint URL is provided by the caller (typically
// the list of Stash-Box instances configured inside the user's Stash server).
package stashbox

import (
	"context"
	"net/http"

	"github.com/Yamashou/gqlgenc/clientv2"
	"github.com/leothevan2444/moji/pkg/stashbox/graphql"

	"golang.org/x/time/rate"
)

// DefaultMaxRequestsPerMinute is the default maximum number of requests per minute.
const DefaultMaxRequestsPerMinute = 240

// Client represents the client interface to access a Stash-Box instance.
type Client struct {
	endpoint   string
	graphql    *graphql.Client
	httpClient *http.Client

	maxRequestsPerMinute int
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

// NewClient creates a new Stash-Box client bound to the given endpoint. The
// API key is sent as the `ApiKey` header on every request. Pass an empty
// `endpoint` to skip constructing a usable client (the caller can still hold
// onto a placeholder for tests); in practice callers should always supply a
// real URL.
func NewClient(endpoint, apiKey string, opts ...ClientOption) *Client {
	ret := &Client{
		endpoint:             endpoint,
		httpClient:           http.DefaultClient,
		maxRequestsPerMinute: DefaultMaxRequestsPerMinute,
	}
	for _, opt := range opts {
		opt(ret)
	}

	authHeader := setApiKeyHeader(apiKey)
	limitRequests := rateLimit(ret.maxRequestsPerMinute)

	gql := graphql.Client{
		Client: clientv2.NewClient(ret.httpClient, endpoint, nil, authHeader, limitRequests),
	}
	ret.graphql = &gql

	return ret
}

// Endpoint returns the GraphQL endpoint URL the client is bound to.
func (c *Client) Endpoint() string {
	return c.endpoint
}

func (c *Client) Me(ctx context.Context) (*graphql.Me_Me, error) {
	resp, err := c.graphql.Me(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Me, nil
}

func (c *Client) GetVersion(ctx context.Context) (*graphql.GetVersion_Version, error) {
	resp, err := c.graphql.GetVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &resp.Version, nil
}
