package service

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/core/usage"
	"zhi-ai-assistant/internal/platform"
	"zhi-ai-assistant/internal/winstate"
)

type settingsCommand struct {
	Action   string         `json:"action"`
	Settings model.Settings `json:"settings"`
	Enabled  bool           `json:"enabled,omitempty"`
	At       time.Time      `json:"at"`
}

const (
	actionPreview   = "preview"
	actionApply     = "apply"
	actionRevert    = "revert"
	actionQuit      = "quit"
	actionUsageAuto = "usageAuto"
)

func settingsCommandPath() string {
	return filepath.Join(store.DataDir(), "data", "settings.cmd.json")
}

func writeSettingsCommand(cmd settingsCommand) error {
	cmd.At = time.Now()
	raw, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	path := settingsCommandPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readSettingsCommand() (settingsCommand, error) {
	var cmd settingsCommand
	raw, err := os.ReadFile(settingsCommandPath())
	if err != nil {
		return cmd, err
	}
	err = json.Unmarshal(raw, &cmd)
	return cmd, err
}

func (a *App) onSettingsCommand() {
	cmd, err := readSettingsCommand()
	if err != nil {
		return
	}
	switch cmd.Action {
	case actionPreview:
		if a.window != nil {
			a.window.ApplySettings(normalizeSettings(cmd.Settings))
		}
		a.emit("settings:preview", normalizeSettings(cmd.Settings))
	case actionApply:
		saved, err := a.UpdateSettings(cmd.Settings)
		if err != nil {
			return
		}
		a.emit("settings:changed", saved)
	case actionRevert:
		a.window.ApplySettings(a.GetSettings())
		a.emit("settings:changed", a.GetSettings())
	case actionUsageAuto:
		if a.usageSvc == nil {
			return
		}
		a.usageSvc.SetAutoCheckinAll(cmd.Enabled)
	case actionQuit:
		a.Quit()
	}
}

func (a *App) Mode() string {
	if a.settingsOnly {
		return "settings"
	}
	return "sidebar"
}

func (a *App) SidebarRunning() bool {
	return platform.SidebarRunning()
}

func (a *App) Preview(s model.Settings) error {
	if !platform.SidebarRunning() {
		return nil
	}
	if err := writeSettingsCommand(settingsCommand{Action: actionPreview, Settings: s}); err != nil {
		return err
	}
	platform.NotifySettingsCommand()
	return nil
}

func (a *App) Apply(s model.Settings) (model.Settings, error) {
	s = normalizeSettings(s)
	if platform.SidebarRunning() {
		if err := writeSettingsCommand(settingsCommand{Action: actionApply, Settings: s}); err != nil {
			return s, err
		}
		if platform.NotifySettingsCommand() {
			return s, nil
		}
	}
	prev := a.GetSettings()
	a.saveSettings(s)
	if prev.Autostart != s.Autostart {
		_ = platform.SetAutostart(s.Autostart)
	}
	return s, nil
}

func (a *App) Revert() {
	if !platform.SidebarRunning() {
		return
	}
	if err := writeSettingsCommand(settingsCommand{Action: actionRevert, Settings: a.GetSettings()}); err != nil {
		return
	}
	platform.NotifySettingsCommand()
}

func (a *App) UsageAutoAll() bool {
	if a.usageSvc != nil {
		return a.usageSvc.AutoCheckinAll()
	}
	return usage.AutoCheckinAllFromDisk()
}

func (a *App) SetUsageAutoAll(enabled bool) error {
	if !platform.SidebarRunning() {
		return errors.New("侧边栏未运行，无法生效")
	}
	if err := writeSettingsCommand(settingsCommand{Action: actionUsageAuto, Enabled: enabled}); err != nil {
		return err
	}
	if !platform.NotifySettingsCommand() {
		return errors.New("无法通知侧边栏")
	}
	return nil
}

func (a *App) showSettingsWindow() {
	if a.ctx == nil || !a.settingsOnly {
		return
	}
	a.settingsShowOnce.Do(func() {
		runtime.WindowShow(a.ctx)
	})
}

func (a *App) FitSettingsWindow(_ int) {
	if a.ctx == nil || !a.settingsOnly {
		return
	}
	defer a.showSettingsWindow()
	width, height := settingsWindowSize()
	if width == a.fitWidth && height == a.fitHeight {
		return
	}
	a.fitWidth, a.fitHeight = width, height
	runtime.WindowSetSize(a.ctx, width, height)
	runtime.WindowCenter(a.ctx)
}

func settingsWindowSize() (int, int) {
	const baseWidth = 1120
	width := baseWidth
	scale := 1.0
	workW, workH := 0, 0
	monitors := platform.Monitors()
	for _, m := range monitors {
		if m.Primary {
			scale = float64(m.Dpi) / 96
			workW, workH = m.Work.W, m.Work.H
			break
		}
	}
	if workW == 0 && len(monitors) > 0 {
		m := monitors[0]
		scale = float64(m.Dpi) / 96
		workW, workH = m.Work.W, m.Work.H
	}
	if scale <= 0 {
		scale = 1
	}
	if workW > 0 {
		if maxW := int(float64(workW)/scale) - 24; width > maxW {
			width = maxW
		}
	}
	if workH > 0 {
		if maxW := (int(float64(workH)/scale) - 24) * 16 / 9; width > maxW {
			width = maxW
		}
	}
	return width, width * 9 / 16
}

func (a *App) OpenSettingsWindow() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "--settings")
	return cmd.Start()
}

func (a *App) CenterSettingsWindow() {
	if a.ctx == nil {
		return
	}
	const winW, winH = 1120, 630

	s := a.GetSettings()
	var mon winstate.Monitor
	found := false
	for _, m := range platform.Monitors() {
		if m.Index == s.TargetMonitor {
			mon, found = m, true
			break
		}
	}
	if !found {
		for _, m := range platform.Monitors() {
			if m.Primary {
				mon, found = m, true
				break
			}
		}
	}
	if !found {
		return
	}

	w, h := mon.Work.W, mon.Work.H
	if w < winW || h < winH {
		return
	}
	x := mon.Work.X + (w-winW)/2
	y := mon.Work.Y + (h-winH)/2
	runtime.WindowSetPosition(a.ctx, x, y)
}

func (a *App) CloseSettings() {
	if a.ctx != nil {
		a.quitRuntime()
	}
}
