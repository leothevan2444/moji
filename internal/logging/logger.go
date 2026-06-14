package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxEntries      = 500
	defaultMaxFileSize     = 10 * 1024 * 1024
	defaultMaxFileBackups  = 5
	defaultMinimumLogLevel = slog.LevelInfo
)

type Entry struct {
	Time    time.Time
	Level   string
	Message string
}

type Options struct {
	Level            string
	ConsoleWriter    io.Writer
	FilePath         string
	MaxEntries       int
	MaxFileSizeBytes int64
	MaxFileBackups   int
}

type Logger struct {
	logger   *slog.Logger
	levelVar *slog.LevelVar

	closers    []io.Closer
	mu         sync.RWMutex
	cache      []Entry
	maxEntries int
}

var (
	defaultMu     sync.RWMutex
	defaultLogger *Logger
)

func New(opts Options) (*Logger, error) {
	level := parseLevel(opts.Level)
	levelVar := &slog.LevelVar{}
	levelVar.Set(level)

	consoleWriter := opts.ConsoleWriter
	if consoleWriter == nil {
		consoleWriter = os.Stderr
	}

	maxEntries := opts.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}

	filePath := strings.TrimSpace(opts.FilePath)
	maxFileSizeBytes := opts.MaxFileSizeBytes
	if maxFileSizeBytes <= 0 {
		maxFileSizeBytes = defaultMaxFileSize
	}
	maxFileBackups := opts.MaxFileBackups
	if maxFileBackups <= 0 {
		maxFileBackups = defaultMaxFileBackups
	}

	l := &Logger{
		levelVar:   levelVar,
		cache:      make([]Entry, 0, maxEntries),
		maxEntries: maxEntries,
	}

	handlers := []slog.Handler{
		slog.NewTextHandler(consoleWriter, &slog.HandlerOptions{Level: levelVar}),
	}
	closers := make([]io.Closer, 0, 1)

	if filePath != "" {
		writer, err := newRollingFileWriter(filePath, maxFileSizeBytes, maxFileBackups)
		if err != nil {
			return nil, err
		}
		closers = append(closers, writer)
		handlers = append(handlers, slog.NewTextHandler(writer, &slog.HandlerOptions{Level: levelVar}))
	}

	cacheHandler := &cacheHandler{
		logger: l,
		next:   newMultiHandler(handlers...),
	}
	l.closers = closers
	l.logger = slog.New(cacheHandler)

	return l, nil
}

func SetDefault(logger *Logger) {
	defaultMu.Lock()
	prev := defaultLogger
	defaultLogger = logger
	defaultMu.Unlock()
	if prev != nil && prev != logger {
		_ = prev.Close()
	}
	if logger != nil {
		slog.SetDefault(logger.logger)
	}
}

func Default() *Logger {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

func ConfigureDefault(opts Options) (*Logger, error) {
	logger, err := New(opts)
	if err != nil {
		return nil, err
	}
	SetDefault(logger)
	return logger, nil
}

func (l *Logger) Slog() *slog.Logger {
	if l == nil {
		return nil
	}
	return l.logger
}

func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	var firstErr error
	for _, closer := range l.closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	l.closers = nil
	return firstErr
}

func (l *Logger) SetLevel(level string) {
	if l == nil || l.levelVar == nil {
		return
	}
	l.levelVar.Set(parseLevel(level))
}

func (l *Logger) Entries(limit int, minLevel string) []Entry {
	if l == nil {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	threshold := parseLevel(minLevel)
	out := make([]Entry, 0, len(l.cache))
	for _, entry := range l.cache {
		if parseLevel(entry.Level) < threshold {
			continue
		}
		out = append(out, entry)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (l *Logger) record(entry Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cache = append([]Entry{entry}, l.cache...)
	if len(l.cache) > l.maxEntries {
		l.cache = l.cache[:l.maxEntries]
	}
}

func (l *Logger) logf(level slog.Level, format string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Log(context.Background(), level, fmt.Sprintf(format, args...))
}

func (l *Logger) log(level slog.Level, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Log(context.Background(), level, fmt.Sprint(args...))
}

func Debugf(format string, args ...any) {
	if logger := Default(); logger != nil {
		logger.logf(slog.LevelDebug, format, args...)
	}
}
func Infof(format string, args ...any) {
	if logger := Default(); logger != nil {
		logger.logf(slog.LevelInfo, format, args...)
	}
}
func Warnf(format string, args ...any) {
	if logger := Default(); logger != nil {
		logger.logf(slog.LevelWarn, format, args...)
	}
}
func Errorf(format string, args ...any) {
	if logger := Default(); logger != nil {
		logger.logf(slog.LevelError, format, args...)
	}
}

func Debug(args ...any) {
	if logger := Default(); logger != nil {
		logger.log(slog.LevelDebug, args...)
	}
}
func Info(args ...any) {
	if logger := Default(); logger != nil {
		logger.log(slog.LevelInfo, args...)
	}
}
func Warn(args ...any) {
	if logger := Default(); logger != nil {
		logger.log(slog.LevelWarn, args...)
	}
}
func Error(args ...any) {
	if logger := Default(); logger != nil {
		logger.log(slog.LevelError, args...)
	}
}

func Fatalf(format string, args ...any) {
	Errorf(format, args...)
	os.Exit(1)
}

func Fatal(args ...any) {
	Error(args...)
	os.Exit(1)
}

func DefaultLogFilePath(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return "moji.log"
	}
	dir := filepath.Dir(configPath)
	if dir == "" || dir == "." {
		return "moji.log"
	}
	return filepath.Join(dir, "moji.log")
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return defaultMinimumLogLevel
	}
}

func formatLevel(level slog.Level) string {
	switch {
	case level <= slog.LevelDebug:
		return "debug"
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warn"
	default:
		return "info"
	}
}
