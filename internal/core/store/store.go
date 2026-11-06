package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultDebounce = 400 * time.Millisecond

type envelope[T any] struct {
	Schema int `json:"schema"`
	Data   T   `json:"data"`
}

type Doc[T any] struct {
	mu       sync.Mutex
	path     string
	schema   int
	data     T
	debounce time.Duration
	timer    *time.Timer
	dirty    bool
	lastErr  error
	onSave   func(error)
}

var (
	dirMu       sync.Mutex
	dirCached   string
	dirCachedAt string
	dirResolved bool
)

func DataDir() string {
	env := os.Getenv("ZHIAI_ASSISTANT_DATA_DIR")
	dirMu.Lock()
	defer dirMu.Unlock()
	if dirResolved && dirCachedAt == env {
		return dirCached
	}
	dir, _ := ResolveDir(env, ReadLocation())
	migrateLegacyDataDir(dir)
	_ = os.MkdirAll(filepath.Join(dir, "data"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "cache", "icons"), 0o755)
	dirCached = dir
	dirCachedAt = env
	dirResolved = true
	return dir
}

func migrateLegacyDataDir(dir string) {
	legacy := filepath.Join(filepath.Dir(dir), "WorkBuddySidebar")
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	if _, err := os.Stat(dir); err == nil {
		return
	}
	_ = os.Rename(legacy, dir)
}

func dataFilePath(name string) string {
	return filepath.Join(DataDir(), "data", name)
}

func Peek[T any](name string) (T, bool) {
	var out T
	raw, err := os.ReadFile(dataFilePath(name))
	if err != nil {
		return out, false
	}
	var env envelope[T]
	if err := json.Unmarshal(raw, &env); err != nil {
		return out, false
	}
	return env.Data, true
}

func Open[T any](name string, schema int, def T) (*Doc[T], error) {
	d := &Doc[T]{
		path:     dataFilePath(name),
		schema:   schema,
		data:     def,
		debounce: DefaultDebounce,
	}
	if err := d.load(def); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Doc[T]) load(def T) error {
	raw, err := os.ReadFile(d.path)
	if err == nil {
		var env envelope[T]
		if err := json.Unmarshal(raw, &env); err == nil {
			d.data = env.Data
			return nil
		}
		_ = os.WriteFile(d.path+".corrupt", raw, 0o644)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		d.lastErr = err
	}

	backup, backupErr := os.ReadFile(d.path + ".bak")
	if backupErr == nil {
		var env envelope[T]
		if err := json.Unmarshal(backup, &env); err == nil {
			d.data = env.Data
			_ = d.saveLocked()
			return nil
		}
	}

	d.data = def
	return d.saveLocked()
}

func (d *Doc[T]) Read(fn func(*T)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fn(&d.data)
}

func (d *Doc[T]) SetData(v T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data = v
	d.dirty = true
	d.scheduleLocked()
}

func (d *Doc[T]) Update(fn func(*T)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	fn(&d.data)
	d.dirty = true
	d.scheduleLocked()
}

func (d *Doc[T]) scheduleLocked() {
	if d.debounce <= 0 {
		_ = d.saveLocked()
		return
	}
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.debounce, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if !d.dirty {
			return
		}
		_ = d.saveLocked()
	})
}

func (d *Doc[T]) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.timer != nil {
		d.timer.Stop()
	}
	if !d.dirty {
		return d.lastErr
	}
	err := d.saveLocked()
	return err
}

func (d *Doc[T]) saveLocked() error {
	env := envelope[T]{Schema: d.schema, Data: d.data}
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		d.lastErr = err
		if d.onSave != nil {
			d.onSave(err)
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.path), 0o755); err != nil {
		d.lastErr = err
		return err
	}
	tmp := d.path + ".tmp"
	if err := writeSync(tmp, b); err != nil {
		d.lastErr = err
		if d.onSave != nil {
			d.onSave(err)
		}
		return err
	}
	if _, statErr := os.Stat(d.path); statErr == nil {
		if copyErr := copyFile(d.path, d.path+".bak"); copyErr != nil {
			d.lastErr = copyErr
		}
	}
	var renameErr error
	for attempt := 0; attempt < 3; attempt++ {
		renameErr = os.Rename(tmp, d.path)
		if renameErr == nil {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if renameErr != nil {
		d.lastErr = renameErr
		if d.onSave != nil {
			d.onSave(renameErr)
		}
		return renameErr
	}
	d.dirty = false
	d.lastErr = nil
	if d.onSave != nil {
		d.onSave(nil)
	}
	return nil
}

func (d *Doc[T]) Path() string { return d.path }

func writeSync(path string, data []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func MustDirErr(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("store: %w", err)
}
