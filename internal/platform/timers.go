package platform

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"zhi-ai-assistant/internal/platform/win32"
)

func MessageBeep() {
	win32.MessageBeep(0x30)
}

func PlayTimerSound(choice string) error {
	choice = strings.TrimSpace(choice)
	if alias, ok := legacySoundAlias[choice]; ok {
		choice = alias
	}
	if hasBuiltinSound(choice) {
		path, err := ensureBuiltinSound(choice)
		if err == nil && win32.PlaySoundFile(path) {
			return nil
		}
		win32.MessageBeep(0x30)
		return nil
	}
	switch choice {
	case "none":
		return nil
	case "asterisk":
		win32.MessageBeep(0x40)
	case "hand":
		win32.MessageBeep(0x10)
	case "question":
		win32.MessageBeep(0x20)
	case "ok":
		win32.MessageBeep(0x0)
	default:
		if strings.HasPrefix(choice, "file:") {
			path := strings.TrimSpace(strings.TrimPrefix(choice, "file:"))
			if path == "" {
				return errors.New("铃音路径为空")
			}
			if !win32.PlaySoundFile(path) {
				return errors.New("无法播放铃音")
			}
			return nil
		}
		win32.MessageBeep(0x30)
	}
	return nil
}

func ScheduleShutdown(seconds int) error {
	if seconds < 60 {
		seconds = 60
	}
	return exec.Command("shutdown.exe", "/s", "/t", strconv.Itoa(seconds)).Run()
}

func AbortShutdown() error {
	return exec.Command("shutdown.exe", "/a").Run()
}
