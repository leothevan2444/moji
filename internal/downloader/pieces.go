package downloader

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrPiecesUnavailable = errors.New("torrent pieces are unavailable")

const maxPieceBuckets = 512
const maxTorrentPieces = 2_000_000

type PieceBucket struct{ Start, Count, Missing, Downloading, Downloaded int }
type PieceMap struct {
	Hash       string
	Total      int
	ObservedAt string
	Buckets    []PieceBucket
}
type pieceReader interface {
	GetTorrentPiecesStates(context.Context, string) ([]int, error)
}

func knownCount(value int64) *int64 {
	if value < 0 {
		return nil
	}
	return &value
}
func completeTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

// Pieces reads only a currently observed torrent. It never synthesizes a map
// from the overall percentage and never joins the global monitoring loop.
func (s *Service) Pieces(ctx context.Context, hash string) (PieceMap, error) {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if !validHash(hash) {
		return PieceMap{}, ErrNotFound
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return PieceMap{}, err
	}
	observed, err := time.Parse(time.RFC3339, snapshot.CheckedAt)
	if err != nil || !snapshot.Connected || time.Since(observed) > 15*time.Second {
		return PieceMap{}, ErrPiecesUnavailable
	}
	found := false
	for _, t := range snapshot.Torrents {
		if t.Hash == hash && t.Present {
			found = true
			break
		}
	}
	if !found {
		return PieceMap{}, ErrNotFound
	}
	reader, ok := s.client.(pieceReader)
	if !ok {
		return PieceMap{}, ErrPiecesUnavailable
	}
	work, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	s.stateMu.Lock()
	if s.pieceSlots == nil {
		s.pieceSlots = make(chan struct{}, 2)
	}
	slots := s.pieceSlots
	s.stateMu.Unlock()
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	case <-work.Done():
		return PieceMap{}, work.Err()
	}
	states, readErr := reader.GetTorrentPiecesStates(work, hash)
	if err = work.Err(); err != nil {
		return PieceMap{}, err
	}
	if readErr != nil {
		return PieceMap{}, ErrPiecesUnavailable
	}
	buckets, err := aggregatePieces(states)
	if err != nil {
		return PieceMap{}, err
	}
	return PieceMap{Hash: hash, Total: len(states), ObservedAt: time.Now().UTC().Format(time.RFC3339), Buckets: buckets}, nil
}
func aggregatePieces(states []int) ([]PieceBucket, error) {
	if len(states) > maxTorrentPieces {
		return nil, ErrPiecesUnavailable
	}
	buckets := []PieceBucket{}
	if len(states) == 0 {
		return buckets, nil
	}
	width := (len(states) + maxPieceBuckets - 1) / maxPieceBuckets
	for start := 0; start < len(states); start += width {
		end := start + width
		if end > len(states) {
			end = len(states)
		}
		b := PieceBucket{Start: start, Count: end - start}
		for _, state := range states[start:end] {
			switch state {
			case 0:
				b.Missing++
			case 1:
				b.Downloading++
			case 2:
				b.Downloaded++
			default:
				return nil, ErrPiecesUnavailable
			}
		}
		buckets = append(buckets, b)
	}
	return buckets, nil
}
