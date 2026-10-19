package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/pkg/jackett"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type candidateValidationError struct {
	code    string
	message string
}

func (e *candidateValidationError) Error() string { return e.message }

func candidateError(code, message string) error {
	return &candidateValidationError{code: code, message: message}
}

// Matching is independent of configurable ranking rules. A fuzzy indexer hit
// must never win merely because all the real matches are unavailable.
func validateCodeEvidence(code string, values ...string) (bool, error) {
	matched := false
	for _, value := range values {
		for _, found := range mediafiles.ExtractCodes(value) {
			if !mediafiles.SameCode(code, found) {
				if mediaLabel(found) {
					continue
				}
				return false, candidateError(TaskStageErrorCandidateMismatch,
					fmt.Sprintf("candidate code %s does not match task code %s", found, code))
			}
			matched = true
		}
	}
	return matched, nil
}

// Quality labels and numbered parts can resemble release codes, for example
// "[HD 720p] OFJE657" or "OFJE657-part01.mp4".
func mediaLabel(value string) bool {
	prefix, number, _ := strings.Cut(value, "-")
	switch prefix {
	case "HD", "FHD", "UHD":
		return number == "720" || number == "1080" || number == "2160" || number == "4320"
	case "CD", "DISC", "DISK", "DVD", "PART", "PT":
		return len(number) <= 2
	}
	return false
}

type preparedTorrent struct {
	candidate Candidate
	options   qbittorrent.AddTorrentOptions
}

func (s *Service) prepareTorrent(ctx context.Context, code string, candidate Candidate, torrentURL string) (preparedTorrent, error) {
	return s.prepareTorrentWithMetadata(ctx, code, candidate, torrentURL, nil)
}

func (s *Service) prepareTorrentWithMetadata(ctx context.Context, code string, candidate Candidate, torrentURL string, metadata *downloadedTorrentMetadata) (preparedTorrent, error) {
	title := candidate.Title
	if title == torrentURL {
		// A URL (and especially its info hash or query parameters) is not
		// evidence of a release code for a manually added magnet.
		title = ""
	}
	matched, err := validateCodeEvidence(code, title, magnetDisplayName(torrentURL))
	if err != nil {
		return preparedTorrent{}, err
	}
	parsed, err := url.Parse(torrentURL)
	if err != nil {
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, "invalid torrent URL")
	}
	if strings.EqualFold(parsed.Scheme, "magnet") {
		if !matched || normalizeMagnetURI(torrentURL) == "" {
			return preparedTorrent{}, candidateError(TaskStageErrorCandidateUnverified, "unable to verify magnet code from its title or display name")
		}
		return preparedTorrent{candidate: candidate, options: qbittorrent.AddTorrentOptions{URLs: []string{torrentURL}}}, nil
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, "torrent URL must use HTTP, HTTPS or magnet")
	}
	if metadata == nil {
		fetched, fetchErr := s.fetchTorrentMetadata(ctx, torrentURL)
		metadata, err = &fetched, fetchErr
	}
	if err != nil {
		// Do not persist download URLs, which may contain indexer API keys.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, fmt.Sprintf("unable to validate torrent metadata: %v", err))
	}
	if err := validateTorrentCode(code, *metadata); err != nil {
		return preparedTorrent{}, err
	}
	if expected := normalizeInfoHash(candidate.InfoHash); expected != "" && expected != metadata.InfoHash {
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, "torrent info hash differs from the selected candidate")
	}
	candidate.InfoHash = metadata.InfoHash
	return preparedTorrent{
		candidate: candidate,
		options:   qbittorrent.AddTorrentOptions{Torrents: []qbittorrent.TorrentFile{{Filename: code + ".torrent", Data: metadata.Data}}},
	}, nil
}

func validateTorrentCode(code string, metadata downloadedTorrentMetadata) error {
	names := append([]string{metadata.Name}, metadata.IdentityNames...)
	matched, err := validateCodeEvidence(code, names...)
	if err != nil {
		return err
	}
	videos := 0
	paths := append(append([]string(nil), metadata.Paths...), metadata.IdentityPaths...)
	for _, file := range paths {
		if !mediafiles.Video(file) {
			continue
		}
		videos++
		fileMatched, err := validateCodeEvidence(code, file)
		if err != nil {
			return err
		}
		matched = matched || fileMatched
	}
	if !matched || videos == 0 {
		return candidateError(TaskStageErrorCandidateUnverified, "torrent name and video files do not establish the task code")
	}
	return nil
}

func (s *Service) selectVerifiedCandidate(ctx context.Context, selector CandidateSelector, code string, results []jackett.SearchResult, cfg config.CandidateSelectionConfig) (jackett.SearchResult, preparedTorrent, error) {
	remaining := make([]jackett.SearchResult, 0, len(results))
	var rejected error
	for _, result := range results {
		if _, err := validateCodeEvidence(code, result.Title, magnetDisplayName(preferredTorrentURL(result))); err != nil {
			rejected = err
			continue
		}
		remaining = append(remaining, result)
	}
	for len(remaining) > 0 {
		if err := ctx.Err(); err != nil {
			return jackett.SearchResult{}, preparedTorrent{}, err
		}
		result, err := selector.Select(ctx, code, remaining, cfg)
		if err != nil {
			if rejected != nil {
				err = rejected
			}
			return jackett.SearchResult{}, preparedTorrent{}, err
		}
		prepared, err := s.prepareTorrent(ctx, code, candidateFromSearchResult(result), preferredTorrentURL(result))
		if err == nil {
			return result, prepared, nil
		}
		rejected = err
		logging.Infof("taskruntime: rejected candidate for code %s: %v", code, err)
		removed := false
		for i, candidate := range remaining {
			if candidate.Title == result.Title && preferredTorrentURL(candidate) == preferredTorrentURL(result) {
				remaining = append(remaining[:i], remaining[i+1:]...)
				removed = true
				break
			}
		}
		if !removed {
			break
		}
	}
	if rejected == nil {
		rejected = errors.New("taskruntime: no downloadable torrent candidate found")
	}
	return jackett.SearchResult{}, preparedTorrent{}, rejected
}

func (s *Service) blockSourcingValidation(ctx context.Context, task *Task, err error) error {
	code := TaskStageErrorTorrentMetadata
	var validation *candidateValidationError
	if errors.As(err, &validation) {
		code = validation.code
	}
	setTaskStage(task, TaskStageSourcing, TaskStageStatusBlocked)
	blockTask(task, code, err.Error(), s.now().UTC())
	if persistErr := s.store.Update(ctx, task); persistErr != nil {
		return fmt.Errorf("persist sourcing validation failure: %w", persistErr)
	}
	return err
}
