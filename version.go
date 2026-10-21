// Package moji exposes the application version set at build time from Git tags.
package moji

// Make and Docker set this with -ldflags -X. Plain go build/run stays a dev build.
var releaseVersion = "dev"

// Version returns the compiled version without requiring Git at runtime.
func Version() string {
	return releaseVersion
}
