package usage

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"zhi-ai-assistant/internal/platform"
)

func qoderOSCryptKey(appDir string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(appDir, "Local State"))
	if err != nil {
		return nil, err
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	crypt, ok := state["os_crypt"].(map[string]any)
	if !ok {
		return nil, errors.New("OSCrypt 配置缺失")
	}
	encoded, ok := crypt["encrypted_key"].(string)
	if !ok || strings.TrimSpace(encoded) == "" {
		return nil, errors.New("OSCrypt 密钥缺失")
	}
	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(blob) <= 5 {
		return nil, errors.New("OSCrypt 密钥格式无效")
	}
	key, err := platform.UnprotectSecret(blob[5:])
	if err != nil || len(key) != 32 {
		return nil, errors.New("OSCrypt 密钥无法解锁")
	}
	return key, nil
}

func qoderDecryptOSCrypt(raw, key []byte) ([]byte, error) {
	if len(raw) < 3+12+16 {
		return nil, errors.New("OSCrypt 数据过短")
	}
	prefix := string(raw[:3])
	if prefix != "v10" && prefix != "v11" {
		return nil, errors.New("OSCrypt 版本不受支持")
	}
	nonce := raw[3:15]
	payload := raw[15:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, errors.New("OSCrypt 解密失败")
	}
	return plain, nil
}
