package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/internal/torrentmeta"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type torrentIdentity struct {
	InfoHash  string
	MagnetURI string
}

func normalizeCode(value string) string {
	return mediafiles.ExtractCode(strings.TrimSpace(value))
}

func extractCode(values ...string) string {
	for _, value := range values {
		if code := normalizeCode(value); code != "" {
			return code
		}
	}
	return ""
}

func normalizeInfoHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value)
}

func normalizeMagnetURI(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(strings.ToLower(value), "magnet:") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	infoHash := ""
	for _, xt := range parsed.Query()["xt"] {
		parts := strings.Split(xt, ":")
		if len(parts) >= 3 && strings.EqualFold(parts[0], "urn") && strings.EqualFold(parts[1], "btih") {
			infoHash = normalizeInfoHash(parts[2])
			break
		}
	}
	if infoHash == "" {
		return ""
	}
	return "magnet:?xt=urn:btih:" + infoHash
}

func magnetDisplayName(value string) string {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "magnet:") {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(firstNonEmpty(parsed.Query()["dn"]))
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func torrentIdentityFromCandidate(candidate Candidate, torrentURL string) torrentIdentity {
	infoHash := normalizeInfoHash(candidate.InfoHash)
	if infoHash == "" {
		infoHash = normalizeInfoHash(infoHashFromMagnet(torrentURL))
	}
	magnet := normalizeMagnetURI(candidate.MagnetURI)
	if magnet == "" {
		magnet = normalizeMagnetURI(torrentURL)
	}
	return torrentIdentity{
		InfoHash:  infoHash,
		MagnetURI: magnet,
	}
}

func (s *Service) ensureTaskCanBeCreated(ctx context.Context, identity torrentIdentity, code string) error {
	code = normalizeCode(code)
	if code == "" {
		return ErrTaskCodeRequired
	}
	if existing, err := s.store.FindByTorrentIdentity(ctx, identity.InfoHash, identity.MagnetURI); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("%w: task %q already uses the same torrent", ErrDuplicateTorrentTask, existing.ID)
	}
	if existing, err := s.store.FindByCode(ctx, code); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("%w: task %q already uses code %s", ErrDuplicateCodeTask, existing.ID, code)
	}
	if s.libraryCodeChecker != nil {
		exists, err := s.libraryCodeChecker.HasCode(ctx, code)
		if err != nil {
			return fmt.Errorf("check stash library code %s: %w", code, err)
		}
		if exists {
			return fmt.Errorf("%w: stash library already contains code %s", ErrDuplicateLibraryCode, code)
		}
	}
	return nil
}

func (s *Service) ensureTaskCodeCanBeCreated(ctx context.Context, code string) error {
	code = normalizeCode(code)
	if code == "" {
		return ErrTaskCodeRequired
	}
	if existing, err := s.store.FindByCode(ctx, code); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("%w: task %q already uses code %s", ErrDuplicateCodeTask, existing.ID, code)
	}
	if s.libraryCodeChecker != nil {
		exists, err := s.libraryCodeChecker.HasCode(ctx, code)
		if err != nil {
			return fmt.Errorf("check stash library code %s: %w", code, err)
		}
		if exists {
			return fmt.Errorf("%w: stash library already contains code %s", ErrDuplicateLibraryCode, code)
		}
	}
	return nil
}

func (s *Service) ensureTaskIdentityAvailable(ctx context.Context, taskID string, identity torrentIdentity) error {
	if existing, err := s.store.FindByTorrentIdentity(ctx, identity.InfoHash, identity.MagnetURI); err != nil {
		return err
	} else if existing != nil && existing.ID != strings.TrimSpace(taskID) {
		return fmt.Errorf("%w: task %q already uses the same torrent", ErrDuplicateTorrentTask, existing.ID)
	}
	return nil
}

type downloadedTorrentMetadata = torrentmeta.Metadata

const MaxTorrentBytes = 8 << 20

func (s *Service) resolveManualTorrent(ctx context.Context, torrentURL string) (Candidate, string, torrentIdentity, *downloadedTorrentMetadata, error) {
	candidate := candidateFromTorrentURL(torrentURL)
	titleCandidates := []string{magnetDisplayName(torrentURL)}
	urlCandidates := []string{torrentURL}

	parsedURL, err := url.Parse(torrentURL)
	if err == nil {
		base := path.Base(strings.TrimSpace(parsedURL.Path))
		if base != "" && base != "." && base != "/" {
			urlCandidates = append(urlCandidates, base)
			titleCandidates = append(titleCandidates, base)
		}
	}

	if candidate.MagnetURI != "" {
		if displayName := firstNonEmpty(titleCandidates); displayName != "" {
			candidate.Title = displayName
		}
		code := extractCode(append(titleCandidates, urlCandidates...)...)
		identity := torrentIdentityFromCandidate(candidate, torrentURL)
		return candidate, code, identity, nil, nil
	}

	metadata, err := s.fetchTorrentMetadata(ctx, torrentURL)
	if err != nil {
		return Candidate{}, "", torrentIdentity{}, nil, err
	}
	if metadata.Name != "" {
		candidate.Title = metadata.Name
		titleCandidates = append([]string{metadata.Name}, titleCandidates...)
	}
	if metadata.FilePath != "" {
		titleCandidates = append(titleCandidates, metadata.FilePath)
	}
	if metadata.InfoHash != "" {
		candidate.InfoHash = metadata.InfoHash
	}
	code := extractCode(titleCandidates...)
	identity := torrentIdentityFromCandidate(candidate, torrentURL)
	return candidate, code, identity, &metadata, nil
}

func (s *Service) fetchTorrentMetadata(ctx context.Context, torrentURL string) (metadata downloadedTorrentMetadata, resultErr error) {
	if s.httpClient == nil {
		return downloadedTorrentMetadata{}, errors.New("taskruntime: http client is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: build torrent metadata request: %w", err)
	}
	finish, err := s.acquireTorrentMetadataHost(ctx, req.URL)
	if err != nil {
		return downloadedTorrentMetadata{}, err
	}
	status := 0
	retryAfter := ""
	defer func() { finish(status, retryAfter, resultErr) }()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			err = urlErr.Err
		}
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: fetch torrent metadata from %s: %w", req.URL.Hostname(), err)
	}
	defer resp.Body.Close()
	status, retryAfter = resp.StatusCode, resp.Header.Get("Retry-After")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: fetch torrent metadata: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxTorrentBytes+1))
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: read torrent metadata: %w", err)
	}
	if len(body) > MaxTorrentBytes {
		return downloadedTorrentMetadata{}, errors.New("taskruntime: torrent metadata exceeds 8 MiB")
	}
	return parseTorrentMetadata(body)
}

func parseTorrentMetadata(data []byte) (downloadedTorrentMetadata, error) {
	return torrentmeta.Parse(data)
}
