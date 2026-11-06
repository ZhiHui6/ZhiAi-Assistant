package usage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/platform"
)

type CredStore struct {
	mu      sync.Mutex
	path    string
	entries map[string]model.UsageCredential
}

func OpenCredentials(path string) *CredStore {
	migrateLegacyCredentials(path)
	v := &CredStore{path: path, entries: map[string]model.UsageCredential{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		return v
	}
	plain, err := platform.UnprotectSecret(raw)
	if err != nil {
		return v
	}
	var entries map[string]model.UsageCredential
	if err := json.Unmarshal(plain, &entries); err != nil {
		return v
	}
	if entries != nil {
		v.entries = entries
	}
	return v
}

func migrateLegacyCredentials(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	legacy := filepath.Join(filepath.Dir(path), "usage.vault")
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	_ = os.Rename(legacy, path)
}

func (v *CredStore) Get(id string) (model.UsageCredential, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	auth, ok := v.entries[id]
	return auth, ok
}

func (v *CredStore) Set(id string, auth model.UsageCredential) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	prev, existed := v.entries[id]
	v.entries[id] = auth
	if err := v.persistLocked(); err != nil {
		if existed {
			v.entries[id] = prev
		} else {
			delete(v.entries, id)
		}
		return err
	}
	return nil
}

func (v *CredStore) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	prev, existed := v.entries[id]
	if !existed {
		return nil
	}
	delete(v.entries, id)
	if err := v.persistLocked(); err != nil {
		v.entries[id] = prev
		return err
	}
	return nil
}

func (v *CredStore) persistLocked() error {
	plain, err := json.Marshal(v.entries)
	if err != nil {
		return err
	}
	sealed, err := platform.ProtectSecret(plain)
	if err != nil {
		return errors.New("无法安全保存登录凭据")
	}
	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "usage.credentials.*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(sealed); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}
