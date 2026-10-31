// Package releasedate defines the calendar gate shared by scene download entry points.
package releasedate

import (
	"errors"
	"strings"
	"time"
	_ "time/tzdata"
)

var ErrNotReleased = errors.New("release date has not arrived")
var ErrUnknown = errors.New("release date is unknown")

func Location(zone string) *time.Location {
	if zone == "" || zone == "auto" || zone == "Local" {
		return time.Local
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Local
	}
	return location
}
func Parse(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"2006-01-02", time.RFC3339Nano} {
		if date, err := time.Parse(layout, raw); err == nil {
			return date.Format("2006-01-02"), true
		}
	}
	return "", false
}
func Check(raw string, now time.Time, zone string) error {
	date, ok := Parse(raw)
	if !ok {
		return ErrUnknown
	}
	if date > now.In(Location(zone)).Format("2006-01-02") {
		return ErrNotReleased
	}
	return nil
}
func Midnight(date, zone string) time.Time {
	value, _ := time.ParseInLocation("2006-01-02", date, Location(zone))
	return value.UTC()
}
