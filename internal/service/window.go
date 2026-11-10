package service

import (
	"log"
	"sync"
	"time"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/platform"
	"zhi-ai-assistant/internal/winstate"
)

type WindowInfo struct {
	State       string `json:"state"`
	Pinned      bool   `json:"pinned"`
	Edge        string `json:"edge"`
	EdgeEnabled bool   `json:"edgeEnabled"`
	PanelW      int    `json:"panelW"`
	PanelH      int    `json:"panelH"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Dpi         int    `json:"dpi"`
	Monitor     int    `json:"monitor"`
	Hotkey      string `json:"hotkey"`
	HotkeyOK    bool   `json:"hotkeyOk"`
}

type Window struct {
	mu       sync.Mutex
	m        *winstate.Machine
	pw       *platform.Window
	host     *platform.Host
	app      *App
	settings model.Settings
}

func NewWindow() *Window { return &Window{} }

func (w *Window) attach(app *App) { w.app = app }

func (w *Window) start(pw *platform.Window, host *platform.Host, s model.Settings) {
	w.pw = pw
	w.host = host
	w.settings = s
	cfg := w.buildConfig(s)
	w.m = winstate.New(pw, cfg)
	w.m.OnChange(func(st winstate.State) {
		info := w.infoLocked()
		if w.app.Debug() {
			log.Printf("[state] %s x=%d y=%d %dx%d cursor=%+v", st, info.X, info.Y, info.PanelW, info.PanelH, w.m.LastCursor())
		}
		w.app.emit("state:changed", info)
		if st == winstate.StateEdgeOpen {
			w.app.emit("panel:shown", map[string]string{"view": ""})
		}
	})
	w.m.ShowIdle(time.Now())
}

func (w *Window) buildConfig(s model.Settings) winstate.Config {
	cfg := winstate.DefaultConfig()
	cfg.Edge = winstate.Edge(s.Edge)
	if cfg.Edge != winstate.EdgeLeft {
		cfg.Edge = winstate.EdgeRight
	}
	monitors := platform.Monitors()
	cfg.Monitors = monitors
	cfg.TargetIndex = s.TargetMonitor
	dpi := 96
	for _, mo := range monitors {
		if mo.Index == s.TargetMonitor {
			dpi = mo.Dpi
		}
	}
	if dpi <= 0 {
		dpi = 96
	}
	width := s.PanelWidth
	if width < 340 {
		width = 340
	}
	cfg.PanelW = width * dpi / 96
	if cfg.PanelW < 340 {
		cfg.PanelW = width
	}
	peek := s.Peek
	if peek < 24 || peek > 40 {
		peek = 30
	}
	cfg.Peek = peek * dpi / 96
	cfg.HandleW = cfg.Peek
	cfg.EdgeEnabled = s.EdgeEnabled
	if !s.Animations {
		cfg.AnimDur = 0
		cfg.CloseHold = 0
	}
	return cfg
}

func (w *Window) infoLocked() WindowInfo {
	if w.m == nil {
		return WindowInfo{State: "hidden", Dpi: w.dpi()}
	}
	p := w.m.Placement()
	info := WindowInfo{
		State:       string(w.m.State()),
		Pinned:      w.m.Pinned(),
		Edge:        string(w.m.Config().Edge),
		EdgeEnabled: w.m.Config().EdgeEnabled,
		PanelW:      p.W,
		PanelH:      p.H,
		X:           p.X,
		Y:           p.Y,
		Dpi:         w.dpi(),
		Monitor:     w.m.Config().TargetIndex,
	}
	if w.host != nil {
		info.Hotkey = w.host.HotkeyLabel()
		info.HotkeyOK = w.host.HotkeyOK()
	}
	return info
}

func (w *Window) dpi() int {
	if w.pw == nil {
		return 96
	}
	return w.pw.Dpi()
}

func (w *Window) GetInfo() WindowInfo {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.infoLocked()
}

func (w *Window) Reveal(view string) {
	w.mu.Lock()
	w.m.Reveal(time.Now())
	w.mu.Unlock()
	if w.host != nil {
		w.host.Kick()
	}
	w.app.emit("panel:shown", map[string]string{"view": view})
}

func (w *Window) Collapse() {
	w.mu.Lock()
	w.m.Collapse(time.Now())
	w.mu.Unlock()
	if w.host != nil {
		w.host.Kick()
	}
}

func (w *Window) Toggle() {
	w.mu.Lock()
	w.m.ToggleFloat(time.Now())
	w.mu.Unlock()
	if w.host != nil {
		w.host.Kick()
	}
	if w.app != nil {
		w.app.emit("panel:shown", map[string]string{"view": ""})
	}
}

func (w *Window) TogglePin() bool {
	w.mu.Lock()
	pinned := !w.m.Pinned()
	w.m.SetPinned(pinned)
	w.mu.Unlock()
	w.app.emit("state:changed", w.GetInfo())
	return pinned
}

func (w *Window) SetPinned(pinned bool) {
	w.mu.Lock()
	w.m.SetPinned(pinned)
	w.mu.Unlock()
}

func (w *Window) Activate() {
	w.mu.Lock()
	w.m.Activate()
	w.mu.Unlock()
	w.app.emit("state:changed", w.GetInfo())
}

func (w *Window) ApplySettings(s model.Settings) {
	w.mu.Lock()
	w.settings = s
	if w.m == nil {
		w.mu.Unlock()
		return
	}
	cfg := w.buildConfig(s)
	w.m.SetConfig(cfg)
	info := w.infoLocked()
	w.mu.Unlock()
	w.app.emit("state:changed", info)
}

func (w *Window) tick(now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pw == nil || w.m == nil {
		return false
	}
	w.pw.UndoMinimize()
	if fg := platform.ForegroundWindow(); fg != 0 && platform.WindowClass(fg) == "WorkerW" {
		w.pw.SetDesktopOverlay(fg)
	} else {
		w.pw.SetDesktopOverlay(0)
	}
	if w.app != nil && w.app.Debug() {
		defer func(t time.Time) {
			if d := time.Since(t); d > 50*time.Millisecond {
				log.Printf("[perf] window.tick slow=%v", d)
			}
		}(time.Now())
	}
	pt, ok := w.pw.CursorPos()
	if !ok {
		return false
	}
	fg := w.pw.IsForeground()
	w.m.OnCursor(pt, now, fg)
	return w.m.Tick(now, fg)
}

func (w *Window) displayChanged() {
	w.mu.Lock()
	w.m.OnDisplayChange(time.Now())
	w.mu.Unlock()
	if w.host != nil {
		w.host.Kick()
	}
	w.app.emit("monitors:changed", platform.Monitors())
}

func (w *Window) State() winstate.State {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.m == nil {
		return winstate.StateHidden
	}
	return w.m.State()
}
