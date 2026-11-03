package indexer

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ipResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type originGate struct {
	gate                    chan struct{}
	next, blocked, lastUsed time.Time
	failures, refs          int
	failureSequence         uint64
	key                     string
	loaded, dirty           bool
	delay                   time.Duration
}

// Network is shared by all indexer instances. Sessions keep separate cookies,
// while request capacity and origin/proxy pacing have one application owner.
type Network struct {
	ctx               context.Context
	cancel            context.CancelFunc
	mu                sync.Mutex
	closed            bool
	wg                sync.WaitGroup
	queue             chan struct{}
	slots             chan struct{}
	origins           map[string]*originGate
	resolver          ipResolver
	transports        map[*http.Transport]struct{}
	solvers           map[string]*sharedSolverClient
	cooldowns         *cooldownStore
	cooldownError     func(error)
	cooldownWarningAt time.Time
	// Test seam substitutes only the socket transport after URL/DNS checks.
	roundTrip func(*http.Request) (*http.Response, error)
}

func NewNetwork(parent context.Context) *Network {
	ctx, cancel := context.WithCancel(parent)
	return &Network{ctx: ctx, cancel: cancel, queue: make(chan struct{}, 68), slots: make(chan struct{}, 4), origins: map[string]*originGate{}, resolver: net.DefaultResolver, transports: map[*http.Transport]struct{}{}}
}

func (n *Network) Close() {
	n.mu.Lock()
	n.closed = true
	n.cancel()
	solvers := n.solvers
	n.solvers = map[string]*sharedSolverClient{}
	n.mu.Unlock()
	n.wg.Wait()
	for _, entry := range solvers {
		entry.close()
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for transport := range n.transports {
		transport.CloseIdleConnections()
	}
	n.transports = map[*http.Transport]struct{}{}
}

func publicIP(ip net.IP) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	for _, cidr := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "::/96", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2002::/16"} {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

func parseHTTPURL(raw string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil || target.User != nil || target.Opaque != "" || target.Hostname() == "" || target.Fragment != "" || target.Scheme != "http" && target.Scheme != "https" || len(raw) > 16384 {
		return nil, ErrUnsafeURL
	}
	if strings.ContainsAny(target.Hostname(), "%\\\x00\r\n") {
		return nil, ErrUnsafeURL
	}
	if port := target.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return nil, ErrUnsafeURL
		}
	}
	return target, nil
}

func parsePublicURL(raw string) (*url.URL, error) {
	target, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(strings.ToLower(target.Hostname()), ".localhost") || strings.EqualFold(target.Hostname(), "localhost") {
		return nil, ErrUnsafeURL
	}
	return target, nil
}

func originOf(target *url.URL) string {
	port := target.Port()
	if port == "" {
		if target.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return target.Scheme + "://" + net.JoinHostPort(strings.ToLower(target.Hostname()), port)
}

func (n *Network) resolve(ctx context.Context, target *url.URL, allowPrivate bool) (net.IP, error) {
	allowed := func(ip net.IP) bool { return publicIP(ip) || allowPrivate && (ip.IsLoopback() || ip.IsPrivate()) }
	if ip := net.ParseIP(target.Hostname()); ip != nil {
		if !allowed(ip) {
			return nil, ErrUnsafeURL
		}
		return ip, nil
	}
	addresses, err := n.resolver.LookupIPAddr(ctx, target.Hostname())
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrTransport
	}
	if len(addresses) == 0 || len(addresses) > 64 {
		return nil, ErrUnsafeURL
	}
	for _, address := range addresses {
		if address.Zone != "" || !allowed(address.IP) {
			return nil, ErrUnsafeURL
		}
	}
	return addresses[0].IP, nil
}

func (n *Network) admit(ctx context.Context) (context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	n.mu.Lock()
	if err := n.ctx.Err(); err != nil {
		n.mu.Unlock()
		return nil, nil, err
	}
	if n.closed {
		n.mu.Unlock()
		return nil, nil, ErrTransport
	}
	select {
	case n.queue <- struct{}{}:
	default:
		n.mu.Unlock()
		return nil, nil, ErrBusy
	}
	n.wg.Add(1)
	n.mu.Unlock()
	merged, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(n.ctx, cancel)
	return merged, func() { stop(); cancel(); <-n.queue; n.wg.Done() }, nil
}

func (n *Network) pace(ctx context.Context, key string, delay time.Duration) (*originGate, func(), error) {
	n.mu.Lock()
	gate := n.origins[key]
	if gate == nil {
		if len(n.origins) >= 2048 {
			for id, old := range n.origins {
				if old.refs == 0 && time.Since(old.lastUsed) > 30*time.Minute {
					delete(n.origins, id)
				}
			}
		}
		if len(n.origins) >= 2048 {
			n.mu.Unlock()
			return nil, nil, ErrBusy
		}
		gate = &originGate{gate: make(chan struct{}, 1), key: cooldownKey(key)}
		n.origins[key] = gate
	}
	gate.refs++
	gate.lastUsed = time.Now()
	n.mu.Unlock()
	releaseRef := func() { n.mu.Lock(); gate.refs--; gate.lastUsed = time.Now(); n.mu.Unlock() }
	select {
	case gate.gate <- struct{}{}:
	case <-ctx.Done():
		releaseRef()
		return nil, nil, ctx.Err()
	}
	release := func() { <-gate.gate; releaseRef() }
	if err := n.loadCooldown(ctx, gate); err != nil {
		release()
		return nil, nil, err
	}
	if time.Now().Before(gate.blocked) {
		until := gate.blocked
		release()
		return nil, nil, &CoolingDownError{Until: until}
	}
	if gate.dirty {
		if err := n.persistCooldown(gate); err != nil {
			release()
			return nil, nil, err
		}
	}
	gate.delay = max(gate.delay, max(delay, time.Second))
	if wait := time.Until(gate.next); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			release()
			return nil, nil, ctx.Err()
		}
	}
	select {
	case n.slots <- struct{}{}:
	case <-ctx.Done():
		release()
		return nil, nil, ctx.Err()
	}
	return gate, func() { <-n.slots; release() }, nil
}

func retryDelay(header string, failures int) time.Duration {
	delay := min(30*time.Second*time.Duration(1<<min(failures-1, 6)), 30*time.Minute)
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		delay = max(delay, time.Duration(min(seconds, 86400))*time.Second)
	} else if instant, err := http.ParseTime(header); err == nil {
		delay = max(delay, min(time.Until(instant), 24*time.Hour))
	}
	return delay
}

type pinnedTransport struct {
	network       *Network
	proxy         *url.URL
	delay         time.Duration
	baseOrigin    string
	certificates  []string
	mu            sync.Mutex
	byOrigin      map[string]*http.Transport
	closed        bool
	privateOrigin string
	solverControl bool
}

func (p *pinnedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	target, err := parseHTTPURL(request.URL.String())
	if err != nil {
		return nil, err
	}
	allowPrivate := originOf(target) == p.privateOrigin
	if !allowPrivate {
		if _, err := parsePublicURL(target.String()); err != nil {
			return nil, err
		}
	}
	ctx, finish, err := p.network.admit(request.Context())
	if err != nil {
		return nil, err
	}
	defer func() { finish() }()
	proxyKey := "direct"
	if p.proxy != nil {
		proxyKey = p.proxy.Scheme + "://" + p.proxy.Host
	}
	gate, release, err := p.network.pace(ctx, originOf(target)+"|"+proxyKey, p.delay)
	if err != nil {
		return nil, err
	}
	defer func() { release() }()
	ip, err := p.network.resolve(ctx, target, allowPrivate)
	if err != nil {
		return nil, err
	}
	pinned := request.Clone(ctx)
	destination := *request.URL
	pinned.URL = &destination
	port := target.Port()
	if port == "" {
		if target.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	pinned.URL.Host = net.JoinHostPort(ip.String(), port)
	pinned.Host = request.Host
	if pinned.Host == "" {
		pinned.Host = target.Host
	}
	var response *http.Response
	if p.network.roundTrip != nil {
		response, err = p.network.roundTrip(pinned)
	} else {
		transport, transportErr := p.forOrigin(target)
		if transportErr != nil {
			return nil, transportErr
		}
		response, err = transport.RoundTrip(pinned)
	}
	gate.next = time.Now().Add(gate.delay)
	var networkErr net.Error
	timeout := err != nil && ctx.Err() == nil && errors.As(err, &networkErr) && networkErr.Timeout()
	wasFailure := gate.failures > 0
	if response != nil && (response.StatusCode == 429 || response.StatusCode >= 500 && !p.solverControl) || timeout {
		gate.failureSequence++
		gate.failures = min(gate.failures+1, 7)
		retry := ""
		if response != nil {
			retry = response.Header.Get("Retry-After")
		}
		gate.blocked = time.Now().Add(retryDelay(retry, gate.failures))
	} else if err == nil {
		gate.failures = 0
		gate.blocked = time.Time{}
	}
	if gate.failures > 0 || wasFailure {
		_ = p.network.persistCooldown(gate)
	}
	if receipt, _ := request.Context().Value(receiptKey{}).(*responseReceipt); receipt != nil {
		receipt.gate = nil
		receipt.sequence = 0
		receipt.retryAt = time.Time{}
		if response != nil && (response.StatusCode == 429 || response.StatusCode >= 500) {
			receipt.gate = gate
			receipt.sequence = gate.failureSequence
			receipt.status = response.StatusCode
			receipt.retryAt = gate.blocked
		}
	}
	if err != nil {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrTransport
	}
	response.Request = request
	// Keep capacity/pacing ownership until the body is consumed, not merely
	// until headers arrive. The wrapper transfers the two release callbacks.
	var once sync.Once
	body := response.Body
	releaseBody, finishBody := release, finish
	response.Body = &ownedBody{ReadCloser: body, close: func() { once.Do(func() { releaseBody(); finishBody() }) }}
	finish = func() {}
	release = func() {}
	return response, nil
}

type ownedBody struct {
	io.ReadCloser
	close func()
}

func (b *ownedBody) Close() error { err := b.ReadCloser.Close(); b.close(); return err }

func (p *pinnedTransport) forOrigin(target *url.URL) (*http.Transport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, ErrTransport
	}
	key := originOf(target)
	if existing := p.byOrigin[key]; existing != nil {
		return existing, nil
	}
	if len(p.byOrigin) >= 32 {
		return nil, ErrBusy
	}
	configuration := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: target.Hostname()}
	if key == p.baseOrigin && len(p.certificates) > 0 {
		// Custom trust is restricted to the declared site's certificate hashes.
		// Normal chains are still fully verified, including host and expiry.
		pins := append([]string(nil), p.certificates...)
		configuration.InsecureSkipVerify = true
		configuration.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return ErrTransport
			}
			leaf := state.PeerCertificates[0]
			intermediates := x509.NewCertPool()
			for _, certificate := range state.PeerCertificates[1:] {
				intermediates.AddCert(certificate)
			}
			_, err := leaf.Verify(x509.VerifyOptions{DNSName: target.Hostname(), Intermediates: intermediates})
			if err == nil {
				return nil
			}
			shaOne := sha1.Sum(leaf.Raw)
			shaTwo := sha256.Sum256(leaf.Raw)
			for _, pin := range pins {
				if strings.EqualFold(pin, hex.EncodeToString(shaOne[:])) || strings.EqualFold(pin, hex.EncodeToString(shaTwo[:])) {
					return nil
				}
			}
			return ErrTransport
		}
	}
	headerTimeout := 30 * time.Second
	if p.solverControl {
		headerTimeout = 0
	}
	transport := &http.Transport{TLSClientConfig: configuration, MaxIdleConns: 4, MaxIdleConnsPerHost: 2, MaxConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 15 * time.Second, ResponseHeaderTimeout: headerTimeout, MaxResponseHeaderBytes: 1 << 20, ForceAttemptHTTP2: true}
	if p.proxy != nil {
		if p.proxy.Scheme == "https" {
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialTLSProxy(ctx, p.proxy, address)
			}
		} else {
			transport.Proxy = http.ProxyURL(p.proxy)
		}
	}
	p.network.mu.Lock()
	defer p.network.mu.Unlock()
	if p.network.closed {
		return nil, ErrTransport
	}
	p.network.transports[transport] = struct{}{}
	p.byOrigin[key] = transport
	return transport, nil
}
