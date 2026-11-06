package service

import (
	"errors"
	"os"
	"os/exec"
	"time"

	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
)

type BackupResult struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
}

type BackupCategoryResult struct {
	ID    string `json:"id"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
}

type BackupContentsResult struct {
	Categories []BackupCategoryResult `json:"categories"`
	Legacy     bool                   `json:"legacy"`
}

func backupResult(info store.BackupInfo) BackupResult {
	return BackupResult{Path: info.Path, Bytes: info.Bytes, Files: info.Files}
}

func (a *App) CreateBackupTo(dir string, categories []string) (BackupResult, error) {
	var selected []string
	if categories != nil {
		selected = categories
	}
	info, err := store.CreateBackup(dir, selected)
	if err != nil {
		return BackupResult{}, err
	}
	return backupResult(info), nil
}

func (a *App) BackupContents(zipPath string) (BackupContentsResult, error) {
	contents, err := store.ReadBackupContents(zipPath)
	if err != nil {
		return BackupContentsResult{}, err
	}
	out := BackupContentsResult{Legacy: contents.Legacy, Categories: make([]BackupCategoryResult, 0, len(contents.Categories))}
	for _, c := range contents.Categories {
		out.Categories = append(out.Categories, BackupCategoryResult{ID: c.ID, Files: c.Files, Bytes: c.Bytes})
	}
	return out, nil
}

func (a *App) PickBackupFile() (string, error) {
	paths, err := platform.OpenFileDialog(i18n.T("选择要导入的备份"), false)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 || paths[0] == "" {
		return "", errors.New("未选择备份文件")
	}
	return paths[0], nil
}

func (a *App) RestoreBackupData(zipPath string, categories []string) error {
	if !a.settingsOnly {
		return errors.New("请在设置窗口中导入备份")
	}
	a.mu.Lock()
	if a.restoreBusy || a.resetBusy || a.locationBusy {
		a.mu.Unlock()
		return errors.New("正在处理数据，请稍候")
	}
	a.restoreBusy = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.restoreBusy = false
		a.mu.Unlock()
	}()

	if _, _, err := store.VerifyBackup(zipPath); err != nil {
		return err
	}

	wasRunning := platform.SidebarRunning()
	if wasRunning {
		if err := writeSettingsCommand(settingsCommand{Action: actionQuit}); err != nil {
			return errors.New("无法通知侧边栏退出")
		}
		platform.NotifySettingsCommand()
		if !waitSidebarExit(20 * time.Second) {
			return errors.New("侧边栏尚未退出，导入已取消")
		}
		time.Sleep(500 * time.Millisecond)
	}

	a.restoring = true
	var selected []string
	if categories != nil {
		selected = categories
	}
	if _, err := store.RestoreBackup(zipPath, selected); err != nil {
		a.restoring = false
		if wasRunning {
			startSidebar()
		}
		return err
	}
	if wasRunning {
		exe, err := os.Executable()
		if err != nil {
			return errors.New("数据已导入，请手动重新打开程序")
		}
		if err := exec.Command(exe).Start(); err != nil {
			return errors.New("数据已导入，但自动启动失败，请手动重新打开程序")
		}
	}
	return nil
}
