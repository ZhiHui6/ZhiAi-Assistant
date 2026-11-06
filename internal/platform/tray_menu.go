package platform

import (
	"zhi-ai-assistant/internal/platform/win32"
)

func (h *Host) showTrayMenu() {
	if h.cb.TrayMenu == nil || !h.Ready() {
		return
	}
	items := h.cb.TrayMenu()
	menu := win32.CreatePopupMenu()
	if menu == 0 {
		return
	}
	defer win32.DestroyMenu(menu)

	for _, item := range items {
		if item.Separator {
			win32.AppendMenu(menu, win32.MF_SEPARATOR, 0, nil)
			continue
		}
		flags := uint32(win32.MF_STRING)
		if item.Checked {
			flags |= win32.MF_CHECKED
		}
		if item.Disabled {
			flags |= win32.MF_GRAYED
		}
		win32.AppendMenu(menu, flags, uintptr(item.ID), win32.UTF16(item.Label))
	}

	x, y := cursorPoint()
	win32.SetForegroundWindow(h.hwnd)
	cmd := win32.TrackPopupMenuEx(
		menu,
		win32.TPM_LEFTALIGN|win32.TPM_RIGHTBUTTON|win32.TPM_RETURNCMD|win32.TPM_NONOTIFY,
		x, y, h.hwnd, 0)
	win32.PostMessage(h.hwnd, win32.WM_NULL, 0, 0)
	if cmd > 0 && h.cb.MenuCommand != nil {
		h.cb.MenuCommand(int(cmd))
	}
}
