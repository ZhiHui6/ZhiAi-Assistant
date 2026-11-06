package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/core/store"
)

type updateCacheState struct {
	AutoCheck   *bool  `json:"autoCheck,omitempty"`
	Dismissed   string `json:"dismissed,omitempty"`
	Latest      string `json:"latest,omitempty"`
	PageURL     string `json:"pageUrl,omitempty"`
	Notes       string `json:"notes,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
	CanInstall  bool   `json:"canInstall,omitempty"`
}

type settingsNavRequest struct {
	Page string `json:"page"`
	At   int64  `json:"at"`
}

func updateCachePath() string {
	return filepath.Join(store.DataDir(), "cache", "update.json")
}

func settingsNavPath() string {
	return filepath.Join(store.DataDir(), "data", "settings.nav.json")
}

func readJSONFile(path string, out interface{}) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func writeJSONFile(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readUpdateCache() updateCacheState {
	var state updateCacheState
	_ = readJSONFile(updateCachePath(), &state)
	return state
}

func writeUpdateCache(state updateCacheState) error {
	return writeJSONFile(updateCachePath(), state)
}

func (a *App) GetAutoUpdateCheck() bool {
	state := readUpdateCache()
	return state.AutoCheck == nil || *state.AutoCheck
}

func (a *App) SetAutoUpdateCheck(enabled bool) error {
	state := readUpdateCache()
	state.AutoCheck = &enabled
	return writeUpdateCache(state)
}

func (a *App) UpdateNotice() *UpdateInfo {
	state := readUpdateCache()
	current := normalizeVersion(buildinfo.Version)
	latest := normalizeVersion(state.Latest)
	if latest == "" || latest == normalizeVersion(state.Dismissed) {
		return nil
	}
	if current != "" && current != "dev" && compareVersions(latest, current) <= 0 {
		return nil
	}
	return &UpdateInfo{
		Current:     current,
		Latest:      latest,
		HasUpdate:   true,
		CanInstall:  state.CanInstall,
		Notes:       state.Notes,
		PageURL:     state.PageURL,
		PublishedAt: state.PublishedAt,
	}
}

func (a *App) DismissUpdate(version string) error {
	state := readUpdateCache()
	state.Dismissed = normalizeVersion(version)
	if err := writeUpdateCache(state); err != nil {
		return err
	}
	a.updateMu.Lock()
	if a.pendingUpdate != nil && normalizeVersion(a.pendingUpdate.Latest) == state.Dismissed {
		a.pendingUpdate = nil
		a.updateNotified = false
	}
	a.updateMu.Unlock()
	return nil
}

func (a *App) loadPendingUpdate() *UpdateInfo {
	notice := a.UpdateNotice()
	a.updateMu.Lock()
	a.pendingUpdate = notice
	if notice != nil {
		a.updateNotified = true
	}
	a.updateMu.Unlock()
	return notice
}

func (a *App) publishUpdate(info *UpdateInfo) {
	if info == nil || info.Latest == "" {
		return
	}
	a.updateMu.Lock()
	if a.updateNotified && a.pendingUpdate != nil && a.pendingUpdate.Latest == info.Latest {
		a.updateMu.Unlock()
		return
	}
	a.pendingUpdate = info
	a.updateNotified = true
	a.updateMu.Unlock()
	a.emit("update:available", info)
}

func (a *App) startUpdateWatch() {
	if a.settingsOnly {
		return
	}
	a.updateMu.Lock()
	if a.updateWatchStarted {
		a.updateMu.Unlock()
		return
	}
	a.updateWatchStarted = true
	a.updateMu.Unlock()
	go func() {
		time.Sleep(15 * time.Second)
		a.updateWatchTick()
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			a.updateWatchTick()
		}
	}()
}

func (a *App) updateWatchTick() {
	if !a.GetAutoUpdateCheck() {
		return
	}
	if notice := a.UpdateNotice(); notice != nil {
		a.publishUpdate(notice)
		return
	}
	info, err := a.CheckUpdate()
	if err != nil || !info.HasUpdate {
		return
	}
	state := readUpdateCache()
	state.Latest = info.Latest
	state.PageURL = info.PageURL
	state.Notes = info.Notes
	state.PublishedAt = info.PublishedAt
	state.CanInstall = info.CanInstall
	_ = writeUpdateCache(state)
	if normalizeVersion(state.Dismissed) == normalizeVersion(info.Latest) {
		return
	}
	a.publishUpdate(&info)
}

func writeSettingsNav(page string) error {
	return writeJSONFile(settingsNavPath(), settingsNavRequest{Page: page, At: time.Now().UnixNano()})
}

func clearSettingsNav() error {
	return writeJSONFile(settingsNavPath(), settingsNavRequest{})
}

func readSettingsNav() settingsNavRequest {
	var nav settingsNavRequest
	_ = readJSONFile(settingsNavPath(), &nav)
	return nav
}

func (a *App) ShowSettingsPage(page string) error {
	if a.settingsOnly {
		return errors.New("请在侧边栏中打开设置")
	}
	switch page {
	case "about", "help", "settings":
	default:
		page = "settings"
	}
	if err := writeSettingsNav(page); err != nil {
		return err
	}
	if err := a.OpenSettingsWindow(); err != nil {
		_ = clearSettingsNav()
		return err
	}
	return nil
}

func (a *App) StartPage() string {
	nav := readSettingsNav()
	if nav.Page == "" || nav.At == 0 {
		return ""
	}
	if time.Since(time.Unix(0, nav.At)) > 30*time.Second {
		return ""
	}
	_ = clearSettingsNav()
	return nav.Page
}

func (a *App) watchSettingsNav() {
	last := readSettingsNav().At
	for {
		time.Sleep(time.Second)
		nav := readSettingsNav()
		if nav.At == 0 || nav.At == last || nav.Page == "" {
			continue
		}
		last = nav.At
		a.emit("settings:page", nav.Page)
	}
}
