package indexer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/pkg/flaresolverr"
)

type HTTPSession struct {
	transport       *pinnedTransport
	jar             http.CookieJar
	base            *url.URL
	timeout         time.Duration
	userAgent       string
	trusted         map[string]bool
	solverURL       string
	solverTimeout   time.Duration
	solver          *flaresolverr.Client
	releaseSolver   func()
	solverUserAgent bool
}

func (n *Network) NewSession(instance InstanceSnapshot, certificates []string) (*HTTPSession, error) {
	return n.newSession(instance, certificates, false)
}

// Only an explicitly configured external service gets a private-origin grant.
// A rule, result URL, DNS answer or redirect cannot create or widen this grant.
func (n *Network) newExternalSession(instance InstanceSnapshot) (*HTTPSession, error) {
	return n.newSession(instance, nil, true)
}

func (n *Network) newSession(instance InstanceSnapshot, certificates []string, external bool) (*HTTPSession, error) {
	base, err := parseHTTPURL(instance.BaseURL)
	if err != nil {
		return nil, err
	}
	if !external {
		if _, err := parsePublicURL(instance.BaseURL); err != nil {
			return nil, err
		}
	}
	var proxy *url.URL
	if instance.Network.ProxyURL != "" {
		proxy, err = url.Parse(instance.Network.ProxyURL)
		if err != nil || proxy.Hostname() == "" || proxy.Fragment != "" || proxy.Path != "" && proxy.Path != "/" || proxy.RawQuery != "" {
			return nil, ErrInvalidSettings
		}
		switch proxy.Scheme {
		case "http", "https", "socks5", "socks5h":
		default:
			return nil, ErrInvalidSettings
		}
	}
	jar, err := newPersistentJar()
	if err != nil {
		return nil, ErrInvalidSettings
	}
	timeout := instance.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 120*time.Second {
		timeout = 120 * time.Second
	}
	trusted := map[string]bool{originOf(base): true}
	for _, raw := range instance.TrustedOrigins {
		target, err := parsePublicURL(raw)
		if err != nil {
			return nil, err
		}
		if target.RawQuery != "" || target.ForceQuery || strings.Contains(raw, "#") || target.Fragment != "" || target.Path != "" && target.Path != "/" {
			return nil, ErrInvalidSettings
		}
		trusted[originOf(target)] = true
	}
	session := &HTTPSession{transport: &pinnedTransport{network: n, proxy: proxy, delay: instance.RequestDelay, baseOrigin: originOf(base), certificates: append([]string(nil), certificates...), byOrigin: map[string]*http.Transport{}}, jar: jar, base: base, timeout: timeout, userAgent: "Moji/Indexer", trusted: trusted}
	session.solverURL = instance.Network.FlareSolverrURL
	session.solverTimeout = instance.ChallengeTimeout
	if session.solverTimeout <= 0 {
		session.solverTimeout = 90 * time.Second
	}
	if session.solverTimeout > 300*time.Second {
		session.solverTimeout = 300 * time.Second
	}
	if external {
		session.transport.privateOrigin = originOf(base)
	}
	return session, nil
}

func (s *HTTPSession) SetCookieHeader(origin, header string) error {
	target, err := parsePublicURL(origin)
	if err != nil || originOf(target) != originOf(s.base) || len(header) > 65536 || strings.ContainsAny(header, "\r\n") {
		return ErrInvalidSettings
	}
	request := &http.Request{Header: http.Header{"Cookie": []string{header}}}
	cookies := request.Cookies()
	for _, cookie := range cookies {
		cookie.Path = "/"
		cookie.Secure = target.Scheme == "https"
	}
	s.jar.SetCookies(target, cookies)
	return nil
}

func (s *HTTPSession) doRaw(ctx context.Context, request cardigann.Request, receipt *responseReceipt) (cardigann.HTTPResponse, error) {
	s.transport.mu.Lock()
	closed := s.transport.closed
	s.transport.mu.Unlock()
	if closed {
		return cardigann.HTTPResponse{}, ErrTransport
	}
	target, err := s.parseTarget(request.URL)
	if err != nil {
		return cardigann.HTTPResponse{}, err
	}
	if request.Method != "" && request.Method != http.MethodGet && request.Method != http.MethodPost {
		return cardigann.HTTPResponse{}, ErrUnsupported
	}
	if len(request.Body) > 1<<20 {
		return cardigann.HTTPResponse{}, ErrResponseBudget
	}
	method := request.Method
	if method == "" {
		method = http.MethodGet
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(request.Body))
	if err != nil {
		return cardigann.HTTPResponse{}, ErrUnsafeURL
	}
	httpRequest.Header = request.Headers.Clone()
	if httpRequest.Header == nil {
		httpRequest.Header = http.Header{}
	}
	for key, values := range httpRequest.Header {
		switch strings.ToLower(key) {
		case "host", "connection", "proxy-authorization", "proxy-connection", "transfer-encoding", "content-length", "upgrade":
			return cardigann.HTTPResponse{}, ErrUnsafeURL
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return cardigann.HTTPResponse{}, ErrUnsafeURL
			}
		}
	}
	if s.solverUserAgent || httpRequest.Header.Get("User-Agent") == "" {
		httpRequest.Header.Set("User-Agent", s.userAgent)
	}
	if !s.trusted[originOf(target)] {
		if method == http.MethodPost && len(request.Body) > 0 {
			return cardigann.HTTPResponse{}, ErrUnsafeURL
		}
		stripPrivateHeaders(httpRequest.Header)
	}
	client := &http.Client{Transport: s.transport, Jar: trustedCookieJar{jar: s.jar, origins: s.trusted}, CheckRedirect: func(next *http.Request, previous []*http.Request) error {
		if !request.FollowRedirect || next.URL.Scheme == "magnet" {
			return http.ErrUseLastResponse
		}
		if len(previous) > 5 {
			return ErrUnsafeURL
		}
		if _, err := s.parseTarget(next.URL.String()); err != nil {
			return err
		}
		from := previous[len(previous)-1].URL
		if from.Scheme == "https" && next.URL.Scheme != "https" {
			return ErrUnsafeURL
		}
		if originOf(from) != originOf(next.URL) {
			next.Header.Del("Cookie")
			next.Header.Del("Referer")
			if !s.trusted[originOf(next.URL)] {
				if next.Method == http.MethodPost {
					return ErrUnsafeURL
				}
				stripPrivateHeaders(next.Header)
			}
		}
		return nil
	}}
	httpRequest = httpRequest.WithContext(context.WithValue(httpRequest.Context(), receiptKey{}, receipt))
	response, err := client.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil {
			return cardigann.HTTPResponse{}, ctx.Err()
		}
		if errors.Is(err, context.Canceled) {
			return cardigann.HTTPResponse{}, context.Canceled
		}
		for _, known := range []error{ErrUnsafeURL, ErrCoolingDown, ErrBusy, ErrStateUnavailable, ErrTransport} {
			if errors.Is(err, known) {
				return cardigann.HTTPResponse{}, unwrapNetworkError(err, known)
			}
		}
		return cardigann.HTTPResponse{}, ErrTransport
	}
	defer response.Body.Close()
	limit := int64(cardigann.MaxPageBytes)
	if request.Torrent {
		limit = cardigann.MaxTorrentBytes
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		if ctx.Err() != nil {
			return cardigann.HTTPResponse{}, ctx.Err()
		}
		return cardigann.HTTPResponse{}, ErrTransport
	}
	if int64(len(body)) > limit {
		return cardigann.HTTPResponse{}, ErrResponseBudget
	}
	if receipt != nil {
		receipt.headers = response.Request.Header.Clone()
	}
	out := cardigann.HTTPResponse{URL: response.Request.URL.String(), Status: response.StatusCode, Headers: response.Header.Clone(), Body: body}
	if response.StatusCode == http.StatusTooManyRequests {
		until := time.Now().Add(retryDelay(response.Header.Get("Retry-After"), 1))
		if receipt != nil && !receipt.retryAt.IsZero() {
			until = receipt.retryAt
		}
		return out, &CoolingDownError{Until: until}
	}
	return out, nil
}

func (s *HTTPSession) parseTarget(raw string) (*url.URL, error) {
	target, err := parseHTTPURL(raw)
	if err != nil {
		return nil, err
	}
	if originOf(target) == s.transport.privateOrigin {
		return target, nil
	}
	return parsePublicURL(raw)
}

func (s *HTTPSession) Close() {
	if s.releaseSolver != nil {
		s.releaseSolver()
		s.releaseSolver = nil
	}
	s.transport.mu.Lock()
	s.transport.closed = true
	transports := s.transport.byOrigin
	s.transport.byOrigin = map[string]*http.Transport{}
	s.transport.mu.Unlock()
	for _, transport := range transports {
		transport.CloseIdleConnections()
		s.transport.network.mu.Lock()
		delete(s.transport.network.transports, transport)
		s.transport.network.mu.Unlock()
	}
}

func unwrapNetworkError(err, target error) error {
	var cooling *CoolingDownError
	if errors.As(err, &cooling) {
		return cooling
	}
	return target
}

func stripPrivateHeaders(headers http.Header) {
	for key := range headers {
		switch strings.ToLower(key) {
		case "accept", "accept-language", "accept-encoding", "user-agent", "content-type":
		default:
			headers.Del(key)
		}
	}
}

// Apply the credential boundary at the CookieJar edge too: net/http adds jar
// cookies after CheckRedirect and can otherwise forward a parent-domain cookie
// or accept an untrusted subdomain's replacement for the account cookie.
type trustedCookieJar struct {
	jar     http.CookieJar
	origins map[string]bool
}

func (j trustedCookieJar) Cookies(target *url.URL) []*http.Cookie {
	if !j.origins[originOf(target)] {
		return nil
	}
	return j.jar.Cookies(target)
}
func (j trustedCookieJar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	if j.origins[originOf(target)] {
		j.jar.SetCookies(target, cookies)
	}
}
