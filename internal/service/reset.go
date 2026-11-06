package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"time"

	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/platform"
)

func (a *App) ResetAll() error {
	if !a.settingsOnly {
		return errors.New("请在设置窗口中执行重置")
	}
	if a.resetting {
		return errors.New("重置正在进行")
	}
	a.mu.Lock()
	a.resetBusy = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.resetBusy = false
		a.mu.Unlock()
	}()
	wasRunning := platform.SidebarRunning()
	if wasRunning {
		if err := writeSettingsCommand(settingsCommand{Action: actionQuit}); err != nil {
			return errors.New("无法通知侧边栏退出")
		}
		platform.NotifySettingsCommand()
		if !waitSidebarExit(8 * time.Second) {
			return errors.New("侧边栏尚未退出，重置已取消")
		}
		time.Sleep(500 * time.Millisecond)
	}
	a.resetting = true
	_ = platform.AbortShutdown()
	_ = platform.SetAutostart(false)
	if err := os.RemoveAll(store.DataDir()); err != nil {
		a.resetting = false
		if wasRunning {
			startSidebar()
		}
		return errors.New("删除数据失败，请关闭程序后手动清理数据目录")
	}
	exe, err := os.Executable()
	if err != nil {
		return errors.New("重置完成，但无法自动启动，请手动打开程序")
	}
	if err := exec.Command(exe).Start(); err != nil {
		return errors.New("重置完成，但自动启动失败，请手动打开程序")
	}
	return nil
}

func startSidebar() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	_ = exec.Command(exe).Start()
}

func (a *App) BeforeCloseSettings(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.resetBusy
}
