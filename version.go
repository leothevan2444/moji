// Package moji exposes the application release embedded from VERSION.
package moji

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var releaseVersion string

// Version returns the release compiled into this binary, independently of its
// working directory and without requiring build flags or a runtime VERSION file.
func Version() string {
	return strings.TrimSpace(releaseVersion)
}
