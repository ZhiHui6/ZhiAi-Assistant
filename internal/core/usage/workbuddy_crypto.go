package usage

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"zhi-ai-assistant/internal/core/store"
	"zhi-ai-assistant/internal/platform"
)

const (
	wbAADDomain        = "WB-AAD\x00"
	wbFieldFormat      = "WBEV1"
	wbSymmetricScheme  = "sym-v1"
	wbKeyFetchTimeout  = 10 * time.Second
	wbKeyFailurePeriod = time.Minute
)

var wbKeyIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

type wbAtRestKeyCache struct {
	mu     sync.Mutex
	key    []byte
	keyErr error
	failAt time.Time
}

var wbKeyCache wbAtRestKeyCache

func wbKeyCachePath(env Env) string {
	return filepath.Join(store.DataDir(), "data", "workbuddy.at-rest.key")
}

func wbLoadCachedKey(env Env) []byte {
	raw, err := os.ReadFile(wbKeyCachePath(env))
	if err != nil {
		return nil
	}
	key, err := platform.UnprotectSecret(raw)
	if err != nil || len(key) != 32 {
		return nil
	}
	return key
}

func wbSaveCachedKey(env Env, key []byte) {
	sealed, err := platform.ProtectSecret(key)
	if err != nil {
		return
	}
	path := wbKeyCachePath(env)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, sealed, 0o644)
}

func wbAtRestKey(ctx context.Context, env Env) ([]byte, error) {
	wbKeyCache.mu.Lock()
	defer wbKeyCache.mu.Unlock()
	if wbKeyCache.key != nil {
		return wbKeyCache.key, nil
	}
	if cached := wbLoadCachedKey(env); cached != nil {
		wbKeyCache.key = cached
		return cached, nil
	}
	now := time.Now()
	if wbKeyCache.keyErr != nil && now.Sub(wbKeyCache.failAt) < wbKeyFailurePeriod {
		return nil, wbKeyCache.keyErr
	}
	exe, err := findClientExecutable(env, "workbuddy")
	if err != nil {
		wbKeyCache.keyErr, wbKeyCache.failAt = err, now
		return nil, err
	}
	script := "try{process.stdout.write(process._linkedBinding('electron_browser_workbuddy_storage').loggerGet())}catch(e){process.exitCode=3;process.stderr.write(String(e&&e.message||e))}"
	cmd := exec.CommandContext(ctx, exe, "-e", script)
	cmd.Env = append(os.Environ(), "ELECTRON_RUN_AS_NODE=1")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		wbKeyCache.keyErr, wbKeyCache.failAt = fmt.Errorf("无法启动 WorkBuddy 客户端：%w", err), now
		return nil, wbKeyCache.keyErr
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(wbKeyFetchTimeout):
		_ = cmd.Process.Kill()
		<-done
		wbKeyCache.keyErr, wbKeyCache.failAt = errors.New("WorkBuddy 客户端响应超时"), now
		return nil, wbKeyCache.keyErr
	case waitErr := <-done:
		payload := strings.TrimSpace(out.String())
		if waitErr != nil || payload == "" {
			wbKeyCache.keyErr, wbKeyCache.failAt = errors.New("WorkBuddy 客户端未能提供静置密钥"), now
			return nil, wbKeyCache.keyErr
		}
	}
	var doc struct {
		AtRestSecretKey string `json:"atRestSecretKey"`
	}
	if err := json.Unmarshal([]byte(out.String()), &doc); err != nil || doc.AtRestSecretKey == "" {
		wbKeyCache.keyErr, wbKeyCache.failAt = errors.New("WorkBuddy 静置密钥格式异常"), now
		return nil, wbKeyCache.keyErr
	}
	sum := sha256.Sum256([]byte(doc.AtRestSecretKey))
	wbKeyCache.key = sum[:]
	wbKeyCache.keyErr = nil
	wbSaveCachedKey(env, wbKeyCache.key)
	return wbKeyCache.key, nil
}

func wbLenPrefixed(value string) []byte {
	out := make([]byte, 4, 4+len(value))
	binary.BigEndian.PutUint32(out, uint32(len(value)))
	return append(out, value...)
}

func wbFieldAAD(keyID string, suite uint32) []byte {
	var out []byte
	out = append(out, wbAADDomain...)
	out = append(out, 1)
	out = append(out, wbLenPrefixed(wbFieldFormat)...)
	out = append(out, wbLenPrefixed(wbSymmetricScheme)...)
	var suiteBytes [4]byte
	binary.BigEndian.PutUint32(suiteBytes[:], suite)
	out = append(out, suiteBytes[:]...)
	out = append(out, wbLenPrefixed(keyID)...)
	out = append(out, 2, 0, 0)
	return out
}

func wbDecodeBase64(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(value))
}

func wbOpenEncryptedField(wrapper map[string]any, key []byte) (string, error) {
	envelopeRaw, ok := wrapper["envelope"].(string)
	if !ok || envelopeRaw == "" {
		return "", errors.New("WorkBuddy 加密字段缺少 envelope")
	}
	var envelope struct {
		Suite      uint32 `json:"suite"`
		KeyID      string `json:"keyId"`
		Nonce      string `json:"nonce"`
		AuthTag    string `json:"authTag"`
		Ciphertext string `json:"ciphertext"`
	}
	envelopeJSON, err := wbDecodeBase64(envelopeRaw)
	if err != nil || json.Unmarshal(envelopeJSON, &envelope) != nil {
		return "", errors.New("WorkBuddy 加密信封无法解析")
	}
	if !wbKeyIDPattern.MatchString(envelope.KeyID) {
		return "", errors.New("WorkBuddy 加密信封 keyId 非法")
	}
	sum := sha256.Sum256(key)
	expectedKeyID := fmt.Sprintf("%x", sum[:16])
	if envelope.KeyID != expectedKeyID {
		return "", fmt.Errorf("WorkBuddy 加密字段属于另一把密钥（%s）", envelope.KeyID)
	}
	nonce, err := wbDecodeBase64(envelope.Nonce)
	if err != nil || len(nonce) != 12 {
		return "", errors.New("WorkBuddy 加密信封 nonce 非法")
	}
	tag, err := wbDecodeBase64(envelope.AuthTag)
	if err != nil || len(tag) != 16 {
		return "", errors.New("WorkBuddy 加密信封 authTag 非法")
	}
	ciphertext, err := wbDecodeBase64(envelope.Ciphertext)
	if err != nil {
		return "", errors.New("WorkBuddy 加密信封 ciphertext 非法")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, nonce, append(ciphertext, tag...), wbFieldAAD(envelope.KeyID, envelope.Suite))
	if err != nil {
		return "", errors.New("WorkBuddy 加密字段校验失败")
	}
	return string(plain), nil
}

func isWbEncryptedWrapper(value any) bool {
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	_, hasFlag := m["$wbEncrypted"]
	_, hasEnvelope := m["envelope"]
	return hasFlag && hasEnvelope && len(m) == 2
}

func wbDocField(doc map[string]any, path []string) (map[string]any, string, bool) {
	node := doc
	for _, segment := range path {
		child, ok := node[segment].(map[string]any)
		if !ok {
			if value, exists := node[segment]; exists && !isWbEncryptedWrapper(value) {
				return nil, "", false
			}
			return nil, "", false
		}
		node = child
	}
	return node, "", false
}

func wbDecryptDocFields(doc map[string]any, key []byte) {
	if key == nil || doc == nil {
		return
	}
	for _, path := range [][]string{
		{"auth"},
		{"account"},
	} {
		parent, ok := doc[path[0]].(map[string]any)
		if !ok {
			continue
		}
		for _, field := range []string{"accessToken", "refreshToken", "nickname", "phoneNumber"} {
			wrapper, ok := parent[field].(map[string]any)
			if !ok || !isWbEncryptedWrapper(wrapper) {
				continue
			}
			if plain, err := wbOpenEncryptedField(wrapper, key); err == nil {
				parent[field] = plain
			}
		}
	}
}

func wbInvalidateKey(env Env) {
	wbKeyCache.mu.Lock()
	wbKeyCache.key = nil
	wbKeyCache.keyErr = nil
	wbKeyCache.mu.Unlock()
	_ = os.Remove(wbKeyCachePath(env))
}

func wbDocHasEncryptedFields(doc map[string]any) bool {
	for _, parent := range []map[string]any{
		func() map[string]any { m, _ := doc["auth"].(map[string]any); return m }(),
		func() map[string]any { m, _ := doc["account"].(map[string]any); return m }(),
	} {
		if parent == nil {
			continue
		}
		for _, field := range []string{"accessToken", "refreshToken", "nickname", "phoneNumber"} {
			if wrapper, ok := parent[field].(map[string]any); ok && isWbEncryptedWrapper(wrapper) {
				return true
			}
		}
	}
	return false
}
