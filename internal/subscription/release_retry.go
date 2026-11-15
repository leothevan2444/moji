package subscription

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"syscall"
	"time"

	"github.com/Yamashou/gqlgenc/clientv2"
	"github.com/leothevan2444/moji/internal/film"
	"github.com/leothevan2444/moji/internal/performer"
	"github.com/leothevan2444/moji/internal/releasedate"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

func temporaryReleaseFailure(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	for _, cause := range []error{syscall.ECONNRESET, syscall.ECONNREFUSED, syscall.ENETUNREACH, syscall.EHOSTUNREACH} {
		if errors.Is(err, cause) {
			return true
		}
	}
	var network net.Error
	if errors.As(err, &network) && (network.Timeout() || network.Temporary()) {
		return true
	}
	var response *clientv2.ErrorResponse
	if errors.As(err, &response) && response.NetworkError != nil {
		code := response.NetworkError.Code
		return code == 408 || code == 429 || code == 500 || code == 502 || code == 503 || code == 504
	}
	return false
}

// ProcessDueReviewRetries retries only explicitly scheduled transient failures.
// Manual checks can additionally retry failures requiring user intervention.
func (s *Service) ProcessDueReviewRetries(ctx context.Context) error {
	store, ok := s.reviews.(releaseRetryStore)
	if !ok {
		return nil
	}
	rows, err := store.DueReleaseRetries(ctx, s.now(), 8)
	if err != nil {
		return err
	}
	for _, candidate := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.reviewMu.Lock()
		err := func() error {
			defer s.reviewMu.Unlock()
			r, err := store.GetReleaseReview(ctx, candidate.Key)
			if err != nil {
				return err
			}
			if r == nil || r.Status != ReviewFailed || r.NextRetryAt == nil || r.NextRetryAt.After(s.now()) {
				return nil
			}
			active := false
			for _, p := range r.Performers {
				actor, err := s.stash.FindPerformerByID(ctx, p.ID)
				if err != nil {
					return err
				}
				if actor != nil && performer.IsSubscribed(actor.CustomFields, s.customFieldKey) {
					active = true
					break
				}
			}
			if !active {
				r.NextRetryAt = nil
				return s.reviews.PutReleaseReviews(ctx, []ReleaseReview{*r})
			}
			attemptCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			err = s.downloadReviewLocked(attemptCtx, r, true)
			cancel()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				s.failReleaseReview(r, err)
			} else {
				clearReleaseFailure(r)
			}
			if err := s.reviews.PutReleaseReviews(ctx, []ReleaseReview{*r}); err != nil {
				return err
			}
			for _, p := range r.Performers {
				s.publishEvent(PerformerSubscriptionEventUpdated, p.ID, nil)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func releaseFailureCode(err error) string {
	switch {
	case temporaryReleaseFailure(err):
		return "UPSTREAM_TEMPORARY"
	case errors.Is(err, context.Canceled):
		return "CANCELLED"
	case errors.Is(err, taskruntime.ErrDuplicateCodeTask):
		return "FILM_IDENTITY_CONFLICT"
	case errors.Is(err, film.ErrRequired):
		return "FILM_METADATA_INVALID"
	case errors.Is(err, releasedate.ErrNotReleased):
		return "RELEASE_NOT_AVAILABLE"
	case errors.Is(err, errNoMatchingStashBoxMapping):
		return "PERFORMER_MAPPING_MISSING"
	default:
		return "MANUAL_REVIEW_REQUIRED"
	}
}

func clearReleaseFailure(record *ReleaseReview) {
	record.Error, record.FailureCode = "", ""
	record.NextRetryAt = nil
	record.RetryAttempts = 0
}

func (s *Service) failReleaseReview(record *ReleaseReview, err error) {
	record.Status, record.Error, record.FailureCode = ReviewFailed, err.Error(), releaseFailureCode(err)
	var response *clientv2.ErrorResponse
	if errors.As(err, &response) && response.NetworkError != nil {
		record.Error = fmt.Sprintf("upstream HTTP %d", response.NetworkError.Code)
	}
	if len(record.Error) > 2048 {
		record.Error = record.Error[:2048] + "…"
	}
	record.RetryAttempts = min(record.RetryAttempts+1, 32)
	record.NextRetryAt = nil
	if temporaryReleaseFailure(err) {
		delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 24 * time.Hour}
		next := s.now().UTC().Add(delays[min(record.RetryAttempts-1, len(delays)-1)])
		record.NextRetryAt = &next
	}
}

func failedReviewsForPerformer(records map[string]ReleaseReview, id string) []string {
	keys := []string{}
	for key, r := range records {
		if r.Status != ReviewFailed {
			continue
		}
		for _, p := range r.Performers {
			if p.ID == id {
				keys = append(keys, key)
				break
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *Service) manualCheck(ctx context.Context) bool {
	id, _ := ctx.Value(checkRunContextKey{}).(string)
	s.checks.mu.Lock()
	defer s.checks.mu.Unlock()
	for _, r := range s.checks.runs {
		if r.ID == id {
			return r.Trigger == "MANUAL"
		}
	}
	return false
}

func (s *Service) retryCheckReleases(ctx context.Context, id string, keys []string, records, updates map[string]ReleaseReview, details *CheckDetails) error {
	if details == nil {
		return nil
	}
	details.Phase, details.RetryTotal = "HISTORICAL_RETRIES", len(keys)
	if err := s.saveCheckDetails(ctx, id, details); err != nil {
		return err
	}
	force := s.manualCheck(ctx)
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		r := records[key]
		if r.Status != ReviewFailed {
			if r.Status == ReviewDownloaded || r.Status == ReviewInLibrary {
				details.RetryAttempted++
				details.RetrySucceeded++
				clearReleaseFailure(&r)
				records[key], updates[key] = r, r
				details.add(r, "HISTORICAL_RETRY", string(r.Status))
			}
			continue
		}
		if details.RetryAttempted >= 32 || (!force && (r.NextRetryAt == nil || r.NextRetryAt.After(s.now()))) {
			details.RetryDeferred++
			details.add(r, "HISTORICAL_RETRY", "DEFERRED")
			continue
		}
		details.RetryAttempted++
		attemptCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		err := s.downloadReviewLocked(attemptCtx, &r, true)
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			s.failReleaseReview(&r, err)
			details.RetryFailed++
		} else {
			clearReleaseFailure(&r)
			details.RetrySucceeded++
		}
		// Each completed attempt is durable before the next external lookup. A
		// restarted check reconciles a previously accepted task by film identity.
		if err := s.reviews.PutReleaseReviews(ctx, []ReleaseReview{r}); err != nil {
			return err
		}
		records[key], updates[key] = r, r
		details.add(r, "HISTORICAL_RETRY", string(r.Status))
		if err := s.saveCheckDetails(ctx, id, details); err != nil {
			return err
		}
	}
	return nil
}
