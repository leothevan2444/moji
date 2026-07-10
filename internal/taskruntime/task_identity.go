package taskruntime

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var codePattern = regexp.MustCompile(`(?i)\b([a-z]{2,10})[-_\s]?(\d{2,6})\b`)

type torrentIdentity struct {
	InfoHash  string
	MagnetURI string
}

func normalizeCode(value string) string {
	match := codePattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) != 3 {
		return ""
	}
	return strings.ToUpper(match[1]) + "-" + match[2]
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

type downloadedTorrentMetadata struct {
	Name     string
	FilePath string
	InfoHash string
	Paths    []string
}

func (s *Service) resolveManualTorrent(ctx context.Context, torrentURL string) (Candidate, string, torrentIdentity, error) {
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
		return candidate, code, identity, nil
	}

	metadata, err := s.fetchTorrentMetadata(ctx, torrentURL)
	if err != nil {
		return Candidate{}, "", torrentIdentity{}, err
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
	return candidate, code, identity, nil
}

func (s *Service) fetchTorrentMetadata(ctx context.Context, torrentURL string) (downloadedTorrentMetadata, error) {
	if s.httpClient == nil {
		return downloadedTorrentMetadata{}, errors.New("taskruntime: http client is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: build torrent metadata request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: fetch torrent metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: fetch torrent metadata: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: read torrent metadata: %w", err)
	}
	return parseTorrentMetadata(body)
}

func parseTorrentMetadata(data []byte) (downloadedTorrentMetadata, error) {
	parser := &bencodeParser{data: data}
	root, infoBytes, err := parser.parseMetainfo()
	if err != nil {
		return downloadedTorrentMetadata{}, fmt.Errorf("taskruntime: parse .torrent metadata: %w", err)
	}
	metadata := downloadedTorrentMetadata{
		Name:     root.name,
		FilePath: root.firstPath,
		Paths:    append([]string(nil), root.paths...),
	}
	if len(metadata.Paths) == 0 && metadata.Name != "" {
		metadata.Paths = []string{metadata.Name}
	}
	if metadata.FilePath == "" && len(metadata.Paths) > 0 {
		metadata.FilePath = metadata.Paths[0]
	}
	if len(infoBytes) > 0 {
		hash := sha1.Sum(infoBytes)
		metadata.InfoHash = strings.ToUpper(fmt.Sprintf("%x", hash[:]))
	}
	return metadata, nil
}

type torrentMetainfo struct {
	name      string
	firstPath string
	paths     []string
}

type bencodeParser struct {
	data []byte
	pos  int
}

func (p *bencodeParser) parseMetainfo() (torrentMetainfo, []byte, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return torrentMetainfo{}, nil, errors.New("root value is not a dictionary")
	}
	p.pos++
	var meta torrentMetainfo
	var infoBytes []byte
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return torrentMetainfo{}, nil, err
		}
		valueStart := p.pos
		switch key {
		case "info":
			infoMeta, err := p.parseInfoDict()
			if err != nil {
				return torrentMetainfo{}, nil, err
			}
			infoBytes = append([]byte(nil), p.data[valueStart:p.pos]...)
			meta = infoMeta
		default:
			if err := p.skipValue(); err != nil {
				return torrentMetainfo{}, nil, err
			}
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return torrentMetainfo{}, nil, errors.New("unterminated root dictionary")
	}
	p.pos++
	return meta, infoBytes, nil
}

func (p *bencodeParser) parseInfoDict() (torrentMetainfo, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return torrentMetainfo{}, errors.New("info is not a dictionary")
	}
	p.pos++
	var meta torrentMetainfo
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return torrentMetainfo{}, err
		}
		switch key {
		case "name":
			value, err := p.parseString()
			if err != nil {
				return torrentMetainfo{}, err
			}
			meta.name = value
		case "files":
			paths, err := p.parseFiles()
			if err != nil {
				return torrentMetainfo{}, err
			}
			meta.paths = paths
			if len(paths) > 0 {
				meta.firstPath = paths[0]
			}
		default:
			if err := p.skipValue(); err != nil {
				return torrentMetainfo{}, err
			}
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return torrentMetainfo{}, errors.New("unterminated info dictionary")
	}
	p.pos++
	return meta, nil
}

func (p *bencodeParser) parseFiles() ([]string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'l' {
		return nil, errors.New("files is not a list")
	}
	p.pos++
	paths := make([]string, 0)
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		pathValue, err := p.parseFileEntry()
		if err != nil {
			return nil, err
		}
		if pathValue != "" {
			paths = append(paths, pathValue)
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return nil, errors.New("unterminated files list")
	}
	p.pos++
	return paths, nil
}

func (p *bencodeParser) parseFileEntry() (string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'd' {
		return "", errors.New("file entry is not a dictionary")
	}
	p.pos++
	firstPath := ""
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		key, err := p.parseString()
		if err != nil {
			return "", err
		}
		if key == "path" {
			pathSegments, err := p.parseStringList()
			if err != nil {
				return "", err
			}
			firstPath = strings.Join(pathSegments, "/")
			continue
		}
		if err := p.skipValue(); err != nil {
			return "", err
		}
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return "", errors.New("unterminated file entry")
	}
	p.pos++
	return firstPath, nil
}

func (p *bencodeParser) parseStringList() ([]string, error) {
	if p.pos >= len(p.data) || p.data[p.pos] != 'l' {
		return nil, errors.New("path is not a list")
	}
	p.pos++
	var items []string
	for p.pos < len(p.data) && p.data[p.pos] != 'e' {
		item, err := p.parseString()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if p.pos >= len(p.data) || p.data[p.pos] != 'e' {
		return nil, errors.New("unterminated string list")
	}
	p.pos++
	return items, nil
}

func (p *bencodeParser) parseString() (string, error) {
	lengthStart := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != ':' {
		if p.data[p.pos] < '0' || p.data[p.pos] > '9' {
			return "", errors.New("invalid string length")
		}
		p.pos++
	}
	if p.pos >= len(p.data) || p.data[p.pos] != ':' {
		return "", errors.New("unterminated string length")
	}
	lengthValue := 0
	for _, digit := range p.data[lengthStart:p.pos] {
		lengthValue = lengthValue*10 + int(digit-'0')
	}
	p.pos++
	if p.pos+lengthValue > len(p.data) {
		return "", errors.New("string exceeds buffer")
	}
	value := string(p.data[p.pos : p.pos+lengthValue])
	p.pos += lengthValue
	return value, nil
}

func (p *bencodeParser) skipValue() error {
	if p.pos >= len(p.data) {
		return io.ErrUnexpectedEOF
	}
	switch p.data[p.pos] {
	case 'd':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			if _, err := p.parseString(); err != nil {
				return err
			}
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated dictionary")
		}
		p.pos++
		return nil
	case 'l':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			if err := p.skipValue(); err != nil {
				return err
			}
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated list")
		}
		p.pos++
		return nil
	case 'i':
		p.pos++
		for p.pos < len(p.data) && p.data[p.pos] != 'e' {
			p.pos++
		}
		if p.pos >= len(p.data) {
			return errors.New("unterminated integer")
		}
		p.pos++
		return nil
	default:
		_, err := p.parseString()
		return err
	}
}
