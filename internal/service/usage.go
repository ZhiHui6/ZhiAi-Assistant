package service

import (
	"context"
	"time"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/usage"
)

type Usage struct {
	mgr      *usage.Manager
	onChange func(model.UsageState)
}

func NewUsage(mgr *usage.Manager, onChange func(model.UsageState)) *Usage {
	if onChange != nil {
		mgr.SetChangeHandler(onChange)
	}
	return &Usage{mgr: mgr, onChange: onChange}
}

func (s *Usage) emit(state model.UsageState) model.UsageState {
	if s.onChange != nil {
		s.onChange(state)
	}
	return state
}

func (s *Usage) List() model.UsageState {
	return s.mgr.List()
}

func (s *Usage) RefreshAll() model.UsageState {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	return s.emit(s.mgr.RefreshAll(ctx))
}

func (s *Usage) Refresh(id string) model.UsageState {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	state, _ := s.mgr.Refresh(ctx, id)
	return s.emit(state)
}

func (s *Usage) Checkin(id string) model.UsageState {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	state, _ := s.mgr.Checkin(ctx, id)
	return s.emit(state)
}

func (s *Usage) CheckinAll() model.UsageState {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	return s.emit(s.mgr.CheckinAll(ctx))
}

func (s *Usage) StartBrowserLogin(id string) (usage.BrowserLoginStart, error) {
	return s.mgr.StartBrowserLogin(id)
}

func (s *Usage) PollBrowserLogin(id, loginID string) (usage.BrowserLoginPoll, error) {
	return s.mgr.PollBrowserLogin(id, loginID)
}

func (s *Usage) CancelBrowserLogin(loginID string) {
	s.mgr.CancelBrowserLogin(loginID)
}

func (s *Usage) AccountFull(id string) (string, error) {
	return s.mgr.AccountFull(id)
}

func (s *Usage) OpenClient(id string) (string, error) {
	return s.mgr.OpenClient(id)
}

func (s *Usage) SignOut(id string) (model.UsageState, error) {
	state, err := s.mgr.SignOut(id)
	return s.emit(state), err
}

func (s *Usage) SetAutoCheckinAll(enabled bool) model.UsageState {
	return s.emit(s.mgr.SetAutoCheckinAll(enabled))
}

func (s *Usage) AutoCheckinAll() bool {
	return s.mgr.AutoCheckinAll()
}

func (s *Usage) StartWorkBuddyLogin() (usage.LoginStart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	return s.mgr.StartWorkBuddyLogin(ctx)
}

func (s *Usage) PollWorkBuddyLogin(loginID string) (usage.LoginPoll, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := s.mgr.PollWorkBuddyLogin(ctx, loginID)
	s.emit(s.mgr.List())
	return result, err
}

func (s *Usage) ReRead(id string) model.UsageState {
	state, _ := s.mgr.ReRead(id)
	return s.emit(state)
}

func (s *Usage) AckNotice() model.UsageState {
	return s.emit(s.mgr.AckNotice())
}

func (s *Usage) Tick(now time.Time) {
	s.mgr.Tick(now)
}
