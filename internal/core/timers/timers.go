package timers

import (
	"errors"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/id"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
)

const maxCountdown = 24 * 60 * 60

type Manager struct {
	doc *store.Doc[model.TimerData]
}

func New() (*Manager, error) {
	doc, err := store.Open("timers.json", 1, model.TimerData{Alarms: []model.Alarm{}})
	if err != nil {
		return nil, err
	}
	m := &Manager{doc: doc}
	m.sanitize()
	return m, nil
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func (m *Manager) sanitize() {
	now := time.Now()
	m.doc.Update(func(d *model.TimerData) {
		if d.Alarms == nil {
			d.Alarms = []model.Alarm{}
		}
		c := &d.Countdown
		if c.Remaining < 0 {
			c.Remaining = 0
		}
		if c.Remaining > maxCountdown {
			c.Remaining = maxCountdown
		}
		if !c.Running {
			c.EndAt = nil
		} else if c.EndAt == nil || !now.Before(*c.EndAt) {
			c.Running = false
			c.EndAt = nil
			c.Remaining = 0
		}
		if d.Shutdown.Active && (d.Shutdown.At == nil || !now.Before(*d.Shutdown.At)) {
			d.Shutdown.Active = false
			d.Shutdown.At = nil
		}
	})
}

func (m *Manager) Snapshot() model.TimerData {
	var out model.TimerData
	m.doc.Read(func(d *model.TimerData) {
		out.Alarms = append([]model.Alarm{}, d.Alarms...)
		out.Countdown = d.Countdown
		out.Shutdown = d.Shutdown
	})
	if out.Alarms == nil {
		out.Alarms = []model.Alarm{}
	}
	return out
}

func (m *Manager) AddAlarm(at, label string, repeat bool) (model.Alarm, error) {
	at = strings.TrimSpace(at)
	if _, err := time.Parse("15:04", at); err != nil {
		return model.Alarm{}, errors.New("时间格式应为 HH:MM")
	}
	alarm := model.Alarm{
		ID:      id.New(),
		Time:    at,
		Label:   strings.TrimSpace(label),
		Repeat:  repeat,
		Enabled: true,
	}
	m.doc.Update(func(d *model.TimerData) {
		d.Alarms = append(d.Alarms, alarm)
	})
	return alarm, nil
}

func (m *Manager) SetAlarmEnabled(alarmID string, enabled bool) error {
	found := false
	m.doc.Update(func(d *model.TimerData) {
		for i := range d.Alarms {
			if d.Alarms[i].ID == alarmID {
				d.Alarms[i].Enabled = enabled
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("闹钟不存在")
	}
	return nil
}

func (m *Manager) RemoveAlarm(alarmID string) error {
	found := false
	m.doc.Update(func(d *model.TimerData) {
		for i := range d.Alarms {
			if d.Alarms[i].ID == alarmID {
				d.Alarms = append(d.Alarms[:i], d.Alarms[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("闹钟不存在")
	}
	return nil
}

func (m *Manager) StartCountdown(seconds int, label string) (model.Countdown, error) {
	if seconds < 1 {
		return model.Countdown{}, errors.New("倒计时至少 1 秒")
	}
	if seconds > maxCountdown {
		return model.Countdown{}, errors.New("倒计时最长 24 小时")
	}
	end := time.Now().Add(time.Duration(seconds) * time.Second)
	next := model.Countdown{
		Running:   true,
		EndAt:     &end,
		Remaining: seconds,
		Label:     strings.TrimSpace(label),
	}
	m.doc.Update(func(d *model.TimerData) {
		d.Countdown = next
	})
	return next, nil
}

func (m *Manager) PauseCountdown() (model.Countdown, error) {
	var out model.Countdown
	m.doc.Update(func(d *model.TimerData) {
		c := &d.Countdown
		if c.Running && c.EndAt != nil {
			remain := int(time.Until(*c.EndAt).Seconds())
			if remain < 0 {
				remain = 0
			}
			c.Running = false
			c.EndAt = nil
			c.Remaining = remain
			if c.Remaining == 0 {
				c.Label = ""
			}
		}
		out = *c
	})
	return out, nil
}

func (m *Manager) ResumeCountdown() (model.Countdown, error) {
	var out model.Countdown
	var err error
	m.doc.Update(func(d *model.TimerData) {
		c := &d.Countdown
		if c.Running {
			out = *c
			return
		}
		if c.Remaining <= 0 {
			err = errors.New("没有可继续的倒计时")
			return
		}
		end := time.Now().Add(time.Duration(c.Remaining) * time.Second)
		c.Running = true
		c.EndAt = &end
		out = *c
	})
	return out, err
}

func (m *Manager) ResetCountdown() {
	m.doc.Update(func(d *model.TimerData) {
		d.Countdown = model.Countdown{}
	})
}

func (m *Manager) SetShutdown(at time.Time) model.ShutdownPlan {
	plan := model.ShutdownPlan{Active: true, At: &at}
	m.doc.Update(func(d *model.TimerData) {
		d.Shutdown = plan
	})
	return plan
}

func (m *Manager) ClearShutdown() {
	m.doc.Update(func(d *model.TimerData) {
		d.Shutdown = model.ShutdownPlan{}
	})
}
