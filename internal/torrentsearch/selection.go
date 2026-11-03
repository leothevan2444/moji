package torrentsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/leothevan2444/moji/internal/indexer"
)

// Selection contains only server-owned metadata and internal opaque references.
// It does not expose locators, account configuration or credentials.
type Selection struct {
	SearchID   string
	Revision   int64
	Query      indexer.Query
	Candidates []SelectionItem
}
type SelectionItem struct {
	Candidate  Candidate
	ResourceID string
}

func (s *Service) Selection(ctx context.Context, id string, ids []string) (Selection, error) {
	return s.store.selection(ctx, id, ids, s.now())
}
func (s *Store) selection(ctx context.Context, id string, ids []string, now time.Time) (Selection, error) {
	if id == "" || len(id) > 128 || len(ids) > 1000 {
		return Selection{}, ErrInvalid
	}
	wanted := map[string]bool{}
	for _, candidate := range ids {
		if candidate == "" || len(candidate) > 128 || wanted[candidate] {
			return Selection{}, ErrInvalid
		}
		wanted[candidate] = true
	}
	tx, err := s.db.Reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Selection{}, err
	}
	defer tx.Rollback()
	var run struct {
		State    string
		Version  int `db:"payload_version"`
		Revision int64
		Query    string `db:"query_json"`
	}
	err = tx.GetContext(ctx, &run, `SELECT state,payload_version,revision,query_json FROM torrent_search_runs WHERE id=? AND expires_at>?`, id, instant(now))
	if errors.Is(err, sql.ErrNoRows) {
		return Selection{}, ErrExpired
	}
	if err != nil {
		return Selection{}, err
	}
	if run.Version != 1 {
		return Selection{}, ErrInvalid
	}
	if !terminal(run.State) {
		return Selection{}, ErrBusy
	}
	out := Selection{SearchID: id, Revision: run.Revision, Candidates: []SelectionItem{}}
	if json.Unmarshal([]byte(run.Query), &out.Query) != nil {
		return Selection{}, ErrInvalid
	}
	var rows []struct {
		Version    int    `db:"payload_version"`
		Payload    string `db:"payload_json"`
		ResourceID string `db:"resource_id"`
	}
	if err = tx.SelectContext(ctx, &rows, `SELECT payload_version,payload_json,resource_id FROM torrent_search_candidates WHERE search_id=? ORDER BY position LIMIT 1001`, id); err != nil {
		return Selection{}, err
	}
	if len(rows) > 1000 {
		return Selection{}, ErrInvalid
	}
	for _, row := range rows {
		var candidate Candidate
		if row.Version != 1 || json.Unmarshal([]byte(row.Payload), &candidate) != nil {
			return Selection{}, ErrInvalid
		}
		if ids != nil && !wanted[candidate.ID] {
			continue
		}
		out.Candidates = append(out.Candidates, SelectionItem{Candidate: candidate, ResourceID: row.ResourceID})
		delete(wanted, candidate.ID)
	}
	if len(wanted) > 0 {
		return Selection{}, ErrExpired
	}
	if err = tx.Commit(); err != nil {
		return Selection{}, err
	}
	return out, nil
}
func (s Selection) RankingCandidates() []indexer.SelectionCandidate {
	out := make([]indexer.SelectionCandidate, 0, len(s.Candidates))
	for _, item := range s.Candidates {
		c := item.Candidate
		release := indexer.Release{InstanceID: c.InstanceID, InstanceName: c.InstanceName, Kind: c.Kind, Title: c.Title, DetailsURL: c.DetailsURL, InfoHash: c.InfoHash, Size: c.Size, Seeders: c.Seeders, Peers: c.Peers, PublishedAt: c.PublishedAt, Categories: c.Categories, Resource: indexer.ResourceRef{ID: item.ResourceID, InstanceID: c.InstanceID, Kind: c.Kind}}
		candidate := indexer.SelectionFromRelease(release)
		candidate.GUID = c.ID
		out = append(out, candidate)
	}
	return out
}

// Project maps a ranking back to immutable public rows before paging. Rules run
// on the complete selected set, so a LOCK match cannot disappear across pages.
func (s Selection) Project(ranked []indexer.SelectionCandidate, page Page) ([]Candidate, int, error) {
	page, err := page.normalized()
	if err != nil {
		return nil, 0, err
	}
	byID := map[string]SelectionItem{}
	for _, item := range s.Candidates {
		byID[item.Candidate.ID] = item
	}
	out := make([]Candidate, 0, len(ranked))
	for _, rank := range ranked {
		original, ok := byID[rank.GUID]
		if !ok || original.ResourceID != rank.ResourceID {
			return nil, 0, indexer.ErrReference
		}
		delete(byID, rank.GUID)
		out = append(out, original.Candidate)
	}
	switch page.Sort {
	case "SIZE_DESC":
		sort.SliceStable(out, func(i, j int) bool { return descendingInt64(out[i].Size, out[j].Size) })
	case "SEEDERS_DESC":
		sort.SliceStable(out, func(i, j int) bool { a, b := out[i].Seeders, out[j].Seeders; return a != nil && (b == nil || *a > *b) })
	case "DATE_DESC":
		sort.SliceStable(out, func(i, j int) bool {
			a, b := out[i].PublishedAt, out[j].PublishedAt
			return a != nil && (b == nil || a.After(*b))
		})
	}
	total := len(out)
	start := min(page.Offset, total)
	return out[start:min(start+page.Limit, total)], total, nil
}
func descendingInt64(a, b *int64) bool { return a != nil && (b == nil || *a > *b) }
