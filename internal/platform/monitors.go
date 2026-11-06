package platform

import (
	"syscall"

	"zhi-ai-assistant/internal/platform/win32"
	"zhi-ai-assistant/internal/winstate"
)

func Monitors() []winstate.Monitor {
	var out []winstate.Monitor
	cb := syscall.NewCallback(func(hMon, hdc, lprc, data uintptr) uintptr {
		var mi win32.MONITORINFO
		if !win32.GetMonitorInfo(hMon, &mi) {
			return 1
		}
		dpi := int(win32.GetDpiForMonitor(hMon))
		if dpi <= 0 {
			dpi = 96
		}
		out = append(out, winstate.Monitor{
			Index: len(out),
			Rect: winstate.Rect{
				X: int(mi.RcMonitor.Left),
				Y: int(mi.RcMonitor.Top),
				W: int(mi.RcMonitor.Width()),
				H: int(mi.RcMonitor.Height()),
			},
			Work: winstate.Rect{
				X: int(mi.RcWork.Left),
				Y: int(mi.RcWork.Top),
				W: int(mi.RcWork.Width()),
				H: int(mi.RcWork.Height()),
			},
			Dpi:     dpi,
			Primary: mi.DwFlags&win32.MONITORINFOF_PRIMARY != 0,
		})
		return 1
	})
	win32.EnumDisplayMonitors(cb, 0)
	return out
}

func MonitorIndexAt(x, y int32) int {
	mon := win32.MonitorFromPoint(win32.POINT{X: x, Y: y}, win32.MONITOR_DEFAULTTONEAREST)
	var mi win32.MONITORINFO
	if mon == 0 || !win32.GetMonitorInfo(mon, &mi) {
		return -1
	}
	for i, m := range Monitors() {
		if m.Rect.X == int(mi.RcMonitor.Left) && m.Rect.Y == int(mi.RcMonitor.Top) {
			return i
		}
	}
	return -1
}
