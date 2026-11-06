package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"zhi-ai-assistant/internal/core/store"
)

const debugBodyLimit = 256 * 1024

var (
	debugMu   sync.Mutex
	debugFile *os.File
	debugOpen bool
)

func debugEnabled() bool {
	return os.Getenv("ZHIAI_ASSISTANT_DEBUG") == "1"
}

func debugLog(platform, url string, raw []byte, err error) {
	if !debugEnabled() {
		return
	}
	debugMu.Lock()
	defer debugMu.Unlock()
	if !debugOpen {
		path := filepath.Join(store.DataDir(), "data", "usage-debug.log")
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		f, ferr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if ferr != nil {
			return
		}
		debugFile = f
		debugOpen = true
	}
	body := raw
	truncated := false
	if len(body) > debugBodyLimit {
		body = body[:debugBodyLimit]
		truncated = true
	}
	status := "ok"
	if err != nil {
		status = "error: " + err.Error()
	}
	line := fmt.Sprintf("==== %s | %s | %s\n%s\n", time.Now().Format("15:04:05"), platform, status, url)
	_, _ = debugFile.WriteString(line)
	if len(raw) > 0 {
		_, _ = debugFile.Write(body)
		_, _ = debugFile.WriteString("\n")
	}
	if truncated {
		_, _ = debugFile.WriteString(fmt.Sprintf("[truncated, origin len=%d]\n", len(raw)))
	}
	_, _ = debugFile.WriteString("\n")
}
