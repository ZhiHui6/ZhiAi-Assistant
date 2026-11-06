package mount

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/id"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
)

type Manager struct {
	doc *store.Doc[model.MountsData]
}

type Entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	Ext     string    `json:"ext"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

type ListOptions struct {
	ShowHidden bool   `json:"showHidden"`
	SortBy     string `json:"sortBy"`
	Desc       bool   `json:"desc"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

func New() (*Manager, error) {
	doc, err := store.Open("mounts.json", 1, model.MountsData{Items: []model.Mount{}})
	if err != nil {
		return nil, err
	}
	return &Manager{doc: doc}, nil
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) List() []model.Mount {
	var out []model.Mount
	m.doc.Read(func(d *model.MountsData) {
		out = append([]model.Mount{}, d.Items...)
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	if out == nil {
		out = []model.Mount{}
	}
	return out
}

func (m *Manager) Add(path string) (model.Mount, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return model.Mount{}, errors.New("路径为空")
	}
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return model.Mount{}, errors.New("目录不存在")
	}
	exists := false
	var added model.Mount
	m.doc.Update(func(d *model.MountsData) {
		for _, it := range d.Items {
			if strings.EqualFold(it.Path, path) {
				exists = true
				added = it
				return
			}
		}
		added = model.Mount{
			ID:    id.New(),
			Path:  path,
			Label: filepath.Base(path),
			Order: len(d.Items),
		}
		d.Items = append(d.Items, added)
	})
	if exists {
		return added, nil
	}
	return added, nil
}

func (m *Manager) Remove(mountID string) error {
	found := false
	m.doc.Update(func(d *model.MountsData) {
		for i := range d.Items {
			if d.Items[i].ID == mountID {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("挂载项不存在")
	}
	return nil
}

func (m *Manager) Get(mountID string) (model.Mount, error) {
	var res model.Mount
	found := false
	m.doc.Read(func(d *model.MountsData) {
		for _, it := range d.Items {
			if it.ID == mountID {
				res = it
				found = true
				return
			}
		}
	})
	if !found {
		return model.Mount{}, errors.New("挂载项不存在")
	}
	return res, nil
}

func Children(root, rel string, o ListOptions) ([]Entry, error) {
	dir := root
	if rel != "" {
		dir = filepath.Join(root, rel)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.New("无法读取目录")
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !o.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if !o.ShowHidden && strings.EqualFold(name, "System Volume Information") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := ""
		if !e.IsDir() {
			ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
		}
		out = append(out, Entry{
			Name:    name,
			Path:    filepath.Join(dir, name),
			IsDir:   e.IsDir(),
			Ext:     ext,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sortEntries(out, o.SortBy, o.Desc)
	if o.Offset > 0 {
		if o.Offset >= len(out) {
			return []Entry{}, nil
		}
		out = out[o.Offset:]
	}
	if o.Limit > 0 && len(out) > o.Limit {
		out = out[:o.Limit]
	}
	return out, nil
}

func sortEntries(list []Entry, by string, desc bool) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		var less bool
		switch by {
		case "time":
			less = a.ModTime.After(b.ModTime)
		case "size":
			less = a.Size > b.Size
		default:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}
