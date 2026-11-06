package launch

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

var programExts = map[string]bool{
	".exe": true, ".lnk": true, ".bat": true, ".cmd": true,
	".com": true, ".msi": true, ".appref-ms": true,
}

type Launcher struct {
	doc *store.Doc[model.LaunchData]
}

func New() (*Launcher, error) {
	migrateLegacyFile()
	doc, err := store.Open("launch.json", 1, model.LaunchData{
		Categories: []model.LaunchCategory{},
		Items:      []model.LaunchShortcut{},
	})
	if err != nil {
		return nil, err
	}
	m := &Launcher{doc: doc}
	m.ensureCategories()
	return m, nil
}

func migrateLegacyFile() {
	dir := filepath.Join(store.DataDir(), "data")
	target := filepath.Join(dir, "launch.json")
	if _, err := os.Stat(target); err == nil {
		return
	}
	legacy := filepath.Join(dir, "apps.json")
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	_ = os.Rename(legacy, target)
}

func (m *Launcher) Flush() error { return m.doc.Flush() }

func (m *Launcher) ensureCategories() {
	m.doc.Update(func(d *model.LaunchData) {
		sort.SliceStable(d.Categories, func(i, j int) bool { return d.Categories[i].Order < d.Categories[j].Order })
		legacy := -1
		for i, c := range d.Categories {
			if c.LegacySystem {
				legacy = i
				break
			}
		}
		if legacy >= 0 {
			old := d.Categories[legacy]
			rest := make([]model.LaunchCategory, 0, len(d.Categories)-1)
			for i, c := range d.Categories {
				if i != legacy {
					rest = append(rest, c)
				}
			}
			if len(rest) == 0 {
				old.Name = "常用"
				d.Categories = []model.LaunchCategory{old}
			} else {
				for i := range d.Items {
					if d.Items[i].CategoryID == old.ID {
						d.Items[i].CategoryID = rest[0].ID
					}
				}
				d.Categories = rest
			}
		}
		if len(d.Categories) == 0 {
			d.Categories = append(d.Categories, model.LaunchCategory{ID: id.New(), Name: "常用"})
		}
		for i := range d.Categories {
			d.Categories[i].Order = i
			d.Categories[i].LegacySystem = false
		}
		valid := map[string]bool{}
		for _, c := range d.Categories {
			valid[c.ID] = true
		}
		first := d.Categories[0].ID
		for i := range d.Items {
			if !valid[d.Items[i].CategoryID] {
				d.Items[i].CategoryID = first
			}
			if strings.TrimSpace(d.Items[i].BaseName) == "" {
				d.Items[i].BaseName = displayName(d.Items[i].Path)
			}
			if strings.TrimSpace(d.Items[i].Name) == "" {
				d.Items[i].Name = d.Items[i].BaseName
			}
		}
		if d.Items == nil {
			d.Items = []model.LaunchShortcut{}
		}
		seen := map[string]bool{}
		kept := make([]model.LaunchShortcut, 0, len(d.Items))
		for _, it := range d.Items {
			key := strings.ToLower(it.Path)
			if seen[key] {
				continue
			}
			seen[key] = true
			kept = append(kept, it)
		}
		d.Items = kept
	})
}

func (m *Launcher) List() model.LaunchData {
	var out model.LaunchData
	m.doc.Read(func(d *model.LaunchData) {
		out.Categories = append([]model.LaunchCategory{}, d.Categories...)
		out.Items = append([]model.LaunchShortcut{}, d.Items...)
	})
	sort.SliceStable(out.Categories, func(i, j int) bool { return out.Categories[i].Order < out.Categories[j].Order })
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].Order < out.Items[j].Order })
	if out.Categories == nil {
		out.Categories = []model.LaunchCategory{}
	}
	if out.Items == nil {
		out.Items = []model.LaunchShortcut{}
	}
	return out
}

func (m *Launcher) Get(itemID string) (model.LaunchShortcut, error) {
	var res model.LaunchShortcut
	found := false
	m.doc.Read(func(d *model.LaunchData) {
		for _, it := range d.Items {
			if it.ID == itemID {
				res = it
				found = true
				return
			}
		}
	})
	if !found {
		return model.LaunchShortcut{}, errors.New("程序不存在")
	}
	return res, nil
}

func isProgram(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return programExts[strings.ToLower(filepath.Ext(path))]
}

func displayName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" && !strings.EqualFold(ext, base) {
		return strings.TrimSuffix(base, ext)
	}
	return base
}

func (m *Launcher) AddPaths(categoryID string, paths []string) ([]model.LaunchShortcut, error) {
	var added []model.LaunchShortcut
	rejected := 0
	noTarget := false
	m.doc.Update(func(d *model.LaunchData) {
		target := ""
		for _, c := range d.Categories {
			if c.ID == categoryID {
				target = c.ID
				break
			}
		}
		if target == "" && len(d.Categories) > 0 {
			target = d.Categories[0].ID
		}
		if target == "" {
			noTarget = true
			return
		}
		categoryID = target
		seen := map[string]bool{}
		for _, it := range d.Items {
			seen[strings.ToLower(it.Path)] = true
		}
		order := 0
		for _, it := range d.Items {
			if it.CategoryID == categoryID && it.Order >= order {
				order = it.Order + 1
			}
		}
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !isProgram(p) || seen[strings.ToLower(p)] {
				rejected++
				continue
			}
			item := model.LaunchShortcut{
				ID:         id.New(),
				CategoryID: categoryID,
				Path:       p,
				Name:       displayName(p),
				BaseName:   displayName(p),
				Order:      order,
				AddedAt:    time.Now(),
			}
			order++
			seen[strings.ToLower(p)] = true
			d.Items = append(d.Items, item)
			added = append(added, item)
		}
	})
	if noTarget {
		return nil, errors.New("请先创建一个分类再添加程序")
	}
	if len(added) == 0 {
		if rejected > 0 {
			return nil, errors.New("只支持程序或快捷方式，且不能重复")
		}
		return nil, errors.New("没有可添加的程序")
	}
	return added, nil
}

func (m *Launcher) Remove(itemID string) error {
	found := false
	m.doc.Update(func(d *model.LaunchData) {
		for i := range d.Items {
			if d.Items[i].ID == itemID {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("程序不存在")
	}
	return nil
}

func (m *Launcher) Rename(itemID, name string) error {
	name = strings.TrimSpace(name)
	found := false
	m.doc.Update(func(d *model.LaunchData) {
		for i := range d.Items {
			if d.Items[i].ID != itemID {
				continue
			}
			found = true
			if name != "" {
				d.Items[i].Name = name
				return
			}
			base := strings.TrimSpace(d.Items[i].BaseName)
			if base == "" {
				base = displayName(d.Items[i].Path)
			}
			d.Items[i].BaseName = base
			d.Items[i].Name = base
			return
		}
	})
	if !found {
		return errors.New("程序不存在")
	}
	return nil
}

func (m *Launcher) Move(itemID, categoryID string) error {
	found := false
	m.doc.Update(func(d *model.LaunchData) {
		valid := false
		for _, c := range d.Categories {
			if c.ID == categoryID {
				valid = true
				break
			}
		}
		if !valid {
			return
		}
		order := 0
		for _, it := range d.Items {
			if it.CategoryID == categoryID && it.Order >= order {
				order = it.Order + 1
			}
		}
		for i := range d.Items {
			if d.Items[i].ID == itemID {
				d.Items[i].CategoryID = categoryID
				d.Items[i].Order = order
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("程序不存在或分类无效")
	}
	return nil
}

func (m *Launcher) AddCategory(name string) (model.LaunchCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.LaunchCategory{}, errors.New("分类名不能为空")
	}
	dup := false
	category := model.LaunchCategory{ID: id.New(), Name: name}
	m.doc.Update(func(d *model.LaunchData) {
		for _, c := range d.Categories {
			if strings.EqualFold(c.Name, name) {
				dup = true
				return
			}
		}
		category.Order = len(d.Categories)
		d.Categories = append(d.Categories, category)
	})
	if dup {
		return model.LaunchCategory{}, errors.New("分类已存在")
	}
	return category, nil
}

func (m *Launcher) RenameCategory(categoryID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("分类名不能为空")
	}
	found := false
	m.doc.Update(func(d *model.LaunchData) {
		for _, c := range d.Categories {
			if c.ID != categoryID && strings.EqualFold(c.Name, name) {
				return
			}
		}
		for i := range d.Categories {
			if d.Categories[i].ID == categoryID {
				d.Categories[i].Name = name
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("分类不存在或名称重复")
	}
	return nil
}

func (m *Launcher) RemoveCategory(categoryID string) error {
	if len(m.List().Categories) <= 1 {
		return errors.New("至少保留一个分类")
	}
	found := false
	m.doc.Update(func(d *model.LaunchData) {
		idx := -1
		for i, c := range d.Categories {
			if c.ID == categoryID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		d.Categories = append(d.Categories[:idx], d.Categories[idx+1:]...)
		if len(d.Categories) == 0 {
			found = true
			return
		}
		target := d.Categories[0].ID
		order := 0
		for _, it := range d.Items {
			if it.CategoryID == target && it.Order >= order {
				order = it.Order + 1
			}
		}
		for i := range d.Items {
			if d.Items[i].CategoryID == categoryID {
				d.Items[i].CategoryID = target
				d.Items[i].Order = order
				order++
			}
		}
		for i := range d.Categories {
			d.Categories[i].Order = i
		}
		found = true
	})
	if !found {
		return errors.New("分类不存在")
	}
	return nil
}
