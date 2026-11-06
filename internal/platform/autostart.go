package platform

import (
	"os"
	"strings"

	"zhi-ai-assistant/internal/platform/win32"
)

const (
	runKeyPath         = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName       = "ZhiAi-Assistant"
	legacyRunValueName = "WorkBuddySidebar"
)

func exePath() string {
	p := win32.ModuleFileName()
	if p == "" {
		if e, err := os.Executable(); err == nil {
			return e
		}
	}
	return p
}

func AutostartEnabled() bool {
	handle, err := win32.RegCreateKey(win32.HKEY_CURRENT_USER, runKeyPath)
	if err != nil {
		return false
	}
	defer win32.RegCloseKey(handle)
	val, ok := win32.RegGetString(handle, runValueName)
	if !ok || val == "" {
		return false
	}
	cur := strings.ToLower(strings.Trim(exePath(), `"`))
	got := strings.ToLower(strings.Trim(strings.TrimSpace(val), `"`))
	got = strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(got, `"`)), `" --autostart`)
	return got == cur
}

func SetAutostart(on bool) error {
	handle, err := win32.RegCreateKey(win32.HKEY_CURRENT_USER, runKeyPath)
	if err != nil {
		return err
	}
	defer win32.RegCloseKey(handle)
	if !on {
		_ = win32.RegDeleteValue(handle, runValueName)
		return nil
	}
	return win32.RegSetString(handle, runValueName, `"`+exePath()+`" --autostart`)
}

func RepairAutostart() {
	handle, err := win32.RegCreateKey(win32.HKEY_CURRENT_USER, runKeyPath)
	if err != nil {
		return
	}
	defer win32.RegCloseKey(handle)
	_ = win32.RegDeleteValue(handle, legacyRunValueName)
	val, ok := win32.RegGetString(handle, runValueName)
	if !ok || val == "" {
		return
	}
	want := `"` + exePath() + `" --autostart`
	if val != want {
		_ = win32.RegSetString(handle, runValueName, want)
	}
}

func AutostartRequested() bool {
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "--autostart") {
			return true
		}
	}
	return false
}
