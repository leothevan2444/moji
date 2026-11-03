package indexer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"

	"github.com/leothevan2444/moji/pkg/flaresolverr"
)

// newSolverClient grants private access only to the explicitly configured
// service origin. Site proxy credentials/cookies travel in the protocol body,
// never as HTTP authentication for the solver endpoint itself.
type sharedSolverClient struct {
	client *flaresolverr.Client
	close  func()
	refs   int
}

func (n *Network) releaseSolver(raw string, entry *sharedSolverClient) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			n.mu.Lock()
			entry.refs--
			last := entry.refs == 0
			if last && n.solvers[raw] == entry {
				delete(n.solvers, raw)
			}
			n.mu.Unlock()
			if last {
				entry.close()
			}
		})
	}
}
func (n *Network) newSolverClient(raw string) (*flaresolverr.Client, func(), error) {
	parsed, err := parseHTTPURL(raw)
	if err != nil {
		return nil, nil, err
	}
	digest := sha256.Sum256([]byte(parsed.JoinPath("v1").String()))
	key := hex.EncodeToString(digest[:])
	n.mu.Lock()
	if n.closed || n.ctx.Err() != nil {
		n.mu.Unlock()
		return nil, nil, context.Canceled
	}
	if existing := n.solvers[key]; existing != nil {
		existing.refs++
		n.mu.Unlock()
		return existing.client, n.releaseSolver(key, existing), nil
	}
	if len(n.solvers) >= 16 {
		n.mu.Unlock()
		return nil, nil, ErrBusy
	}
	n.mu.Unlock()
	client, closeClient, err := n.buildSolverClient(raw)
	if err != nil {
		return nil, nil, err
	}
	n.mu.Lock()
	if n.closed || n.ctx.Err() != nil {
		n.mu.Unlock()
		closeClient()
		return nil, nil, context.Canceled
	}
	if existing := n.solvers[key]; existing != nil {
		existing.refs++
		n.mu.Unlock()
		closeClient()
		return existing.client, n.releaseSolver(key, existing), nil
	}
	if len(n.solvers) >= 16 {
		n.mu.Unlock()
		closeClient()
		return nil, nil, ErrBusy
	}
	entry := &sharedSolverClient{client: client, close: closeClient, refs: 1}
	if n.solvers == nil {
		n.solvers = map[string]*sharedSolverClient{}
	}
	n.solvers[key] = entry
	n.mu.Unlock()
	return client, n.releaseSolver(key, entry), nil
}
func (n *Network) buildSolverClient(raw string) (*flaresolverr.Client, func(), error) {
	target, err := parseHTTPURL(raw)
	if err != nil {
		return nil, nil, err
	}
	target = target.JoinPath("v1")
	session, err := n.newExternalSession(InstanceSnapshot{BaseURL: raw})
	if err != nil {
		return nil, nil, err
	}
	// HTTP 500 is a documented protocol rejection, not a network cooldown. The
	// parent solve deadline bounds header wait; the usual 30s header limit would
	// incorrectly preempt the configured browser challenge timeout.
	session.transport.solverControl = true
	client := &http.Client{Transport: session.transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	caller := flaresolverr.CallFunc(func(ctx context.Context, body []byte) (int, []byte, error) {
		session.transport.mu.Lock()
		closed := session.transport.closed
		session.transport.mu.Unlock()
		if closed {
			return 0, nil, ErrTransport
		}
		if len(body) > 8<<20 {
			return 0, nil, flaresolverr.ErrBudget
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
		if err != nil {
			return 0, nil, ErrUnsafeURL
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return 0, nil, err
		}
		defer response.Body.Close()
		data, err := io.ReadAll(io.LimitReader(response.Body, flaresolverr.MaxResponseBytes+1))
		if err != nil {
			return 0, nil, err
		}
		if len(data) > flaresolverr.MaxResponseBytes {
			return 0, nil, flaresolverr.ErrBudget
		}
		return response.StatusCode, data, nil
	})
	protocol, err := flaresolverr.New(caller)
	if err != nil {
		session.Close()
		return nil, nil, err
	}
	return protocol, session.Close, nil
}
