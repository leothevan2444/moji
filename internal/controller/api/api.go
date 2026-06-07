package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/leothevan2444/moji/internal/tracker"
)

type Handler struct {
	tracker tracker.Tracker
}

func NewHandler(tr tracker.Tracker) *Handler {
	return &Handler{tracker: tr}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /api/tracker/search", h.handleTrackerSearch)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

type errorResponse struct {
	Error string `json:"error"`
}

type trackerSearchResponse struct {
	Results any `json:"results"`
}

func (h *Handler) handleTrackerSearch(w http.ResponseWriter, r *http.Request) {
	markDeprecatedRESTDebugEndpoint(w)

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "missing query parameter: q"})
		return
	}

	var options []tracker.SearchOption

	if raw := strings.TrimSpace(r.URL.Query().Get("trackers")); raw != "" {
		trackers := splitCSV(raw)
		if len(trackers) > 0 {
			options = append(options, tracker.WithTrackers(trackers))
		}
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("categories")); raw != "" {
		categories, err := parseCSVInts(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid categories: " + err.Error()})
			return
		}
		if len(categories) > 0 {
			options = append(options, tracker.WithCategories(categories))
		}
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid limit"})
			return
		}
		if limit > 0 {
			options = append(options, tracker.WithLimit(limit))
		}
	}

	results, err := h.tracker.Search(q, options...)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, trackerSearchResponse{Results: results})
}

func markDeprecatedRESTDebugEndpoint(w http.ResponseWriter) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</graphql>; rel="successor-version"`)
	w.Header().Set("Warning", `299 - "REST debug endpoint is deprecated; use GraphQL"`)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseCSVInts(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}
