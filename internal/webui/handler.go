package webui

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const notBuiltMessage = "moji web ui is not built; run `make web-build` or `make web-dev`\n"

type Handler struct {
	distDir string
}

func NewHandler(distDir string) Handler {
	return Handler{distDir: distDir}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(notBuiltMessage))
		return
	}

	requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestPath == "" {
		http.ServeFile(w, r, indexPath)
		return
	}

	filePath := filepath.Join(h.distDir, filepath.FromSlash(requestPath))
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}

	http.ServeFile(w, r, indexPath)
}
