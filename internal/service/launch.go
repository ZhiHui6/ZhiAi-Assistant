package service

import (
	"errors"
	"os"
	"os/exec"

	"zhi-ai-assistant/internal/core/launch"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/platform"
)

type Launch struct {
	mgr *launch.Launcher
}

func NewLaunch(mgr *launch.Launcher) *Launch { return &Launch{mgr: mgr} }

func (s *Launch) List() model.LaunchData { return s.mgr.List() }

func (s *Launch) AddPaths(categoryID string, paths []string) ([]model.LaunchShortcut, error) {
	return s.mgr.AddPaths(categoryID, paths)
}

func (s *Launch) Remove(itemID string) error { return s.mgr.Remove(itemID) }

func (s *Launch) Rename(itemID, name string) error { return s.mgr.Rename(itemID, name) }

func (s *Launch) Move(itemID, categoryID string) error { return s.mgr.Move(itemID, categoryID) }

func (s *Launch) AddCategory(name string) (model.LaunchCategory, error) {
	return s.mgr.AddCategory(name)
}

func (s *Launch) RenameCategory(categoryID, name string) error {
	return s.mgr.RenameCategory(categoryID, name)
}

func (s *Launch) RemoveCategory(categoryID string) error {
	return s.mgr.RemoveCategory(categoryID)
}

func (s *Launch) Pick(categoryID string) ([]model.LaunchShortcut, error) {
	paths, err := openFileDialog("选择要添加的程序", true)
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	return s.mgr.AddPaths(categoryID, paths)
}

func (s *Launch) Launch(itemID string) error {
	it, err := s.mgr.Get(itemID)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(it.Path); statErr != nil {
		return errors.New("程序不存在: " + it.Path)
	}
	return platform.ShellExecute(0, "open", it.Path, "", "", 1)
}

func (s *Launch) Reveal(itemID string) error {
	it, err := s.mgr.Get(itemID)
	if err != nil {
		return err
	}
	cmd := exec.Command("explorer.exe", "/select,"+it.Path)
	return cmd.Start()
}
