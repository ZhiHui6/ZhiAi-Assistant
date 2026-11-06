package notes

import (
	"errors"
	"sort"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/id"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
)

type Manager struct {
	doc *store.Doc[model.NotesData]
}

func New() (*Manager, error) {
	doc, err := store.Open("notes.json", 1, model.NotesData{Items: []model.Note{}})
	if err != nil {
		return nil, err
	}
	return &Manager{doc: doc}, nil
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) List(keyword string, limit int) []model.Note {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	var out []model.Note
	m.doc.Read(func(d *model.NotesData) {
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
		out = []model.Note{}
	}
	return out
}

func match(n model.Note, kw string) bool {
	if strings.Contains(strings.ToLower(n.Text), kw) {
		return true
	}
	for _, t := range n.Tags {
		if strings.Contains(strings.ToLower(t), kw) {
			return true
		}
	}
	return false
}

func (m *Manager) Create(text string, tags []string) (model.Note, error) {
	text = strings.TrimSpace(text)
	now := time.Now()
	n := model.Note{
		ID:        id.New(),
		Text:      text,
		Tags:      normalizeTags(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.doc.Update(func(d *model.NotesData) {
		d.Items = append([]model.Note{n}, d.Items...)
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

func (m *Manager) SetTint(id string, tint string) (model.Note, error) {
	tint = normalizeTint(tint)
	var res model.Note
	found := false
	m.doc.Update(func(d *model.NotesData) {
		for i := range d.Items {
			if d.Items[i].ID != id {
				continue
			}
			d.Items[i].Tint = tint
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.Note{}, errors.New("记事不存在")
	}
	return res, nil
}

func (m *Manager) Update(id string, text *string, tags *[]string, pinned *bool) (model.Note, error) {
	var res model.Note
	found := false
	m.doc.Update(func(d *model.NotesData) {
		for i := range d.Items {
			if d.Items[i].ID != id {
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
		return model.Note{}, errors.New("记事不存在")
	}
	return res, nil
}

func (m *Manager) Delete(id string) error {
	found := false
	m.doc.Update(func(d *model.NotesData) {
		for i := range d.Items {
			if d.Items[i].ID == id {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("记事不存在")
	}
	return nil
}

func (m *Manager) Get(id string) (model.Note, error) {
	var res model.Note
	found := false
	m.doc.Read(func(d *model.NotesData) {
		for _, n := range d.Items {
			if n.ID == id {
				res = n
				found = true
				return
			}
		}
	})
	if !found {
		return model.Note{}, errors.New("记事不存在")
	}
	return res, nil
}
