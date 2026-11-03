package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerdownloads"
)

type preparedDownloads interface {
	Open(string) (*indexerdownloads.Lease, error)
}

func WithIndexerDownloads(downloads preparedDownloads) Option {
	return func(h *Handler) { h.downloads = downloads }
}

// This endpoint only serves an already acquired immutable artifact. It never
// uses a browser-supplied source URL, redirects upstream or starts a task.
func (h *Handler) handlePreparedDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if h.downloads == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: indexer.ErrNotConfigured.Error()})
		return
	}
	lease, err := h.downloads.Open(r.PathValue("handle"))
	if err != nil {
		status := http.StatusServiceUnavailable
		code := "INDEXER_DOWNLOAD_UNAVAILABLE"
		if errors.Is(err, indexer.ErrExpired) {
			status, code = http.StatusGone, indexer.ErrExpired.Error()
		}
		if errors.Is(err, indexer.ErrBusy) {
			status, code = http.StatusTooManyRequests, indexer.ErrBusy.Error()
		}
		writeJSON(w, status, errorResponse{Error: code})
		return
	}
	defer lease.Release()
	w.Header().Set("Content-Type", lease.ContentType)
	// Both names are constants owned by the download service, never a site title.
	w.Header().Set("Content-Disposition", `attachment; filename="`+lease.Filename+`"`)
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = io.Copy(w, lease.Reader)
}
