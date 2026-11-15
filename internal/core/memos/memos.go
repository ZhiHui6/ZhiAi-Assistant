package memos

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/id"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/i18n"
)

const filesDirName = "memo-files"

type Manager struct {
	doc      *store.Doc[model.MemosData]
	filesDir string
}

type Payload struct {
	Name string
	Data []byte
}

func New() (*Manager, error) {
	migrateLegacyData()
	doc, err := store.Open("memos.json", 1, model.MemosData{Items: []model.Memo{}})
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(store.DataRoot(), filesDirName)
	_ = os.MkdirAll(filesDir, 0o755)
	return &Manager{doc: doc, filesDir: filesDir}, nil
}

func migrateLegacyData() {
	base := store.DataRoot()
	oldPath := filepath.Join(base, "notes.json")
	newPath := filepath.Join(base, "memos.json")
	if _, err := os.Stat(newPath); err == nil {
		return
	}
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		if data, readErr := os.ReadFile(oldPath); readErr == nil {
			_ = os.WriteFile(newPath, data, 0o644)
		}
	}
	oldBak := oldPath + ".bak"
	newBak := newPath + ".bak"
	if _, err := os.Stat(newBak); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(oldBak); err == nil {
			_ = os.Rename(oldBak, newBak)
		}
	}
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) List(keyword string, limit int) []model.Memo {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	var out []model.Memo
	m.doc.Read(func(d *model.MemosData) {
		for _, n := range d.Items {
			if kw != "" && !match(n, kw) {
				continue
			}
			out = append(out, n)
		}
	})
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	if out == nil {
		out = []model.Memo{}
	}
	return out
}

func match(n model.Memo, kw string) bool {
	if strings.Contains(strings.ToLower(n.Text), kw) {
		return true
	}
	for _, t := range n.Tags {
		if strings.Contains(strings.ToLower(t), kw) {
			return true
		}
	}
	for _, a := range n.Attachments {
		if strings.Contains(strings.ToLower(a.Name), kw) || strings.Contains(strings.ToLower(a.URL), kw) {
			return true
		}
	}
	return false
}

func (m *Manager) Create(text string, tags []string) (model.Memo, error) {
	text = strings.TrimSpace(text)
	now := time.Now()
	n := model.Memo{
		ID:        id.New(),
		Text:      text,
		Tags:      normalizeTags(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.doc.Update(func(d *model.MemosData) {
		d.Items = append([]model.Memo{n}, d.Items...)
	})
	return n, nil
}

func normalizeTags(tags []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func normalizeTint(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if len(v) != 7 || v[0] != '#' {
		return ""
	}
	for _, c := range v[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return ""
		}
	}
	return strings.ToLower(v)
}

func (m *Manager) SetTint(memoID string, tint string) (model.Memo, error) {
	tint = normalizeTint(tint)
	var res model.Memo
	found := false
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			d.Items[i].Tint = tint
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.Memo{}, errors.New(i18n.T("速记不存在"))
	}
	return res, nil
}

func (m *Manager) Update(memoID string, text *string, tags *[]string, pinned *bool) (model.Memo, error) {
	var res model.Memo
	found := false
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			if text != nil {
				trimmed := strings.TrimSpace(*text)
				if trimmed == "" {
					d.Items[i].Text = ""
				} else {
					d.Items[i].Text = *text
				}
			}
			if tags != nil {
				d.Items[i].Tags = normalizeTags(*tags)
			}
			if pinned != nil {
				d.Items[i].Pinned = *pinned
			}
			d.Items[i].UpdatedAt = time.Now()
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.Memo{}, errors.New(i18n.T("速记不存在"))
	}
	return res, nil
}

func (m *Manager) Delete(memoID string) error {
	found := false
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID == memoID {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New(i18n.T("速记不存在"))
	}
	_ = os.RemoveAll(filepath.Join(m.filesDir, memoID))
	return nil
}

func (m *Manager) Get(memoID string) (model.Memo, error) {
	var res model.Memo
	found := false
	m.doc.Read(func(d *model.MemosData) {
		for _, n := range d.Items {
			if n.ID == memoID {
				res = n
				found = true
				return
			}
		}
	})
	if !found {
		return model.Memo{}, errors.New(i18n.T("速记不存在"))
	}
	return res, nil
}

func (m *Manager) Attachment(memoID, attachmentID string) (model.MemoAttachment, error) {
	memo, err := m.Get(memoID)
	if err != nil {
		return model.MemoAttachment{}, err
	}
	for _, a := range memo.Attachments {
		if a.ID == attachmentID {
			return a, nil
		}
	}
	return model.MemoAttachment{}, errors.New(i18n.T("附件不存在"))
}

func (m *Manager) AddFiles(memoID string, paths []string, maxBytes int64) (model.Memo, int, []string) {
	var added []model.MemoAttachment
	var skipped []string
	if _, err := m.Get(memoID); err != nil {
		return model.Memo{}, 0, []string{err.Error()}
	}
	dir := filepath.Join(m.filesDir, memoID)
	if len(paths) > 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return model.Memo{}, 0, []string{"无法创建附件目录"}
		}
	}
	for _, src := range paths {
		base := filepath.Base(strings.TrimSpace(src))
		info, err := os.Stat(src)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s：文件不存在", base))
			continue
		}
		if info.IsDir() {
			skipped = append(skipped, fmt.Sprintf("%s：暂不支持文件夹", base))
			continue
		}
		if maxBytes > 0 && info.Size() > maxBytes {
			skipped = append(skipped, fmt.Sprintf("%s：超过 %s 上限", base, sizeText(maxBytes)))
			continue
		}
		name := safeFileName(base)
		attID := id.New()
		rel := filepath.ToSlash(filepath.Join(memoID, attID+"_"+name))
		dst := filepath.Join(m.filesDir, filepath.FromSlash(rel))
		if err := copyFile(src, dst); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s：复制失败", base))
			continue
		}
		added = append(added, model.MemoAttachment{
			ID:   attID,
			Kind: kindOf(name),
			Name: name,
			Path: rel,
			Size: info.Size(),
		})
	}
	if len(added) == 0 {
		_ = os.Remove(dir)
		return m.mustGet(memoID), 0, skipped
	}
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			d.Items[i].Attachments = append(d.Items[i].Attachments, added...)
			d.Items[i].UpdatedAt = time.Now()
			return
		}
	})
	return m.mustGet(memoID), len(added), skipped
}

func (m *Manager) AddPayloads(memoID string, items []Payload, maxBytes int64) (model.Memo, int, []string) {
	var added []model.MemoAttachment
	var skipped []string
	if _, err := m.Get(memoID); err != nil {
		return model.Memo{}, 0, []string{err.Error()}
	}
	dir := filepath.Join(m.filesDir, memoID)
	if len(items) > 0 {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return model.Memo{}, 0, []string{i18n.T("无法创建附件目录")}
		}
	}
	for _, item := range items {
		name := safeFileName(item.Name)
		size := int64(len(item.Data))
		if size == 0 {
			skipped = append(skipped, fmt.Sprintf("%s：文件为空", name))
			continue
		}
		if maxBytes > 0 && size > maxBytes {
			skipped = append(skipped, fmt.Sprintf("%s：超过 %s 上限", name, sizeText(maxBytes)))
			continue
		}
		attID := id.New()
		rel := filepath.ToSlash(filepath.Join(memoID, attID+"_"+name))
		if err := os.WriteFile(filepath.Join(m.filesDir, filepath.FromSlash(rel)), item.Data, 0o644); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s：写入失败", name))
			continue
		}
		added = append(added, model.MemoAttachment{
			ID:   attID,
			Kind: kindOf(name),
			Name: name,
			Path: rel,
			Size: size,
		})
	}
	if len(added) == 0 {
		_ = os.Remove(dir)
		return m.mustGet(memoID), 0, skipped
	}
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			d.Items[i].Attachments = append(d.Items[i].Attachments, added...)
			d.Items[i].UpdatedAt = time.Now()
			return
		}
	})
	return m.mustGet(memoID), len(added), skipped
}

func (m *Manager) AddLink(memoID, raw string) (model.Memo, error) {
	link, err := normalizeLink(raw)
	if err != nil {
		return model.Memo{}, err
	}
	att := model.MemoAttachment{
		ID:   id.New(),
		Kind: "link",
		Name: linkName(link),
		URL:  link,
	}
	found := false
	var res model.Memo
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			d.Items[i].Attachments = append(d.Items[i].Attachments, att)
			d.Items[i].UpdatedAt = time.Now()
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.Memo{}, errors.New(i18n.T("速记不存在"))
	}
	return res, nil
}

func (m *Manager) RemoveAttachment(memoID, attachmentID string) (model.Memo, error) {
	var removed *model.MemoAttachment
	found := false
	var res model.Memo
	m.doc.Update(func(d *model.MemosData) {
		for i := range d.Items {
			if d.Items[i].ID != memoID {
				continue
			}
			for j := range d.Items[i].Attachments {
				if d.Items[i].Attachments[j].ID != attachmentID {
					continue
				}
				att := d.Items[i].Attachments[j]
				removed = &att
				d.Items[i].Attachments = append(d.Items[i].Attachments[:j], d.Items[i].Attachments[j+1:]...)
				found = true
				break
			}
			if found {
				d.Items[i].UpdatedAt = time.Now()
				res = d.Items[i]
			}
			return
		}
	})
	if !found {
		return model.Memo{}, errors.New(i18n.T("附件不存在"))
	}
	if removed != nil && removed.Kind != "link" && removed.Path != "" {
		_ = os.Remove(filepath.Join(m.filesDir, filepath.FromSlash(removed.Path)))
	}
	return res, nil
}

func (m *Manager) AttachmentFilePath(att model.MemoAttachment) (string, error) {
	if att.Kind == "link" || att.Path == "" {
		return "", errors.New(i18n.T("该附件不是本地文件"))
	}
	full := filepath.Join(m.filesDir, filepath.FromSlash(att.Path))
	rel, err := filepath.Rel(m.filesDir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New(i18n.T("附件路径无效"))
	}
	if _, err := os.Stat(full); err != nil {
		return "", errors.New(i18n.T("附件文件不存在"))
	}
	return full, nil
}

func (m *Manager) mustGet(memoID string) model.Memo {
	memo, err := m.Get(memoID)
	if err != nil {
		return model.Memo{ID: memoID}
	}
	return memo
}

func normalizeLink(raw string) (string, error) {
	link := strings.TrimSpace(raw)
	if link == "" {
		return "", errors.New(i18n.T("请输入链接"))
	}
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}
	if len(link) > 2048 {
		return "", errors.New(i18n.T("链接过长"))
	}
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return "", errors.New(i18n.T("链接格式不正确"))
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return "", errors.New(i18n.T("只支持 http 或 https 链接"))
	}
	return link, nil
}

func linkName(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return link
	}
	name := u.Host
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	if path != "" {
		name += path
	}
	if u.RawQuery != "" {
		name += "?" + u.RawQuery
	}
	return truncateRunes(name, 80)
}

func kindOf(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".heic":
		return "image"
	case ".mp3", ".wav", ".m4a", ".flac", ".ogg", ".aac", ".wma", ".opus":
		return "audio"
	case ".mp4", ".mkv", ".mov", ".avi", ".webm", ".wmv", ".flv", ".m4v":
		return "video"
	case ".txt", ".md", ".markdown", ".json", ".csv", ".tsv", ".log", ".ini", ".yaml", ".yml",
		".xml", ".html", ".htm", ".css", ".js", ".ts", ".go", ".py", ".java", ".c", ".h",
		".cpp", ".sh", ".bat", ".ps1", ".sql":
		return "text"
	}
	return "file"
}

func safeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(os.PathSeparator) {
		base = "file"
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stem = sanitizeRunes(stem)
	stem = strings.Trim(stem, " .")
	if stem == "" {
		stem = "file"
	}
	ext = sanitizeRunes(ext)
	if len(ext) > 16 {
		ext = ext[:16]
	}
	return truncateRunes(stem, 60) + ext
}

func sanitizeRunes(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if r < 0x20 {
			return -1
		}
		return r
	}, value)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func sizeText(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(mb))
	default:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(kb))
	}
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}
