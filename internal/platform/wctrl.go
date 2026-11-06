package platform

import (
	"errors"
	"time"

	"zhi-ai-assistant/internal/platform/win32"
	"zhi-ai-assistant/internal/winstate"
)

type Window struct {
	hwnd   uintptr
	lastEx uintptr
	shown  bool
}

func FindMainWindow(className string, timeout time.Duration) (uintptr, error) {
	deadline := time.Now().Add(timeout)
	for {
		hwnd := win32.FindWindow(win32.UTF16(className), nil)
		if hwnd != 0 {
			return hwnd, nil
		}
		if time.Now().After(deadline) {
			return 0, errors.New("未找到主窗口: " + className)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func NewWindow(hwnd uintptr) *Window {
	w := &Window{hwnd: hwnd}
	w.shown = win32.IsWindowVisible(hwnd)
	w.prepareExStyle()
	return w
}

func (w *Window) Handle() uintptr { return w.hwnd }

func (w *Window) Dpi() int {
	if dpi := win32.GetDpiForWindow(w.hwnd); dpi > 0 {
		return int(dpi)
	}
	return 96
}

func (w *Window) prepareExStyle() {
	ex := win32.GetWindowLongPtr(w.hwnd, win32.GWL_EXSTYLE)
	ex |= win32.WS_EX_TOOLWINDOW
	ex &^= win32.WS_EX_APPWINDOW
	w.lastEx = ex
	win32.SetWindowLongPtr(w.hwnd, win32.GWL_EXSTYLE, ex)
	win32.SetWindowPos(w.hwnd, 0, 0, 0, 0, 0,
		win32.SWP_NOMOVE|win32.SWP_NOSIZE|win32.SWP_NOZORDER|win32.SWP_NOACTIVATE|win32.SWP_FRAMECHANGED)
}

func (w *Window) Apply(p winstate.Placement) {
	ex := w.lastEx
	if p.NoActivate {
		ex |= win32.WS_EX_NOACTIVATE
	} else {
		ex &^= win32.WS_EX_NOACTIVATE
	}
	if ex != w.lastEx {
		win32.SetWindowLongPtr(w.hwnd, win32.GWL_EXSTYLE, ex)
		w.lastEx = ex
	}

	flags := uint32(win32.SWP_NOZORDER | win32.SWP_NOACTIVATE | win32.SWP_ASYNCWINDOWPOS)
	if p.Visible {
		flags |= win32.SWP_SHOWWINDOW
		if !w.shown {
			win32.ShowWindow(w.hwnd, win32.SW_SHOWNOACTIVATE)
			w.shown = true
		}
	} else {
		flags |= win32.SWP_HIDEWINDOW
		w.shown = false
	}
	win32.SetWindowPos(w.hwnd, 0, int32(p.X), int32(p.Y), int32(p.W), int32(p.H), flags)
}

func (w *Window) ActivateWindow() {
	win32.ShowWindow(w.hwnd, win32.SW_SHOW)
	fg := win32.GetForegroundWindow()
	if fg == w.hwnd {
		return
	}
	ourThread := win32.GetCurrentThreadID()
	fgThread := win32.GetWindowThreadProcessID(fg)
	attached := false
	if fgThread != 0 && fgThread != ourThread {
		attached = win32.AttachThreadInput(ourThread, fgThread, true)
	}
	win32.BringWindowToTop(w.hwnd)
	win32.SetForegroundWindow(w.hwnd)
	win32.SetFocus(w.hwnd)
	if attached {
		win32.AttachThreadInput(ourThread, fgThread, false)
	}
}

func (w *Window) IsForeground() bool {
	return win32.GetForegroundWindow() == w.hwnd
}

func (w *Window) CursorPos() (winstate.Point, bool) {
	var p win32.POINT
	if !win32.GetCursorPos(&p) {
		return winstate.Point{}, false
	}
	return winstate.Point{X: int(p.X), Y: int(p.Y)}, true
}

func (w *Window) ScreenPoint() (int32, int32) {
	var p win32.POINT
	win32.GetCursorPos(&p)
	return p.X, p.Y
}
