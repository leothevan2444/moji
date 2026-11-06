// Package torrentsearch owns bounded, restart-readable interactive searches.
package torrentsearch

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/leothevan2444/moji/internal/indexer"
)

var ErrExpired = errors.New("TORRENT_SEARCH_EXPIRED")
var ErrBusy = errors.New("TORRENT_SEARCH_BUSY")
var ErrNotRunning = errors.New("TORRENT_SEARCH_NOT_RUNNING")
var ErrInvalid = errors.New("TORRENT_SEARCH_INVALID")
var ErrTaskScope = errors.New("TORRENT_SEARCH_TASK_SCOPE_INVALID")

const retention = 30 * time.Minute
const maxRuns = 50
const maxCandidates = 10000
const maxBytes = 64 << 20

type Candidate struct {
	ID, InstanceID, InstanceName string
	Kind                         indexer.Kind
	Title, DetailsURL, InfoHash  string
	Size                         *int64
	Seeders, Peers               *int
	PublishedAt                  *time.Time
	Categories                   []int
}
type Site struct {
	InstanceID, Name string
	Kind             indexer.Kind
	State            string
	Report           indexer.SiteReport
	FetchedAt        *time.Time
	CandidateCount   int
}
type Snapshot struct {
	FilterInstanceID, CandidateQuery            string
	ID, State, TaskID, ErrorCode, CorrelationID string
	Revision                                    int64
	Query                                       indexer.Query
	CreatedAt, UpdatedAt, ExpiresAt             time.Time
	FinishedAt                                  *time.Time
	CancelRequested, Partial, Truncated         bool
	Total, Offset, Limit                        int
	Sites                                       []Site
	Candidates                                  []Candidate
}
type Input struct {
	Query  indexer.Query
	TaskID string
}
type Page struct {
	Offset, Limit              int
	Sort                       string
	InstanceID, CandidateQuery string
}

func (p Page) normalized() (Page, error) {
	p.CandidateQuery = strings.TrimSpace(p.CandidateQuery)
	if len(p.InstanceID) > 128 || !utf8.ValidString(p.CandidateQuery) || utf8.RuneCountInString(p.CandidateQuery) > 512 {
		return p, ErrInvalid
	}
	if p.Limit == 0 {
		p.Limit = 100
	}
	if p.Sort == "" {
		p.Sort = "RELEVANCE"
	}
	if p.Limit < 1 || p.Limit > 100 || p.Offset < 0 || p.Offset > 1000 {
		return p, ErrInvalid
	}
	switch p.Sort {
	case "RELEVANCE", "SEEDERS_DESC", "SIZE_DESC", "DATE_DESC":
	default:
		return p, ErrInvalid
	}
	return p, nil
}
func terminal(state string) bool { return state != "QUEUED" && state != "RUNNING" }
func instant(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z") }
