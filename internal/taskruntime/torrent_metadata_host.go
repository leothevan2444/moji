package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Jackett can translate an upstream 429 into a local 404. Share pacing and
// cooldown across every task using this host, including manual submissions.
var errTorrentMetadataCoolingDown = errors.New("torrent metadata host cooling down")

type torrentMetadataHost struct {
	gate         chan struct{}
	next         time.Time
	blockedUntil time.Time
	failures     int
}

func (s *Service) acquireTorrentMetadataHost(ctx context.Context, target *url.URL) (func(int, string, error), error) {
	key := strings.ToLower(target.Host)
	if parts := strings.Split(strings.Trim(target.Path, "/"), "/"); len(parts) >= 2 && parts[0] == "dl" {
		key += "/dl/" + parts[1]
	}
	s.metadataHostsMu.Lock()
	if s.metadataHosts == nil {
		s.metadataHosts = make(map[string]*torrentMetadataHost)
	}
	host := s.metadataHosts[key]
	if host == nil {
		host = &torrentMetadataHost{gate: make(chan struct{}, 1)}
		s.metadataHosts[key] = host
	}
	s.metadataHostsMu.Unlock()
	select {
	case host.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-host.gate }
	if time.Now().Before(host.blockedUntil) {
		until := host.blockedUntil
		release()
		return nil, fmt.Errorf("%w: %s until %s", errTorrentMetadataCoolingDown, target.Hostname(), until.UTC().Format(time.RFC3339))
	}
	if delay := time.Until(host.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		}
	}
	return func(status int, retryAfter string, requestErr error) {
		defer release()
		host.next = time.Now().Add(time.Second)
		var networkErr net.Error
		timeout := ctx.Err() == nil && errors.As(requestErr, &networkErr) && networkErr.Timeout()
		jackettFailure := status == http.StatusNotFound && strings.HasPrefix(target.Path, "/dl/")
		if status == http.StatusTooManyRequests || status >= 500 || timeout || jackettFailure {
			host.failures++
			delay := 30 * time.Second * time.Duration(1<<min(host.failures-1, 4))
			if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
				delay = max(delay, time.Duration(min(seconds, 3600))*time.Second)
			} else if at, err := http.ParseTime(retryAfter); err == nil {
				delay = max(delay, min(time.Until(at), time.Hour))
			}
			host.blockedUntil = time.Now().Add(delay)
		} else if requestErr == nil {
			host.failures = 0
		}
	}, nil
}
