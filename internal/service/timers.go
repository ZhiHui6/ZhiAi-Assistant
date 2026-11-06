package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/timers"
	"zhi-ai-assistant/internal/i18n"
	"zhi-ai-assistant/internal/platform"
)

type Timers struct {
	mgr *timers.Manager

	mu        sync.Mutex
	lastCheck time.Time
	fired     map[string]string
	onFire    func(kind, label string)
}

func NewTimers(mgr *timers.Manager) *Timers {
	return &Timers{mgr: mgr, fired: map[string]string{}}
}

func (s *Timers) SetFireHandler(fn func(kind, label string)) {
	s.mu.Lock()
	s.onFire = fn
	s.mu.Unlock()
}

func (s *Timers) State() model.TimerData { return s.mgr.Snapshot() }

func (s *Timers) AddAlarm(at, label string, repeat bool) (model.Alarm, error) {
	return s.mgr.AddAlarm(at, label, repeat)
}

func (s *Timers) ToggleAlarm(alarmID string, enabled bool) error {
	return s.mgr.SetAlarmEnabled(alarmID, enabled)
}

func (s *Timers) RemoveAlarm(alarmID string) error {
	return s.mgr.RemoveAlarm(alarmID)
}

func (s *Timers) StartCountdown(seconds int, label string) (model.Countdown, error) {
	return s.mgr.StartCountdown(seconds, label)
}

func (s *Timers) PauseCountdown() (model.Countdown, error) {
	return s.mgr.PauseCountdown()
}

func (s *Timers) ResumeCountdown() (model.Countdown, error) {
	return s.mgr.ResumeCountdown()
}

func (s *Timers) ResetCountdown() {
	s.mgr.ResetCountdown()
}

func (s *Timers) ScheduleShutdown(atUnix int64) (model.ShutdownPlan, error) {
	now := time.Now()
	at := time.Unix(atUnix, 0)
	delay := int(at.Sub(now).Seconds())
	if delay < 60 {
		delay = 60
	}
	if delay > 7*24*3600 {
		return model.ShutdownPlan{}, errors.New("关机时间不能超过 7 天")
	}
	if s.mgr.Snapshot().Shutdown.Active {
		_ = platform.AbortShutdown()
	}
	if err := platform.ScheduleShutdown(delay); err != nil {
		return model.ShutdownPlan{}, errors.New("无法安排系统关机")
	}
	target := now.Add(time.Duration(delay) * time.Second)
	return s.mgr.SetShutdown(target), nil
}

func (s *Timers) CancelShutdown() error {
	if s.mgr.Snapshot().Shutdown.Active {
		_ = platform.AbortShutdown()
	}
	s.mgr.ClearShutdown()
	return nil
}

func (a *App) SetTimerSound(kind, sound string) (model.Settings, error) {
	current := a.GetSettings()
	sound = strings.TrimSpace(sound)
	if strings.HasPrefix(sound, "file:") {
		path := strings.TrimSpace(strings.TrimPrefix(sound, "file:"))
		if path == "" {
			return current, errors.New("铃音路径为空")
		}
		if !strings.EqualFold(filepath.Ext(path), ".wav") {
			return current, errors.New("铃音需为 .wav 文件")
		}
		if _, err := os.Stat(path); err != nil {
			return current, errors.New("铃音文件不存在")
		}
		sound = "file:" + path
	}
	switch kind {
	case "alarm":
		current.AlarmSound = sound
	case "countdown":
		current.CountdownSound = sound
	default:
		return current, errors.New("未知计时类型")
	}
	return a.UpdateSettings(current)
}

func (a *App) PreviewTimerSound(sound string) error {
	return platform.PlayTimerSound(sound)
}

func (a *App) PickSound() (string, error) {
	paths, err := openFileDialog(i18n.T("选择提示音（WAV）"), false)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("未选择文件")
	}
	path := paths[0]
	if !strings.EqualFold(filepath.Ext(path), ".wav") {
		return "", errors.New(i18n.T("请选择 .wav 文件"))
	}
	return path, nil
}

func (s *Timers) Tick(now time.Time) {
	s.mu.Lock()
	if now.Sub(s.lastCheck) < 900*time.Millisecond {
		s.mu.Unlock()
		return
	}
	s.lastCheck = now
	s.mu.Unlock()

	data := s.mgr.Snapshot()
	today := now.Format("2006-01-02")
	minute := now.Format("15:04")
	type firedEvent struct {
		kind  string
		label string
	}
	var events []firedEvent

	for _, alarm := range data.Alarms {
		if !alarm.Enabled || alarm.Time != minute {
			continue
		}
		s.mu.Lock()
		already := s.fired[alarm.ID] == today
		if !already {
			s.fired[alarm.ID] = today
		}
		s.mu.Unlock()
		if already {
			continue
		}
		repeat := alarm.Repeat
		if !repeat {
			_ = s.mgr.SetAlarmEnabled(alarm.ID, false)
		}
		label := alarm.Label
		if label == "" {
			label = "闹钟 " + alarm.Time
		}
		events = append(events, firedEvent{kind: "alarm", label: label})
	}

	if data.Countdown.Running && data.Countdown.EndAt != nil && !now.Before(*data.Countdown.EndAt) {
		s.mgr.ResetCountdown()
		label := data.Countdown.Label
		if label == "" {
			label = "倒计时结束"
		}
		events = append(events, firedEvent{kind: "countdown", label: label})
	}

	for _, event := range events {
		s.mu.Lock()
		handler := s.onFire
		s.mu.Unlock()
		if handler != nil {
			handler(event.kind, event.label)
		}
	}
}
