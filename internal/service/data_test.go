package service

import (
	"encoding/base64"
	"testing"

	"zhi-ai-assistant/internal/core/memos"
)

func TestMemosAddPayloads(t *testing.T) {
	t.Setenv("ZHIAI_ASSISTANT_DATA_DIR", t.TempDir())
	mgr, err := memos.New()
	if err != nil {
		t.Fatal(err)
	}
	memo, err := mgr.Create("paste", nil)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewMemos(mgr, func() int64 { return 1 << 20 })

	res, err := svc.AddPayloads(memo.ID, []MemoPayload{{Name: "a.png", Data: base64.StdEncoding.EncodeToString([]byte("abc"))}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || len(res.Memo.Attachments) != 1 || res.Memo.Attachments[0].Size != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}

	if _, err := svc.AddPayloads(memo.ID, []MemoPayload{{Name: "b.png", Data: "!!not-base64"}}); err == nil {
		t.Fatal("invalid base64 accepted")
	}
}
