package memos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKindOf(t *testing.T) {
	cases := map[string]string{
		"a.PNG":     "image",
		"voice.mp3": "audio",
		"clip.mp4":  "video",
		"notes.md":  "text",
		"data.bin":  "file",
	}
	for name, want := range cases {
		if got := kindOf(name); got != want {
			t.Errorf("kindOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSafeFileName(t *testing.T) {
	got := safeFileName(`..\evil:name?.txt`)
	if strings.ContainsAny(got, `\/:*?"<>|`) {
		t.Fatalf("safeFileName left illegal characters: %q", got)
	}
	if !strings.HasSuffix(got, ".txt") {
		t.Fatalf("safeFileName dropped extension: %q", got)
	}
	long := safeFileName(strings.Repeat("a", 120) + ".md")
	if len([]rune(long)) > 64 {
		t.Fatalf("safeFileName did not truncate: %d runes", len([]rune(long)))
	}
}

func TestNormalizeLink(t *testing.T) {
	link, err := normalizeLink("example.com/a?b=1")
	if err != nil {
		t.Fatalf("normalizeLink returned error: %v", err)
	}
	if link != "https://example.com/a?b=1" {
		t.Fatalf("normalizeLink = %q", link)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "ftp://example.com", "http://"} {
		if _, err := normalizeLink(bad); err == nil {
			t.Errorf("normalizeLink(%q) should fail", bad)
		}
	}
}

func TestAttachmentLifecycle(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("ZHIAI_ASSISTANT_DATA_DIR", dataDir)
	mgr, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	memo, err := mgr.Create("hello", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "photo.png")
	if err := os.WriteFile(src, []byte("image-bytes"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	updated, added, skipped := mgr.AddFiles(memo.ID, []string{src}, 1024)
	if added != 1 || len(skipped) != 0 {
		t.Fatalf("AddFiles added=%d skipped=%v", added, skipped)
	}
	if len(updated.Attachments) != 1 || updated.Attachments[0].Kind != "image" {
		t.Fatalf("unexpected attachments: %+v", updated.Attachments)
	}
	path, err := mgr.AttachmentFilePath(updated.Attachments[0])
	if err != nil {
		t.Fatalf("AttachmentFilePath: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "image-bytes" {
		t.Fatalf("copied file mismatch: %q %v", data, err)
	}

	if _, _, skipped := mgr.AddFiles(memo.ID, []string{src}, 4); len(skipped) != 1 {
		t.Fatalf("oversized file should be skipped, got %v", skipped)
	}

	if _, err := mgr.RemoveAttachment(memo.ID, updated.Attachments[0].ID); err != nil {
		t.Fatalf("RemoveAttachment: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("attachment file still exists: %v", err)
	}

	linked, err := mgr.AddLink(memo.ID, "example.com")
	if err != nil || len(linked.Attachments) != 1 || linked.Attachments[0].Kind != "link" {
		t.Fatalf("AddLink: %+v %v", linked, err)
	}
	if err := mgr.Delete(memo.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(mgr.filesDir, memo.ID)); !os.IsNotExist(err) {
		t.Fatalf("memo attachment dir still exists: %v", err)
	}
}

func TestMigrateLegacyNotes(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("ZHIAI_ASSISTANT_DATA_DIR", dataDir)
	dataPath := filepath.Join(dataDir, "data")
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	legacy := `{"schema":1,"data":{"items":[{"id":"old","text":"legacy","pinned":false,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}]}}`
	if err := os.WriteFile(filepath.Join(dataPath, "notes.json"), []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	mgr, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	items := mgr.List("", 0)
	if len(items) != 1 || items[0].Text != "legacy" {
		t.Fatalf("legacy data not migrated: %+v", items)
	}
	if _, err := os.Stat(filepath.Join(dataPath, "notes.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy notes.json still present")
	}
}

func TestAddPayloads(t *testing.T) {
	t.Setenv("ZHIAI_ASSISTANT_DATA_DIR", t.TempDir())
	m, err := New()
	if err != nil {
		t.Fatal(err)
	}
	memo, err := m.Create("paste test", nil)
	if err != nil {
		t.Fatal(err)
	}

	updated, added, skipped := m.AddPayloads(memo.ID, []Payload{{Name: "截图.png", Data: []byte("png-bytes")}}, 1<<20)
	if added != 1 || len(skipped) != 0 || len(updated.Attachments) != 1 {
		t.Fatalf("added=%d skipped=%v attachments=%d", added, skipped, len(updated.Attachments))
	}
	att := updated.Attachments[0]
	if att.Kind != "image" || att.Size != int64(len("png-bytes")) || att.Path == "" {
		t.Fatalf("unexpected attachment: %+v", att)
	}
	full := filepath.Join(m.filesDir, filepath.FromSlash(att.Path))
	if data, err := os.ReadFile(full); err != nil || string(data) != "png-bytes" {
		t.Fatalf("stored file = %q err=%v", data, err)
	}

	_, added, skipped = m.AddPayloads(memo.ID, []Payload{{Name: "big.bin", Data: make([]byte, 2048)}}, 1024)
	if added != 0 || len(skipped) != 1 {
		t.Fatalf("size limit not enforced: added=%d skipped=%v", added, skipped)
	}
}
