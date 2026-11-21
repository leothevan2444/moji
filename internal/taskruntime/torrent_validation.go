package taskruntime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/leothevan2444/moji/internal/config"
	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/logging"
	"github.com/leothevan2444/moji/internal/mediafiles"
	"github.com/leothevan2444/moji/pkg/qbittorrent"
)

type candidateValidationError struct {
	code    string
	message string
	cause   error
}

func (e *candidateValidationError) Error() string { return e.message }
func (e *candidateValidationError) Unwrap() error { return e.cause }

func candidateError(code, message string) error {
	return &candidateValidationError{code: code, message: message}
}

func SourcingValidationErrorCode(err error) string {
	var validation *candidateValidationError
	if errors.As(err, &validation) {
		return validation.code
	}
	return ""
}

// Matching is independent of configurable ranking rules. A fuzzy indexer hit
// must never win merely because all the real matches are unavailable.
func validateCodeEvidence(code string, values ...string) (bool, error) {
	nonempty := false
	for _, value := range values {
		if film.Match(code, value) {
			return true, nil
		}
		nonempty = nonempty || strings.TrimSpace(value) != ""
	}
	if nonempty {
		return false, candidateError(TaskStageErrorCandidateUnverified, "resource does not establish the complete target film code")
	}
	return false, nil
}

type preparedTorrent struct {
	candidate Candidate
	options   qbittorrent.AddTorrentOptions
}

func (s *Service) prepareTorrent(ctx context.Context, code string, candidate Candidate, torrentURL string) (preparedTorrent, error) {
	return s.prepareTorrentWithMetadata(ctx, code, candidate, torrentURL, nil)
}

func (s *Service) prepareTorrentWithMetadata(ctx context.Context, code string, candidate Candidate, torrentURL string, metadata *downloadedTorrentMetadata) (preparedTorrent, error) {
	if strings.HasPrefix(torrentURL, "indexer-resource:") {
		if metadata != nil {
			return prepareTorrentFile(code, candidate, *metadata)
		}
		resource, magnet, err := s.resolveIndexedResource(ctx, torrentURL)
		if err != nil {
			return preparedTorrent{}, &candidateValidationError{code: TaskStageErrorTorrentMetadata, message: "unable to acquire indexer resource", cause: err}
		}
		if magnet != "" {
			candidate.MagnetURI = magnet
			return s.prepareTorrentWithMetadata(ctx, code, candidate, magnet, nil)
		}
		return prepareTorrentFile(code, candidate, resource)
	}
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
		s.inspectionCacheMu.Lock()
		entry, ok := s.inspectionCache[normalizeTorrentInspectionCacheKey(torrentURL, "")]
		s.inspectionCacheMu.Unlock()
		if ok && entry.metadata != nil && entry.expiresAt.After(s.now().UTC()) {
			metadata = entry.metadata
		} else {
			fetched, fetchErr := s.fetchTorrentMetadata(ctx, torrentURL)
			metadata, err = &fetched, fetchErr
		}
	}
	if err != nil {
		// Do not persist download URLs, which may contain indexer API keys.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, fmt.Sprintf("unable to validate torrent metadata: %v", err))
	}
	return prepareTorrentFile(code, candidate, *metadata)
}

func prepareTorrentFile(code string, candidate Candidate, metadata downloadedTorrentMetadata) (preparedTorrent, error) {
	if _, err := validateCodeEvidence(code, candidate.Title); err != nil {
		return preparedTorrent{}, err
	}
	if err := validateTorrentCode(code, metadata); err != nil {
		return preparedTorrent{}, err
	}
	if expected := normalizeInfoHash(candidate.InfoHash); expected != "" && expected != metadata.InfoHash {
		return preparedTorrent{}, candidateError(TaskStageErrorTorrentMetadata, "torrent info hash differs from the selected candidate")
	}
	candidate.InfoHash = metadata.InfoHash
	return preparedTorrent{
		candidate: candidate,
		options:   qbittorrent.AddTorrentOptions{Torrents: []qbittorrent.TorrentFile{{Filename: film.SafeName(code) + ".torrent", Data: metadata.Data}}},
	}, nil
}

func validateTorrentCode(code string, metadata downloadedTorrentMetadata) error {
	matched := false
	for _, name := range append([]string{metadata.Name}, metadata.IdentityNames...) {
		matched = film.Match(code, name) || matched
	}
	videos := 0
	paths := append(append([]string(nil), metadata.Paths...), metadata.IdentityPaths...)
	for _, file := range paths {
		if !mediafiles.Video(file) {
			continue
		}
		videos++
		matched = film.Match(code, file) || matched
	}
	if !matched || videos == 0 {
		return candidateError(TaskStageErrorCandidateUnverified, "torrent name and video files do not establish the task code")
	}
	return nil
}

func (s *Service) selectVerifiedCandidate(ctx context.Context, selector CandidateSelector, code string, results []indexer.SelectionCandidate, cfg config.CandidateSelectionConfig, taskID string) (indexer.SelectionCandidate, preparedTorrent, error) {
	remaining := make([]indexer.SelectionCandidate, 0, len(results))
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
			return indexer.SelectionCandidate{}, preparedTorrent{}, err
		}
		result, err := selector.Select(ctx, code, remaining, cfg)
		if err != nil {
			if rejected != nil {
				err = rejected
			}
			return indexer.SelectionCandidate{}, preparedTorrent{}, err
		}
		prepared, err := s.prepareTorrent(ctx, code, candidateFromSearchResult(result), preferredTorrentURL(result))
		if err == nil {
			identity := torrentIdentityFromCandidate(prepared.candidate, preferredTorrentURL(result))
			err = s.ensureTaskIdentityAvailable(ctx, taskID, identity)
			if err == nil {
				return result, prepared, nil
			}
		}
		if errors.Is(err, indexer.ErrStateUnavailable) {
			return indexer.SelectionCandidate{}, preparedTorrent{}, err
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
	return indexer.SelectionCandidate{}, preparedTorrent{}, rejected
}

func (s *Service) blockSourcingValidation(ctx context.Context, task *Task, err error) error {
	code := TaskStageErrorTorrentMetadata
	var validation *candidateValidationError
	if errors.As(err, &validation) {
		code = validation.code
	}
	stage := TaskStageSourcing
	if task.SourcingMode == "USER_SELECTED" {
		stage = TaskStageDownloading
	}
	setTaskStage(task, stage, TaskStageStatusBlocked)
	if persistErr := s.recordSourcingFailure(ctx, task, code, err.Error()); persistErr != nil {
		return fmt.Errorf("persist sourcing validation failure: %w", persistErr)
	}
	return err
}
