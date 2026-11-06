package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/platform"
)

type DataLocationInfo struct {
	Mode       string `json:"mode"`
	Custom     string `json:"custom"`
	Effective  string `json:"effective"`
	DefaultDir string `json:"defaultDir"`
	Portable   string `json:"portableDir"`
	FellBack   bool   `json:"fellBack"`
	Bytes      int64  `json:"bytes"`
}

func (a *App) DataLocation() DataLocationInfo {
	loc, _, fellBack := store.LocationStatus()
	dir := store.DataDir()
	return DataLocationInfo{
		Mode:       loc.Mode,
		Custom:     loc.Custom,
		Effective:  dir,
		DefaultDir: store.DefaultDir(),
		Portable:   store.PortableDir(),
		FellBack:   fellBack,
		Bytes:      store.DataTreeSize(dir),
	}
}

func (a *App) PickFolder(title string) (string, error) {
	return platform.BrowseFolder(title)
}

func dataLocationTarget(mode, custom string) (string, error) {
	switch mode {
	case store.LocationDefault:
		return store.DefaultDir(), nil
	case store.LocationPortable:
		dir := store.PortableDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", errors.New("无法在程序目录创建数据文件夹")
		}
		return dir, nil
	case store.LocationCustom:
		path := filepath.Clean(custom)
		if err := store.ValidateDir(path); err != nil {
			return "", errors.New("自定义路径需要是一个已存在且可写入的文件夹")
		}
		return path, nil
	default:
		return "", errors.New("未知的存储方式")
	}
}

func dataLocationRecord(mode, custom string) store.Location {
	if mode != store.LocationCustom {
		return store.Location{Mode: mode}
	}
	return store.Location{Mode: mode, Custom: filepath.Clean(custom)}
}

func (a *App) SetDataLocation(mode, custom string) error {
	if !a.settingsOnly {
		return errors.New("请在设置窗口中修改存储位置")
	}
	a.mu.Lock()
	if a.locationBusy {
		a.mu.Unlock()
		return errors.New("存储位置正在修改")
	}
	a.locationBusy = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.locationBusy = false
		a.mu.Unlock()
	}()

	target, err := dataLocationTarget(mode, custom)
	if err != nil {
		return err
	}
	loc := dataLocationRecord(mode, custom)
	current := store.DataDir()
	if filepath.Clean(current) == filepath.Clean(target) {
		return store.WriteLocation(loc)
	}

	wasRunning := platform.SidebarRunning()
	if wasRunning {
		if err := writeSettingsCommand(settingsCommand{Action: actionQuit}); err != nil {
			return errors.New("无法通知侧边栏退出")
		}
		platform.NotifySettingsCommand()
		if !waitSidebarExit(20 * time.Second) {
			return errors.New("侧边栏尚未退出，存储位置未修改")
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err := store.CopyDataTree(current, target); err != nil {
		if wasRunning {
			startSidebar()
		}
		return errors.New("复制数据失败，仍使用原位置：" + err.Error())
	}
	if err := store.WriteLocation(loc); err != nil {
		if wasRunning {
			startSidebar()
		}
		return errors.New("写入位置记录失败，仍使用原位置")
	}
	if wasRunning {
		exe, err := os.Executable()
		if err != nil {
			return errors.New("存储位置已修改，请手动重新打开程序")
		}
		if err := exec.Command(exe).Start(); err != nil {
			return errors.New("存储位置已修改，但自动启动失败，请手动重新打开程序")
		}
	}
	return nil
}
