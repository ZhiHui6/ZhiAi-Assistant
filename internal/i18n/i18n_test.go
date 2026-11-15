package i18n

import "testing"

func TestTranslate(t *testing.T) {
	Set("en")
	if got := T("退出"); got != "Quit" {
		t.Fatalf("en T(退出) = %q", got)
	}
	if got := T("标题不能为空"); got != "Title cannot be empty" {
		t.Fatalf("en T(error) = %q", got)
	}
	if got := TF("写入不完整: %s", "a.txt"); got != "Write incomplete: a.txt" {
		t.Fatalf("en TF = %q", got)
	}

	Set("zh-TW")
	if got := T("退出"); got != "結束" {
		t.Fatalf("zh-TW T(退出) = %q", got)
	}
	if got := T("标题不能为空"); got != "標題不能為空" {
		t.Fatalf("zh-TW T(error) = %q", got)
	}

	Set("zh-CN")
	if got := T("退出"); got != "退出" {
		t.Fatalf("zh-CN T = %q", got)
	}
	if got := T("不存在的词条"); got != "不存在的词条" {
		t.Fatalf("fallback T = %q", got)
	}

	Set("fr")
	if got := Current(); got != "zh-CN" {
		t.Fatalf("Current after invalid locale = %q", got)
	}
}
