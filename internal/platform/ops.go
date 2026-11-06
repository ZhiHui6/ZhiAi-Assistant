package platform

import "zhi-ai-assistant/internal/platform/win32"

func ShellExecute(hwnd uintptr, verb, file, params, dir string, show int32) error {
	return win32.ShellExecute(hwnd, verb, file, params, dir, show)
}
