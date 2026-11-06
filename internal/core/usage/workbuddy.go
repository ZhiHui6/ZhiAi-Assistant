package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/model"
)

const (
	wbResourcePath = "/v2/billing/meter/get-user-resource"
	wbStatusPath   = "/v2/billing/meter/checkin-activity-status"
	wbClaimPath    = "/v2/billing/meter/daily-checkin"
	wbUsagePath    = "/billing/meter/get-enterprise-user-usage"
)

type workBuddyProvider struct {
	doer  Doer
	oauth *workBuddyOAuth
}

func newWorkBuddyProvider(doer Doer) *workBuddyProvider {
	return &workBuddyProvider{doer: doer, oauth: newWorkBuddyOAuth()}
}

func (p *workBuddyProvider) ID() string { return "workbuddy" }

func (p *workBuddyProvider) ReadLocal(env Env) (Auth, error) {
	dirs := []string{
		filepath.Join(env.LocalAppData, "CodeBuddyExtension", "Data", "Public", "auth"),
		filepath.Join(env.AppData, "CodeBuddyExtension", "Data", "Public", "auth"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		type candidate struct {
			path string
			mod  int64
		}
		var files []candidate
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasPrefix(name, "workbuddy-desktop") || !strings.HasSuffix(name, ".info") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, candidate{path: filepath.Join(dir, name), mod: info.ModTime().UnixNano()})
		}
		sort.Slice(files, func(i, j int) bool { return files[i].mod > files[j].mod })
		for _, file := range files {
			raw, err := os.ReadFile(file.path)
			if err != nil {
				continue
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				continue
			}
			if wbDocHasEncryptedFields(doc) {
				ctx, cancel := context.WithTimeout(context.Background(), wbKeyFetchTimeout+2*time.Second)
				key, keyErr := wbAtRestKey(ctx, env)
				cancel()
				if keyErr == nil {
					wbDecryptDocFields(doc, key)
				}
			}
			auth, err := parseWorkBuddyDoc(doc)
			if err == nil {
				auth.Source = SourceLocal
				return auth, nil
			}
			if errors.Is(err, ErrEncrypted) {
				if wbDocHasEncryptedFields(doc) {
					wbInvalidateKey(env)
				}
				return Auth{}, err
			}
		}
	}
	return Auth{}, ErrNoAuth
}

func parseWorkBuddyAuth(raw []byte) (Auth, error) {
	var root map[string]any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return Auth{}, fmt.Errorf("%w：登录文件无法解析", ErrNoAuth)
	}
	return parseWorkBuddyDoc(root)
}

func parseWorkBuddyDoc(root map[string]any) (Auth, error) {
	authMap := unwrapMap(root, "auth")
	if authMap == nil {
		authMap = root
	}
	tokenValue, ok := lookup(authMap, "accessToken", "access_token", "token")
	if !ok {
		return Auth{}, ErrNoAuth
	}
	token, ok := tokenValue.(string)
	if !ok {
		return Auth{}, fmt.Errorf("%w：请使用浏览器登录", ErrEncrypted)
	}
	if strings.TrimSpace(token) == "" {
		return Auth{}, ErrNoAuth
	}
	identity := unwrapMap(root, "identity", "user", "account")
	auth := Auth{
		Token:        token,
		RefreshToken: text(authMap, "refreshToken", "refresh_token"),
		UserID:       firstText(identity, root, "uid", "userId", "user_id", "id"),
		Name:         firstText(identity, root, "nickname", "displayName", "name"),
		Domain:       text(authMap, "domain"),
	}
	if strings.TrimSpace(auth.Domain) == "" {
		auth.Domain = "workbuddy.cn"
	}
	lowerDomain := strings.ToLower(auth.Domain)
	if strings.Contains(lowerDomain, "workbuddy.ai") || strings.Contains(lowerDomain, "codebuddy.ai") {
		return Auth{}, fmt.Errorf("%w：暂不支持 WorkBuddy 国际区", ErrUnsupported)
	}
	if expires, ok := lookup(authMap, "expiresAt", "expires_at", "expireAt", "expire_at"); ok {
		auth.ExpiresAt = timeValue(expires)
	}
	if !auth.ExpiresAt.IsZero() && auth.ExpiresAt.Before(pNow()) {
		return Auth{}, fmt.Errorf("%w：请在 WorkBuddy 中重新登录", ErrExpired)
	}
	return auth, nil
}

func wbBaseFor(domain string) string {
	lower := strings.ToLower(domain)
	if strings.Contains(lower, "copilot.tencent.com") {
		return "https://copilot.tencent.com"
	}
	return "https://www.workbuddy.cn"
}

func (p *workBuddyProvider) fetchNickname(ctx context.Context, auth Auth) string {
	headers := workBuddyHeaders(auth)
	headers["Accept"] = "application/json"
	raw, err := requestJSON(ctx, p.doer, http.MethodGet, wbPluginBase+"/login/account", headers, nil)
	if err != nil {
		return ""
	}
	root, derr := decodeMap(raw)
	if derr != nil {
		return ""
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	return text(data, "nickname", "NickName", "displayName", "name")
}

func workBuddyHeaders(auth Auth) map[string]string {
	base := wbBaseFor(auth.Domain)
	return map[string]string{
		"Authorization":       "Bearer " + auth.Token,
		"X-User-Id":           auth.UserID,
		"X-Domain":            auth.Domain,
		"User-Agent":          "WorkBuddy/1.0",
		"X-Requested-With":    "XMLHttpRequest",
		"X-CodeBuddy-Request": "1",
		"Accept-Language":     "zh-CN",
		"Origin":              base,
		"Referer":             base + "/",
	}
}

func (p *workBuddyProvider) Fetch(ctx context.Context, auth Auth) (model.UsageSnapshot, error) {
	if strings.TrimSpace(auth.Name) == "" {
		if nick := p.fetchNickname(ctx, auth); nick != "" {
			auth.Name = nick
		}
	}
	snapshot := model.UsageSnapshot{
		Account:    maskAccount(firstNonEmpty(auth.Name, auth.UserID)),
		AccountRaw: firstNonEmpty(auth.Name, auth.UserID),
		Unit:       "credits",
		UpdatedAt:  pNow(),
	}
	resourceBody := map[string]any{
		"PageNumber":               1,
		"PageSize":                 100,
		"ProductCode":              "p_tcaca",
		"Status":                   []int{0, 3},
		"PackageEndTimeRangeBegin": time.Now().Format("2006-01-02 15:04:05"),
		"PackageEndTimeRangeEnd":   time.Now().AddDate(101, 0, 0).Format("2006-01-02 15:04:05"),
	}
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, wbBaseFor(auth.Domain)+wbResourcePath, workBuddyHeaders(auth), resourceBody)
	debugLog("workbuddy", wbBaseFor(auth.Domain)+wbResourcePath, raw, err)
	var lastErr error
	if err == nil {
		total, used, remaining, plan, reset, packs, parseErr := workBuddyQuota(raw)
		if parseErr == nil {
			snapshot.Total = total
			snapshot.Used = used
			snapshot.Remaining = remaining
			snapshot.Percent = normalizedPercent(used, total, 0)
			snapshot.Plan = plan
			snapshot.Packs = packs
			snapshot.ResetAt = reset
		} else {
			lastErr = parseErr
			snapshot.Message = parseErr.Error()
		}
	} else {
		lastErr = workBuddyError(err)
		snapshot.Message = lastErr.Error()
	}
	if snapshot.Total <= 0 {
		raw, fallbackErr := requestJSON(ctx, p.doer, http.MethodPost, wbBaseFor(auth.Domain)+wbUsagePath, workBuddyHeaders(auth), map[string]any{})
		debugLog("workbuddy", wbBaseFor(auth.Domain)+wbUsagePath, raw, fallbackErr)
		if fallbackErr == nil {
			if total, used, remaining, plan, reset, packs, parseErr := workBuddyEnterpriseQuota(raw); parseErr == nil {
				snapshot.Total = total
				snapshot.Used = used
				snapshot.Remaining = remaining
				snapshot.Percent = normalizedPercent(used, total, 0)
				snapshot.Plan = plan
				snapshot.Packs = packs
				snapshot.ResetAt = reset
				snapshot.Message = ""
			} else {
				lastErr = parseErr
			}
		} else {
			lastErr = workBuddyError(fallbackErr)
		}
	}
	if snapshot.Total <= 0 {
		if lastErr != nil {
			return model.UsageSnapshot{}, lastErr
		}
		return model.UsageSnapshot{}, errors.New("WorkBuddy 额度数据为空")
	}
	checked, available, daily, message := p.workBuddyCheckin(ctx, auth)
	snapshot.CheckedIn = checked
	snapshot.CheckinAvailable = available
	snapshot.DailyCredit = daily
	if message != "" {
		snapshot.Message = message
	}
	return snapshot, nil
}

func (p *workBuddyProvider) Claim(ctx context.Context, auth Auth) (ClaimResult, error) {
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, wbBaseFor(auth.Domain)+wbClaimPath, workBuddyHeaders(auth), map[string]any{})
	debugLog("workbuddy", wbBaseFor(auth.Domain)+wbClaimPath, raw, err)
	if err != nil {
		return ClaimResult{}, workBuddyError(err)
	}
	root, _ := decodeMap(raw)
	if code := workBuddyCode(root); code != 0 && code != 14018 {
		return ClaimResult{}, fmt.Errorf("WorkBuddy 返回错误码 %d", code)
	}
	amount, _ := num(root, "credit", "credits", "amount", "dailyCredit")
	if data := unwrapMap(root, "data"); data != nil {
		if v, ok := num(data, "credit", "credits", "amount", "dailyCredit"); ok {
			amount = v
		}
	}
	return ClaimResult{Claimed: true, Amount: amount, Message: "签到成功"}, nil
}

func (p *workBuddyProvider) workBuddyCheckin(ctx context.Context, auth Auth) (bool, bool, float64, string) {
	raw, err := requestJSON(ctx, p.doer, http.MethodPost, wbBaseFor(auth.Domain)+wbStatusPath, workBuddyHeaders(auth), map[string]any{})
	debugLog("workbuddy", wbBaseFor(auth.Domain)+wbStatusPath, raw, err)
	if err != nil {
		return false, false, 0, "签到状态读取失败：" + workBuddyError(err).Error()
	}
	root, err := decodeMap(raw)
	if err != nil {
		return false, false, 0, "签到状态读取失败"
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	checked, _ := boolean(data, "today_checked_in", "todayCheckedIn", "checked_in", "checkedIn")
	active, activeOK := boolean(data, "active", "enabled")
	if !activeOK {
		active = true
	}
	daily, _ := num(data, "daily_credit", "dailyCredit", "credit")
	return checked, active && !checked, daily, ""
}

func workBuddyCode(root map[string]any) int {
	if root == nil {
		return -1
	}
	if value, ok := num(root, "code", "errorCode", "error_code"); ok {
		return int(value)
	}
	return 0
}

func workBuddyPrecise(m map[string]any, names ...string) (float64, bool) {
	for _, name := range names {
		if value, ok := lookup(m, name+"Precise"); ok {
			if n, ok := numberValue(value); ok {
				return n, true
			}
		}
		if n, ok := num(m, name); ok {
			return n, true
		}
	}
	return 0, false
}

func hasField(m map[string]any, names ...string) bool {
	_, ok := lookup(m, names...)
	return ok
}

func workBuddyQuota(raw []byte) (float64, float64, float64, string, string, []model.UsagePack, error) {
	root, err := decodeMap(raw)
	if err != nil {
		return 0, 0, 0, "", "", nil, err
	}
	if code := workBuddyCode(root); code != 0 {
		if code == 14018 {
			return 0, 0, 0, "", "", nil, errors.New("额度已用尽")
		}
		return 0, 0, 0, "", "", nil, fmt.Errorf("WorkBuddy 返回错误码 %d", code)
	}
	accounts := findArray(root, "Accounts", "accounts")
	var total, used, remaining float64
	plan := ""
	reset := ""
	var packs []model.UsagePack
	for _, item := range accounts {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		size, sizeOK := workBuddyPrecise(m, "CycleCapacitySize", "CapacitySize")
		if !sizeOK {
			size, sizeOK = workBuddyPrecise(m, "TotalCredit", "Total", "LimitNum", "Limit", "Amount")
		}
		remain, remainOK := workBuddyPrecise(m, "CycleCapacityRemain", "CapacityRemain", "RemainCredit", "Remaining", "Remain", "Balance", "Available", "Credit")
		if !remainOK {
			continue
		}
		capacityType, _ := num(m, "CapacityType")
		monthly := int(capacityType) == 4
		if !monthly && remain <= 0 {
			continue
		}
		if expiredRaw, ok := lookup(m, "ExpiredTime"); ok && !monthly {
			if expired := timeValue(expiredRaw); !expired.IsZero() && expired.Before(time.Now()) {
				continue
			}
		}
		if remain < 0 {
			remain = 0
		}
		if !sizeOK || size <= 0 {
			size = remain
			if !hasField(m, "CycleCapacitySize", "CapacitySize", "CycleCapacityRemain", "CapacityRemain") {
				continue
			}
		}
		if size < remain {
			remain = size
		}
		total += size
		remaining += remain
		used += size - remain
		if plan == "" {
			plan = text(m, "PackageName", "ProductName", "Name", "Title")
		}
		if reset == "" {
			reset = text(m, "CycleEndTime", "ExpiredTime")
		}
		packs = append(packs, model.UsagePack{
			Name:      text(m, "PackageName", "ProductName", "Name", "Title"),
			Total:     size,
			Remaining: remain,
			Unit:      "credits",
			ExpiresAt: text(m, "CycleEndTime", "ExpiredTime"),
		})
	}
	if total > 0 {
		return total, used, remaining, plan, reset, packs, nil
	}
	return workBuddyEnterpriseQuota(raw)
}

func workBuddyEnterpriseQuota(raw []byte) (float64, float64, float64, string, string, []model.UsagePack, error) {
	root, err := decodeMap(raw)
	if err != nil {
		return 0, 0, 0, "", "", nil, err
	}
	data := unwrapMap(root, "data")
	if data == nil {
		data = root
	}
	total, ok := num(data, "limitNum", "limit", "total")
	if !ok || total <= 0 {
		return 0, 0, 0, "", "", nil, errors.New("WorkBuddy 额度数据为空")
	}
	used, _ := num(data, "credit", "used", "consumed")
	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}
	plan := text(data, "productName", "planName", "plan", "name")
	reset := text(data, "cycleResetTime", "resetTime", "resetAt")
	return total, used, remaining, plan, reset, nil, nil
}

func workBuddyError(err error) error {
	if err == nil {
		return errors.New("WorkBuddy 请求失败")
	}
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Status {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w：请在 WorkBuddy 中重新登录或填写 Token", ErrExpired)
		case http.StatusForbidden:
			return fmt.Errorf("%w：WorkBuddy 拒绝了非官方客户端请求", ErrBlocked)
		}
		return fmt.Errorf("WorkBuddy 请求失败（HTTP %d）", httpErr.Status)
	}
	return err
}
