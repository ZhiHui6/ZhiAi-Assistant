package service

import (
	"strings"

	"zhi-ai-assistant/internal/platform"
	"zhi-ai-assistant/internal/platform/win32"
)

var keyNames = map[string]uint32{
	"space": win32.VK_SPACE,
	"`":     win32.VK_OEM_3,
	"~":     win32.VK_OEM_3,
	"f1":    win32.VK_F1 + 0,
	"f2":    win32.VK_F1 + 1,
	"f3":    win32.VK_F1 + 2,
	"f4":    win32.VK_F1 + 3,
	"f5":    win32.VK_F1 + 4,
	"f6":    win32.VK_F1 + 5,
	"f7":    win32.VK_F1 + 6,
	"f8":    win32.VK_F1 + 7,
	"f9":    win32.VK_F1 + 8,
	"f10":   win32.VK_F1 + 9,
	"f11":   win32.VK_F1 + 10,
	"f12":   win32.VK_F1 + 11,
}

func vkFor(key string) uint32 {
	if v, ok := keyNames[key]; ok {
		return v
	}
	if len(key) == 1 {
		c := key[0]
		if c >= 'a' && c <= 'z' {
			return uint32(c - 'a' + 0x41)
		}
		if c >= '0' && c <= '9' {
			return uint32(c - '0' + 0x30)
		}
	}
	return 0
}

func ParseHotkey(s string) (platform.HotkeySpec, bool) {
	raw := strings.ToLower(strings.TrimSpace(s))
	raw = strings.ReplaceAll(raw, "control", "ctrl")
	raw = strings.ReplaceAll(raw, " ", "")
	if raw == "" {
		return platform.HotkeySpec{}, false
	}
	parts := strings.Split(raw, "+")
	var mods uint32
	key := ""
	for _, p := range parts {
		switch p {
		case "ctrl":
			mods |= win32.MOD_CONTROL
		case "alt":
			mods |= win32.MOD_ALT
		case "shift":
			mods |= win32.MOD_SHIFT
		case "win":
			mods |= win32.MOD_WIN
		case "":
		default:
			key = p
		}
	}
	vk := vkFor(key)
	if vk == 0 || mods == 0 {
		return platform.HotkeySpec{}, false
	}
	return platform.HotkeySpec{Mods: mods, VK: vk, Label: displayHotkey(mods, key)}, true
}

func displayHotkey(mods uint32, key string) string {
	var parts []string
	if mods&win32.MOD_CONTROL != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&win32.MOD_ALT != 0 {
		parts = append(parts, "Alt")
	}
	if mods&win32.MOD_SHIFT != 0 {
		parts = append(parts, "Shift")
	}
	if mods&win32.MOD_WIN != 0 {
		parts = append(parts, "Win")
	}
	switch key {
	case "space":
		parts = append(parts, "Space")
	case "`":
		parts = append(parts, "`")
	default:
		parts = append(parts, strings.ToUpper(key))
	}
	return strings.Join(parts, "+")
}

func hotkeyCandidates(preferred string) []platform.HotkeySpec {
	var out []platform.HotkeySpec
	seen := map[string]bool{}
	add := func(s string) {
		spec, ok := ParseHotkey(s)
		if !ok || seen[spec.Label] {
			return
		}
		seen[spec.Label] = true
		out = append(out, spec)
	}
	add(preferred)
	add("ctrl+alt+space")
	add("ctrl+alt+`")
	add("ctrl+shift+space")
	add("alt+`")
	add("ctrl+alt+f10")
	return out
}
