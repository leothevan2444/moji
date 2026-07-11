package imagecache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	DefaultMaxSizeMB     = 1024
	DefaultRetentionDays = 30
	maxImageBytes        = 10 << 20
)

type SourceKind string

const (
	SourceStash    SourceKind = "stash"
	SourceStashBox SourceKind = "stashbox"
)

type Descriptor struct {
	Kind        SourceKind
	InstanceURL string
	ImageURL    string
	APIKey      string
}

type Config struct {
	Enabled       bool
	MaxSizeMB     int
	RetentionDays int
}

func DefaultConfig() Config {
	return Config{Enabled: true, MaxSizeMB: DefaultMaxSizeMB, RetentionDays: DefaultRetentionDays}
}

type ConfigProvider func() Config

type Status struct {
	UsedBytes      int64
	EntryCount     int
	CacheDirectory string
	LastCleanupAt  *time.Time
	LastError      string
}

type registration struct {
	Kind        SourceKind `json:"kind"`
	InstanceURL string     `json:"instanceUrl"`
	OriginURL   string     `json:"originUrl"`
}

type imageResponse struct {
	Path        string
	ContentType string
	Data        []byte
}

type Service struct {
	dir              string
	registrationsDir string
	objectsDir       string
	config           ConfigProvider
	client           *http.Client
	group            singleflight.Group
	credentialMu     sync.RWMutex
	credentials      map[string]string
	statusMu         sync.RWMutex
	lastCleanupAt    *time.Time
	lastError        string
	now              func() time.Time
}

type requestSourceKey struct{}
type requestSource struct {
	Kind     SourceKind
	Instance string
}

func New(cacheDir string, provider ConfigProvider) (*Service, error) {
	if provider == nil {
		provider = func() Config { return DefaultConfig() }
	}
	s := &Service{
		dir:              cacheDir,
		registrationsDir: filepath.Join(cacheDir, "registrations"),
		objectsDir:       filepath.Join(cacheDir, "objects"),
		config:           provider,
		credentials:      map[string]string{},
		now:              time.Now,
	}
	for _, dir := range []string{s.registrationsDir, s.objectsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("imagecache: create cache directory: %w", err)
		}
	}
	s.client = &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: s.checkRedirect,
		Transport:     &http.Transport{Proxy: nil, DialContext: s.safeDialContext},
	}
	return s, nil
}

func (s *Service) Close() error { return nil }

func normalizeConfig(c Config) Config {
	if c.MaxSizeMB == 0 {
		c.MaxSizeMB = DefaultMaxSizeMB
	}
	if c.RetentionDays == 0 {
		c.RetentionDays = DefaultRetentionDays
	}
	return c
}

func (s *Service) Register(ctx context.Context, d Descriptor) (string, error) {
	resolved, instance, err := resolveDescriptor(d)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(string(d.Kind) + "\n" + instance + "\n" + resolved))
	key := hex.EncodeToString(sum[:])
	reg := registration{Kind: d.Kind, InstanceURL: instance, OriginURL: resolved}
	data, err := json.Marshal(reg)
	if err != nil {
		return "", fmt.Errorf("imagecache: encode registration: %w", err)
	}
	registrationPath := s.registrationPath(key)
	current, readErr := os.ReadFile(registrationPath)
	if readErr != nil || string(current) != string(data) {
		if err := writeAtomic(registrationPath, data, 0o600); err != nil {
			return "", fmt.Errorf("imagecache: register: %w", err)
		}
	}
	if strings.TrimSpace(d.APIKey) != "" {
		s.credentialMu.Lock()
		s.credentials[credentialKey(d.Kind, instance)] = d.APIKey
		s.credentialMu.Unlock()
	}
	return "/api/images/" + key, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func resolveDescriptor(d Descriptor) (string, string, error) {
	base, err := url.Parse(strings.TrimSpace(d.InstanceURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", "", errors.New("imagecache: invalid instance URL")
	}
	raw, err := url.Parse(strings.TrimSpace(d.ImageURL))
	if err != nil {
		return "", "", errors.New("imagecache: invalid image URL")
	}
	resolved := base.ResolveReference(raw)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", "", errors.New("imagecache: unsupported image URL scheme")
	}
	base.Path, base.RawQuery, base.Fragment = "", "", ""
	return resolved.String(), base.String(), nil
}

func credentialKey(kind SourceKind, instance string) string { return string(kind) + "\n" + instance }

func (s *Service) registrationPath(key string) string {
	return filepath.Join(s.registrationsDir, key+".json")
}

func (s *Service) RegisterHandler(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/images/{key}", s.ServeHTTP)
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !validKey(key) {
		http.NotFound(w, r)
		return
	}
	v, err, _ := s.group.Do(key, func() (any, error) { return s.load(r.Context(), key) })
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		s.setError(err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	if err := serveImage(w, r, key, v.(imageResponse)); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func validKey(key string) bool {
	if len(key) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(key)
	return err == nil
}

func (s *Service) load(ctx context.Context, key string) (imageResponse, error) {
	if path, contentType, ok := s.findObject(key); ok {
		return imageResponse{Path: path, ContentType: contentType}, nil
	}
	reg, err := s.readRegistration(key)
	if err != nil {
		return imageResponse{}, err
	}
	result, err := s.fetch(ctx, key, reg)
	if err == nil {
		s.setError(nil)
	}
	return result, err
}

func (s *Service) readRegistration(key string) (registration, error) {
	data, err := os.ReadFile(s.registrationPath(key))
	if err != nil {
		return registration{}, err
	}
	var reg registration
	if err := json.Unmarshal(data, &reg); err != nil {
		return registration{}, fmt.Errorf("imagecache: decode registration: %w", err)
	}
	if reg.Kind != SourceStash && reg.Kind != SourceStashBox {
		return registration{}, errors.New("imagecache: invalid registered source")
	}
	return reg, nil
}

func (s *Service) findObject(key string) (string, string, bool) {
	for contentType, ext := range contentTypeExtensions {
		path := filepath.Join(s.objectsDir, key+ext)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path, contentType, true
		}
	}
	return "", "", false
}

func (s *Service) fetch(ctx context.Context, key string, reg registration) (imageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reg.OriginURL, nil)
	if err != nil {
		return imageResponse{}, err
	}
	source := requestSource{Kind: reg.Kind, Instance: reg.InstanceURL}
	if err := s.validateTargetURL(ctx, req.URL, source); err != nil {
		return imageResponse{}, err
	}
	req = req.WithContext(context.WithValue(req.Context(), requestSourceKey{}, source))
	if sameOrigin(reg.OriginURL, reg.InstanceURL) {
		s.credentialMu.RLock()
		apiKey := s.credentials[credentialKey(reg.Kind, reg.InstanceURL)]
		s.credentialMu.RUnlock()
		if apiKey != "" {
			req.Header.Set("ApiKey", apiKey)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return imageResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return imageResponse{}, fmt.Errorf("imagecache: upstream status %d", resp.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if !allowedContentType(contentType) {
		return imageResponse{}, fmt.Errorf("imagecache: rejected content type %q", contentType)
	}
	tmp, err := os.CreateTemp(s.objectsDir, ".tmp-*")
	if err != nil {
		return imageResponse{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	n, copyErr := io.Copy(tmp, io.LimitReader(resp.Body, maxImageBytes+1))
	if copyErr != nil {
		_ = tmp.Close()
		return imageResponse{}, copyErr
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return imageResponse{}, err
	}
	if err := tmp.Close(); err != nil {
		return imageResponse{}, err
	}
	if n > maxImageBytes {
		return imageResponse{}, errors.New("imagecache: image exceeds 10 MB")
	}
	if !normalizeConfig(s.config()).Enabled {
		data, err := os.ReadFile(tmpName)
		return imageResponse{ContentType: contentType, Data: data}, err
	}
	path := filepath.Join(s.objectsDir, key+extensionFor(contentType))
	if err := os.Rename(tmpName, path); err != nil {
		return imageResponse{}, err
	}
	return imageResponse{Path: path, ContentType: contentType}, nil
}

var contentTypeExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/avif": ".avif",
}

func allowedContentType(value string) bool {
	_, ok := contentTypeExtensions[value]
	return ok
}

func extensionFor(contentType string) string { return contentTypeExtensions[contentType] }

func sameOrigin(a, b string) bool {
	ua, _ := url.Parse(a)
	ub, _ := url.Parse(b)
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

func serveImage(w http.ResponseWriter, r *http.Request, key string, result imageResponse) error {
	if result.Data != nil {
		w.Header().Set("Content-Type", result.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(result.Data)))
		w.Header().Set("Cache-Control", "private, no-store")
		_, err := w.Write(result.Data)
		return err
	}
	f, err := os.Open(result.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	etag := fmt.Sprintf("\"%s-%x-%x\"", key, info.Size(), info.ModTime().UnixNano())
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	_, err = io.Copy(w, f)
	return err
}

func (s *Service) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return errors.New("imagecache: too many redirects")
	}
	source, _ := req.Context().Value(requestSourceKey{}).(requestSource)
	return s.validateTargetURL(req.Context(), req.URL, source)
}

func (s *Service) safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, address)
}

func (s *Service) validateTargetURL(ctx context.Context, u *url.URL, source requestSource) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("imagecache: unsafe redirect")
	}
	if source.Kind == SourceStash && sameOrigin(u.String(), source.Instance) {
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", u.Hostname())
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return errors.New("imagecache: unsafe target")
		}
	}
	return nil
}

func (s *Service) Cleanup(_ context.Context) error {
	cfg := normalizeConfig(s.config())
	cutoff := s.now().UTC().Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)
	entries, err := os.ReadDir(s.objectsDir)
	if err != nil {
		return err
	}
	for _, item := range entries {
		path := filepath.Join(s.objectsDir, item.Name())
		info, statErr := item.Info()
		if statErr != nil {
			continue
		}
		if strings.HasPrefix(item.Name(), ".tmp-") {
			if s.now().Sub(info.ModTime()) > time.Hour {
				_ = os.Remove(path)
			}
			continue
		}
		if info.Mode().IsRegular() && info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
	}
	registrationEntries, _ := os.ReadDir(s.registrationsDir)
	for _, item := range registrationEntries {
		if strings.HasPrefix(item.Name(), ".tmp-") {
			if info, err := item.Info(); err == nil && s.now().Sub(info.ModTime()) > time.Hour {
				_ = os.Remove(filepath.Join(s.registrationsDir, item.Name()))
			}
		}
	}
	now := s.now().UTC()
	s.statusMu.Lock()
	s.lastCleanupAt = &now
	s.lastError = ""
	s.statusMu.Unlock()
	return nil
}

func (s *Service) Clear(_ context.Context) (Status, error) {
	entries, err := os.ReadDir(s.objectsDir)
	if err != nil {
		return Status{}, err
	}
	for _, item := range entries {
		if item.Type().IsRegular() {
			if err := os.Remove(filepath.Join(s.objectsDir, item.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return Status{}, err
			}
		}
	}
	return s.Status(context.Background())
}

func (s *Service) Status(_ context.Context) (Status, error) {
	st := Status{CacheDirectory: s.dir}
	entries, err := os.ReadDir(s.objectsDir)
	if err != nil {
		return st, err
	}
	for _, item := range entries {
		if strings.HasPrefix(item.Name(), ".tmp-") {
			continue
		}
		info, statErr := item.Info()
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		st.EntryCount++
		st.UsedBytes += info.Size()
	}
	s.statusMu.RLock()
	st.LastCleanupAt = s.lastCleanupAt
	st.LastError = s.lastError
	s.statusMu.RUnlock()
	return st, nil
}

func (s *Service) setError(err error) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if err == nil {
		s.lastError = ""
		return
	}
	s.lastError = err.Error()
}

func (s *Service) StartCleanup(ctx context.Context) {
	go func() {
		_ = s.Cleanup(ctx)
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.Cleanup(ctx)
			}
		}
	}()
}
