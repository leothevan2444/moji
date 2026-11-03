package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/leothevan2444/moji/internal/indexerdefs"
)

var (
	ErrNotConfigured    = errors.New("INDEXER_NOT_CONFIGURED")
	ErrCatalogNotLoaded = errors.New("INDEXER_CATALOG_NOT_LOADED")
	ErrInvalidScope     = errors.New("INDEXER_SCOPE_INVALID")
	ErrInvalidQuery     = errors.New("INDEXER_QUERY_INVALID")
	ErrInvalidSettings  = errors.New("INDEXER_SETTINGS_INVALID")
	ErrUnsupported      = errors.New("INDEXER_CAPABILITY_UNSUPPORTED")
	ErrReference        = errors.New("INDEXER_RESOURCE_REFERENCE_INVALID")
	ErrExpired          = errors.New("INDEXER_RESOURCE_EXPIRED")
	ErrIncomplete       = errors.New("INDEXER_SEARCH_INCOMPLETE")
	ErrBusy             = errors.New("INDEXER_BUSY")
	ErrCoolingDown      = errors.New("INDEXER_COOLING_DOWN")
	ErrStateUnavailable = errors.New("INDEXER_STATE_UNAVAILABLE")
	ErrTransport        = errors.New("INDEXER_REQUEST_FAILED")
	ErrUnsafeURL        = errors.New("INDEXER_URL_NOT_ALLOWED")
	ErrResponseBudget   = errors.New("INDEXER_RESPONSE_TOO_LARGE")
	ErrAuthentication   = errors.New("INDEXER_AUTHENTICATION_FAILED")
	ErrSolver           = errors.New("INDEXER_SOLVER_FAILED")
	ErrChallenge        = errors.New("INDEXER_CHALLENGE_REQUIRED")
)

type CoolingDownError struct{ Until time.Time }

func (e *CoolingDownError) Error() string        { return ErrCoolingDown.Error() }
func (e *CoolingDownError) Is(target error) bool { return target == ErrCoolingDown }

func ErrorCode(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "INDEXER_TIMEOUT"
	}
	if errors.Is(err, context.Canceled) {
		return "INDEXER_CANCELLED"
	}
	for _, known := range []error{ErrCatalogNotLoaded, ErrNotConfigured, ErrInvalidScope, ErrInvalidQuery, ErrInvalidSettings, ErrUnsupported, ErrReference, ErrExpired, ErrIncomplete, ErrBusy, ErrCoolingDown, ErrStateUnavailable, ErrTransport, ErrUnsafeURL, ErrResponseBudget, ErrAuthentication, ErrChallenge, ErrSolver} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	for _, known := range []error{indexerdefs.ErrNotInstalled, indexerdefs.ErrUnavailable, indexerdefs.ErrInvalidBundle, indexerdefs.ErrIncompatible} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "INDEXER_OPERATION_FAILED"
}
