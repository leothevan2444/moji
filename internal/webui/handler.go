package webui

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

const (
	notBuiltMessage   = "moji web ui is not built; run `make web-build` or `make web-dev`\n"
	immutableCaching  = "public, max-age=31536000, immutable"
	revalidateCaching = "no-cache"
	minCompressSize   = 1024
)

var hashedAssetPattern = regexp.MustCompile(`-[A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`)

type compressedCacheKey struct {
	path     string
	encoding string
	size     int64
	modified int64
}

type Handler struct {
	distDir    string
	compressed sync.Map
}

func NewHandler(distDir string) *Handler {
	return &Handler{distDir: distDir}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(notBuiltMessage))
		return
	}

	requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	filePath := filepath.Join(h.distDir, filepath.FromSlash(requestPath))
	info, err := os.Stat(filePath)
	if requestPath == "" || err != nil || info.IsDir() {
		filePath = indexPath
		info, err = os.Stat(indexPath)
	}

	if isImmutableAsset(requestPath) && filePath != indexPath {
		w.Header().Set("Cache-Control", immutableCaching)
	} else {
		w.Header().Set("Cache-Control", revalidateCaching)
	}

	if err != nil {
		http.ServeFile(w, r, filePath)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	encoding := ""
	if info.Size() >= minCompressSize && contentType != "" && isCompressible(contentType) {
		appendVary(w.Header(), "Accept-Encoding")
		encoding = negotiateEncoding(r.Header.Get("Accept-Encoding"))
	}

	var content []byte
	if encoding != "" {
		key := compressedCacheKey{path: filePath, encoding: encoding, size: info.Size(), modified: info.ModTime().UnixNano()}
		if cached, ok := h.compressed.Load(key); ok {
			content = cached.([]byte)
		} else {
			source, readErr := os.ReadFile(filePath)
			if readErr != nil {
				http.ServeFile(w, r, filePath)
				return
			}
			encoded, encodeErr := h.compressedContent(key, source)
			if encodeErr == nil {
				content = encoded
			} else {
				content = source
				encoding = ""
			}
		}
	} else {
		content, err = os.ReadFile(filePath)
		if err != nil {
			http.ServeFile(w, r, filePath)
			return
		}
	}

	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	w.Header().Set("Content-Type", contentType)
	if encoding != "" {
		w.Header().Set("Content-Encoding", encoding)
	}

	http.ServeContent(w, r, filepath.Base(filePath), info.ModTime(), bytes.NewReader(content))
}

func (h *Handler) compressedContent(key compressedCacheKey, content []byte) ([]byte, error) {
	if cached, ok := h.compressed.Load(key); ok {
		return cached.([]byte), nil
	}

	var output bytes.Buffer
	var writer io.WriteCloser
	if key.encoding == "br" {
		writer = brotli.NewWriterLevel(&output, 5)
	} else {
		gzipWriter, err := gzip.NewWriterLevel(&output, gzip.BestCompression)
		if err != nil {
			return nil, err
		}
		writer = gzipWriter
	}
	if _, err := writer.Write(content); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	encoded := output.Bytes()
	h.compressed.Store(key, encoded)
	return encoded, nil
}

func isImmutableAsset(requestPath string) bool {
	return strings.HasPrefix(requestPath, "assets/") && hashedAssetPattern.MatchString(requestPath)
}

func isCompressible(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	return strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/javascript" ||
		mediaType == "application/json" ||
		mediaType == "application/manifest+json" ||
		mediaType == "application/wasm" ||
		mediaType == "application/xml" ||
		mediaType == "image/svg+xml"
}

func negotiateEncoding(header string) string {
	brQuality := encodingQuality(header, "br")
	gzipQuality := encodingQuality(header, "gzip")
	if brQuality > 0 && brQuality >= gzipQuality {
		return "br"
	}
	if gzipQuality > 0 {
		return "gzip"
	}
	return ""
}

func encodingQuality(header, encoding string) float64 {
	quality := -1.0
	wildcardQuality := -1.0
	for _, value := range strings.Split(header, ",") {
		parts := strings.Split(value, ";")
		name := strings.ToLower(strings.TrimSpace(parts[0]))
		q := 1.0
		for _, parameter := range parts[1:] {
			key, raw, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				q = 0
			} else {
				q = parsed
			}
		}
		if name == encoding {
			quality = q
		} else if name == "*" {
			wildcardQuality = q
		}
	}
	if quality >= 0 {
		return quality
	}
	if wildcardQuality >= 0 {
		return wildcardQuality
	}
	return 0
}

func appendVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, item := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
