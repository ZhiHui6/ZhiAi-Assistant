package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/core/festival"
	"zhi-ai-assistant/internal/core/holiday"
	"zhi-ai-assistant/internal/core/launch"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/mount"
	"zhi-ai-assistant/internal/core/notes"
	"zhi-ai-assistant/internal/core/shelf"
	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/core/timers"
	"zhi-ai-assistant/internal/core/todos"
	"zhi-ai-assistant/internal/core/usage"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
	"zhi-ai-assistant/internal/platform/win32"
	"zhi-ai-assistant/internal/winstate"
)

const WindowClass = "ZhiAi-Assistant-Main"

const SettingsWindowClass = "ZhiAi-Assistant-Settings"

const (
	menuToggle = iota + 1
	menuPin
	menuSettings
	menuAutostart
	menuEdge
	menuQuit
)

type App struct {
	ctx context.Context

	settingsDoc  *store.Doc[model.Settings]
	notesMgr     *notes.Manager
	todosMgr     *todos.Manager
	shelfMgr     *shelf.Manager
	launchMgr    *launch.Launcher
	mountMgr     *mount.Manager
	timersMgr    *timers.Manager
	holidayStore *holiday.Store

	host   *platform.Host
	window *Window

	notesSvc    *Notes
	todosSvc    *Todos
	calendarSvc *Calendar
	shelfSvc    *Shelf
	launchSvc   *Launch
	mountSvc    *Mount
	timersSvc   *Timers
	iconsSvc    *Icons
	usageMgr    *usage.Manager
	usageSvc    *Usage

	settingsOnly     bool
	resetting        bool
	resetBusy        bool
	locationBusy     bool
	restoring        bool
	restoreBusy      bool
	fitWidth         int
	fitHeight        int
	settingsShowOnce sync.Once

	mu        sync.Mutex
	quitting  bool
	tickCount int
	lastTip   string

	updateMu           sync.Mutex
	pendingUpdate      *UpdateInfo
	updateNotified     bool
	updateWatchStarted bool
}

func NewApp() (*App, error) {
	var (
		settingsDoc *store.Doc[model.Settings]
		notesMgr    *notes.Manager
		todosMgr    *todos.Manager
		shelfMgr    *shelf.Manager
		launchMgr   *launch.Launcher
		mountMgr    *mount.Manager
		timersMgr   *timers.Manager
		usageMgr    *usage.Manager
	)

	errs := make(chan error, 8)
	var wg sync.WaitGroup

	loadAsync(&wg, errs, func() (*store.Doc[model.Settings], error) {
		return store.Open("settings.json", 1, model.DefaultSettings())
	}, &settingsDoc)
	loadAsync(&wg, errs, notes.New, &notesMgr)
	loadAsync(&wg, errs, todos.New, &todosMgr)
	loadAsync(&wg, errs, shelf.New, &shelfMgr)
	loadAsync(&wg, errs, launch.New, &launchMgr)
	loadAsync(&wg, errs, mount.New, &mountMgr)
	loadAsync(&wg, errs, timers.New, &timersMgr)
	loadAsync(&wg, errs, usage.New, &usageMgr)

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return nil, err
		}
	}

	if settingsDoc.Path() == "" {
		return nil, fmt.Errorf("数据目录不可写")
	}

	holidayStore := holiday.New(filepath.Join(store.DataDir(), "data", "holidays"))
	holidayStore.Load()
	festival.SetHolidayProvider(holidayProvider{store: holidayStore})

	a := &App{
		settingsDoc:  settingsDoc,
		notesMgr:     notesMgr,
		todosMgr:     todosMgr,
		shelfMgr:     shelfMgr,
		launchMgr:    launchMgr,
		mountMgr:     mountMgr,
		timersMgr:    timersMgr,
		usageMgr:     usageMgr,
		holidayStore: holidayStore,
		window:       NewWindow(),
	}
	a.window.attach(a)
	a.migrateSettingsKeys()
	a.notesSvc = NewNotes(notesMgr)
	a.todosSvc = NewTodos(todosMgr)
	a.calendarSvc = NewCalendar(todosMgr)
	a.shelfSvc = NewShelf(shelfMgr)
	a.launchSvc = NewLaunch(launchMgr)
	a.mountSvc = NewMount(mountMgr)
	a.timersSvc = NewTimers(timersMgr)
	a.timersSvc.SetFireHandler(a.onTimerFired)
	a.iconsSvc = NewIcons()
	a.usageSvc = NewUsage(usageMgr, func(state model.UsageState) {
		a.emit("usage:changed", state)
	})
	return a, nil
}

func loadAsync[T any](wg *sync.WaitGroup, errs chan<- error, fn func() (T, error), out *T) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		v, err := fn()
		if err != nil {
			errs <- err
			return
		}
		*out = v
	}()
}

func NewSettingsApp() (*App, error) {
	doc, err := store.Open("settings.json", 1, model.DefaultSettings())
	if err != nil {
		return nil, err
	}
	return &App{settingsDoc: doc, settingsOnly: true}, nil
}

func (a *App) quitRuntime() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func normalizeBackground(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) != 7 || v[0] != '#' {
		return "#0a0c11"
	}
	n, err := strconv.ParseUint(v[1:], 16, 32)
	if err != nil {
		return "#0a0c11"
	}
	r := int(n >> 16 & 0xff)
	g := int(n >> 8 & 0xff)
	b := int(n & 0xff)
	lum := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
	if lum > 0.2 {
		f := 0.2 / lum
		r = int(float64(r) * f)
		g = int(float64(g) * f)
		b = int(float64(b) * f)
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func resolveLanguage(value string) string {
	switch value {
	case "zh-CN", "zh-TW", "en":
		return value
	}
	return platform.SystemLanguage()
}

func (a *App) Locale() string { return i18n.Current() }

func (a *App) SystemLocale() string { return platform.SystemLanguage() }

func (a *App) migrateSettingsKeys() {
	raw, err := os.ReadFile(a.settingsDoc.Path())
	if err != nil {
		return
	}
	var legacy struct {
		Data struct {
			AppColumns   *int  `json:"appColumns"`
			AppRows      *int  `json:"appRows"`
			AppIconSize  *int  `json:"appIconSize"`
			HideAppNames *bool `json:"hideAppNames"`
			HideApps     *bool `json:"hideApps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return
	}
	d := legacy.Data
	if d.AppColumns == nil && d.AppRows == nil && d.AppIconSize == nil && d.HideAppNames == nil && d.HideApps == nil {
		return
	}
	a.settingsDoc.Update(func(s *model.Settings) {
		if d.AppColumns != nil {
			s.LaunchColumns = *d.AppColumns
		}
		if d.AppRows != nil {
			s.LaunchRows = *d.AppRows
		}
		if d.AppIconSize != nil {
			s.LaunchIconSize = *d.AppIconSize
		}
		if d.HideAppNames != nil {
			s.HideLaunchNames = *d.HideAppNames
		}
		if d.HideApps != nil {
			s.HideLaunch = *d.HideApps
		}
	})
}

const maxPanelWidth = 560

func normalizeSettings(s model.Settings) model.Settings {
	if s.PanelWidth < 340 {
		s.PanelWidth = 340
	}
	maxWidth := maxPanelWidth
	for _, m := range platform.Monitors() {
		if m.Rect.W > 0 && m.Rect.W < maxWidth {
			maxWidth = m.Rect.W
		}
	}
	if s.PanelWidth > maxWidth {
		s.PanelWidth = maxWidth
	}
	if s.Peek < 24 || s.Peek > 40 {
		s.Peek = 30
	}
	if s.Edge != "left" {
		s.Edge = "right"
	}
	switch s.Language {
	case "zh-CN", "zh-TW", "en":
	default:
		s.Language = "auto"
	}
	if s.Opacity < 0.25 {
		s.Opacity = 0.25
	}
	if s.Opacity > 0.9 {
		s.Opacity = 0.9
	}
	s.Background = normalizeBackground(s.Background)
	if strings.TrimSpace(s.Hotkey) == "" {
		s.Hotkey = "ctrl+alt+space"
	}
	if s.LaunchColumns < 3 || s.LaunchColumns > 8 {
		s.LaunchColumns = 5
	}
	if s.LaunchRows < 1 || s.LaunchRows > 3 {
		s.LaunchRows = 1
	}
	if s.LaunchIconSize < 24 || s.LaunchIconSize > 40 {
		s.LaunchIconSize = 32
	}
	return s
}

func (a *App) Notes() *Notes          { return a.notesSvc }
func (a *App) Todos() *Todos          { return a.todosSvc }
func (a *App) Calendar() *Calendar    { return a.calendarSvc }
func (a *App) Shelf() *Shelf          { return a.shelfSvc }
func (a *App) Launch() *Launch        { return a.launchSvc }
func (a *App) Mount() *Mount          { return a.mountSvc }
func (a *App) Timers() *Timers        { return a.timersSvc }
func (a *App) Icons() *Icons          { return a.iconsSvc }
func (a *App) Usage() *Usage          { return a.usageSvc }
func (a *App) WindowService() *Window { return a.window }

func (a *App) RevealWindow() {
	if !a.Ready() {
		return
	}
	if a.window != nil {
		a.window.Reveal("")
	}
}

func (a *App) PrepositionWindow() {
	go func() {
		s := a.GetSettings()
		monitors := platform.Monitors()
		idx := 0
		dpi := 96
		for i, mo := range monitors {
			if mo.Primary || (s.TargetMonitor >= 0 && mo.Index == s.TargetMonitor) {
				idx = i
				dpi = mo.Dpi
				if s.TargetMonitor >= 0 && mo.Index == s.TargetMonitor {
					break
				}
			}
		}
		work := winstate.Rect{X: 0, Y: 0, W: 1920, H: 1040}
		if len(monitors) > 0 {
			work = monitors[idx].Work
		}
		width := s.PanelWidth
		if width < 340 {
			width = 340
		}
		panelW := width * dpi / 96
		peek := s.Peek
		if peek < 24 || peek > 40 {
			peek = 30
		}
		visible := peek * dpi / 96
		idleX := work.Right() - visible
		if s.Edge == "left" {
			idleX = work.X - panelW + visible
		}

		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			hwnd := win32.FindWindow(win32.UTF16(WindowClass), nil)
			if hwnd != 0 {
				ex := win32.GetWindowLongPtr(hwnd, win32.GWL_EXSTYLE)
				ex |= win32.WS_EX_TOOLWINDOW | win32.WS_EX_NOACTIVATE
				ex &^= win32.WS_EX_APPWINDOW
				win32.SetWindowLongPtr(hwnd, win32.GWL_EXSTYLE, ex)
				win32.SetWindowPos(hwnd, 0, int32(idleX), int32(work.Y), int32(panelW), int32(work.H),
					win32.SWP_NOZORDER|win32.SWP_NOACTIVATE|win32.SWP_SHOWWINDOW|win32.SWP_FRAMECHANGED)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
	}()
}

func (a *App) HandleFileDrop(x, y int, paths []string) {
	a.emit("drop:paths", map[string]interface{}{"x": x, "y": y, "paths": paths})
}

func (a *App) emit(name string, payload interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, payload)
}

func (a *App) Debug() bool {
	return os.Getenv("SIDEBAR_DEBUG") == "1"
}

func (a *App) DebugLog(message string) {
	if !a.Debug() {
		return
	}
	log.Println("[ui]", message)
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	settings := a.GetSettings()
	i18n.Set(resolveLanguage(settings.Language))
	if a.settingsOnly {
		a.CenterSettingsWindow()
		time.AfterFunc(2*time.Second, a.showSettingsWindow)
		go a.watchSettingsNav()
		return
	}

	a.StartHost()
	if a.host != nil {
		a.host.SetStage(i18n.T("正在加载界面…"))
	}

	a.startHolidaySync()

	hwnd, err := platform.FindMainWindow(WindowClass, 8*time.Second)
	if err != nil {
		log.Println("查找主窗口失败:", err)
		if a.host != nil {
			a.host.FailStartup(i18n.T("界面初始化失败"))
		}
		return
	}
	pw := platform.NewWindow(hwnd)
	a.window.start(pw, a.host, settings)

	go func() {
		platform.RepairAutostart()
		if settings.Autostart {
			_ = platform.SetAutostart(true)
		}
	}()
	pending := a.loadPendingUpdate()
	a.lastTip = a.trayTip()
	if a.host != nil {
		a.host.SetTip(a.lastTip)
	}

	if a.host != nil && !a.host.HotkeyOK() {
		a.emit("hotkey:error", map[string]interface{}{
			"message": "全局热键被占用，请到设置里更换",
		})
	}
	a.emit("app:ready", map[string]interface{}{
		"state":  a.window.GetInfo(),
		"config": settings,
		"debug":  a.Debug(),
	})
	if pending != nil {
		a.emit("update:available", pending)
	}
	a.startUpdateWatch()
}

func (a *App) StartHost() {
	if a.host != nil {
		return
	}
	settings := a.GetSettings()
	i18n.Set(resolveLanguage(settings.Language))

	host := platform.NewHost()
	host.SetCallbacks(platform.HostCallbacks{
		Tick: a.onTick,
		Hotkey: func() {
			if a.window != nil {
				a.window.Toggle()
			}
		},
		TrayLeftClick: func() {
			if a.window != nil {
				a.window.Reveal("")
			}
		},
		TrayMenu:        a.trayMenu,
		MenuCommand:     a.menuCommand,
		DisplayChange:   a.onDisplayChange,
		SettingsCommand: a.onSettingsCommand,
	})
	host.SetHotkeyCandidates(hotkeyCandidates(settings.Hotkey))
	host.SetTip(i18n.T(buildinfo.AppName))
	if err := host.Start(); err != nil {
		log.Println("原生宿主启动失败:", err)
		return
	}
	a.host = host
	host.BeginStartup(i18n.T("正在启动…"))
	time.AfterFunc(25*time.Second, func() {
		if !host.Ready() {
			host.FailStartup(i18n.T("启动超时"))
		}
	})
}

func (a *App) SignalReady() {
	if a.host != nil {
		a.host.MarkReady()
	}
}

func (a *App) Ready() bool {
	return a.host != nil && a.host.Ready()
}

func (a *App) onTick(now time.Time) bool {
	if a.window == nil {
		return false
	}
	a.tickCount++
	if a.tickCount%30 == 0 {
		a.refreshTrayTip()
	}
	anim := a.window.tick(now)
	if a.timersSvc != nil {
		a.timersSvc.Tick(now)
	}
	if a.usageSvc != nil {
		a.usageSvc.Tick(now)
	}
	return anim
}

func (a *App) onTimerFired(kind, label string) {
	a.mu.Lock()
	quitting := a.quitting
	a.mu.Unlock()
	if quitting {
		return
	}
	if a.window != nil {
		a.window.Reveal("timers")
	}
	settings := a.GetSettings()
	sound := settings.AlarmSound
	if kind == "countdown" {
		sound = settings.CountdownSound
	}
	if err := platform.PlayTimerSound(sound); err != nil {
		platform.MessageBeep()
	}
	a.emit("timer:fired", map[string]string{"kind": kind, "label": label})
}

func (a *App) refreshTrayTip() {
	if a.host == nil {
		return
	}
	tip := a.trayTip()
	if tip == a.lastTip {
		return
	}
	a.lastTip = tip
	a.host.SetTip(tip)
}

func (a *App) onDisplayChange() {
	if a.window != nil {
		a.window.displayChanged()
	}
}

func (a *App) trayTip() string {
	name := i18n.T(buildinfo.AppName)
	a.updateMu.Lock()
	pending := a.pendingUpdate
	a.updateMu.Unlock()
	if pending != nil && pending.Latest != "" {
		return i18n.TF("%s · 有新版本 v%s", name, pending.Latest)
	}
	return name
}

func (a *App) trayMenu() []platform.MenuItem {
	info := a.window.GetInfo()
	toggleLabel := i18n.T("显示面板")
	if info.State != "hidden" && info.State != "edge-idle" {
		toggleLabel = i18n.T("收起面板")
	}
	return []platform.MenuItem{
		{ID: menuToggle, Label: toggleLabel},
		{ID: menuPin, Label: i18n.T("钉住面板"), Checked: info.Pinned},
		{Separator: true},
		{ID: menuEdge, Label: i18n.T("贴边自动隐藏"), Checked: info.EdgeEnabled},
		{ID: menuAutostart, Label: i18n.T("开机自启"), Checked: platform.AutostartEnabled()},
		{ID: menuSettings, Label: i18n.T("设置…")},
		{Separator: true},
		{ID: menuQuit, Label: i18n.T("退出")},
	}
}

func (a *App) menuCommand(id int) {
	switch id {
	case menuToggle:
		a.window.Toggle()
	case menuPin:
		a.window.TogglePin()
	case menuSettings:
		_ = a.OpenSettingsWindow()
	case menuAutostart:
		enabled := platform.AutostartEnabled()
		_ = platform.SetAutostart(!enabled)
		settings := a.GetSettings()
		settings.Autostart = !enabled
		a.saveSettings(settings)
	case menuEdge:
		settings := a.GetSettings()
		settings.EdgeEnabled = !settings.EdgeEnabled
		a.saveSettings(settings)
	case menuQuit:
		a.Quit()
	}
}

func (a *App) GetSettings() model.Settings {
	var s model.Settings
	a.settingsDoc.Read(func(d *model.Settings) { s = *d })
	if s.PanelWidth <= 0 {
		s = model.DefaultSettings()
	}
	if s.PanelWidth > maxPanelWidth {
		s.PanelWidth = maxPanelWidth
	}
	s.Background = normalizeBackground(s.Background)
	return s
}

func (a *App) saveSettings(s model.Settings) {
	if a.restoring {
		return
	}
	i18n.Set(resolveLanguage(s.Language))
	a.settingsDoc.Update(func(d *model.Settings) { *d = s })
	if a.host != nil && s.Hotkey != "" {
		if spec, ok := ParseHotkey(s.Hotkey); ok {
			a.host.SetHotkey(spec)
		}
	}
	if a.window != nil {
		a.window.ApplySettings(s)
	}
	a.emit("settings:changed", s)
}

func (a *App) UpdateSettings(s model.Settings) (model.Settings, error) {
	s = normalizeSettings(s)
	prev := a.GetSettings()
	a.saveSettings(s)
	if prev.Autostart != s.Autostart {
		_ = platform.SetAutostart(s.Autostart)
	}
	return s, nil
}

func (a *App) Defaults() model.Settings { return model.DefaultSettings() }

func (a *App) ClipboardText() string {
	if a.ctx == nil {
		return ""
	}
	text, err := runtime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	return text
}

func (a *App) SetClipboardText(text string) {
	if a.ctx == nil {
		return
	}
	_ = runtime.ClipboardSetText(a.ctx, text)
}

func (a *App) GetMonitors() interface{} { return platform.Monitors() }

func (a *App) GetAutostart() bool { return platform.AutostartEnabled() }

func (a *App) SetAutostart(on bool) bool {
	_ = platform.SetAutostart(on)
	s := a.GetSettings()
	s.Autostart = on
	a.saveSettings(s)
	return platform.AutostartEnabled()
}

func (a *App) OpenDataDir() error {
	return platform.ShellExecute(0, "open", store.DataDir(), "", "", 1)
}

func (a *App) OpenExternal(target string) error {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return fmt.Errorf("仅支持打开网页链接")
	}
	return platform.ShellExecute(0, "open", target, "", "", 1)
}

func (a *App) DataDir() string { return store.DataDir() }

func (a *App) HotkeyInfo() map[string]interface{} {
	ok := false
	label := ""
	if a.host != nil {
		ok = a.host.HotkeyOK()
		label = a.host.HotkeyLabel()
	}
	return map[string]interface{}{"ok": ok, "label": label}
}

func (a *App) Quit() {
	a.mu.Lock()
	if a.quitting {
		a.mu.Unlock()
		return
	}
	a.quitting = true
	a.mu.Unlock()
	go func() {
		a.flushAll()
		if a.host != nil {
			a.host.Stop()
		}
		runtime.Quit(a.ctx)
	}()
}

func (a *App) BeforeClose(ctx context.Context) bool {
	a.mu.Lock()
	q := a.quitting
	a.mu.Unlock()
	if q {
		return false
	}
	a.flushAll()
	if a.window != nil {
		a.window.Collapse()
	}
	return true
}

func (a *App) Shutdown(ctx context.Context) {
	if !a.resetting {
		a.flushAll()
	}
	if a.host != nil {
		a.host.Stop()
	}
}

func (a *App) flushAll() {
	if a.notesMgr != nil {
		_ = a.notesMgr.Flush()
	}
	if a.todosMgr != nil {
		_ = a.todosMgr.Flush()
	}
	if a.shelfMgr != nil {
		_ = a.shelfMgr.Flush()
	}
	if a.launchMgr != nil {
		_ = a.launchMgr.Flush()
	}
	if a.timersMgr != nil {
		_ = a.timersMgr.Flush()
	}
	if a.mountMgr != nil {
		_ = a.mountMgr.Flush()
	}
	if a.settingsDoc != nil {
		_ = a.settingsDoc.Flush()
	}
	if a.usageMgr != nil {
		_ = a.usageMgr.Flush()
	}
}

var _ = os.Getenv
var _ = win32.GetModuleHandle
