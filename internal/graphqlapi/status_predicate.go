package graphqlapi

import (
	"strings"
	"time"
)

// ServiceReadinessWindow is the freshness window used to derive
// ServiceStatus.ready from the last successful probe. A probe that
// succeeded more than this ago, or a probe that failed, yields ready=false.
// The slow ticker is 120s; 240s (2×) tolerates one missed probe without
// flipping the chip.
const ServiceReadinessWindow = 240 * time.Second

// EvaluateServiceReadiness returns true iff:
//   - the service's relevant config fields are complete (configured), and
//   - a probe of the upstream service has succeeded within
//     ServiceReadinessWindow (okAt non-zero and within window), and
//   - the most recent probe did not record a lastError.
//
// lastError is "" when the latest probe succeeded; the predicate treats a
// non-empty lastError as "probe failed" regardless of okAt's value.
// Whitespace-only lastError is treated as empty so user input drift in the
// settings form (e.g. accidental spaces) does not poison the readiness signal.
func EvaluateServiceReadiness(configured bool, okAt time.Time, lastError string, now time.Time) bool {
	if !configured {
		return false
	}
	if okAt.IsZero() {
		return false
	}
	if strings.TrimSpace(lastError) != "" {
		return false
	}
	return now.Sub(okAt) <= ServiceReadinessWindow
}
