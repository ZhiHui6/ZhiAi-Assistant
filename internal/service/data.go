package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/festival"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/mount"
	"zhi-ai-assistant/internal/core/notes"
	"zhi-ai-assistant/internal/core/shelf"
	"zhi-ai-assistant/internal/core/todos"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
)

type Notes struct {
	mgr *notes.Manager
}

func NewNotes(mgr *notes.Manager) *Notes { return &Notes{mgr: mgr} }

func (s *Notes) List(keyword string, limit int) []model.Note {
	return s.mgr.List(keyword, limit)
}

func (s *Notes) Create(text string, tags []string) (model.Note, error) {
	return s.mgr.Create(text, tags)
}

func (s *Notes) Update(id string, text *string, tags *[]string, pinned *bool) (model.Note, error) {
	return s.mgr.Update(id, text, tags, pinned)
}

func (s *Notes) SetTint(id string, tint string) (model.Note, error) {
	return s.mgr.SetTint(id, tint)
}

func (s *Notes) Delete(id string) error { return s.mgr.Delete(id) }

func (s *Notes) Get(id string) (model.Note, error) { return s.mgr.Get(id) }

type Todos struct {
	mgr *todos.Manager
}

func NewTodos(mgr *todos.Manager) *Todos { return &Todos{mgr: mgr} }

func (s *Todos) List(f todos.Filter) []model.Todo { return s.mgr.List(f) }

func (s *Todos) Create(in todos.Input) (model.Todo, error) { return s.mgr.Create(in) }

func (s *Todos) Update(id string, p todos.Patch) (model.Todo, error) { return s.mgr.Update(id, p) }

func (s *Todos) Toggle(id string, done bool) (model.Todo, error) { return s.mgr.Toggle(id, done) }

func (s *Todos) Delete(id string) error { return s.mgr.Delete(id) }

func (s *Todos) Restore(t model.Todo) (model.Todo, error) { return s.mgr.Restore(t) }

func (s *Todos) Reorder(ids []string) error { return s.mgr.Reorder(ids) }

func (s *Todos) Stats() todos.Stats { return s.mgr.Stats() }

type Calendar struct {
	mgr *todos.Manager
}

func NewCalendar(mgr *todos.Manager) *Calendar { return &Calendar{mgr: mgr} }

func (s *Calendar) Month(year, month int) []todos.DaySummary {
	return s.mgr.MonthSummary(year, month)
}

func (s *Calendar) Day(date string) []model.Todo { return s.mgr.ForDate(date) }

func (s *Calendar) Festivals(year, month int) map[string]festival.Day {
	return festival.Month(year, month)
}

func (s *Calendar) Today() festival.Day {
	return festival.Of(time.Now())
}

type Shelf struct {
	mgr *shelf.Manager
}

func NewShelf(mgr *shelf.Manager) *Shelf { return &Shelf{mgr: mgr} }

func (s *Shelf) List() model.ShelfData { return s.mgr.List() }

func (s *Shelf) AddPaths(groupID string, paths []string) ([]model.ShelfItem, error) {
	return s.mgr.AddPaths(groupID, paths)
}

func (s *Shelf) Remove(itemID string) error { return s.mgr.Remove(itemID) }

func (s *Shelf) Update(itemID string, name, args, workDir *string) (model.ShelfItem, error) {
	return s.mgr.Update(itemID, name, args, workDir)
}

func (s *Shelf) Move(itemID, toGroupID string, toIndex int) error {
	return s.mgr.Move(itemID, toGroupID, toIndex)
}

func (s *Shelf) CreateGroup(name string) (model.ShelfGroup, error) { return s.mgr.CreateGroup(name) }

func (s *Shelf) RenameGroup(groupID, name string) error { return s.mgr.RenameGroup(groupID, name) }

func (s *Shelf) ToggleGroup(groupID string, collapsed bool) error {
	return s.mgr.ToggleGroup(groupID, collapsed)
}

func (s *Shelf) DeleteGroup(groupID string) error { return s.mgr.DeleteGroup(groupID) }

func (s *Shelf) Launch(itemID string) error {
	it, err := s.mgr.Get(itemID)
	if err != nil {
		return err
	}
	if it.Kind == "url" {
		return platform.ShellExecute(0, "open", it.Path, "", "", 1)
	}
	if _, statErr := os.Stat(it.Path); statErr != nil {
		return errors.New("目标不存在: " + it.Path)
	}
	if it.Kind == "dir" || it.Kind == "file" {
		return platform.ShellExecute(0, "open", it.Path, "", "", 1)
	}
	return platform.ShellExecute(0, "open", it.Path, it.Args, it.WorkDir, 1)
}

func (s *Shelf) Reveal(itemID string) error {
	it, err := s.mgr.Get(itemID)
	if err != nil {
		return err
	}
	if it.Kind == "url" {
		return errors.New("链接无法定位")
	}
	cmd := exec.Command("explorer.exe", "/select,"+it.Path)
	return cmd.Start()
}

func (s *Shelf) PickFiles(groupID string) ([]model.ShelfItem, error) {
	paths, err := openFileDialog(i18n.T("选择要收纳的文件"), true)
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	return s.mgr.AddPaths(groupID, paths)
}

func (s *Shelf) PickFolder(groupID string) ([]model.ShelfItem, error) {
	path, err := openFolderDialog(i18n.T("选择要收纳的文件夹"))
	if err != nil || path == "" {
		return nil, err
	}
	return s.mgr.AddPaths(groupID, []string{path})
}

type Mount struct {
	mgr *mount.Manager
}

func NewMount(mgr *mount.Manager) *Mount { return &Mount{mgr: mgr} }

func (s *Mount) List() []model.Mount { return s.mgr.List() }

func (s *Mount) Add(path string) (model.Mount, error) { return s.mgr.Add(path) }

func (s *Mount) Pick() (model.Mount, error) {
	path, err := openFolderDialog("选择要挂载的文件夹")
	if err != nil {
		return model.Mount{}, err
	}
	if path == "" {
		return model.Mount{}, errors.New("未选择文件夹")
	}
	return s.mgr.Add(path)
}

func (s *Mount) Remove(mountID string) error { return s.mgr.Remove(mountID) }

func (s *Mount) Children(mountID, relPath string, opts mount.ListOptions) ([]mount.Entry, error) {
	m, err := s.mgr.Get(mountID)
	if err != nil {
		return nil, err
	}
	return mount.Children(m.Path, relPath, opts)
}

func (s *Mount) Open(mountID, relPath string) error {
	m, err := s.mgr.Get(mountID)
	if err != nil {
		return err
	}
	target := m.Path
	if relPath != "" {
		target = filepath.Join(m.Path, relPath)
	}
	return platform.ShellExecute(0, "open", target, "", "", 1)
}

func (s *Mount) Reveal(mountID, relPath string) error {
	m, err := s.mgr.Get(mountID)
	if err != nil {
		return err
	}
	target := m.Path
	if relPath != "" {
		target = filepath.Join(m.Path, relPath)
	}
	return exec.Command("explorer.exe", "/select,"+target).Start()
}

type Icons struct{}

func NewIcons() *Icons { return &Icons{} }

func (s *Icons) Get(paths []string) map[string]string {
	out := make(map[string]string, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if url := platform.IconDataURL(p); url != "" {
			out[strings.ToLower(p)] = url
		}
	}
	return out
}

func (s *Icons) GetOne(path string) string { return platform.IconDataURL(path) }

func (s *Icons) ClearCache() { platform.ClearIconCache() }

func openFileDialog(title string, multi bool) ([]string, error) {
	return platform.OpenFileDialog(title, multi)
}

func openFolderDialog(title string) (string, error) {
	return platform.BrowseFolder(title)
}
