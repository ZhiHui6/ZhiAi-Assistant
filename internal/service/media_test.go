package service

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMediaPath(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "memo-files", "a.txt")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inside, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outside, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, ok := resolveMediaPath(root, inside); !ok || got != inside {
		t.Fatalf("inside file rejected: %q %v", got, ok)
	}
	if got, ok := resolveMediaPath(root, filepath.Join("memo-files", "a.txt")); !ok || got != inside {
		t.Fatalf("relative path rejected: %q %v", got, ok)
	}
	for _, raw := range []string{"", "  ", filepath.Dir(root), outside, filepath.Join(root, "..", "outside.txt"), filepath.Join(root, "missing.txt"), "../outside.txt"} {
		if _, ok := resolveMediaPath(root, raw); ok {
			t.Fatalf("unexpected accept: %q", raw)
		}
	}
}

func TestMediaHandlerServesMemoFile(t *testing.T) {
	base := t.TempDir()
	t.Setenv("ZHIAI_ASSISTANT_DATA_DIR", base)
	target := filepath.Join(base, "data", memoFilesDir, "note.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	serve := func(rawURL string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		MediaHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, rawURL, nil))
		return rec
	}

	ok := serve(mediaRoute + "?p=" + url.QueryEscape(target))
	if ok.Code != http.StatusOK || ok.Body.String() != "hello" {
		t.Fatalf("serve memo file: %d %q", ok.Code, ok.Body.String())
	}
	if got := serve("/other"); got.Code != http.StatusNotFound {
		t.Fatalf("unknown route: %d", got.Code)
	}
	if got := serve(mediaRoute); got.Code != http.StatusNotFound {
		t.Fatalf("missing param: %d", got.Code)
	}
	if got := serve(mediaRoute + "?p=" + url.QueryEscape(filepath.Join(base, "data", "settings.json"))); got.Code != http.StatusNotFound {
		t.Fatalf("outside memo-files: %d", got.Code)
	}
}
