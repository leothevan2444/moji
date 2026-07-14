package graphqlapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/leothevan2444/moji/internal/taskruntime"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	ErrorDuplicateTorrentTask = "DUPLICATE_TORRENT_TASK"
	ErrorDuplicateCodeTask    = "DUPLICATE_CODE_TASK"
	ErrorDuplicateLibraryCode = "DUPLICATE_LIBRARY_CODE"
	ErrorTaskCodeRequired     = "TASK_CODE_REQUIRED"
	ErrorTrackerNotConfigured = "TRACKER_NOT_CONFIGURED"
	ErrorDownloaderDisabled   = "DOWNLOADER_NOT_CONFIGURED"
	ErrorStashNotConfigured   = "STASH_NOT_CONFIGURED"
	ErrorScanPathRequired     = "SCAN_PATH_REQUIRED"
	ErrorTransferPathFailed   = "TRANSFER_PATH_FAILED"
	ErrorStashScanFailed      = "STASH_SCAN_FAILED"
	ErrorNoTorrentCandidate   = "NO_TORRENT_CANDIDATE"
	ErrorTorrentURLRequired   = "TORRENT_URL_REQUIRED"
	ErrorAddTorrentFailed     = "ADD_TORRENT_FAILED"
	ErrorInternal             = "INTERNAL_ERROR"
	ErrorTaskBatchEmpty       = "TASK_BATCH_EMPTY"
	ErrorTaskBatchTooLarge    = "TASK_BATCH_TOO_LARGE"
)

// ConfigureGraphQLServer installs the production error contract in one place.
func ConfigureGraphQLServer(server *handler.Server) {
	server.SetErrorPresenter(ErrorPresenter)
}

func NewGraphQLServer(schema graphql.ExecutableSchema) *handler.Server {
	server := handler.New(schema)
	server.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader:              websocket.Upgrader{CheckOrigin: SameOriginWebSocketRequest},
		ErrorFunc: func(_ context.Context, err error) {
			slog.Warn("graphql websocket transport error", "error", err)
		},
		CloseFunc: func(_ context.Context, code int) {
			slog.Info("graphql websocket connection closed", "code", code)
		},
	})
	server.AddTransport(transport.Options{})
	server.AddTransport(transport.GET{})
	server.AddTransport(transport.POST{})
	server.AddTransport(transport.MultipartForm{})
	server.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	server.Use(extension.Introspection{})
	server.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
	ConfigureGraphQLServer(server)
	return server
}

func SameOriginWebSocketRequest(request *http.Request) bool {
	origin := strings.TrimSpace(request.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, request.Host)
}

// ErrorPresenter exposes stable machine-readable codes while keeping root
// causes in server logs. User-facing clients should localize extensions.code.
func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	presented := graphql.DefaultErrorPresenter(ctx, err)
	code := classifyError(err)
	correlationID := uuid.NewString()

	slog.ErrorContext(ctx, "graphql request failed", "correlation_id", correlationID, "code", code, "error", err)
	presented.Message = "request failed"
	presented.Extensions = map[string]any{
		"code":          code,
		"params":        errorParams(err),
		"correlationId": correlationID,
	}
	return presented
}

func classifyError(err error) string {
	switch {
	case errors.Is(err, taskruntime.ErrDuplicateTorrentTask):
		return ErrorDuplicateTorrentTask
	case errors.Is(err, taskruntime.ErrDuplicateCodeTask):
		return ErrorDuplicateCodeTask
	case errors.Is(err, taskruntime.ErrDuplicateLibraryCode):
		return ErrorDuplicateLibraryCode
	case errors.Is(err, taskruntime.ErrTaskCodeRequired):
		return ErrorTaskCodeRequired
	case errors.Is(err, taskruntime.ErrTaskBatchEmpty):
		return ErrorTaskBatchEmpty
	case errors.Is(err, taskruntime.ErrTaskBatchTooLarge):
		return ErrorTaskBatchTooLarge
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "tracker is not configured"):
		return ErrorTrackerNotConfigured
	case strings.Contains(message, "qbittorrent client is required"), strings.Contains(message, "qbittorrent client is not configured"):
		return ErrorDownloaderDisabled
	case strings.Contains(message, "stash client is required"), strings.Contains(message, "stash client is not configured"):
		return ErrorStashNotConfigured
	case strings.Contains(message, "at least one scan path is required"):
		return ErrorScanPathRequired
	case strings.Contains(message, "resolve qb relative path failed"), strings.Contains(message, "build moji transfer source path failed"), strings.Contains(message, "build moji transfer target path failed"):
		return ErrorTransferPathFailed
	case strings.Contains(message, "trigger stash scan"), strings.Contains(message, "build stash scan path failed"):
		return ErrorStashScanFailed
	case strings.Contains(message, "no downloadable torrent candidate found"):
		return ErrorNoTorrentCandidate
	case strings.Contains(message, "torrent url is required"):
		return ErrorTorrentURLRequired
	case strings.Contains(message, "add torrent"):
		return ErrorAddTorrentFailed
	default:
		return ErrorInternal
	}
}

func errorParams(err error) map[string]any {
	// Params are deliberately conservative: never expose paths, URLs, upstream
	// responses, credentials, or raw SQL details to clients.
	return map[string]any{}
}
