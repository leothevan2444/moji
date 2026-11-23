package taskruntime

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/torrentmeta"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

func (s *Service) queueManualResource(ctx context.Context, req AddTorrentRequest) (*Task, error) {
	if req.Film == nil || req.Film.Validate() != nil {
		return nil, film.ErrRequired
	}
	u, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || len(req.URL) > 16384 || u.User != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "magnet") || (u.Scheme != "magnet" && u.Host == "") {
		return nil, ErrInvalidTorrentInput
	}
	_, magnetErr := torrentmeta.ParseMagnet(req.URL)
	if u.Scheme == "magnet" && (normalizeMagnetURI(req.URL) == "" || magnetErr != nil) {
		return nil, ErrInvalidTorrentInput
	}
	code := film.Code(req.Film.Code)
	unlock := s.lockTask("sourcing-code:" + code)
	defer unlock()
	if err = s.ensureTaskCodeCanBeCreated(ctx, code); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	request, err := marshalSourcingRequest(DownloadRequest{Film: req.Film, Code: code, Paused: req.Paused, SavePath: req.SavePath, Category: req.Category, Tags: req.Tags})
	if err != nil {
		return nil, err
	}
	t := &Task{ID: s.newID(), Film: req.Film, Code: code, Source: TaskSourceManual, SourcingMode: "USER_SELECTED", ResourceConfirmed: true, TorrentURL: req.URL, SavePath: req.SavePath, Category: req.Category, Tags: req.Tags, SourcingRequestJSON: string(request), CreatedAt: now, UpdatedAt: now}
	t.SelectionReportJSON = s.manualSelectionReport(t)
	setTaskStage(t, TaskStageDownloading, TaskStageStatusPending)
	if err = s.store.Create(ctx, t); err != nil {
		return nil, err
	}
	select {
	case s.sourcingWake <- struct{}{}:
	default:
	}
	return cloneTask(t), nil
}

func (s *Service) prepareConfirmedResource(ctx context.Context, t *Task, rawURL string, metadata *downloadedTorrentMetadata) (preparedTorrent, error) {
	if !t.ResourceConfirmed {
		return preparedTorrent{}, film.ErrResourceConfirmation
	}
	if metadata == nil && len(t.TorrentFileData) > 0 {
		parsed, err := parseTorrentMetadata(t.TorrentFileData)
		if err != nil {
			return preparedTorrent{}, err
		}
		metadata = &parsed
	}
	if metadata == nil && strings.HasPrefix(rawURL, "magnet:") {
		_, magnetErr := torrentmeta.ParseMagnet(rawURL)
		if normalizeMagnetURI(rawURL) == "" || magnetErr != nil {
			return preparedTorrent{}, ErrInvalidTorrentInput
		}
		title := t.Candidate.Title
		if title == rawURL {
			title = ""
		}
		matched, err := validateCodeEvidence(t.Code, title, magnetDisplayName(rawURL))
		if err != nil {
			return preparedTorrent{}, err
		}
		if !matched {
			return preparedTorrent{}, candidateError(TaskStageErrorCandidateUnverified, "selected magnet does not establish the target film code")
		}
		return preparedTorrent{candidate: t.Candidate, options: qbittorrent.AddTorrentOptions{URLs: []string{rawURL}}}, nil
	}
	if metadata == nil {
		v, err := s.fetchTorrentMetadata(ctx, rawURL)
		if err != nil {
			return preparedTorrent{}, err
		}
		metadata = &v
	}
	if expected := normalizeInfoHash(t.Candidate.InfoHash); expected != "" && expected != metadata.InfoHash {
		return preparedTorrent{}, errors.New("torrent info hash differs from selected resource")
	}
	if err := validateTorrentCode(t.Code, *metadata); err != nil {
		return preparedTorrent{}, err
	}
	c := t.Candidate
	c.InfoHash = metadata.InfoHash
	if c.Title == "" {
		c.Title = metadata.Name
	}
	return preparedTorrent{candidate: c, options: qbittorrent.AddTorrentOptions{Torrents: []qbittorrent.TorrentFile{{Filename: film.SafeName(t.Code) + ".torrent", Data: metadata.Data}}}}, nil
}
