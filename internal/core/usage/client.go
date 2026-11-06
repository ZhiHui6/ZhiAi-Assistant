package usage

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"zhi-ai-assistant/internal/platform"
)

type clientSpec struct {
	keywords  []string
	dirs      []string
	execNames []string
}

var clientSpecs = map[string]clientSpec{
	"qoder": {
		keywords:  []string{"qoder"},
		dirs:      []string{"Qoder CN", "Qoder"},
		execNames: []string{"Qoder CN.exe", "Qoder.exe"},
	},
	"trae": {
		keywords:  []string{"trae"},
		dirs:      []string{"Trae CN", "Trae"},
		execNames: []string{"Trae CN.exe", "Trae.exe"},
	},
	"workbuddy": {
		keywords:  []string{"workbuddy", "codebuddy"},
		dirs:      []string{"WorkBuddyAI", "WorkBuddy", "CodeBuddy CN", "CodeBuddy"},
		execNames: []string{"WorkBuddy.exe", "CodeBuddy.exe"},
	},
}

var helperExeHints = []string{
	"unins", "uninstall", "repair", "elevate", "update", "updater", "setup",
	"helper", "doctor", "crash", "report", "share-target", "task_host",
}

func (m *Manager) OpenClient(id string) (string, error) {
	path, err := findClientExecutable(m.env, id)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := platform.ShellExecute(0, "open", path, "", dir, 1); err != nil {
		cmd := exec.Command(path)
		cmd.Dir = dir
		if startErr := cmd.Start(); startErr != nil {
			return path, fmt.Errorf("启动失败：%v", startErr)
		}
	}
	return path, nil
}

func clientRoots(env Env) []string {
	return []string{
		filepath.Join(env.LocalAppData, "Programs"),
		filepath.Join(env.AppData, "Programs"),
		env.LocalAppData,
		env.AppData,
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
	}
}

func findClientExecutable(env Env, id string) (string, error) {
	spec, ok := clientSpecs[id]
	if !ok {
		return "", errors.New("未知平台")
	}
	roots := clientRoots(env)
	for _, root := range roots {
		if root == "" {
			continue
		}
		for _, dir := range spec.dirs {
			for _, name := range spec.execNames {
				path := filepath.Join(root, dir, name)
				if isExecutableFile(path) {
					return path, nil
				}
			}
		}
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		best := ""
		bestScore := 0
		for _, entry := range entries {
			if !entry.IsDir() || !matchesAny(entry.Name(), spec.keywords) {
				continue
			}
			if path, score := bestExecutable(filepath.Join(root, entry.Name()), entry.Name(), spec); score > bestScore {
				best, bestScore = path, score
			}
		}
		if best != "" {
			return best, nil
		}
	}
	return "", errors.New("未找到客户端程序，请先安装客户端")
}

func bestExecutable(dir, dirName string, spec clientSpec) (string, int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0
	}
	best := ""
	bestScore := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if matchesAny(base, helperExeHints) {
			continue
		}
		score := 1
		if strings.EqualFold(base, dirName) {
			score += 5
		}
		if matchesAny(base, spec.keywords) {
			score += 3
		}
		for _, exec := range spec.execNames {
			if strings.EqualFold(entry.Name(), exec) {
				score += 4
			}
		}
		if score > bestScore {
			best, bestScore = filepath.Join(dir, entry.Name()), score
		}
	}
	return best, bestScore
}

func matchesAny(name string, keywords []string) bool {
	lower := strings.ToLower(name)
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
