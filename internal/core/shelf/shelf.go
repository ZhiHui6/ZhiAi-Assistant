package shelf

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

var appExts = map[string]bool{
	".exe": true, ".lnk": true, ".bat": true, ".cmd": true,
	".msi": true, ".appref-ms": true, ".ps1": true, ".url": true,
}

type Manager struct {
	doc *store.Doc[model.ShelfData]
}

func New() (*Manager, error) {
	doc, err := store.Open("shelf.json", 1, model.ShelfData{
		Groups: []model.ShelfGroup{},
		Items:  []model.ShelfItem{},
	})
	if err != nil {
		return nil, err
	}
	m := &Manager{doc: doc}
	m.ensureDefaultGroup()
	return m, nil
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) ensureDefaultGroup() {
	has := false
	m.doc.Read(func(d *model.ShelfData) { has = len(d.Groups) > 0 })
	if has {
		return
	}
	m.doc.Update(func(d *model.ShelfData) {
		d.Groups = append(d.Groups, model.ShelfGroup{ID: id.New(), Name: "常用", Order: 0})
	})
}

func (m *Manager) List() model.ShelfData {
	var out model.ShelfData
	m.doc.Read(func(d *model.ShelfData) {
		out.Groups = append([]model.ShelfGroup{}, d.Groups...)
		out.Items = append([]model.ShelfItem{}, d.Items...)
	})
	sort.SliceStable(out.Groups, func(i, j int) bool { return out.Groups[i].Order < out.Groups[j].Order })
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].Order < out.Items[j].Order })
	if out.Groups == nil {
		out.Groups = []model.ShelfGroup{}
	}
	if out.Items == nil {
		out.Items = []model.ShelfItem{}
	}
	return out
}

func kindOf(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "url"
	}
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return "dir"
	}
	ext := strings.ToLower(filepath.Ext(path))
	if appExts[ext] {
		return "app"
	}
	return "file"
}

func displayName(path string) string {
	if strings.HasPrefix(path, "http") {
		return path
	}
	base := filepath.Base(strings.TrimRight(path, `\/`))
	ext := filepath.Ext(base)
	if ext != "" && !strings.EqualFold(ext, base) {
		return strings.TrimSuffix(base, ext)
	}
	return base
}

func (m *Manager) AddPaths(groupID string, paths []string) ([]model.ShelfItem, error) {
	if len(paths) == 0 {
		return nil, errors.New("没有可添加的项目")
	}
	var added []model.ShelfItem
	m.doc.Update(func(d *model.ShelfData) {
		if groupID == "" {
			if len(d.Groups) == 0 {
				d.Groups = append(d.Groups, model.ShelfGroup{ID: id.New(), Name: "常用", Order: 0})
			}
			groupID = d.Groups[0].ID
		}
		order := 0
		for _, it := range d.Items {
			if it.GroupID == groupID && it.Order >= order {
				order = it.Order + 1
			}
		}
		seen := map[string]bool{}
		for _, it := range d.Items {
			seen[strings.ToLower(it.Path)+"|"+it.GroupID] = true
		}
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if seen[strings.ToLower(p)+"|"+groupID] {
				continue
			}
			item := model.ShelfItem{
				ID:      id.New(),
				GroupID: groupID,
				Kind:    kindOf(p),
				Path:    p,
				Name:    displayName(p),
				Order:   order,
				AddedAt: time.Now(),
			}
			order++
			d.Items = append(d.Items, item)
			added = append(added, item)
		}
	})
	if len(added) == 0 {
		return nil, errors.New("这些项目已经在收纳区里了")
	}
	return added, nil
}

func (m *Manager) Remove(itemID string) error {
	found := false
	m.doc.Update(func(d *model.ShelfData) {
		for i := range d.Items {
			if d.Items[i].ID == itemID {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("项目不存在")
	}
	return nil
}

func (m *Manager) Update(itemID string, name, args, workDir *string) (model.ShelfItem, error) {
	var res model.ShelfItem
	found := false
	m.doc.Update(func(d *model.ShelfData) {
		for i := range d.Items {
			if d.Items[i].ID != itemID {
				continue
			}
			if name != nil && strings.TrimSpace(*name) != "" {
				d.Items[i].Name = strings.TrimSpace(*name)
			}
			if args != nil {
				d.Items[i].Args = *args
			}
			if workDir != nil {
				d.Items[i].WorkDir = *workDir
			}
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.ShelfItem{}, errors.New("项目不存在")
	}
	return res, nil
}

func (m *Manager) Move(itemID, toGroupID string, toIndex int) error {
	found := false
	m.doc.Update(func(d *model.ShelfData) {
		idx := -1
		for i := range d.Items {
			if d.Items[i].ID == itemID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		groupOK := false
		for _, g := range d.Groups {
			if g.ID == toGroupID {
				groupOK = true
				break
			}
		}
		if !groupOK {
			return
		}

		moved := d.Items[idx]
		d.Items = append(d.Items[:idx], d.Items[idx+1:]...)
		moved.GroupID = toGroupID

		var target []model.ShelfItem
		var rest []model.ShelfItem
		for _, it := range d.Items {
			if it.GroupID == toGroupID {
				target = append(target, it)
			} else {
				rest = append(rest, it)
			}
		}
		sort.SliceStable(target, func(i, j int) bool { return target[i].Order < target[j].Order })
		if toIndex < 0 {
			toIndex = 0
		}
		if toIndex > len(target) {
			toIndex = len(target)
		}
		target = append(target, model.ShelfItem{})
		copy(target[toIndex+1:], target[toIndex:])
		target[toIndex] = moved
		for i := range target {
			target[i].Order = i
		}

		sort.SliceStable(rest, func(i, j int) bool {
			if rest[i].GroupID != rest[j].GroupID {
				return rest[i].GroupID < rest[j].GroupID
			}
			return rest[i].Order < rest[j].Order
		})
		counter := map[string]int{}
		for i := range rest {
			g := rest[i].GroupID
			rest[i].Order = counter[g]
			counter[g]++
		}

		d.Items = append(target, rest...)
		found = true
	})
	if !found {
		return errors.New("项目或分组不存在")
	}
	return nil
}

func (m *Manager) CreateGroup(name string) (model.ShelfGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.ShelfGroup{}, errors.New("分组名不能为空")
	}
	g := model.ShelfGroup{ID: id.New(), Name: name}
	m.doc.Update(func(d *model.ShelfData) {
		g.Order = len(d.Groups)
		d.Groups = append(d.Groups, g)
	})
	return g, nil
}

func (m *Manager) RenameGroup(groupID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("分组名不能为空")
	}
	found := false
	m.doc.Update(func(d *model.ShelfData) {
		for i := range d.Groups {
			if d.Groups[i].ID == groupID {
				d.Groups[i].Name = name
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("分组不存在")
	}
	return nil
}

func (m *Manager) ToggleGroup(groupID string, collapsed bool) error {
	m.doc.Update(func(d *model.ShelfData) {
		for i := range d.Groups {
			if d.Groups[i].ID == groupID {
				d.Groups[i].Collapsed = collapsed
				return
			}
		}
	})
	return nil
}

func (m *Manager) DeleteGroup(groupID string) error {
	found := false
	m.doc.Update(func(d *model.ShelfData) {
		for i := range d.Groups {
			if d.Groups[i].ID != groupID {
				continue
			}
			d.Groups = append(d.Groups[:i], d.Groups[i+1:]...)
			kept := d.Items[:0]
			for _, it := range d.Items {
				if it.GroupID != groupID {
					kept = append(kept, it)
				}
			}
			d.Items = kept
			found = true
			return
		}
	})
	if !found {
		return errors.New("分组不存在")
	}
	return nil
}

func (m *Manager) Get(itemID string) (model.ShelfItem, error) {
	var res model.ShelfItem
	found := false
	m.doc.Read(func(d *model.ShelfData) {
		for _, it := range d.Items {
			if it.ID == itemID {
				res = it
				found = true
				return
			}
		}
	})
	if !found {
		return model.ShelfItem{}, errors.New("项目不存在")
	}
	return res, nil
}
