// Package indexerdefs owns runtime installation and activation of immutable
// Cardigann definition bundles. It does not perform resource selection.
package indexerdefs

import (
	"context"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/leothevan2444/moji/internal/cardigann"
)

const SchemaVersion = 11
const ManifestVersion = 1
const EngineVersion = "cardigann-v11-1"

var (
	ErrNotInstalled  = errors.New("DEFINITION_LIBRARY_NOT_INSTALLED")
	ErrBusy          = errors.New("DEFINITION_UPDATE_BUSY")
	ErrConflict      = errors.New("DEFINITION_REVISION_CONFLICT")
	ErrInvalidBundle = errors.New("DEFINITION_BUNDLE_INVALID")
	ErrIncompatible  = errors.New("DEFINITION_BUNDLE_INCOMPATIBLE")
	ErrUnavailable   = errors.New("DEFINITION_LIBRARY_UNAVAILABLE")
	ErrNotRunning    = errors.New("DEFINITION_SERVICE_NOT_RUNNING")
	ErrNoPrevious    = errors.New("DEFINITION_PREVIOUS_UNAVAILABLE")
	ErrBudget        = errors.New("DEFINITION_STORAGE_BUDGET_EXCEEDED")
	ErrSource        = errors.New("DEFINITION_SOURCE_UNAVAILABLE")
	ErrJobTerminal   = errors.New("DEFINITION_UPDATE_ALREADY_TERMINAL")
	ErrCancelling    = errors.New("DEFINITION_ACTIVATION_ALREADY_STARTED")
)

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var definitionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func ValidRevision(revision string) bool { return revisionPattern.MatchString(revision) }

type Revision struct {
	Commit      string
	ETag        string
	PublishedAt time.Time
}

// Source implementations must return one immutable revision per download.
// Only the official source is exposed by application configuration.
type Source interface {
	Latest(context.Context, string) (Revision, bool, error)
	Download(context.Context, string, io.Writer) error
}

type File struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type Manifest struct {
	Version         int       `json:"version"`
	Schema          int       `json:"schema"`
	Repository      string    `json:"repository"`
	Revision        string    `json:"revision"`
	Engine          string    `json:"engine"`
	InstalledAt     time.Time `json:"installedAt"`
	Files           []File    `json:"files"`
	DefinitionCount int       `json:"definitionCount"`
}

type Bundle struct {
	Manifest       Manifest
	ManifestSHA256 string
	Directory      string
	Plans          map[string]*cardigann.Plan
}

type JobState string

const (
	Queued            JobState = "QUEUED"
	Downloading       JobState = "DOWNLOADING"
	Validating        JobState = "VALIDATING"
	CheckingInstances JobState = "CHECKING_INSTANCES"
	Activating        JobState = "ACTIVATING"
	Completed         JobState = "COMPLETED"
	Failed            JobState = "FAILED"
	Cancelled         JobState = "CANCELLED"
	Interrupted       JobState = "INTERRUPTED"
)

func (s JobState) Terminal() bool {
	return s == Completed || s == Failed || s == Cancelled || s == Interrupted
}

type Job struct {
	ID              string     `json:"id"`
	Kind            string     `json:"kind"`
	State           JobState   `json:"state"`
	Target          string     `json:"target"`
	ExpectedActive  string     `json:"expectedActive"`
	DownloadedBytes int64      `json:"downloadedBytes"`
	ErrorCode       string     `json:"errorCode"`
	CorrelationID   string     `json:"correlationId"`
	StartedAt       time.Time  `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	Revision        int64      `json:"revision"`
}

type State struct {
	Active        string
	Previous      string
	Revision      int64
	Available     string
	ETag          string
	LastCheckedAt *time.Time
	CheckError    string
}

type Status struct {
	StreamID  string
	State     State
	Installed []Manifest
	Job       *Job
	Available bool
	ErrorCode string
}

// Validator checks a candidate against current configured instances. It must
// provide a short activation guard that rechecks the configuration revision;
// compilation/network work occurs before that guard and outside SQL writes.
type Validator interface {
	Validate(context.Context, *Bundle) (ActivationGuard, error)
}

type ActivationGuard interface {
	WithCurrent(context.Context, func() error) error
}
