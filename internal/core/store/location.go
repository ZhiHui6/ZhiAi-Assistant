package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	LocationDefault  = "default"
	LocationPortable = "portable"
	LocationCustom   = "custom"
)

type Location struct {
	Mode   string `json:"mode"`
	Custom string `json:"custom,omitempty"`
}

var defaultDirOverride string

func DefaultDir() string {
	if defaultDirOverride != "" {
		return defaultDirOverride
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "ZhiAi-Assistant")
}

func PortableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "ZhiAiData")
	}
	return filepath.Join(filepath.Dir(exe), "ZhiAiData")
}

func LocationFile() string {
	return filepath.Join(DefaultDir(), "location.json")
}

func ReadLocation() Location {
	loc := Location{Mode: LocationDefault}
	raw, err := os.ReadFile(LocationFile())
	if err != nil {
		return loc
	}
	var saved Location
	if err := json.Unmarshal(raw, &saved); err != nil {
		return loc
	}
	switch saved.Mode {
	case LocationPortable:
		loc.Mode = saved.Mode
	case LocationCustom:
		if saved.Custom != "" {
			loc.Mode = saved.Mode
			loc.Custom = filepath.Clean(saved.Custom)
		}
	}
	return loc
}

func WriteLocation(loc Location) error {
	if !ValidModes[loc.Mode] {
		return errors.New("未知的存储方式")
	}
	dir := DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(loc)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "location.json.tmp")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, LocationFile()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

var ValidModes = map[string]bool{
	LocationDefault:  true,
	LocationPortable: true,
	LocationCustom:   true,
}

func ValidateDir(path string) error {
	if path == "" {
		return errors.New("路径为空")
	}
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("路径不存在")
	}
	if !info.IsDir() {
		return errors.New("路径不是文件夹")
	}
	probe := filepath.Join(path, ".zhi-ai-write-test")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.New("该文件夹不可写入")
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return nil
}

func ResolveDir(env string, loc Location) (dir string, fellBack bool) {
	if env != "" {
		return filepath.Clean(env), false
	}
	switch loc.Mode {
	case LocationPortable:
		return PortableDir(), false
	case LocationCustom:
		if ValidateDir(loc.Custom) == nil {
			return loc.Custom, false
		}
		return DefaultDir(), true
	default:
		return DefaultDir(), false
	}
}

func LocationStatus() (Location, string, bool) {
	env := os.Getenv("ZHIAI_ASSISTANT_DATA_DIR")
	loc := ReadLocation()
	dir, fellBack := ResolveDir(env, loc)
	return loc, dir, fellBack
}
