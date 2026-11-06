package usage

import (
	"context"
	"errors"
	"os"
	"time"

	"zhi-ai-assistant/internal/core/model"
)

const (
	SourceNone    = "none"
	SourceLocal   = "local"
	SourceBrowser = "browser"

	AuthOK        = "ok"
	AuthMissing   = "missing"
	AuthExpired   = "expired"
	AuthEncrypted = "encrypted"
	AuthBlocked   = "blocked"
)

var (
	ErrNoAuth       = errors.New("未找到本机登录态")
	ErrNotLoggedIn  = errors.New("请先登录")
	ErrExpired      = errors.New("登录状态已过期")
	ErrEncrypted    = errors.New("登录信息已加密")
	ErrBlocked      = errors.New("平台拒绝了该请求")
	ErrUnsupported  = errors.New("暂不支持该区域")
	ErrAlreadyClaim = errors.New("今日已领取")
	ErrSignedOut    = errors.New("已退出登录")
)

type Env struct {
	AppData      string
	LocalAppData string
	UserProfile  string
}

func SystemEnv() Env {
	return Env{
		AppData:      os.Getenv("APPDATA"),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
		UserProfile:  os.Getenv("USERPROFILE"),
	}
}

type Auth struct {
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
	UserID       string
	Name         string
	Email        string
	Domain       string
	DeviceID     string
	Source       string
}

type ClaimResult struct {
	Claimed bool
	Already bool
	Amount  float64
	Message string
}

type LoginStart struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	ExpiresIn int    `json:"expiresIn"`
}

type LoginPoll struct {
	Done  bool   `json:"done"`
	Error string `json:"error,omitempty"`
}

type Provider interface {
	ID() string
	ReadLocal(Env) (Auth, error)
	Fetch(context.Context, Auth) (model.UsageSnapshot, error)
	Claim(context.Context, Auth) (ClaimResult, error)
}

func authStateFor(err error) string {
	switch {
	case err == nil:
		return AuthOK
	case errors.Is(err, ErrExpired):
		return AuthExpired
	case errors.Is(err, ErrEncrypted):
		return AuthEncrypted
	case errors.Is(err, ErrBlocked), errors.Is(err, ErrUnsupported):
		return AuthBlocked
	default:
		return AuthMissing
	}
}
