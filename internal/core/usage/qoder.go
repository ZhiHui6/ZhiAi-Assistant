package usage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/platform"
)

const (
	qoderOpenAPI     = "https://openapi.qoder.com.cn"
	qoderUsageURL    = qoderOpenAPI + "/sash/api/v2/me/usage"
	qoderUsageAltURL = qoderOpenAPI + "/api/v2/quota/usage"
	qoderCampaignURL = qoderOpenAPI + "/sash/api/v1/me/campaigns"
)

type qoderProvider struct {
	doer Doer
	now  func() time.Time
}

func newQoderProvider(doer Doer) *qoderProvider {
	return &qoderProvider{doer: doer, now: time.Now}
}

func (p *qoderProvider) ID() string { return "qoder" }

func (p *qoderProvider) ReadLocal(env Env) (Auth, error) {
	dirs := []string{"com.qodercn.app.stable", "QoderCN", "Qoder CN", "QoderWork CN"}
	var lastErr error
	for _, name := range dirs {
		path := filepath.Join(env.AppData, name, "auth.v1.dat")
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		auth, err := parseQoderAuth(raw)
		if err == nil {
			auth.Source = SourceLocal
			return auth, nil
		}
		if key, keyErr := qoderOSCryptKey(filepath.Dir(path)); keyErr == nil {
			if plain, decryptErr := qoderDecryptOSCrypt(raw, key); decryptErr == nil {
				if decoded, parseErr := parseQoderAuth(plain); parseErr == nil {
					decoded.Source = SourceLocal
					return decoded, nil
				}
			}
		}
		lastErr = err
	}
	if lastErr != nil {
		return Auth{}, lastErr
	}
	return Auth{}, ErrNoAuth
}

func parseQoderAuth(raw []byte) (Auth, error) {
	raw = bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}))
	raw = bytes.Trim(raw, "\x00")
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		var encoded string
		if json.Unmarshal(raw, &encoded) == nil {
			encoded = strings.TrimSpace(encoded)
			if strings.HasPrefix(encoded, "{") {
				dec = json.NewDecoder(strings.NewReader(encoded))
				dec.UseNumber()
				if dec.Decode(&root) == nil {
					return parseQoderAuthMap(root)
				}
			}
		}
		if plain, derr := platform.UnprotectSecret(raw); derr == nil {
			dec = json.NewDecoder(bytes.NewReader(plain))
			dec.UseNumber()
			if dec.Decode(&root) == nil {
				return parseQoderAuthMap(root)
			}
		}
		return Auth{}, fmt.Errorf("%w：登录文件格式暂不支持，请填写 Token", ErrEncrypted)
	}
	return parseQoderAuthMap(root)
}

func parseQoderAuthMap(root map[string]any) (Auth, error) {
	authRoot := root
	if nested := unwrapMap(root, "auth", "data", "credential", "credentials", "payload"); nested != nil {
		authRoot = nested
	}
	token := text(authRoot, "token", "accessToken", "access_token")
	if token == "" {
		if _, ok := lookup(authRoot, "token", "accessToken", "access_token"); ok {
			return Auth{}, fmt.Errorf("%w：请在 Qoder 中重新登录", ErrEncrypted)
		}
		return Auth{}, ErrNoAuth
	}
	user := unwrapMap(authRoot, "user", "userInfo", "account")
	auth := Auth{
		Token:        token,
		RefreshToken: text(authRoot, "refreshToken", "refresh_token"),
		UserID:       firstText(user, authRoot, "id", "uid", "userId", "user_id"),
		Name:         firstText(user, authRoot, "name", "nickname", "displayName"),
		Email:        firstText(user, authRoot, "email"),
		DeviceID:     text(authRoot, "machineId", "machine_id", "machineid"),
	}
	if expires, ok := lookup(authRoot, "expiresAt", "expires_at", "expireAt", "expire_at"); ok {
		auth.ExpiresAt = timeValue(expires)
	}
	if !auth.ExpiresAt.IsZero() && auth.ExpiresAt.Before(pNow()) {
		return Auth{}, fmt.Errorf("%w：请在 Qoder 中重新登录", ErrExpired)
	}
	return auth, nil
}

func pNow() time.Time { return time.Now() }

func firstText(primary, fallback map[string]any, names ...string) string {
	if value := text(primary, names...); value != "" {
		return value
	}
	return text(fallback, names...)
}

func qoderHeaders(auth Auth) map[string]string {
	return map[string]string{
		"Authorization":   "Bearer " + auth.Token,
		"Cosy-ClientType": "10",
		"User-Agent":      "Qoder",
	}
}

func (p *qoderProvider) Fetch(ctx context.Context, auth Auth) (model.UsageSnapshot, error) {
	headers := qoderHeaders(auth)
	raw, err := requestJSON(ctx, p.doer, http.MethodGet, qoderUsageURL, headers, nil)
	debugLog("qoder", qoderUsageURL, raw, err)
	if err != nil {
		raw, err = requestJSON(ctx, p.doer, http.MethodGet, qoderUsageAltURL, headers, nil)
		debugLog("qoder", qoderUsageAltURL, raw, err)
	}
	if err != nil {
		return model.UsageSnapshot{}, qoderHTTPError(err)
	}
	root, err := decodeMap(raw)
	if err != nil {
		return model.UsageSnapshot{}, err
	}
	total, used, remaining, unit, plan, reset, packs, ok := qoderQuota(root)
	if !ok {
		return model.UsageSnapshot{}, fmt.Errorf("Qoder 用量数据格式变化，暂时无法解析")
	}
	snapshot := model.UsageSnapshot{
		Account:    maskAccount(firstNonEmpty(auth.Name, auth.Email, auth.UserID)),
		AccountRaw: firstNonEmpty(auth.Name, auth.Email, auth.UserID),
		Plan:       plan,
		Unit:       unit,
		Total:      total,
		Used:       used,
		Remaining:  remaining,
		Percent:    normalizedPercent(used, total, 0),
		ResetAt:    reset,
		Packs:      packs,
		UpdatedAt:  p.now(),
	}
	if snapshot.Unit == "" {
		snapshot.Unit = "credits"
	}
	checked, available, daily, message := p.qoderCampaign(ctx, auth)
	snapshot.CheckedIn = checked
	snapshot.CheckinAvailable = available
	snapshot.DailyCredit = daily
	snapshot.Message = message
	return snapshot, nil
}

func (p *qoderProvider) Claim(ctx context.Context, auth Auth) (ClaimResult, error) {
	campaigns, err := p.qoderCampaigns(ctx, auth)
	if err != nil {
		return ClaimResult{}, err
	}
	var claimable *qoderCampaign
	already := false
	for i := range campaigns {
		if campaigns[i].claimed {
			already = true
		}
		if campaigns[i].claimable && claimable == nil {
			claimable = &campaigns[i]
		}
	}
	if claimable == nil {
		if already {
			return ClaimResult{Already: true, Message: "今日已领取"}, nil
		}
		return ClaimResult{}, errors.New("当前没有可领取的签到活动")
	}
	url := qoderCampaignURL + "/" + claimable.id + "/claim"
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, url, qoderHeaders(auth), map[string]any{})
	if err != nil {
		return ClaimResult{}, qoderHTTPError(err)
	}
	root, _ := decodeMap(raw)
	replayed, _ := boolean(root, "replayed")
	amount, _ := num(root, "amount", "credit", "credits", "reward")
	message := "签到成功"
	if replayed {
		message = "今日已领取"
	}
	return ClaimResult{Claimed: true, Already: replayed, Amount: amount, Message: message}, nil
}

type qoderCampaign struct {
	id        string
	claimable bool
	claimed   bool
	amount    float64
}

func (p *qoderProvider) qoderCampaigns(ctx context.Context, auth Auth) ([]qoderCampaign, error) {
	raw, err := requestJSON(ctx, p.doer, http.MethodGet, qoderCampaignURL, qoderHeaders(auth), nil)
	if err != nil {
		return nil, qoderHTTPError(err)
	}
	root, err := decodeMap(raw)
	if err != nil {
		return nil, err
	}
	list := findArray(root, "campaigns")
	if list == nil {
		if data, ok := lookup(root, "data"); ok {
			if array, ok := data.([]any); ok {
				list = array
			}
		}
	}
	out := make([]qoderCampaign, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		action := strings.ToUpper(text(m, "actionType", "action_type", "type"))
		status := strings.ToUpper(text(m, "claimStatus", "claim_status", "status"))
		id := text(m, "id", "campaignId", "campaign_id")
		if action != "CLAIM_BENEFIT" || id == "" {
			continue
		}
		amount, _ := num(m, "amount", "credit", "credits", "reward", "dailyCredit")
		out = append(out, qoderCampaign{
			id:        id,
			claimable: status != "CLAIMED" && status != "EXPIRED" && status != "ENDED",
			claimed:   status == "CLAIMED",
			amount:    amount,
		})
	}
	return out, nil
}

func (p *qoderProvider) qoderCampaign(ctx context.Context, auth Auth) (bool, bool, float64, string) {
	campaigns, err := p.qoderCampaigns(ctx, auth)
	if err != nil {
		return false, false, 0, fmt.Sprintf("签到状态读取失败：%v", err)
	}
	checked := false
	available := false
	amount := 0.0
	for _, campaign := range campaigns {
		if campaign.claimed {
			checked = true
		}
		if campaign.claimable {
			available = true
			if amount == 0 {
				amount = campaign.amount
			}
		}
	}
	return checked, available, amount, ""
}

func qoderQuota(root map[string]any) (float64, float64, float64, string, string, string, []model.UsagePack, bool) {
	bestTotal := -1.0
	var best struct {
		total, used, remaining float64
		unit, plan, reset      string
	}
	walkMaps(root, func(m map[string]any) {
		total, ok := num(m, "total", "totalAmount", "total_amount", "limit", "limitNum", "quotaTotal", "amount")
		if !ok || total <= 0 {
			return
		}
		used, usedOK := num(m, "used", "usedAmount", "used_amount", "consumed", "consumedAmount", "credit", "usedCredit")
		remaining, remainOK := num(m, "remaining", "remain", "balance", "available", "remainCredit", "remainingCredit")
		if !usedOK && !remainOK {
			return
		}
		if !usedOK {
			used = total - remaining
		}
		if !remainOK {
			remaining = total - used
		}
		if used < 0 {
			used = 0
		}
		if remaining < 0 {
			remaining = 0
		}
		if total > bestTotal {
			bestTotal = total
			best.total = total
			best.used = used
			best.remaining = remaining
			best.unit = text(m, "unit", "creditUnit", "credit_unit")
			best.plan = text(m, "planName", "plan_name", "plan", "packageName", "name", "title")
			best.reset = text(m, "resetAt", "reset_at", "resetTime", "reset_time", "expireAt", "expiresAt", "periodEnd")
		}
	})
	if bestTotal < 0 {
		return 0, 0, 0, "", "", "", nil, false
	}
	if best.unit == "" {
		best.unit = text(root, "unit")
	}
	if best.plan == "" {
		best.plan = text(root, "planName", "plan_name", "plan", "packageName", "name", "title")
	}
	return best.total, best.used, best.remaining, best.unit, best.plan, best.reset, qoderPacks(root, best.unit), true
}

func qoderPacks(root map[string]any, unit string) []model.UsagePack {
	container := root
	if qu, ok := root["qoderUsage"].(map[string]any); ok {
		container = qu
	}
	names := map[string]string{"userQuota": "订阅额度", "addOnQuota": "加量包"}
	var packs []model.UsagePack
	for _, section := range []string{"userQuota", "addOnQuota"} {
		m, ok := container[section].(map[string]any)
		if !ok {
			continue
		}
		total, ok := num(m, "total")
		if !ok || total <= 0 {
			continue
		}
		remaining, _ := num(m, "remaining")
		if remaining < 0 {
			remaining = 0
		}
		sectionUnit := text(m, "unit")
		if sectionUnit == "" {
			sectionUnit = unit
		}
		packs = append(packs, model.UsagePack{
			Name:      names[section],
			Total:     total,
			Remaining: remaining,
			Unit:      sectionUnit,
			URL:       text(m, "detailUrl"),
		})
	}
	return packs
}

func qoderHTTPError(err error) error {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w：请在 Qoder 中重新登录", ErrExpired)
		case http.StatusForbidden:
			return fmt.Errorf("%w：Qoder 拒绝了该请求", ErrBlocked)
		case http.StatusTooManyRequests:
			return fmt.Errorf("%w：请求过于频繁，请稍后再试", ErrBlocked)
		}
		return fmt.Errorf("Qoder 请求失败（HTTP %d）", httpErr.Status)
	}
	return err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
