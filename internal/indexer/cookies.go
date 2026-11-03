package indexer

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

type cookieRecord struct {
	Origin string      `json:"origin"`
	Cookie http.Cookie `json:"cookie"`
}
type persistentJar struct {
	mu       sync.Mutex
	jar      http.CookieJar
	records  map[string]cookieRecord
	dirty    bool
	overflow bool
}

func newPersistentJar() (*persistentJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	return &persistentJar{jar: jar, records: map[string]cookieRecord{}}, nil
}
func (j *persistentJar) Cookies(target *url.URL) []*http.Cookie { return j.jar.Cookies(target) }
func (j *persistentJar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.jar.SetCookies(target, cookies)
	for _, cookie := range cookies {
		if cookie.Name == "" || len(cookie.Name)+len(cookie.Value) > 8192 {
			j.overflow = true
			continue
		}
		copy := *cookie
		copy.Unparsed = nil
		copy.Raw = ""
		if copy.MaxAge > 0 {
			copy.Expires = time.Now().Add(time.Duration(min(copy.MaxAge, 365*86400)) * time.Second)
			copy.MaxAge = 0
		}
		domain := strings.TrimPrefix(strings.ToLower(copy.Domain), ".")
		if domain == "" {
			domain = strings.ToLower(target.Hostname())
		}
		path := copy.Path
		if path == "" || path[0] != '/' {
			path = target.Path
			if cut := strings.LastIndex(path, "/"); cut > 0 {
				path = path[:cut]
			} else {
				path = "/"
			}
		}
		key := domain + "\n" + path + "\n" + copy.Name
		if copy.MaxAge < 0 || !copy.Expires.IsZero() && copy.Expires.Before(time.Now()) {
			delete(j.records, key)
			j.dirty = true
			continue
		}
		if _, exists := j.records[key]; !exists && len(j.records) >= 512 {
			j.overflow = true
			continue
		}
		copy.Path = path
		j.records[key] = cookieRecord{Origin: target.Scheme + "://" + target.Host + "/", Cookie: copy}
		j.dirty = true
	}
}

func (j *persistentJar) snapshot() ([]cookieRecord, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.overflow {
		return nil, false, ErrResponseBudget
	}
	records := make([]cookieRecord, 0, len(j.records))
	for _, record := range j.records {
		if !record.Cookie.Expires.IsZero() && record.Cookie.Expires.Before(time.Now()) {
			continue
		}
		records = append(records, record)
	}
	return records, j.dirty, nil
}

func (j *persistentJar) restore(records []cookieRecord) error {
	if len(records) > 512 {
		return ErrResponseBudget
	}
	for _, record := range records {
		origin, err := parsePublicURL(record.Origin)
		if err != nil {
			return ErrInvalidSettings
		}
		j.SetCookies(origin, []*http.Cookie{&record.Cookie})
	}
	j.mu.Lock()
	j.dirty = false
	j.mu.Unlock()
	return nil
}
