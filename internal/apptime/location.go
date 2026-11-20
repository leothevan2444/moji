// Package apptime owns Moji's configured calendar timezone. Instants remain UTC.
package apptime

import (
	"sync"
	"time"
	_ "time/tzdata"
)

var locations sync.Map

func Location(zone string) *time.Location {
	if zone == "" || zone == "auto" || zone == "Local" {
		return time.Local
	}
	if cached, ok := locations.Load(zone); ok {
		return cached.(*time.Location)
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return time.Local
	}
	loaded, _ := locations.LoadOrStore(zone, location)
	return loaded.(*time.Location)
}
