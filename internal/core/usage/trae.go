package usage

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zhi-ai-assistant/internal/buildinfo"
	"zhi-ai-assistant/internal/core/model"
)

const (
	traeStorageKey = "iCubeAuthInfo://icube.cloudide"
	traeUsageURL   = "https://api.trae.cn/trae/api/v2/pay/web_user_ent_usage"
	traeStatusURL  = "https://api.trae.cn/trae/api/v2/ug/checkin_credits/status"
	traeClaimURL   = "https://api.trae.cn/trae/api/v2/ug/checkin_credits/claim"
)

type traeProvider struct {
	doer Doer
}

func newTraeProvider(doer Doer) *traeProvider {
	return &traeProvider{doer: doer}
}

func (p *traeProvider) ID() string { return "trae" }

func (p *traeProvider) ReadLocal(env Env) (Auth, error) {
	appNames := []string{"Trae CN", "trae-cn", "Trae", "TRAE SOLO CN"}
	var encryptedErr error
	for _, name := range appNames {
		path := filepath.Join(env.AppData, name, "User", "globalStorage", "storage.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		appDir := filepath.Join(env.AppData, name)
		auth, err := parseTraeStorage(raw, appDir)
		if err != nil {
			if errors.Is(err, ErrEncrypted) {
				encryptedErr = err
			}
			continue
		}
		auth.Source = SourceLocal
		return auth, nil
	}
	if encryptedErr != nil {
		return Auth{}, encryptedErr
	}
	cliPath := filepath.Join(env.UserProfile, ".trae-cn", "trae-jwt-token")
	if raw, err := os.ReadFile(cliPath); err == nil {
		token := strings.TrimSpace(string(raw))
		if token != "" {
			uid, exp := traeJWTInfo(token)
			if !exp.IsZero() && exp.Before(pNow()) {
				return Auth{}, fmt.Errorf("%w：请在 Trae 中重新登录", ErrExpired)
			}
			return Auth{Token: token, UserID: uid, ExpiresAt: exp, Source: SourceLocal}, nil
		}
	}
	return Auth{}, ErrNoAuth
}

func parseTraeStorage(raw []byte, appDir string) (Auth, error) {
	var root map[string]any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return Auth{}, fmt.Errorf("%w：Trae 登录文件无法解析", ErrNoAuth)
	}
	value, ok := root[traeStorageKey]
	if !ok {
		return Auth{}, ErrNoAuth
	}
	var authMap map[string]any
	switch typed := value.(type) {
	case string:
		plain := strings.TrimSpace(typed)
		if !strings.HasPrefix(plain, "{") {
			decoded, err := decryptTraeStorageValue(plain)
			if err != nil {
				return Auth{}, fmt.Errorf("%w：%v", ErrEncrypted, err)
			}
			plain = decoded
		}
		if err := json.Unmarshal([]byte(plain), &authMap); err != nil {
			return Auth{}, fmt.Errorf("%w：Trae 登录内容无法解析", ErrEncrypted)
		}
	case map[string]any:
		authMap = typed
	default:
		return Auth{}, ErrNoAuth
	}
	token := text(authMap, "token", "accessToken", "access_token")
	if token == "" {
		return Auth{}, ErrNoAuth
	}
	machineID := text(root, "telemetry.machineId", "machineId", "machine_id")
	if machineID == "" && appDir != "" {
		if value, err := os.ReadFile(filepath.Join(appDir, "machineid")); err == nil {
			machineID = strings.TrimSpace(string(value))
		}
	}
	deviceID := traeDeviceID(root)
	if deviceID == "" && machineID != "" {
		sum := sha256.Sum256([]byte(machineID))
		deviceID = hex.EncodeToString(sum[:])[:32]
	}
	auth := Auth{
		Token:        token,
		RefreshToken: text(authMap, "refreshToken", "refresh_token"),
		UserID:       firstText(authMap, root, "userId", "user_id", "uid", "id"),
		DeviceID:     deviceID,
		Domain:       text(authMap, "host"),
	}
	if account, ok := authMap["account"].(map[string]any); ok {
		auth.Name = text(account, "username", "nickname", "name")
	}
	if auth.UserID == "" || auth.ExpiresAt.IsZero() {
		if uid, exp := traeJWTInfo(token); uid != "" || !exp.IsZero() {
			if auth.UserID == "" {
				auth.UserID = uid
			}
			auth.ExpiresAt = exp
		}
	}
	if value, ok := lookup(authMap, "expiresAt", "expires_at", "expireAt"); ok {
		auth.ExpiresAt = timeValue(value)
	}
	host := strings.ToLower(auth.Domain)
	if host != "" && !strings.Contains(host, "trae.cn") {
		return Auth{}, fmt.Errorf("%w：暂不支持 Trae 国际区", ErrUnsupported)
	}
	if !auth.ExpiresAt.IsZero() && auth.ExpiresAt.Before(pNow()) {
		return Auth{}, fmt.Errorf("%w：请在 Trae 中重新登录", ErrExpired)
	}
	return auth, nil
}

func traeDeviceID(root map[string]any) string {
	const prefix = "iCubeAuthInfo://icube-dc:"
	for key := range root {
		if strings.HasPrefix(key, prefix) {
			id := strings.TrimSpace(strings.TrimPrefix(key, prefix))
			if id != "" {
				return id
			}
		}
	}
	if id := text(root, "telemetry.devDeviceId", "devDeviceId", "deviceId"); id != "" {
		return id
	}
	return ""
}

func traeJWTInfo(token string) (string, time.Time) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", time.Time{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return "", time.Time{}
	}
	uid := firstNonEmpty(text(claims, "user_id", "userId", "sub", "uid"), text(claims, "id"))
	var exp time.Time
	if value, ok := lookup(claims, "exp"); ok {
		exp = timeValue(value)
	}
	return uid, exp
}

func traeHeaders(auth Auth) map[string]string {
	headers := map[string]string{
		"Authorization": "Cloud-IDE-JWT " + auth.Token,
		"Origin":        "https://www.trae.cn",
		"Referer":       "https://www.trae.cn/",
		"User-Agent":    buildinfo.AppSlug,
	}
	if auth.DeviceID != "" {
		headers["x-device-id"] = auth.DeviceID
	}
	return headers
}

func (p *traeProvider) Fetch(ctx context.Context, auth Auth) (model.UsageSnapshot, error) {
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, traeUsageURL, traeHeaders(auth), map[string]any{"require_usage": true})
	debugLog("trae", traeUsageURL, raw, err)
	if err != nil {
		return model.UsageSnapshot{}, traeHTTPError(err)
	}
	root, err := decodeMap(raw)
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	if code := traeCode(root); code != 0 {
		return model.UsageSnapshot{}, traeBusinessError(code, root)
	}
	total, ratio, plan, reset, packList, ok := traeUsage(root)
	if !ok {
		return model.UsageSnapshot{}, errors.New("Trae 用量数据格式变化，暂时无法解析")
	}
	used := total * ratio
	if used < 0 {
		used = 0
	}
	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}
	snapshot := model.UsageSnapshot{
		Account:    maskAccount(firstNonEmpty(auth.Name, auth.UserID)),
		AccountRaw: firstNonEmpty(auth.Name, auth.UserID),
		Plan:       plan,
		Unit:       "credits",
		Total:      total,
		Used:       used,
		Remaining:  remaining,
		Percent:    normalizedPercent(used, total, ratio),
		ResetAt:    reset,
		Packs:      packList,
		UpdatedAt:  pNow(),
	}
	checked, available, daily, message := p.traeCheckin(ctx, auth)
	snapshot.CheckedIn = checked
	snapshot.CheckinAvailable = available
	snapshot.DailyCredit = daily
	snapshot.Message = message
	return snapshot, nil
}

func (p *traeProvider) Claim(ctx context.Context, auth Auth) (ClaimResult, error) {
	if auth.DeviceID == "" {
		return ClaimResult{}, errors.New("缺少设备标识，请先在 Trae 客户端登录并签到一次")
	}
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, traeClaimURL, traeHeaders(auth), map[string]any{})
	if err != nil {
		return ClaimResult{}, traeHTTPError(err)
	}
	root, _ := decodeMap(raw)
	code := traeCode(root)
	if code == 9095 {
		return ClaimResult{Already: true, Message: "今日已领取"}, nil
	}
	if code != 0 {
		return ClaimResult{}, traeBusinessError(code, root)
	}
	amount, _ := num(root, "credits", "credit", "amount", "reward")
	if data := unwrapMap(root, "data"); data != nil {
		if v, ok := num(data, "credits", "credit", "amount", "reward"); ok {
			amount = v
		}
	}
	return ClaimResult{Claimed: true, Amount: amount, Message: "签到成功"}, nil
}

func (p *traeProvider) traeCheckin(ctx context.Context, auth Auth) (bool, bool, float64, string) {
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, traeStatusURL, traeHeaders(auth), map[string]any{})
	if err != nil {
		return false, false, 0, "签到状态读取失败：" + traeHTTPError(err).Error()
	}
	root, err := decodeMap(raw)
	if err != nil {
		return false, false, 0, "签到状态读取失败"
	}
	if code := traeCode(root); code != 0 && code != 9095 {
		return false, false, 0, traeBusinessError(code, root).Error()
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	checked, _ := boolean(data, "checked_in", "checkedIn")
	did, _ := boolean(data, "did_checked_in", "didCheckedIn")
	if code := traeCode(root); code == 9095 {
		checked = true
	}
	available := !checked
	if did && !checked {
		available = false
	}
	daily, _ := num(data, "credits", "daily_credits", "dailyCredit", "reward", "amount")
	return checked, available, daily, ""
}

func traeUsage(root map[string]any) (float64, float64, string, string, []model.UsagePack, bool) {
	var total, ratio float64
	found := false
	walkMaps(root, func(m map[string]any) {
		t, ok := num(m, "total_amount", "totalAmount", "total")
		if !ok || t <= 0 {
			return
		}
		r, rok := num(m, "consumption_ratio", "consumptionRatio", "percentage", "ratio")
		if !rok {
			return
		}
		if !found || t > total {
			total = t
			ratio = r
			found = true
		}
	})
	if !found {
		return 0, 0, "", "", nil, false
	}
	if ratio > 1 {
		ratio = ratio / 100
	}
	plan := ""
	reset := ""
	var packList []model.UsagePack
	var latest time.Time
	now := time.Now()
	for _, item := range findArray(root, "user_entitlement_pack_list", "packs", "entitlementPacks") {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		limit, limitOK := num(m, "credits_limit")
		if !limitOK {
			if base, ok := m["entitlement_base_info"].(map[string]any); ok {
				for _, holder := range []any{base} {
					if quota, qok := holder.(map[string]any)["quota"].(map[string]any); qok {
						if limit, limitOK = num(quota, "credits_limit"); limitOK {
							break
						}
					}
				}
			}
			if !limitOK {
				if extra, eok := m["entitlement_base_info"].(map[string]any); eok {
					if pe, pok := extra["product_extra"].(map[string]any); pok {
						for _, nested := range []string{"package_extra", "subscription_extra"} {
							if sub, sok := pe[nested].(map[string]any); sok {
								if quota, qok := sub["quota"].(map[string]any); qok {
									if limit, limitOK = num(quota, "credits_limit"); limitOK {
										break
									}
								}
							}
						}
					}
				}
			}
		}
		if !limitOK || limit <= 0 {
			continue
		}
		expireAt := time.Time{}
		if raw, ok := lookup(m, "expire_time", "expireTime", "end_time"); ok {
			expireAt = timeValue(raw)
		}
		if !expireAt.IsZero() && expireAt.Before(now) {
			continue
		}
		used, usedOK := num(m, "credits_amount")
		if !usedOK {
			if usage, uok := m["usage"].(map[string]any); uok {
				used, usedOK = num(usage, "credits_amount")
			}
		}
		if !usedOK {
			continue
		}
		remaining := limit - used
		if remaining <= 0 {
			continue
		}
		name := text(m, "group_name", "groupName", "name", "packName", "productName", "title")
		if name == "" {
			name = text(m, "entitlement_id")
		}
		expires := ""
		if !expireAt.IsZero() {
			expires = expireAt.Format("2006-01-02 15:04:05")
		}
		packList = append(packList, model.UsagePack{
			Name:      name,
			Total:     limit,
			Remaining: remaining,
			Unit:      "credits",
			ExpiresAt: expires,
		})
		if expireAt.After(latest) {
			latest = expireAt
		}
	}
	if plan == "" {
		plan = text(root, "planName", "plan", "name")
	}
	if reset == "" && !latest.IsZero() {
		reset = latest.Format("2006-01-02 15:04:05")
	}
	if reset == "" {
		reset = text(root, "resetAt", "resetTime", "expireAt", "expiresAt", "cycleResetTime")
	}
	return total, ratio, plan, reset, packList, true
}

func traeCode(root map[string]any) int {
	if root == nil {
		return 0
	}
	if value, ok := num(root, "code", "statusCode", "status_code"); ok {
		return int(value)
	}
	return 0
}

func traeBusinessError(code int, root map[string]any) error {
	message := text(root, "message", "msg", "error")
	if message == "" {
		message = fmt.Sprintf("Trae 返回错误码 %d", code)
	}
	switch code {
	case 9004:
		return errors.New("缺少设备标识：" + message)
	case 9095:
		return ErrAlreadyClaim
	case 20403:
		return errors.New("Trae 设备校验失败：" + message)
	}
	return errors.New(message)
}

func traeHTTPError(err error) error {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w：请在 Trae 中重新登录", ErrExpired)
		case http.StatusForbidden:
			return fmt.Errorf("%w：Trae 拒绝了该请求", ErrBlocked)
		}
		return fmt.Errorf("Trae 请求失败（HTTP %d）", httpErr.Status)
	}
	return err
}
