package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zhi-ai-assistant/internal/buildinfo"
)

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	body := strings.TrimSpace(e.Body)
	if len(body) > 180 {
		body = body[:180]
	}
	if body == "" {
		return fmt.Sprintf("请求失败（HTTP %d）", e.Status)
	}
	return fmt.Sprintf("请求失败（HTTP %d）", e.Status)
}

func defaultDoer() Doer {
	return &http.Client{Timeout: 12 * time.Second}
}

func requestJSON(ctx context.Context, doer Doer, method, url string, headers map[string]string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if payload != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", buildinfo.AppSlug+"/"+buildinfo.Version)
	}
	resp, err := doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络不可用")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, &HTTPError{Status: resp.StatusCode, Body: string(raw)}
	}
	return raw, nil
}

func decodeMap(raw []byte) (map[string]any, error) {
	var value any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, fmt.Errorf("响应格式无法解析")
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("响应格式无法解析")
	}
	return m, nil
}
