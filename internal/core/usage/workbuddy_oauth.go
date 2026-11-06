package usage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	wbPluginBase = "https://www.workbuddy.cn/v2/plugin"
	wbOAuthTTL   = 10 * time.Minute
)

type wbOAuthSession struct {
	state     string
	expiresAt time.Time
}

type workBuddyOAuth struct {
	mu       sync.Mutex
	sessions map[string]wbOAuthSession
}

func newWorkBuddyOAuth() *workBuddyOAuth {
	return &workBuddyOAuth{sessions: map[string]wbOAuthSession{}}
}

func (o *workBuddyOAuth) start(ctx context.Context, doer Doer) (LoginStart, error) {
	raw, err := requestJSON(ctx, doer, http.MethodPost, wbPluginBase+"/auth/state?platform=workbuddy", nil, map[string]any{})
	if err != nil {
		return LoginStart{}, workBuddyError(err)
	}
	root, err := decodeMap(raw)
	if err != nil {
		return LoginStart{}, errors.New("WorkBuddy 登录服务返回异常")
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	state := text(data, "state")
	if state == "" {
		return LoginStart{}, errors.New("WorkBuddy 未返回登录 state")
	}
	authURL := text(data, "authUrl", "auth_url", "url", "Url")
	if authURL == "" {
		authURL = "https://www.workbuddy.cn/login?state=" + url.QueryEscape(state)
	}
	loginID := randomLoginID()
	o.mu.Lock()
	o.sessions[loginID] = wbOAuthSession{state: state, expiresAt: time.Now().Add(wbOAuthTTL)}
	o.mu.Unlock()
	return LoginStart{ID: loginID, URL: authURL, ExpiresIn: int(wbOAuthTTL.Seconds())}, nil
}

func (o *workBuddyOAuth) poll(ctx context.Context, doer Doer, loginID string) (Auth, bool, string) {
	o.mu.Lock()
	session, ok := o.sessions[loginID]
	if !ok {
		o.mu.Unlock()
		return Auth{}, true, "登录会话不存在，请重新发起登录"
	}
	if time.Now().After(session.expiresAt) {
		delete(o.sessions, loginID)
		o.mu.Unlock()
		return Auth{}, true, "登录已超时，请重新发起登录"
	}
	o.mu.Unlock()

	raw, err := requestJSON(ctx, doer, http.MethodGet, wbPluginBase+"/auth/token?state="+url.QueryEscape(session.state), nil, nil)
	if err != nil {
		return Auth{}, false, "等待浏览器完成登录"
	}
	root, err := decodeMap(raw)
	if err != nil {
		return Auth{}, false, "等待浏览器完成登录"
	}
	code, _ := num(root, "code")
	if code != 0 && code != 200 {
		return Auth{}, false, "等待浏览器完成登录"
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	token := text(data, "accessToken", "access_token", "AccessToken")
	if token == "" {
		return Auth{}, false, "等待浏览器完成登录"
	}
	domain := text(data, "domain", "Domain")
	if domain == "" {
		domain = "workbuddy.cn"
	}
	uid := text(data, "uid", "userId", "user_id")
	name := text(data, "nickname", "nickName", "displayName")
	profileHeaders := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
		"X-Domain":      domain,
	}
	if profileRaw, profileErr := requestJSON(ctx, doer, http.MethodGet, wbPluginBase+"/login/account", profileHeaders, nil); profileErr == nil {
		if profileRoot, decodeErr := decodeMap(profileRaw); decodeErr == nil {
			profile := unwrapMap(profileRoot, "data")
			if profile == nil {
				profile = profileRoot
			}
			if uid == "" {
				uid = text(profile, "uid", "Uid", "userId", "UserID")
			}
			if name == "" {
				name = text(profile, "nickname", "NickName", "ScreenName")
			}
		}
	}
	if uid == "" {
		uid, _ = traeJWTInfo(token)
	}
	auth := Auth{
		Token:        token,
		RefreshToken: text(data, "refreshToken", "refresh_token", "RefreshToken"),
		UserID:       uid,
		Name:         name,
		Domain:       domain,
		Source:       SourceBrowser,
	}
	if expires, ok := lookup(data, "expiresAt", "expires_at"); ok {
		auth.ExpiresAt = timeValue(expires)
	} else if seconds, ok := num(data, "expiresIn", "expires_in"); ok && seconds > 0 {
		auth.ExpiresAt = time.Now().Add(time.Duration(seconds) * time.Second)
	}
	o.mu.Lock()
	delete(o.sessions, loginID)
	o.mu.Unlock()
	return auth, true, ""
}

func randomLoginID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("wb_%d", time.Now().UnixNano())
	}
	return "wb_" + hex.EncodeToString(raw[:])
}
