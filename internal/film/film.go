// Package film owns explicit StashBox identities. Codes are data, never parsed
// from titles here. Resource-to-film suggestions live at the creation boundary.
package film

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
)

type Person struct {
	ID   string
	Name string
}
type Snapshot struct {
	Endpoint        string
	SceneID         string
	Code            string
	Title           string
	Date            string
	DurationSeconds *int
	Studio          Person
	Performers      []Person
	ImageURL        string
	URL             string
	ConfirmedAt     time.Time
}

var ErrRequired = errors.New("STASHBOX_FILM_REQUIRED")
var ErrChanged = errors.New("FILM_CONFIRMATION_CHANGED")
var ErrResourceConfirmation = errors.New("RESOURCE_CONFIRMATION_REQUIRED")

func Endpoint(value string) string {
	u, err := url.Parse(strings.TrimSpace(value))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}
func Code(value string) string { return strings.ToUpper(strings.TrimSpace(value)) }
func (s Snapshot) Validate() error {
	if Endpoint(s.Endpoint) == "" || strings.TrimSpace(s.SceneID) == "" || Code(s.Code) == "" || strings.ContainsAny(s.Code, "\x00\r\n") || len(s.Code) > 512 {
		return ErrRequired
	}
	return nil
}
func (s Snapshot) Key() string { return Endpoint(s.Endpoint) + "\x00" + strings.TrimSpace(s.SceneID) }

// Digest excludes observation time; it binds confirmation to displayed data.
func (s Snapshot) Digest() string {
	s.ConfirmedAt = time.Time{}
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (s Snapshot) Clone() Snapshot {
	s.Performers = append([]Person(nil), s.Performers...)
	if s.DurationSeconds != nil {
		v := *s.DurationSeconds
		s.DurationSeconds = &v
	}
	return s
}

// Match locates a known complete code. It never extracts a different identity,
// normalizes number padding or discards provider suffixes.
func Match(code, text string) bool { _, ok := Suffix(code, text); return ok }
func Suffix(code, text string) (string, bool) {
	code = Code(code)
	upper := []rune(strings.ToUpper(text))
	original := []rune(text)
	target := []rune(code)
	if code == "" {
		return "", false
	}
	for i := 0; i <= len(upper)-len(target); i++ {
		if string(upper[i:i+len(target)]) != code {
			continue
		}
		end := i + len(target)
		left := i == 0 || !alnum(upper[i-1])
		right := end == len(upper) || !alnum(upper[end])
		if left && right {
			return string(original[end:]), true
		}
	}
	return "", false
}
func alnum(c rune) bool { return c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' }

// SafeName preserves ordinary codes but prevents metadata from choosing paths.
func SafeName(code string) string {
	v := strings.TrimSpace(code)
	if v != "" && v != "." && v != ".." && len(v) <= 160 && !strings.ContainsAny(v, "/\\\x00\r\n") {
		return v
	}
	h := sha256.Sum256([]byte(v))
	return "film-" + hex.EncodeToString(h[:8])
}
