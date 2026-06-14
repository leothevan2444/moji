package logging

import (
	"context"
	"log/slog"
	"time"
)

type cacheHandler struct {
	logger *Logger
	next   slog.Handler
}

func (h *cacheHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *cacheHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.logger != nil {
		timestamp := record.Time
		if timestamp.IsZero() {
			timestamp = time.Now()
		}
		h.logger.record(Entry{
			Time:    timestamp,
			Level:   formatLevel(record.Level),
			Message: record.Message,
		})
	}
	return h.next.Handle(ctx, record)
}

func (h *cacheHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &cacheHandler{
		logger: h.logger,
		next:   h.next.WithAttrs(attrs),
	}
}

func (h *cacheHandler) WithGroup(name string) slog.Handler {
	return &cacheHandler{
		logger: h.logger,
		next:   h.next.WithGroup(name),
	}
}

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}
	return &multiHandler{handlers: next}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}
	return &multiHandler{handlers: next}
}
