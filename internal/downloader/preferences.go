package downloader

import (
	"context"
	"errors"
	"math"
	"reflect"
	"time"

	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

var ErrInvalidPreferences = errors.New("invalid qB transfer preferences")
var ErrPreferencesUnavailable = errors.New("qB transfer preferences unavailable")

type preferencesClient interface {
	GetTorrentPreferences(context.Context) (*qbittorrent.TorrentPreferences, error)
	PatchTransferPreferences(context.Context, qbittorrent.TransferPreferences) error
}

func (s *Service) TorrentPreferences(ctx context.Context) (*qbittorrent.TorrentPreferences, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client, ok := s.client.(preferencesClient)
	if !ok {
		return nil, ErrPreferencesUnavailable
	}
	return client.GetTorrentPreferences(ctx)
}

// PatchTransferPreferences never writes an entire fetched preferences snapshot.
// This preserves unrelated settings and edits made through qB's own WebUI.
func (s *Service) PatchTransferPreferences(ctx context.Context, p qbittorrent.TransferPreferences) (*qbittorrent.TorrentPreferences, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	client, ok := s.client.(preferencesClient)
	if !ok {
		return nil, ErrPreferencesUnavailable
	}
	for _, v := range []*int{p.MaxActiveDownloads, p.MaxActiveUploads, p.MaxActiveTorrents} {
		if v != nil && (*v < -1 || *v > 1000000) {
			return nil, ErrInvalidPreferences
		}
	}
	for _, v := range []*int{p.SlowTorrentDLRateThreshold, p.SlowTorrentULRateThreshold, p.SlowTorrentInactiveTimer} {
		if v != nil && (*v < 1 || *v > 2147483647) {
			return nil, ErrInvalidPreferences
		}
	}
	for _, v := range []*int{p.MaxSeedingTime, p.MaxInactiveSeedingTime} {
		if v != nil && (*v < 0 || *v > 2147483647) {
			return nil, ErrInvalidPreferences
		}
	}
	if p.MaxRatio != nil && (math.IsNaN(*p.MaxRatio) || math.IsInf(*p.MaxRatio, 0) || *p.MaxRatio < 0 || *p.MaxRatio > 1000000) {
		return nil, ErrInvalidPreferences
	}
	if p.MaxRatioAction != nil && (*p.MaxRatioAction < 0 || *p.MaxRatioAction > 3) {
		return nil, ErrInvalidPreferences
	}
	if p.ShareLimitMode != nil && *p.ShareLimitMode != "MatchAny" && *p.ShareLimitMode != "MatchAll" {
		return nil, ErrInvalidPreferences
	}
	current, err := client.GetTorrentPreferences(ctx)
	if err != nil {
		return nil, err
	}
	// Reject unsupported fields before any mutation, including false and zero patches.
	requested := reflect.ValueOf(p)
	supported := reflect.ValueOf(current.TransferPreferences)
	for i := 0; i < requested.NumField(); i++ {
		if !requested.Field(i).IsNil() && supported.Field(i).IsNil() {
			return nil, ErrPreferencesUnavailable
		}
	}

	if err := client.PatchTransferPreferences(ctx, p); err != nil {
		return nil, err
	}
	observed, err := client.GetTorrentPreferences(ctx)
	if err != nil {
		return nil, ErrObservationPending
	}
	for i := 0; i < requested.NumField(); i++ {
		if !requested.Field(i).IsNil() && !reflect.DeepEqual(requested.Field(i).Interface(), reflect.ValueOf(observed.TransferPreferences).Field(i).Interface()) {
			return nil, ErrObservationPending
		}
	}
	return observed, nil
}
