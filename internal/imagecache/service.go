package imagecache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	_ "modernc.org/sqlite"
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

type entry struct {
	Key, Kind, InstanceURL, OriginURL, FileName, ContentType, ETag, LastModified string
	Size                                                                         int64
	FetchedAt, ExpiresAt, LastAccessed                                           time.Time
}
type imageResponse struct {
	Entry entry
	Data  []byte
}

type Service struct {
	db            *sql.DB
	dir           string
	config        ConfigProvider
	client        *http.Client
	group         singleflight.Group
	credentialMu  sync.RWMutex
	credentials   map[string]string
	statusMu      sync.RWMutex
	lastCleanupAt *time.Time
	lastError     string
	now           func() time.Time
}
type requestSourceKey struct{}
type requestSource struct {
	Kind     SourceKind
	Instance string
}

func New(dbPath, cacheDir string, provider ConfigProvider) (*Service, error) {
	if provider == nil {
		provider = func() Config { return DefaultConfig() }
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("imagecache: create cache dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000", `CREATE TABLE IF NOT EXISTS image_cache_entries (
		cache_key TEXT PRIMARY KEY, source_kind TEXT NOT NULL, instance_url TEXT NOT NULL, origin_url TEXT NOT NULL,
		file_name TEXT NOT NULL DEFAULT '', content_type TEXT NOT NULL DEFAULT '', etag TEXT NOT NULL DEFAULT '',
		last_modified TEXT NOT NULL DEFAULT '', size_bytes INTEGER NOT NULL DEFAULT 0, fetched_at TEXT,
		expires_at TEXT, last_accessed_at TEXT NOT NULL)`} {
		if _, err := db.Exec(q); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("imagecache: init db: %w", err)
		}
	}
	s := &Service{db: db, dir: cacheDir, config: provider, credentials: map[string]string{}, now: time.Now}
	s.client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: s.checkRedirect, Transport: &http.Transport{Proxy: nil, DialContext: s.safeDialContext}}
	return s, nil
}

func (s *Service) Close() error { return s.db.Close() }

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
	now := s.now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.ExecContext(ctx, `INSERT INTO image_cache_entries(cache_key,source_kind,instance_url,origin_url,last_accessed_at)
		VALUES(?,?,?,?,?) ON CONFLICT(cache_key) DO UPDATE SET origin_url=excluded.origin_url,last_accessed_at=excluded.last_accessed_at`, key, d.Kind, instance, resolved, now)
	if err != nil {
		return "", fmt.Errorf("imagecache: register: %w", err)
	}
	if strings.TrimSpace(d.APIKey) != "" {
		s.credentialMu.Lock()
		s.credentials[credentialKey(d.Kind, instance)] = d.APIKey
		s.credentialMu.Unlock()
	}
	return "/api/images/" + key, nil
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

func (s *Service) RegisterHandler(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/images/{key}", s.ServeHTTP)
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if len(key) != 64 {
		http.NotFound(w, r)
		return
	}
	v, err, _ := s.group.Do(key, func() (any, error) { return s.load(r.Context(), key) })
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		s.statusMu.Lock()
		s.lastError = err.Error()
		s.statusMu.Unlock()
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	result := v.(imageResponse)
	if err := serveEntry(w, r, s.dir, result); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Service) load(ctx context.Context, key string) (imageResponse, error) {
	e, err := s.get(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return imageResponse{}, os.ErrNotExist
	}
	if err != nil {
		return imageResponse{}, err
	}
	now := s.now().UTC()
	if e.FileName != "" {
		if _, statErr := os.Stat(filepath.Join(s.dir, e.FileName)); statErr != nil {
			e.FileName = ""
			e.Size = 0
			e.ETag = ""
			e.LastModified = ""
		}
	}
	if e.FileName != "" && now.Before(e.ExpiresAt) {
		_, _ = s.db.ExecContext(ctx, "UPDATE image_cache_entries SET last_accessed_at=? WHERE cache_key=?", now.Format(time.RFC3339Nano), key)
		return imageResponse{Entry: e}, nil
	}
	fresh, err := s.fetch(ctx, e)
	if err != nil && e.FileName != "" {
		s.statusMu.Lock()
		s.lastError = err.Error()
		s.statusMu.Unlock()
		return imageResponse{Entry: e}, nil
	}
	if err == nil {
		s.statusMu.Lock()
		s.lastError = ""
		s.statusMu.Unlock()
	}
	return fresh, err
}

func (s *Service) get(ctx context.Context, key string) (entry, error) {
	var e entry
	var fetched, expires, accessed sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT cache_key,source_kind,instance_url,origin_url,file_name,content_type,etag,last_modified,size_bytes,fetched_at,expires_at,last_accessed_at FROM image_cache_entries WHERE cache_key=?`, key).
		Scan(&e.Key, &e.Kind, &e.InstanceURL, &e.OriginURL, &e.FileName, &e.ContentType, &e.ETag, &e.LastModified, &e.Size, &fetched, &expires, &accessed)
	if err != nil {
		return e, err
	}
	e.FetchedAt = parseTime(fetched.String)
	e.ExpiresAt = parseTime(expires.String)
	e.LastAccessed = parseTime(accessed.String)
	return e, nil
}

func parseTime(v string) time.Time { t, _ := time.Parse(time.RFC3339Nano, v); return t }

func (s *Service) fetch(ctx context.Context, e entry) (imageResponse, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, e.OriginURL, nil)
	source := requestSource{Kind: SourceKind(e.Kind), Instance: e.InstanceURL}
	if err := s.validateTargetURL(ctx, req.URL, source); err != nil {
		return imageResponse{}, err
	}
	req = req.WithContext(context.WithValue(req.Context(), requestSourceKey{}, source))
	if e.ETag != "" {
		req.Header.Set("If-None-Match", e.ETag)
	}
	if e.LastModified != "" {
		req.Header.Set("If-Modified-Since", e.LastModified)
	}
	if sameOrigin(e.OriginURL, e.InstanceURL) {
		s.credentialMu.RLock()
		key := s.credentials[credentialKey(SourceKind(e.Kind), e.InstanceURL)]
		s.credentialMu.RUnlock()
		if key != "" {
			req.Header.Set("ApiKey", key)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return imageResponse{}, err
	}
	defer resp.Body.Close()
	now := s.now().UTC()
	expires := expiryFromHeaders(resp.Header, now)
	if resp.StatusCode == http.StatusNotModified && e.FileName != "" {
		_, err = s.db.ExecContext(ctx, "UPDATE image_cache_entries SET fetched_at=?,expires_at=?,last_accessed_at=? WHERE cache_key=?", now.Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), e.Key)
		e.ExpiresAt = expires
		return imageResponse{Entry: e}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return imageResponse{}, fmt.Errorf("imagecache: upstream status %d", resp.StatusCode)
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if !allowedContentType(contentType) {
		return imageResponse{}, fmt.Errorf("imagecache: rejected content type %q", contentType)
	}
	tmp, err := os.CreateTemp(s.dir, ".image-*")
	if err != nil {
		return imageResponse{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	n, copyErr := io.Copy(tmp, io.LimitReader(resp.Body, maxImageBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return imageResponse{}, copyErr
	}
	if closeErr != nil {
		return imageResponse{}, closeErr
	}
	if n > maxImageBytes {
		return imageResponse{}, errors.New("imagecache: image exceeds 10 MB")
	}
	cfg := normalizeConfig(s.config())
	fileName := ""
	if cfg.Enabled {
		fileName = e.Key + extensionFor(contentType)
		if err := os.Rename(tmpName, filepath.Join(s.dir, fileName)); err != nil {
			return imageResponse{}, err
		}
	}
	_, err = s.db.ExecContext(ctx, `UPDATE image_cache_entries SET file_name=?,content_type=?,etag=?,last_modified=?,size_bytes=?,fetched_at=?,expires_at=?,last_accessed_at=? WHERE cache_key=?`, fileName, contentType, resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"), n, now.Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), e.Key)
	if err != nil {
		return imageResponse{}, err
	}
	if !cfg.Enabled {
		data, readErr := os.ReadFile(tmpName)
		if readErr != nil {
			return imageResponse{}, readErr
		}
		e.ContentType = contentType
		e.Size = n
		return imageResponse{Entry: e, Data: data}, nil
	}
	e.FileName = fileName
	e.ContentType = contentType
	e.Size = n
	e.ETag = resp.Header.Get("ETag")
	e.LastModified = resp.Header.Get("Last-Modified")
	e.ExpiresAt = expires
	_ = s.Cleanup(ctx)
	return imageResponse{Entry: e}, nil
}

func allowedContentType(v string) bool {
	switch v {
	case "image/jpeg", "image/png", "image/webp", "image/gif", "image/avif":
		return true
	}
	return false
}
func extensionFor(v string) string {
	return map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif", "image/avif": ".avif"}[v]
}
func expiryFromHeaders(h http.Header, now time.Time) time.Time {
	for _, p := range strings.Split(h.Get("Cache-Control"), ",") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "max-age=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(p, "max-age=")); err == nil && n >= 0 {
				if n > 86400 {
					n = 86400
				}
				return now.Add(time.Duration(n) * time.Second)
			}
		}
	}
	return now.Add(24 * time.Hour)
}
func sameOrigin(a, b string) bool {
	ua, _ := url.Parse(a)
	ub, _ := url.Parse(b)
	return strings.EqualFold(ua.Scheme, ub.Scheme) && strings.EqualFold(ua.Host, ub.Host)
}

func serveEntry(w http.ResponseWriter, r *http.Request, dir string, result imageResponse) error {
	e := result.Entry
	if e.ETag != "" {
		w.Header().Set("ETag", e.ETag)
	}
	if e.LastModified != "" {
		w.Header().Set("Last-Modified", e.LastModified)
	}
	if e.ETag != "" && r.Header.Get("If-None-Match") == e.ETag {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	if result.Data != nil {
		w.Header().Set("Content-Type", e.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(result.Data)))
		w.Header().Set("Cache-Control", "private, no-store")
		_, err := w.Write(result.Data)
		return err
	}
	f, err := os.Open(filepath.Join(dir, e.FileName))
	if err != nil {
		return err
	}
	defer f.Close()
	w.Header().Set("Content-Type", e.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(e.Size, 10))
	w.Header().Set("Cache-Control", "private, max-age=86400")
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

func (s *Service) Cleanup(ctx context.Context) error {
	cfg := normalizeConfig(s.config())
	cutoff := s.now().UTC().Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)
	max := int64(cfg.MaxSizeMB) << 20
	rows, err := s.db.QueryContext(ctx, `SELECT cache_key,file_name,size_bytes,last_accessed_at FROM image_cache_entries WHERE file_name<>'' ORDER BY last_accessed_at DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type item struct {
		k, f string
		s    int64
		t    time.Time
	}
	var items []item
	var total int64
	for rows.Next() {
		var x item
		var ts string
		if err := rows.Scan(&x.k, &x.f, &x.s, &ts); err != nil {
			return err
		}
		x.t = parseTime(ts)
		items = append(items, x)
		total += x.s
	}
	for i := len(items) - 1; i >= 0; i-- {
		x := items[i]
		if x.t.Before(cutoff) || total > max {
			_ = os.Remove(filepath.Join(s.dir, x.f))
			_, _ = s.db.ExecContext(ctx, `UPDATE image_cache_entries SET file_name='',content_type='',etag='',last_modified='',size_bytes=0,fetched_at=NULL,expires_at=NULL WHERE cache_key=?`, x.k)
			total -= x.s
		}
	}
	temps, _ := filepath.Glob(filepath.Join(s.dir, ".image-*"))
	for _, name := range temps {
		if info, err := os.Stat(name); err == nil && s.now().Sub(info.ModTime()) > time.Hour {
			_ = os.Remove(name)
		}
	}
	now := s.now().UTC()
	s.statusMu.Lock()
	s.lastCleanupAt = &now
	s.lastError = ""
	s.statusMu.Unlock()
	return nil
}

func (s *Service) Clear(ctx context.Context) (Status, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT file_name FROM image_cache_entries WHERE file_name<>''")
	if err != nil {
		return Status{}, err
	}
	var files []string
	for rows.Next() {
		var f string
		_ = rows.Scan(&f)
		files = append(files, f)
	}
	rows.Close()
	for _, f := range files {
		_ = os.Remove(filepath.Join(s.dir, f))
	}
	_, err = s.db.ExecContext(ctx, `UPDATE image_cache_entries SET file_name='',content_type='',etag='',last_modified='',size_bytes=0,fetched_at=NULL,expires_at=NULL`)
	if err != nil {
		return Status{}, err
	}
	return s.Status(ctx)
}
func (s *Service) Status(ctx context.Context) (Status, error) {
	var st Status
	st.CacheDirectory = s.dir
	err := s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(size_bytes),0),COUNT(*) FROM image_cache_entries WHERE file_name<>''").Scan(&st.UsedBytes, &st.EntryCount)
	s.statusMu.RLock()
	st.LastCleanupAt = s.lastCleanupAt
	st.LastError = s.lastError
	s.statusMu.RUnlock()
	return st, err
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
