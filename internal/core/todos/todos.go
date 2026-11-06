package todos

import (
	"errors"
	"sort"
	"strings"
	"time"

	"zhi-ai-assistant/internal/core/id"
	"zhi-ai-assistant/internal/core/model"
	"zhi-ai-assistant/internal/core/store"
)

const dateLayout = "2006-01-02"

type Filter struct {
	Scope string `json:"scope"`
	Date  string `json:"date,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type Input struct {
	Title    string  `json:"title"`
	Note     string  `json:"note,omitempty"`
	Priority int     `json:"priority"`
	DueDate  *string `json:"dueDate,omitempty"`
	Group    string  `json:"group,omitempty"`
}

type Patch struct {
	Title    *string `json:"title,omitempty"`
	Note     *string `json:"note,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	DueDate  *string `json:"dueDate,omitempty"`
	ClearDue bool    `json:"clearDue,omitempty"`
	Group    *string `json:"group,omitempty"`
	Done     *bool   `json:"done,omitempty"`
}

type Stats struct {
	Today     int `json:"today"`
	TodayDone int `json:"todayDone"`
	Week      int `json:"week"`
	Overdue   int `json:"overdue"`
	Open      int `json:"open"`
	Done      int `json:"done"`
	Total     int `json:"total"`
}

type DaySummary struct {
	Date    string `json:"date"`
	Open    int    `json:"open"`
	Done    int    `json:"done"`
	Overdue int    `json:"overdue"`
}

type Manager struct {
	doc *store.Doc[model.TodosData]
}

func New() (*Manager, error) {
	doc, err := store.Open("todos.json", 1, model.TodosData{Items: []model.Todo{}})
	if err != nil {
		return nil, err
	}
	return &Manager{doc: doc}, nil
}

func (m *Manager) Flush() error { return m.doc.Flush() }

func Today() string { return time.Now().Format(dateLayout) }

func dayString(t time.Time) string { return t.Format(dateLayout) }

func weekRange(now time.Time) (string, string) {
	offset := (int(now.Weekday()) + 6) % 7
	start := now.AddDate(0, 0, -offset)
	return dayString(start), dayString(start.AddDate(0, 0, 6))
}

func (m *Manager) List(f Filter) []model.Todo {
	today := Today()
	weekStart, weekEnd := weekRange(time.Now())
	var out []model.Todo
	m.doc.Read(func(d *model.TodosData) {
		for _, t := range d.Items {
			if keep(t, f, today, weekStart, weekEnd) {
				out = append(out, t)
			}
		}
	})
	if f.Scope == "done" {
		sort.SliceStable(out, func(i, j int) bool {
			ci, cj := out[i].CompletedAt, out[j].CompletedAt
			if ci == nil || cj == nil {
				return cj == nil && ci != nil
			}
			return ci.After(*cj)
		})
	} else {
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Done != out[j].Done {
				return !out[i].Done
			}
			if out[i].Priority != out[j].Priority {
				return out[i].Priority > out[j].Priority
			}
			di, dj := out[i].DueDate, out[j].DueDate
			if di != nil && dj != nil && *di != *dj {
				return *di < *dj
			}
			if (di == nil) != (dj == nil) {
				return di != nil
			}
			return out[i].Order < out[j].Order
		})
	}
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	if out == nil {
		out = []model.Todo{}
	}
	return out
}

func keep(t model.Todo, f Filter, today, weekStart, weekEnd string) bool {
	switch f.Scope {
	case "today":
		return t.DueDate != nil && *t.DueDate == today
	case "week":
		if t.Done {
			return false
		}
		return t.DueDate != nil && *t.DueDate >= weekStart && *t.DueDate <= weekEnd
	case "overdue":
		return !t.Done && t.DueDate != nil && *t.DueDate < today
	case "open":
		return !t.Done
	case "done":
		return t.Done
	case "all":
		return true
	case "date":
		return t.DueDate != nil && *t.DueDate == f.Date
	default:
		return !t.Done
	}
}

func (m *Manager) Create(in Input) (model.Todo, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Todo{}, errors.New("标题不能为空")
	}
	if in.Priority < 0 || in.Priority > 2 {
		in.Priority = 1
	}
	t := model.Todo{
		ID:        id.New(),
		Title:     title,
		Note:      in.Note,
		Priority:  in.Priority,
		DueDate:   cleanDate(in.DueDate),
		Group:     in.Group,
		CreatedAt: time.Now(),
	}
	m.doc.Update(func(d *model.TodosData) {
		min := 0
		for _, x := range d.Items {
			if x.Order < min {
				min = x.Order
			}
		}
		t.Order = min - 1
		d.Items = append(d.Items, t)
	})
	return t, nil
}

func cleanDate(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	if len(v) > 10 {
		v = v[:10]
	}
	if _, err := time.Parse(dateLayout, v); err != nil {
		return nil
	}
	return &v
}

func (m *Manager) Update(id string, p Patch) (model.Todo, error) {
	var res model.Todo
	found := false
	m.doc.Update(func(d *model.TodosData) {
		for i := range d.Items {
			if d.Items[i].ID != id {
				continue
			}
			if p.Title != nil {
				if v := strings.TrimSpace(*p.Title); v != "" {
					d.Items[i].Title = v
				}
			}
			if p.Note != nil {
				d.Items[i].Note = *p.Note
			}
			if p.Priority != nil {
				d.Items[i].Priority = *p.Priority
			}
			if p.ClearDue {
				d.Items[i].DueDate = nil
			} else if p.DueDate != nil {
				d.Items[i].DueDate = cleanDate(p.DueDate)
			}
			if p.Group != nil {
				d.Items[i].Group = *p.Group
			}
			if p.Done != nil {
				d.Items[i].Done = *p.Done
				if *p.Done {
					now := time.Now()
					d.Items[i].CompletedAt = &now
				} else {
					d.Items[i].CompletedAt = nil
				}
			}
			res = d.Items[i]
			found = true
			return
		}
	})
	if !found {
		return model.Todo{}, errors.New("待办不存在")
	}
	return res, nil
}

func (m *Manager) Toggle(id string, done bool) (model.Todo, error) {
	return m.Update(id, Patch{Done: &done})
}

func (m *Manager) Delete(id string) error {
	found := false
	m.doc.Update(func(d *model.TodosData) {
		for i := range d.Items {
			if d.Items[i].ID == id {
				d.Items = append(d.Items[:i], d.Items[i+1:]...)
				found = true
				return
			}
		}
	})
	if !found {
		return errors.New("待办不存在")
	}
	return nil
}

func (m *Manager) Restore(t model.Todo) (model.Todo, error) {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return model.Todo{}, errors.New("标题不能为空")
	}
	if t.Priority < 0 || t.Priority > 2 {
		t.Priority = 1
	}
	t.DueDate = cleanDate(t.DueDate)
	if t.ID != "" {
		exists := false
		m.doc.Read(func(d *model.TodosData) {
			for _, x := range d.Items {
				if x.ID == t.ID {
					exists = true
					return
				}
			}
		})
		if exists {
			t.ID = ""
		}
	}
	if t.ID == "" {
		t.ID = id.New()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	m.doc.Update(func(d *model.TodosData) {
		d.Items = append(d.Items, t)
	})
	return t, nil
}

func (m *Manager) Reorder(orderedIDs []string) error {
	idx := make(map[string]int, len(orderedIDs))
	for i, v := range orderedIDs {
		idx[v] = i
	}
	m.doc.Update(func(d *model.TodosData) {
		for i := range d.Items {
			if pos, ok := idx[d.Items[i].ID]; ok {
				d.Items[i].Order = pos
			}
		}
	})
	return nil
}

func (m *Manager) Stats() Stats {
	today := Today()
	weekStart, weekEnd := weekRange(time.Now())
	var s Stats
	m.doc.Read(func(d *model.TodosData) {
		for _, t := range d.Items {
			s.Total++
			if !t.Done {
				s.Open++
				if t.DueDate != nil {
					switch {
					case *t.DueDate < today:
						s.Overdue++
					case *t.DueDate == today:
						s.Today++
					}
					if *t.DueDate >= weekStart && *t.DueDate <= weekEnd {
						s.Week++
					}
				}
				continue
			}
			s.Done++
			if t.DueDate != nil && *t.DueDate == today {
				s.TodayDone++
			}
		}
	})
	return s
}

func (m *Manager) MonthSummary(year, month int) []DaySummary {
	today := time.Now()
	todayStr := dayString(today)
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)
	byDate := map[string]*DaySummary{}
	m.doc.Read(func(d *model.TodosData) {
		for _, t := range d.Items {
			if t.DueDate == nil {
				continue
			}
			due, err := time.ParseInLocation(dateLayout, *t.DueDate, time.Local)
			if err != nil || due.Before(first) || due.After(last) {
				continue
			}
			s, ok := byDate[*t.DueDate]
			if !ok {
				s = &DaySummary{Date: *t.DueDate}
				byDate[*t.DueDate] = s
			}
			if t.Done {
				s.Done++
			} else {
				s.Open++
				if *t.DueDate < todayStr {
					s.Overdue++
				}
			}
		}
	})
	out := make([]DaySummary, 0, len(byDate))
	for _, v := range byDate {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}

func (m *Manager) ForDate(date string) []model.Todo {
	return m.List(Filter{Scope: "date", Date: date})
}
