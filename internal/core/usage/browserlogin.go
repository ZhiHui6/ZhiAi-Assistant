package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"zhi-ai-assistant/internal/core/model"
)

const (
	browserLoginTTL         = 5 * time.Minute
	browserLoginPollTimeout = 8 * time.Second
	browserCandidateTimeout = 9 * time.Second
)

var (
	jwtTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}$`)
	loginURLs       = map[string]string{
		"qoder":     "https://qoder.com.cn/login",
		"workbuddy": "https://www.workbuddy.cn/login",
		"trae":      "https://www.trae.cn/login",
	}
	tokenHarvestJS = `(function(){
  var out = {};
  var jwt = /^[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}$/;
  var add = function(v){
    if (typeof v !== 'string') return;
    var s = v.trim();
    if (!s || s.length > 4096) return;
    var bare = s.replace(/^"|"$/g, '').replace(/^(Bearer|Cloud-IDE-JWT)\s+/i, '');
    if (jwt.test(bare) || /^[A-Za-z0-9_.:\-]{24,}$/.test(bare)) out[bare] = 1;
  };
  var scan = function(v){
    add(v);
    try {
      var o = JSON.parse(v);
      if (o && typeof o === 'object') { for (var k in o) { if (typeof o[k] === 'string') add(o[k]); } }
    } catch (e) {}
  };
  try { for (var i = 0; i < localStorage.length; i++) scan(localStorage.getItem(localStorage.key(i)) || ''); } catch (e) {}
  try { for (var j = 0; j < sessionStorage.length; j++) scan(sessionStorage.getItem(sessionStorage.key(j)) || ''); } catch (e) {}
  try { document.cookie.split(';').forEach(function(p){ add(p.split('=').slice(1).join('=')); }); } catch (e) {}
  return Object.keys(out).slice(0, 24);
})()`
)

type BrowserLoginStart struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	ExpiresIn int    `json:"expiresIn"`
}

type BrowserLoginPoll struct {
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

type browserLoginSession struct {
	id       string
	platform string
	expires  time.Time
	proc     *exec.Cmd
	done     chan struct{}
	profile  string
	debug    string
	host     string
}

func (m *Manager) StartBrowserLogin(platform string) (BrowserLoginStart, error) {
	if !m.valid(platform) {
		return BrowserLoginStart{}, errors.New("未知平台")
	}
	loginURL, ok := loginURLs[platform]
	if !ok {
		return BrowserLoginStart{}, errors.New("该平台暂不支持浏览器登录")
	}
	exe, err := findBrowserExecutable()
	if err != nil {
		return BrowserLoginStart{}, err
	}
	port, err := freePort()
	if err != nil {
		return BrowserLoginStart{}, err
	}
	profile, err := os.MkdirTemp("", "zhi-ai-login-")
	if err != nil {
		return BrowserLoginStart{}, err
	}
	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--user-data-dir=" + profile,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--window-size=980,760",
		loginURL,
	}
	cmd := exec.Command(exe, args...)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(profile)
		return BrowserLoginStart{}, fmt.Errorf("无法启动浏览器：%w", err)
	}
	session := &browserLoginSession{
		id:       randomLoginID(),
		platform: platform,
		expires:  time.Now().Add(browserLoginTTL),
		proc:     cmd,
		done:     make(chan struct{}),
		profile:  profile,
		debug:    fmt.Sprintf("http://127.0.0.1:%d", port),
		host:     loginHost(loginURL),
	}
	go func() {
		_ = cmd.Wait()
		close(session.done)
	}()
	m.loginMu.Lock()
	if m.logins == nil {
		m.logins = map[string]*browserLoginSession{}
	}
	m.logins[session.id] = session
	m.loginMu.Unlock()
	return BrowserLoginStart{ID: session.id, URL: loginURL, ExpiresIn: int(browserLoginTTL.Seconds())}, nil
}

func (m *Manager) PollBrowserLogin(platform, loginID string) (BrowserLoginPoll, error) {
	m.loginMu.Lock()
	session := m.logins[loginID]
	m.loginMu.Unlock()
	if session == nil || session.platform != platform {
		return BrowserLoginPoll{Done: true, Error: "登录会话不存在，请重新发起"}, nil
	}
	if time.Now().After(session.expires) {
		m.closeBrowserLogin(session)
		return BrowserLoginPoll{Done: true, Error: "登录已超时，请重新发起"}, nil
	}
	if session.exited() {
		m.closeBrowserLogin(session)
		return BrowserLoginPoll{Done: true, Error: "登录窗口已关闭，请重新发起"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), browserLoginPollTimeout)
	candidates, err := harvestTokens(ctx, session.debug, session.host)
	cancel()
	if err != nil || len(candidates) == 0 {
		return BrowserLoginPoll{Error: "等待浏览器完成登录"}, nil
	}

	cred, snapshot, err := m.validateCandidates(platform, candidates)
	if err != nil {
		return BrowserLoginPoll{Error: "未能读取到可用的登录信息，请确认已完成登录"}, nil
	}
	if err := m.creds.Set(platform, cred); err != nil {
		m.closeBrowserLogin(session)
		return BrowserLoginPoll{Done: true, Error: err.Error()}, err
	}
	now := m.now()
	snapshot.Account = maskAccount(snapshot.Account)
	snapshot.UpdatedAt = now
	if snapshot.Unit == "" {
		snapshot.Unit = "credits"
	}
	m.update(platform, func(st *model.UsagePlatformState) {
		st.Snapshot = snapshot
		st.Source = SourceBrowser
		st.AuthState = AuthOK
		st.LastError = ""
		st.LastFetchAt = now
		st.Attempts = 0
		st.AttemptDate = ""
		st.RetryAt = time.Time{}
		st.SignedOut = false
		st.Preferred = SourceBrowser
	})
	m.closeBrowserLogin(session)
	return BrowserLoginPoll{Done: true}, nil
}

func (m *Manager) CancelBrowserLogin(loginID string) {
	m.loginMu.Lock()
	session := m.logins[loginID]
	m.loginMu.Unlock()
	if session != nil {
		m.closeBrowserLogin(session)
	}
}

func (m *Manager) cleanupBrowserLogins() {
	m.loginMu.Lock()
	var stale []*browserLoginSession
	for _, session := range m.logins {
		if time.Now().After(session.expires) || session.exited() {
			stale = append(stale, session)
		}
	}
	m.loginMu.Unlock()
	for _, session := range stale {
		m.closeBrowserLogin(session)
	}
}

func (m *Manager) closeBrowserLogin(session *browserLoginSession) {
	m.loginMu.Lock()
	delete(m.logins, session.id)
	m.loginMu.Unlock()
	if session.proc != nil && session.proc.Process != nil {
		_ = session.proc.Process.Kill()
	}
	if session.profile != "" {
		_ = os.RemoveAll(session.profile)
	}
}

func (s *browserLoginSession) exited() bool {
	if s == nil || s.done == nil {
		return true
	}
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

func (m *Manager) validateCandidates(platform string, candidates []string) (model.UsageCredential, model.UsageSnapshot, error) {
	provider := m.providers[platform]
	if provider == nil {
		return model.UsageCredential{}, model.UsageSnapshot{}, errors.New("未知平台")
	}
	for _, token := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), browserCandidateTimeout)
		auth := Auth{Token: token, Source: SourceBrowser}
		if platform == "workbuddy" {
			auth.Domain = "workbuddy.cn"
		}
		snapshot, err := provider.Fetch(ctx, auth)
		cancel()
		if err != nil {
			continue
		}
		return model.UsageCredential{Token: token}, snapshot, nil
	}
	return model.UsageCredential{}, model.UsageSnapshot{}, errors.New("没有可用凭据")
}

func harvestTokens(ctx context.Context, debugBase, host string) ([]string, error) {
	wsURL, err := pageWebSocketURL(ctx, debugBase, host)
	if err != nil {
		return nil, err
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(6 * time.Second))

	request := map[string]any{
		"id":     1,
		"method": "Runtime.evaluate",
		"params": map[string]any{
			"expression":    tokenHarvestJS,
			"returnByValue": true,
		},
	}
	if err := conn.WriteJSON(request); err != nil {
		return nil, err
	}
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		var message struct {
			ID     int `json:"id"`
			Result struct {
				Result struct {
					Value []string `json:"value"`
				} `json:"result"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &message); err != nil {
			continue
		}
		if message.ID != 1 {
			continue
		}
		return message.Result.Result.Value, nil
	}
}

func pageWebSocketURL(ctx context.Context, debugBase, host string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, debugBase+"/json/list", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var targets []struct {
		Type                 string `json:"type"`
		URL                  string `json:"url"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", err
	}
	fallback := ""
	for _, target := range targets {
		if target.Type != "page" || target.WebSocketDebuggerURL == "" {
			continue
		}
		if !strings.HasPrefix(target.URL, "http://") && !strings.HasPrefix(target.URL, "https://") {
			continue
		}
		if host != "" && strings.Contains(target.URL, host) {
			return target.WebSocketDebuggerURL, nil
		}
		if fallback == "" {
			fallback = target.WebSocketDebuggerURL
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", errors.New("未找到可调试的登录页面")
}

func loginHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func findBrowserExecutable() (string, error) {
	candidates := []string{}
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
		if root == "" {
			continue
		}
		candidates = append(candidates,
			filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
		)
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("未找到可用的浏览器，请先安装 Edge 或 Chrome")
}
