package store

import (
	"testing"
)

func TestBackupCategoryForMemos(t *testing.T) {
	cases := map[string]string{
		"memos.json":                "memos",
		"notes.json":                "memos",
		"memo-files/m1/a_photo.png": "memos",
		"todos.json":                "todos",
	}
	for path, want := range cases {
		if got := categoryOf(path); got != want {
			t.Errorf("categoryOf(%q) = %q, want %q", path, got, want)
		}
	}
}
