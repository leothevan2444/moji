package indexer

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
	"github.com/leothevan2444/moji/pkg/flaresolverr"
	"golang.org/x/net/publicsuffix"
)

type httpPurpose uint8

const (
	purposeNone httpPurpose = iota
	purposeAuth
	purposeSearch
	purposeFetch
)

type purposeHTTP struct {
	session *HTTPSession
	purpose httpPurpose
}

func (p *purposeHTTP) Do(ctx context.Context, r cardigann.Request) (cardigann.HTTPResponse, error) {
	return p.session.doScoped(ctx, r, p.purpose)
}

type receiptKey struct{}
type responseReceipt struct {
	retryAt  time.Time
	gate     *originGate
	sequence uint64
	status   int
	headers  http.Header
}

func (s *HTTPSession) Do(ctx context.Context, r cardigann.Request) (cardigann.HTTPResponse, error) {
	return s.doScoped(ctx, r, purposeNone)
}
func protectedResponse(r cardigann.HTTPResponse) bool {
	if strings.EqualFold(r.Headers.Get("cf-mitigated"), "challenge") {
		return true
	}
	server := strings.ToLower(r.Headers.Get("Server"))
	if r.Status != 403 && r.Status != 503 {
		return false
	}
	body := strings.ToLower(string(r.Body))
	if server == "ddos-guard" {
		return strings.Contains(body, "<title>ddos-guard</title>")
	}
	if server != "cloudflare" && server != "cloudflare-nginx" {
		return false
	}
	return strings.Contains(body, "/cdn-cgi/challenge-platform") || strings.Contains(body, "cf-chl-") || strings.Contains(body, "<title>just a moment...</title>") || strings.Contains(body, "<title>attention required! | cloudflare</title>")
}
func (s *HTTPSession) doScoped(ctx context.Context, r cardigann.Request, purpose httpPurpose) (cardigann.HTTPResponse, error) {
	receipt := &responseReceipt{}
	out, err := s.doRaw(ctx, r, receipt)
	if err != nil || !protectedResponse(out) {
		return out, err
	}
	if s.solverURL == "" || r.Torrent || (purpose != purposeSearch && purpose != purposeAuth) {
		return out, ErrChallenge
	}
	target, err := s.parseTarget(out.URL)
	if err != nil || !s.trusted[originOf(target)] {
		return out, ErrUnsafeURL
	}
	// An external browser is an explicit trusted service. Validate the site
	// destination before delegation as well as the final reported origin.
	if _, err = s.transport.network.resolve(ctx, target, false); err != nil {
		return out, err
	}
	if s.solver == nil {
		s.solver, s.releaseSolver, err = s.transport.network.newSolverClient(s.solverURL)
		if err != nil {
			return out, err
		}
	}
	cookies := map[string]flaresolverr.Cookie{}
	for _, c := range s.jar.Cookies(target) {
		cookies[c.Name] = flaresolverr.Cookie{Name: c.Name, Value: c.Value}
	}
	for _, c := range (&http.Request{Header: receipt.headers}).Cookies() {
		cookies[c.Name] = flaresolverr.Cookie{Name: c.Name, Value: c.Value}
	}
	request := flaresolverr.Request{URL: target.String(), Timeout: s.solverTimeout}
	for _, cookie := range cookies {
		request.Cookies = append(request.Cookies, cookie)
	}
	if proxy := s.transport.proxy; proxy != nil {
		copy := *proxy
		var username, password string
		if proxy.User != nil {
			username = proxy.User.Username()
			password, _ = proxy.User.Password()
		}
		copy.User = nil
		if copy.Scheme == "socks5h" {
			copy.Scheme = "socks5"
		}
		request.Proxy = &flaresolverr.Proxy{URL: copy.String(), Username: username, Password: password}
	}
	// Solve only a GET challenge navigation; form credentials are never replayed
	// by the browser. The original auth/search request is verified once afterwards.
	solution, err := s.solver.Solve(ctx, request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return out, err
		}
		if errors.Is(err, flaresolverr.ErrBudget) {
			return out, ErrResponseBudget
		}
		return out, ErrSolver
	}
	solved, err := s.parseTarget(solution.URL)
	if err != nil || originOf(solved) != originOf(target) {
		return out, ErrUnsafeURL
	}
	if len(solution.Cookies) == 0 {
		return out, ErrChallenge
	}
	if err = s.adoptSolution(target, solution); err != nil {
		return out, err
	}
	if err = s.clearChallengePenalty(ctx, receipt); err != nil {
		return out, err
	}
	verified, err := s.doRaw(ctx, r, &responseReceipt{})
	if err == nil && protectedResponse(verified) {
		return verified, ErrChallenge
	}
	return verified, err
}
func (s *HTTPSession) clearChallengePenalty(ctx context.Context, receipt *responseReceipt) error {
	if receipt.gate == nil || receipt.status == 429 {
		return nil
	}
	gate := receipt.gate
	select {
	case gate.gate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-gate.gate }()
	if gate.failureSequence == receipt.sequence {
		gate.blocked = time.Time{}
		gate.failures = 0
		return s.transport.network.persistCooldown(gate)
	}
	return nil
}
func (s *HTTPSession) adoptSolution(target *url.URL, solution flaresolverr.Solution) error {
	var cookies []*http.Cookie
	for _, value := range solution.Cookies {
		domain := strings.TrimPrefix(strings.ToLower(value.Domain), ".")
		host := strings.ToLower(target.Hostname())
		if suffix, _ := publicsuffix.PublicSuffix(domain); domain != "" && domain != host && suffix == domain {
			return ErrUnsafeURL
		}
		if domain != "" && domain != host && !strings.HasSuffix(host, "."+domain) {
			return ErrUnsafeURL
		}
		cookie := &http.Cookie{Name: value.Name, Value: value.Value, Domain: value.Domain, Path: value.Path, Secure: value.Secure, HttpOnly: value.HTTPOnly}
		if cookie.Path == "" {
			cookie.Path = "/"
		}
		if value.Expires > 0 {
			if value.Expires > 253402300799 {
				return ErrResponseBudget
			}
			cookie.Expires = time.Unix(int64(value.Expires), 0)
		}
		switch strings.ToLower(value.SameSite) {
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		}
		if err := cookie.Valid(); err != nil {
			return ErrSolver
		}
		cookies = append(cookies, cookie)
	}
	jar, ok := s.jar.(*persistentJar)
	if !ok {
		return ErrSolver
	}
	records, _, err := jar.snapshot()
	if err != nil {
		return err
	}
	preview, err := newPersistentJar()
	if err != nil {
		return err
	}
	if err = preview.restore(records); err != nil {
		return err
	}
	preview.SetCookies(target, cookies)
	if _, _, err = preview.snapshot(); err != nil {
		return err
	}
	jar.SetCookies(target, cookies)
	s.userAgent = solution.UserAgent
	s.solverUserAgent = true
	return nil
}

func (p *purposeHTTP) SetCookieHeader(origin, header string) error {
	return p.session.SetCookieHeader(origin, header)
}
