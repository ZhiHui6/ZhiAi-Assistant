package platform

import "syscall"

var (
	kernel32DLL        = syscall.NewLazyDLL("kernel32.dll")
	procGetProcAddress = kernel32DLL.NewProc("GetProcAddress")
	uxthemeDLL         = syscall.NewLazyDLL("uxtheme.dll")

	darkMenusApplied bool
)

func enableDarkMenus(hwnd uintptr) {
	if darkMenusApplied {
		return
	}
	if err := uxthemeDLL.Load(); err != nil {
		return
	}
	module := uxthemeDLL.Handle()
	if module == 0 {
		return
	}
	setPreferredAppMode, _, _ := procGetProcAddress.Call(module, 135)
	allowDarkModeForWindow, _, _ := procGetProcAddress.Call(module, 133)
	flushMenuThemes, _, _ := procGetProcAddress.Call(module, 136)
	if setPreferredAppMode == 0 {
		return
	}
	syscall.Syscall(setPreferredAppMode, 1, 2, 0, 0)
	if allowDarkModeForWindow != 0 && hwnd != 0 {
		syscall.Syscall(allowDarkModeForWindow, 2, hwnd, 1, 0)
	}
	if flushMenuThemes != 0 {
		syscall.Syscall(flushMenuThemes, 0, 0, 0, 0)
	}
	darkMenusApplied = true
}
