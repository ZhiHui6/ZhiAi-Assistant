package usage

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
)

var platformOrder = []string{"qoder", "workbuddy", "trae"}

type Manager struct {
	doc       *store.Doc[model.UsageState]
	creds     *CredStore
	providers map[string]Provider
	order     []string
	env       Env
	doer      Doer
	now       func() time.Time

	mu         sync.Mutex
	busy       map[string]bool
	loginMu    sync.Mutex
	logins     map[string]*browserLoginSession
	lastTickAt time.Time
	onChange   func(model.UsageState)
}

func New() (*Manager, error) {
	return NewWith(SystemEnv(), defaultDoer(), time.Now)
}

func NewWith(env Env, doer Doer, now func() time.Time) (*Manager, error) {
	if doer == nil {
		doer = defaultDoer()
	}
	if now == nil {
		now = time.Now
	}
	doc, err := store.Open("usage.json", 1, model.UsageState{
		Platforms:   map[string]model.UsagePlatformState{},
		AutoCheckin: map[string]bool{},
	})
	if err != nil {
		return nil, err
	}
	m := &Manager{
		doc:       doc,
		creds:     OpenCredentials(filepath.Join(store.DataDir(), "data", "usage.credentials")),
		providers: map[string]Provider{},
		order:     append([]string(nil), platformOrder...),
		env:       env,
		doer:      doer,
		now:       now,
		busy:      map[string]bool{},
		logins:    map[string]*browserLoginSession{},
	}
	m.providers["qoder"] = newQoderProvider(doer)
	m.providers["workbuddy"] = newWorkBuddyProvider(doer)
	m.providers["trae"] = newTraeProvider(doer)
	m.ensureState()
	return m, nil
}

func (m *Manager) ensureState() {
	m.doc.Update(func(state *model.UsageState) {
		if state.Platforms == nil {
			state.Platforms = map[string]model.UsagePlatformState{}
		}
		if state.AutoCheckin == nil {
			state.AutoCheckin = map[string]bool{}
		}
		for id, st := range state.Platforms {
			if st.Source == "manual" || st.Source == "account" {
				st.Source = SourceBrowser
				state.Platforms[id] = st
			}
		}
		for _, id := range m.order {
			if _, ok := state.Platforms[id]; !ok {
				state.Platforms[id] = model.UsagePlatformState{Source: SourceNone, AuthState: AuthMissing}
			}
		}
	})
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) SetChangeHandler(fn func(model.UsageState)) {
	m.mu.Lock()
	m.onChange = fn
	m.mu.Unlock()
}

func (m *Manager) notify() {
	m.mu.Lock()
	fn := m.onChange
	m.mu.Unlock()
	if fn != nil {
		fn(m.List())
	}
}

func (m *Manager) List() model.UsageState {
	var out model.UsageState
	m.doc.Read(func(state *model.UsageState) {
		out.NoticeAcked = state.NoticeAcked
		out.Platforms = make(map[string]model.UsagePlatformState, len(state.Platforms))
		for id, value := range state.Platforms {
			out.Platforms[id] = value
		}
		out.AutoCheckin = make(map[string]bool, len(state.AutoCheckin))
		for id, value := range state.AutoCheckin {
			out.AutoCheckin[id] = value
		}
	})
	for _, id := range m.order {
		if _, ok := out.Platforms[id]; !ok {
			out.Platforms[id] = model.UsagePlatformState{Source: SourceNone, AuthState: AuthMissing}
		}
	}
	return out
}

func (m *Manager) valid(id string) bool {
	_, ok := m.providers[id]
	return ok
}

func (m *Manager) begin(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.busy[id] {
		return false
	}
	m.busy[id] = true
	return true
}

func (m *Manager) end(id string) {
	m.mu.Lock()
	delete(m.busy, id)
	m.mu.Unlock()
}

func (m *Manager) browserAuth(id string) (Auth, error) {
	cred, ok := m.creds.Get(id)
	if !ok || strings.TrimSpace(cred.Token) == "" {
		return Auth{}, ErrNoAuth
	}
	if !cred.ExpiresAt.IsZero() && m.now().After(cred.ExpiresAt) {
		return Auth{}, ErrExpired
	}
	auth := Auth{
		Token:        strings.TrimSpace(cred.Token),
		RefreshToken: strings.TrimSpace(cred.RefreshToken),
		ExpiresAt:    cred.ExpiresAt,
		UserID:       strings.TrimSpace(cred.UserID),
		Domain:       strings.TrimSpace(cred.Domain),
		DeviceID:     strings.TrimSpace(cred.DeviceID),
		Source:       SourceBrowser,
	}
	if id == "workbuddy" && auth.Domain == "" {
		auth.Domain = "workbuddy.cn"
	}
	return auth, nil
}

func (m *Manager) preferred(id string) string {
	var out string
	m.doc.Read(func(state *model.UsageState) {
		out = state.Platforms[id].Preferred
	})
	return out
}

func (m *Manager) resolve(id string) (Auth, string, error) {
	provider := m.providers[id]
	if provider == nil {
		return Auth{}, SourceNone, errors.New("未知平台")
	}
	localAuth, localErr := provider.ReadLocal(m.env)
	if localErr == nil && localAuth.Source == "" {
		localAuth.Source = SourceLocal
	}
	browserAuth, browserErr := m.browserAuth(id)

	if m.preferred(id) == SourceBrowser {
		if browserErr == nil {
			return browserAuth, SourceBrowser, nil
		}
		if localErr == nil {
			return localAuth, localAuth.Source, nil
		}
	} else {
		if localErr == nil {
			return localAuth, localAuth.Source, nil
		}
		if browserErr == nil {
			return browserAuth, SourceBrowser, nil
		}
	}

	if errors.Is(browserErr, ErrExpired) && (localErr == nil || errors.Is(localErr, ErrNoAuth)) {
		return Auth{}, SourceBrowser, ErrExpired
	}
	if localErr != nil {
		return Auth{}, SourceNone, localErr
	}
	return Auth{}, SourceNone, ErrNoAuth
}

func (m *Manager) update(id string, fn func(*model.UsagePlatformState)) {
	m.doc.Update(func(state *model.UsageState) {
		value := state.Platforms[id]
		fn(&value)
		state.Platforms[id] = value
	})
}

func (m *Manager) loggedIn(id string) bool {
	var ok bool
	m.doc.Read(func(state *model.UsageState) {
		ok = state.Platforms[id].AuthState == AuthOK
	})
	return ok
}

func (m *Manager) Refresh(ctx context.Context, id string) (model.UsageState, error) {
	if !m.valid(id) {
		return m.List(), errors.New("未知平台")
	}
	if !m.loggedIn(id) {
		return m.List(), ErrNotLoggedIn
	}
	if !m.begin(id) {
		m.update(id, func(st *model.UsagePlatformState) {
			st.LastError = "正在刷新"
		})
		return m.List(), errors.New("正在刷新")
	}
	defer m.end(id)
	err := m.refresh(ctx, id)
	return m.List(), err
}

func (m *Manager) RefreshAll(ctx context.Context) model.UsageState {
	for _, id := range m.order {
		if !m.loggedIn(id) {
			continue
		}
		if !m.begin(id) {
			continue
		}
		_ = m.refresh(ctx, id)
		m.end(id)
	}
	return m.List()
}

func (m *Manager) signedOut(id string) bool {
	var out bool
	m.doc.Read(func(state *model.UsageState) {
		out = state.Platforms[id].SignedOut
	})
	return out
}

func (m *Manager) refresh(ctx context.Context, id string) error {
	if m.signedOut(id) {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = SourceNone
			st.AuthState = AuthMissing
			st.LastError = ""
		})
		return nil
	}
	provider := m.providers[id]
	auth, source, err := m.resolve(id)
	now := m.now()
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = source
			st.AuthState = authStateFor(err)
			st.LastFetchAt = now
			if errors.Is(err, ErrNoAuth) {
				st.LastError = ""
			} else {
				st.LastError = err.Error()
			}
		})
		return err
	}
	snapshot, err := provider.Fetch(ctx, auth)
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = source
			if isAuthError(err) {
				st.AuthState = authStateFor(err)
			} else if st.AuthState == "" || st.AuthState == AuthMissing {
				st.AuthState = AuthOK
			}
			st.LastError = err.Error()
			st.LastFetchAt = now
		})
		return err
	}
	snapshot.Account = maskAccount(snapshot.Account)
	if snapshot.Unit == "" {
		snapshot.Unit = "credits"
	}
	snapshot.UpdatedAt = now
	if snapshot.AccountRaw != "" {
		cred, found := m.creds.Get(id)
		if !found {
			cred = model.UsageCredential{}
		}
		cred.Account = snapshot.AccountRaw
		_ = m.creds.Set(id, cred)
	}
	m.update(id, func(st *model.UsagePlatformState) {
		st.Snapshot = snapshot
		st.Source = source
		st.AuthState = AuthOK
		st.LastError = ""
		st.LastFetchAt = now
	})
	return nil
}

func isAuthError(err error) bool {
	return errors.Is(err, ErrNoAuth) ||
		errors.Is(err, ErrExpired) ||
		errors.Is(err, ErrEncrypted) ||
		errors.Is(err, ErrBlocked) ||
		errors.Is(err, ErrUnsupported)
}

func (m *Manager) Checkin(ctx context.Context, id string) (model.UsageState, error) {
	if !m.valid(id) {
		return m.List(), errors.New("未知平台")
	}
	if m.signedOut(id) {
		return m.List(), ErrSignedOut
	}
	if !m.begin(id) {
		m.update(id, func(st *model.UsagePlatformState) {
			st.LastError = "正在处理中"
		})
		return m.List(), errors.New("正在处理中")
	}
	defer m.end(id)
	provider := m.providers[id]
	auth, source, err := m.resolve(id)
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = source
			st.AuthState = authStateFor(err)
			st.LastError = err.Error()
		})
		return m.List(), err
	}
	result, err := provider.Claim(ctx, auth)
	now := m.now()
	today := now.Format("2006-01-02")
	if err != nil {
		if errors.Is(err, ErrAlreadyClaim) {
			m.update(id, func(st *model.UsagePlatformState) {
				st.Snapshot.CheckedIn = true
				st.Snapshot.CheckinAvailable = false
				st.LastCheckinDate = today
				st.LastError = ""
				st.Attempts = 0
				st.RetryAt = time.Time{}
			})
			return m.List(), nil
		}
		m.markFailure(id, now, err)
		return m.List(), err
	}
	m.update(id, func(st *model.UsagePlatformState) {
		st.Snapshot.CheckedIn = true
		st.Snapshot.CheckinAvailable = false
		if result.Amount > 0 {
			st.Snapshot.DailyCredit = result.Amount
		}
		if result.Message != "" {
			st.Snapshot.Message = result.Message
		}
		st.LastCheckinDate = today
		st.LastError = ""
		st.Attempts = 0
		st.AttemptDate = ""
		st.RetryAt = time.Time{}
	})
	_ = m.refresh(ctx, id)
	return m.List(), nil
}

func (m *Manager) CheckinAll(ctx context.Context) model.UsageState {
	today := m.now().Format("2006-01-02")
	state := m.List()
	for _, id := range m.order {
		st := state.Platforms[id]
		if st.AuthState != AuthOK || st.Snapshot.CheckedIn || st.LastCheckinDate == today {
			continue
		}
		one, cancel := context.WithTimeout(ctx, 25*time.Second)
		_, _ = m.Checkin(one, id)
		cancel()
	}
	return m.List()
}

func (m *Manager) markFailure(id string, now time.Time, err error) {
	today := now.Format("2006-01-02")
	m.update(id, func(st *model.UsagePlatformState) {
		if st.AttemptDate != today {
			st.AttemptDate = today
			st.Attempts = 0
		}
		st.Attempts++
		delay := 15 * time.Minute
		if st.Attempts >= 2 {
			delay = 60 * time.Minute
		}
		st.RetryAt = now.Add(delay)
		if errors.Is(err, ErrNoAuth) {
			st.LastError = ""
		} else {
			st.LastError = err.Error()
		}
	})
}

func (m *Manager) SignOut(id string) (model.UsageState, error) {
	if !m.valid(id) {
		return m.List(), errors.New("未知平台")
	}
	if err := m.creds.Delete(id); err != nil {
		return m.List(), err
	}
	m.update(id, func(st *model.UsagePlatformState) {
		*st = model.UsagePlatformState{Source: SourceNone, AuthState: AuthMissing, SignedOut: true}
	})
	return m.List(), nil
}

func (m *Manager) ReRead(id string) (model.UsageState, error) {
	if !m.valid(id) {
		return m.List(), errors.New("未知平台")
	}
	m.update(id, func(st *model.UsagePlatformState) { st.SignedOut = false })
	if !m.begin(id) {
		m.update(id, func(st *model.UsagePlatformState) {
			st.LastError = "正在处理中"
		})
		return m.List(), errors.New("正在处理中")
	}
	defer m.end(id)
	provider := m.providers[id]
	auth, err := provider.ReadLocal(m.env)
	now := m.now()
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = SourceNone
			st.AuthState = authStateFor(err)
			st.LastError = err.Error()
		})
		return m.List(), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	snapshot, err := provider.Fetch(ctx, auth)
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = SourceLocal
			st.AuthState = authStateFor(err)
			st.LastError = err.Error()
		})
		return m.List(), err
	}
	snapshot.Account = maskAccount(snapshot.Account)
	snapshot.UpdatedAt = now
	m.update(id, func(st *model.UsagePlatformState) {
		st.Snapshot = snapshot
		st.Source = SourceLocal
		st.AuthState = AuthOK
		st.LastError = ""
		st.LastFetchAt = now
		st.Attempts = 0
		st.AttemptDate = ""
		st.RetryAt = time.Time{}
		st.Preferred = SourceLocal
	})
	return m.List(), nil
}

func (m *Manager) AccountFull(id string) (string, error) {
	if !m.valid(id) {
		return "", errors.New("未知平台")
	}
	cred, ok := m.creds.Get(id)
	if !ok {
		return "", nil
	}
	return strings.TrimSpace(cred.Account), nil
}

func (m *Manager) SetAutoCheckinAll(enabled bool) model.UsageState {
	m.doc.Update(func(state *model.UsageState) {
		if state.AutoCheckin == nil {
			state.AutoCheckin = map[string]bool{}
		}
		for _, id := range platformOrder {
			state.AutoCheckin[id] = enabled
		}
	})
	return m.List()
}

func (m *Manager) AutoCheckinAll() bool {
	return autoCheckinAllEnabled(m.List().AutoCheckin)
}

func AutoCheckinAllFromDisk() bool {
	state, ok := store.Peek[model.UsageState]("usage.json")
	if !ok {
		return false
	}
	return autoCheckinAllEnabled(state.AutoCheckin)
}

func autoCheckinAllEnabled(auto map[string]bool) bool {
	for _, id := range platformOrder {
		if !auto[id] {
			return false
		}
	}
	return true
}

func (m *Manager) StartWorkBuddyLogin(ctx context.Context) (LoginStart, error) {
	provider, ok := m.providers["workbuddy"].(*workBuddyProvider)
	if !ok {
		return LoginStart{}, errors.New("WorkBuddy 不可用")
	}
	return provider.oauth.start(ctx, m.doer)
}

func (m *Manager) PollWorkBuddyLogin(ctx context.Context, loginID string) (LoginPoll, error) {
	provider, ok := m.providers["workbuddy"].(*workBuddyProvider)
	if !ok {
		return LoginPoll{Done: true, Error: "WorkBuddy 不可用"}, errors.New("WorkBuddy 不可用")
	}
	auth, done, message := provider.oauth.poll(ctx, m.doer, loginID)
	if !done {
		return LoginPoll{Error: message}, nil
	}
	if message != "" {
		return LoginPoll{Done: true, Error: message}, nil
	}
	cred := model.UsageCredential{
		Token:        auth.Token,
		Account:      firstNonEmpty(auth.Name, auth.UserID),
		RefreshToken: auth.RefreshToken,
		UserID:       auth.UserID,
		Domain:       auth.Domain,
		ExpiresAt:    auth.ExpiresAt,
	}
	if err := m.creds.Set("workbuddy", cred); err != nil {
		return LoginPoll{Done: true, Error: err.Error()}, err
	}
	m.update("workbuddy", func(st *model.UsagePlatformState) {
		st.Source = SourceBrowser
		st.AuthState = AuthOK
		st.LastError = ""
		st.Attempts = 0
		st.AttemptDate = ""
		st.RetryAt = time.Time{}
		st.SignedOut = false
	})
	if m.begin("workbuddy") {
		_ = m.refresh(ctx, "workbuddy")
		m.end("workbuddy")
	}
	return LoginPoll{Done: true}, nil
}

func (m *Manager) AckNotice() model.UsageState {
	m.doc.Update(func(state *model.UsageState) {
		state.NoticeAcked = true
	})
	return m.List()
}

func (m *Manager) Tick(now time.Time) {
	m.cleanupBrowserLogins()
	m.mu.Lock()
	if !m.lastTickAt.IsZero() && now.Sub(m.lastTickAt) < time.Minute {
		m.mu.Unlock()
		return
	}
	m.lastTickAt = now
	m.mu.Unlock()

	state := m.List()
	if !state.NoticeAcked {
		return
	}
	if now.Hour() == 0 && now.Minute() < 5 {
		return
	}
	today := now.Format("2006-01-02")
	for _, id := range m.order {
		if !state.AutoCheckin[id] {
			continue
		}
		st := state.Platforms[id]
		if st.SignedOut || st.AuthState != AuthOK {
			continue
		}
		if st.LastCheckinDate == today {
			continue
		}
		if st.AttemptDate == today && st.Attempts >= 3 {
			continue
		}
		if !st.RetryAt.IsZero() && now.Before(st.RetryAt) {
			continue
		}
		needClaim := st.Snapshot.CheckinAvailable && !st.Snapshot.CheckedIn
		stale := st.LastFetchAt.IsZero() || now.Sub(st.LastFetchAt) >= 30*time.Minute
		if !needClaim && !stale {
			continue
		}
		go m.autoOne(id, now)
	}
}

func (m *Manager) autoOne(id string, now time.Time) {
	if m.signedOut(id) {
		return
	}
	if !m.begin(id) {
		return
	}
	defer m.end(id)
	defer m.notify()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	provider := m.providers[id]
	auth, source, err := m.resolve(id)
	if err != nil {
		m.update(id, func(st *model.UsagePlatformState) {
			st.Source = source
			st.AuthState = authStateFor(err)
			if !errors.Is(err, ErrNoAuth) {
				st.LastError = err.Error()
			}
		})
		m.markFailure(id, now, err)
		return
	}
	st := m.List().Platforms[id]
	if !st.Snapshot.CheckinAvailable || st.Snapshot.CheckedIn {
		if err := m.refresh(ctx, id); err != nil {
			m.markFailure(id, now, err)
			return
		}
		st = m.List().Platforms[id]
	}
	if !st.Snapshot.CheckinAvailable || st.Snapshot.CheckedIn {
		return
	}
	result, err := provider.Claim(ctx, auth)
	if err != nil {
		if errors.Is(err, ErrAlreadyClaim) {
			m.update(id, func(ps *model.UsagePlatformState) {
				ps.Snapshot.CheckedIn = true
				ps.Snapshot.CheckinAvailable = false
				ps.LastCheckinDate = now.Format("2006-01-02")
				ps.LastError = ""
			})
			return
		}
		m.markFailure(id, now, err)
		return
	}
	today := now.Format("2006-01-02")
	m.update(id, func(ps *model.UsagePlatformState) {
		ps.Snapshot.CheckedIn = true
		ps.Snapshot.CheckinAvailable = false
		if result.Amount > 0 {
			ps.Snapshot.DailyCredit = result.Amount
		}
		ps.LastCheckinDate = today
		ps.LastError = ""
		ps.Attempts = 0
		ps.AttemptDate = ""
		ps.RetryAt = time.Time{}
	})
	refreshCtx, cancelRefresh := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRefresh()
	_ = m.refresh(refreshCtx, id)
}
