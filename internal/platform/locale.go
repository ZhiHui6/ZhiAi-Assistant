package platform

import "zhi-ai-assistant/internal/platform/win32"

func SystemLanguage() string {
	id := uint32(win32.UserDefaultUILanguage())
	if id&0x3ff != 0x04 {
		return "en"
	}
	switch id & 0xffff {
	case 0x0404, 0x0c04, 0x1404:
		return "zh-TW"
	}
	return "zh-CN"
}
