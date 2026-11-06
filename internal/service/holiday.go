package service

import (
	"log"
	"time"

	"zhi-ai-assistant/internal/core/holiday"
)

type holidayProvider struct{ store *holiday.Store }

func (p holidayProvider) Lookup(date string) (string, bool, bool) {
	day, ok := p.store.Lookup(date)
	return day.Name, day.Off, ok
}

func (a *App) startHolidaySync() {
	if a.holidayStore == nil {
		return
	}
	if a.Debug() {
		a.holidayStore.SetLogger(func(format string, args ...interface{}) {
			log.Printf("[holiday] "+format, args...)
		})
	}

	go a.syncHolidays()
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			a.mu.Lock()
			quitting := a.quitting
			a.mu.Unlock()
			if quitting {
				return
			}
			a.syncHolidays()
		}
	}()
}

func (a *App) syncHolidays() {
	if a.holidayStore == nil {
		return
	}
	now := time.Now()
	if a.holidayStore.Ensure(now.Year(), now.Year()+1) {
		a.emit("calendar:holidays", a.holidayStore.Coverage())
	}
}

func (a *App) RefreshHolidays() map[string]interface{} {
	if a.holidayStore == nil {
		return map[string]interface{}{"years": []int{}, "current": false}
	}
	now := time.Now()
	a.holidayStore.Refresh(now.Year(), now.Year()+1)
	coverage := a.holidayStore.Coverage()
	a.emit("calendar:holidays", coverage)
	return map[string]interface{}{
		"years":   coverage,
		"current": a.holidayStore.Has(now.Year()),
	}
}
