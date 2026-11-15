package service

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"zhi-ai-assistant/internal/core/store"
)

const (
	mediaRoute   = "/memo-media"
	memoFilesDir = "memo-files"
)

func MediaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != mediaRoute {
			http.NotFound(w, r)
			return
		}
		root := filepath.Join(store.DataRoot(), memoFilesDir)
		target, ok := resolveMediaPath(root, r.URL.Query().Get("p"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, target)
	})
}

func resolveMediaPath(root, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsRune(raw, 0) {
		return "", false
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	target := raw
	if !filepath.IsAbs(target) {
		target = filepath.Join(base, target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", false
	}
	return abs, true
}
