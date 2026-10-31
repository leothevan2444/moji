package releasedate

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Auto uses the timezone of the Moji process (including a container's TZ).
const DefaultTimezone = "auto"

var systemTimezoneOnce sync.Once
var systemTimezoneName string

func EffectiveTimezone(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" && configured != "auto" {
		return configured
	}
	systemTimezoneOnce.Do(func() { systemTimezoneName = detectSystemTimezone() })
	return systemTimezoneName
}
func detectSystemTimezone() string {
	if tz, ok := os.LookupEnv("TZ"); ok {
		tz = strings.TrimPrefix(tz, ":")
		if tz == "" {
			return "UTC"
		}
		if name := zoneName(tz); name != "" {
			return name
		}
		if _, err := time.LoadLocation(tz); err == nil {
			return tz
		}
	}
	if name := time.Local.String(); name != "Local" {
		if _, err := time.LoadLocation(name); err == nil {
			return name
		}
	}
	if path, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
		if name := zoneName(path); name != "" {
			return name
		}
	}
	// Some systems copy /etc/localtime instead of linking it. Match TZif data,
	// rather than guessing a timezone from its current UTC offset.
	local, err := os.ReadFile("/etc/localtime")
	if err != nil {
		return "Local"
	}
	roots := []string{"/usr/share/zoneinfo", "/usr/share/lib/zoneinfo", "/usr/lib/locale/TZ"}
	if raw, err := os.ReadFile("/etc/timezone"); err == nil {
		name := strings.TrimSpace(string(raw))
		for _, root := range roots {
			if data, err := os.ReadFile(filepath.Join(root, name)); err == nil && bytes.Equal(data, local) {
				if _, err = time.LoadLocation(name); err == nil {
					return name
				}
			}
		}
	}
	for _, root := range roots {
		match := ""
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if entry.Name() == "posix" || entry.Name() == "right" {
					return filepath.SkipDir
				}
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(data, local) {
				return nil
			}
			name, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			if _, err = time.LoadLocation(name); err == nil {
				match = name
				return fs.SkipAll
			}
			return nil
		})
		if match != "" {
			return match
		}
	}
	// Custom system TZif files still work through Go's Local location. The UI
	// receives the server's calendar date separately and never guesses an IANA ID.
	return "Local"
}
func zoneName(path string) string {
	_, name, ok := strings.Cut(filepath.ToSlash(path), "/zoneinfo/")
	if !ok || strings.HasPrefix(name, "posix/") || strings.HasPrefix(name, "right/") {
		return ""
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ""
	}
	return name
}
