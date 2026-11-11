package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/leothevan2444/moji/internal/taskruntime"
)

// HomeStatsProvider reads local counts without loading task details or upstreams.
type HomeStatsProvider interface {
	HomeStats(context.Context, time.Time) (taskruntime.HomeStats, error)
}

func WithHomeStats(provider HomeStatsProvider) Option {
	return func(h *Handler) { h.homeStats = provider }
}

type homepageResponse struct {
	Attention            int `json:"attention"`
	Active               int `json:"active"`
	Waiting              int `json:"waiting"`
	CompletedLast24Hours int `json:"completedLast24Hours"`
}

type widgetError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func (h *Handler) handleHomepage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.homeStats == nil {
		writeWidgetError(w, r, http.StatusServiceUnavailable, "WIDGET_UNAVAILABLE", nil)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	stats, err := h.homeStats.HomeStats(ctx, time.Now())
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			writeWidgetError(w, r, http.StatusGatewayTimeout, "WIDGET_QUERY_TIMEOUT", err)
			return
		}
		writeWidgetError(w, r, http.StatusInternalServerError, "WIDGET_QUERY_FAILED", err)
		return
	}
	writeJSON(w, http.StatusOK, homepageResponse{Attention: stats.Attention, Active: stats.Active, Waiting: stats.Waiting, CompletedLast24Hours: stats.CompletedLast24Hours})
}

func writeWidgetError(w http.ResponseWriter, r *http.Request, status int, code string, err error) {
	correlationID := uuid.NewString()
	slog.ErrorContext(r.Context(), "homepage widget request failed", "code", code, "correlation_id", correlationID, "error", err)
	writeJSON(w, status, struct {
		Error widgetError `json:"error"`
	}{
		Error: widgetError{Code: code, Message: "Unable to read Moji widget statistics.", CorrelationID: correlationID},
	})
}
