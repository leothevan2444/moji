// Package indexer owns source-independent search and resource acquisition.
package indexer

import (
	"context"
	"math"
	"strconv"
	"time"
)

type Kind string

const (
	Builtin Kind = "BUILTIN"
	Jackett Kind = "JACKETT"
)

// SettingValue is the typed dynamic setting contract. Exactly one member is set.
type SettingValue struct {
	String  *string  `json:"string,omitempty"`
	Boolean *bool    `json:"boolean,omitempty"`
	Number  *float64 `json:"number,omitempty"`
}

func (v SettingValue) Text() (string, error) {
	count := 0
	out := ""
	if v.String != nil {
		count++
		out = *v.String
	}
	if v.Boolean != nil {
		count++
		out = strconv.FormatBool(*v.Boolean)
	}
	if v.Number != nil {
		count++
		if math.IsNaN(*v.Number) || math.IsInf(*v.Number, 0) {
			return "", ErrInvalidSettings
		}
		out = strconv.FormatFloat(*v.Number, 'f', -1, 64)
	}
	if count != 1 {
		return "", ErrInvalidSettings
	}
	return out, nil
}

type NetworkProfile struct {
	ID              string
	Revision        string
	ProxyURL        string
	FlareSolverrURL string
}

// InstanceSnapshot is a server-only frozen configuration. It may contain
// credentials and must never be returned as a GraphQL object or logged.
type InstanceSnapshot struct {
	SiteURL          string
	SiteType         string
	Language         string
	ID               string
	Kind             Kind
	Name             string
	Enabled          bool
	DefinitionID     string
	BundleRevision   string
	RemoteID         string
	ConfigRevision   string
	BaseURL          string
	TrustedOrigins   []string
	Settings         map[string]SettingValue
	Network          NetworkProfile
	ChallengeTimeout time.Duration
	RequestTimeout   time.Duration
	SiteTimeout      time.Duration
	RequestDelay     time.Duration
	JackettAPIKey    string
}

func (s InstanceSnapshot) String() string { return "indexer instance " + s.ID }

type Query struct {
	Mode        string
	Keywords    string
	Categories  []int
	Parameters  map[string]string
	InstanceIDs []string
	Limit       int
	Offset      int
	CachePolicy CachePolicy
}
type CachePolicy string

const (
	CacheDefault CachePolicy = "DEFAULT"
	CacheRefresh CachePolicy = "REFRESH"
)

// ResourceRef is created and stored by the server. APIs accept its opaque ID,
// not replacement locator/configuration data supplied by the browser.
type ResourceRef struct {
	Version          int
	ID               string
	InstanceID       string
	Kind             Kind
	BundleRevision   string
	DefinitionID     string
	DefinitionDigest string
	ConfigRevision   string
	Locator          Locator
}

func (r ResourceRef) String() string { return "indexer resource " + r.ID }

type Locator struct {
	DownloadURL string
	MagnetURI   string
	InfoHash    string
	Title       string
	Fields      map[string]string
}

type Release struct {
	Title        string
	GUID         string
	InstanceID   string
	InstanceName string
	Kind         Kind
	DetailsURL   string
	PublishedAt  *time.Time
	Categories   []int
	Size         *int64
	Seeders      *int
	Peers        *int
	InfoHash     string
	Resource     ResourceRef
}

type Artifact struct {
	infoHash     string
	TorrentBytes []byte
	Magnet       string
	Resource     ResourceRef
	AcquiredAt   time.Time
}

type Batch struct {
	ObservationCorrelationID string
	FetchedAt                time.Time
	InstanceID               string
	Releases                 []Release
	CacheHit                 bool
	Truncated                bool
	Partial                  bool
	IssueCount               int
	ReportedCount            *int64
	release                  func()
}

// Close releases the rule lease retained for this owned result batch.
func (b *Batch) Close() {
	if b != nil && b.release != nil {
		release := b.release
		b.release = nil
		release()
	}
}

type SiteReport struct {
	InstanceID    string
	Code          string
	CorrelationID string
	Results       int
	Elapsed       time.Duration
	CacheHit      bool
	Truncated     bool
	Partial       bool
	RetryAt       *time.Time
}

type SearchReport struct {
	Sites              []SiteReport
	AcceptedInstanceID string
	Partial            bool
	Truncated          bool
}
type SearchResult struct {
	Releases []Release
	Report   SearchReport
}

type Registry struct {
	Revision       string
	Instances      []InstanceSnapshot
	Order          []string
	Diagnostics    []SiteReport
	JackettCatalog CatalogStatus
}

// CatalogStatus describes only the last confirmed directory for one account.
// Known distinguishes a successful empty directory from an unloaded directory.
type CatalogStatus struct {
	Known     bool
	Stale     bool
	UpdatedAt *time.Time
}
type RegistryProvider interface {
	Snapshot(context.Context) (Registry, error)
}

// Adapter executes a single frozen instance. Catalog/configuration and run
// lifecycle are deliberately separate application capabilities.
type Adapter interface {
	Search(context.Context, InstanceSnapshot, Query) (Batch, error)
	Fetch(context.Context, InstanceSnapshot, ResourceRef) (Artifact, error)
}

// SearchProgress contains public source identities and bounded observations,
// never the account/locator snapshot used to execute the request.
type SearchSource struct {
	ID, Name string
	Kind     Kind
}
type SearchProgress struct {
	Phase       string
	Sources     []SearchSource
	Diagnostics []SiteReport
	InstanceID  string
	Report      SiteReport
	FetchedAt   time.Time
}
