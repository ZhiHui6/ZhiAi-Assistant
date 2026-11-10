package model

import "time"

type Note struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Tags      []string  `json:"tags,omitempty"`
	Tint      string    `json:"tint,omitempty"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type NotesData struct {
	Items []Note `json:"items"`
}

type Todo struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Note        string     `json:"note,omitempty"`
	Done        bool       `json:"done"`
	Priority    int        `json:"priority"`
	DueDate     *string    `json:"dueDate,omitempty"`
	Group       string     `json:"group,omitempty"`
	Order       int        `json:"order"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type TodosData struct {
	Items  []Todo   `json:"items"`
	Groups []string `json:"groups,omitempty"`
}

type ShelfItem struct {
	ID      string    `json:"id"`
	GroupID string    `json:"groupId"`
	Kind    string    `json:"kind"`
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Args    string    `json:"args,omitempty"`
	WorkDir string    `json:"workDir,omitempty"`
	Order   int       `json:"order"`
	AddedAt time.Time `json:"addedAt"`
}

type ShelfGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Order     int    `json:"order"`
	Collapsed bool   `json:"collapsed"`
}

type ShelfData struct {
	Groups []ShelfGroup `json:"groups"`
	Items  []ShelfItem  `json:"items"`
}

type LaunchCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
	LegacySystem bool `json:"system,omitempty"`
}

type LaunchShortcut struct {
	ID         string    `json:"id"`
	CategoryID string    `json:"categoryId"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	BaseName   string    `json:"baseName,omitempty"`
	Order      int       `json:"order"`
	AddedAt    time.Time `json:"addedAt"`
}

type LaunchData struct {
	Categories []LaunchCategory `json:"categories"`
	Items      []LaunchShortcut `json:"items"`
}

type Alarm struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	Label   string `json:"label,omitempty"`
	Repeat  bool   `json:"repeat"`
	Enabled bool   `json:"enabled"`
}

type Countdown struct {
	Running   bool       `json:"running"`
	EndAt     *time.Time `json:"endAt,omitempty"`
	Remaining int        `json:"remaining"`
	Label     string     `json:"label,omitempty"`
}

type ShutdownPlan struct {
	Active bool       `json:"active"`
	At     *time.Time `json:"at,omitempty"`
}

type TimerData struct {
	Alarms    []Alarm      `json:"alarms"`
	Countdown Countdown    `json:"countdown"`
	Shutdown  ShutdownPlan `json:"shutdown"`
}

type Mount struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type MountsData struct {
	Items []Mount `json:"items"`
}

type Settings struct {
	Edge                  string   `json:"edge"`
	PanelWidth            int      `json:"panelWidth"`
	Peek                  int      `json:"peek"`
	TargetMonitor         int      `json:"targetMonitor"`
	Hotkey                string   `json:"hotkey"`
	Language              string   `json:"language"`
	EdgeEnabled           bool     `json:"edgeEnabled"`
	Autostart             bool     `json:"autostart"`
	Opacity               float64  `json:"opacity"`
	Background            string   `json:"background"`
	Animations            bool     `json:"animations"`
	ShowClock             bool     `json:"showClock"`
	ShowCalendar          bool     `json:"showCalendar"`
	HideCalendarFestivals bool     `json:"hideCalendarFestivals"`
	AlarmSound            string   `json:"alarmSound"`
	CountdownSound        string   `json:"countdownSound"`
	LaunchColumns         int      `json:"launchColumns"`
	LaunchRows            int      `json:"launchRows"`
	LaunchIconSize        int      `json:"launchIconSize"`
	HideLaunchNames       bool     `json:"hideLaunchNames"`
	HideLaunch            bool     `json:"hideLaunch"`
	Clock12               bool     `json:"clock12"`
	ShelfView             string   `json:"shelfView"`
	ShelfScale            string   `json:"shelfScale"`
	MountView             string   `json:"mountView"`
	MountScale            string   `json:"mountScale"`
}

func DefaultSettings() Settings {
	return Settings{
		Edge:           "right",
		PanelWidth:     340,
		Peek:           30,
		TargetMonitor:  -1,
		Hotkey:         "ctrl+alt+space",
		Language:       "auto",
		EdgeEnabled:    false,
		Autostart:      false,
		Opacity:        0.8,
		Background:     "#0a0c11",
		Animations:     true,
		ShowClock:      true,
		ShowCalendar:   true,
		AlarmSound:     "chime",
		CountdownSound: "chime",
		LaunchColumns:  5,
		LaunchRows:     1,
		LaunchIconSize: 32,
		ShelfView:      "grid",
		ShelfScale:     "normal",
		MountView:      "list",
		MountScale:     "normal",
	}
}
