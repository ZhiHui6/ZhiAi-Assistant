package holiday

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Day struct {
	Name string `json:"name"`
	Off  bool   `json:"off"`
}

type yearFile struct {
	Year      int            `json:"year"`
	FetchedAt time.Time      `json:"fetchedAt"`
	Source    string         `json:"source"`
	Days      map[string]Day `json:"days"`
}

type Store struct {
	mu    sync.RWMutex
	years map[int]*yearFile
	dir   string
	http  *http.Client
	now   func() time.Time

	endpoint string
	refresh  time.Duration
	retry    time.Duration
	log      func(format string, args ...interface{})

	fetching map[int]bool
	attempts map[int]time.Time
}

const (
	defaultEndpoint = "https://timor.tech/api/holiday/year/%d"
	sourceRemote    = "timor.tech"
	sourceCache     = "cache"
)

func New(dir string) *Store {
	return &Store{
		years:    make(map[int]*yearFile),
		dir:      dir,
		http:     &http.Client{Timeout: 8 * time.Second},
		now:      time.Now,
		endpoint: defaultEndpoint,
		refresh:  7 * 24 * time.Hour,
		retry:    6 * time.Hour,
		log:      func(string, ...interface{}) {},
		fetching: make(map[int]bool),
		attempts: make(map[int]time.Time),
	}
}

func (s *Store) SetLogger(fn func(string, ...interface{})) { s.log = fn }

func (s *Store) Load() {
	if s.dir == "" {
		return
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.dir, name))
		if err != nil {
			continue
		}
		var f yearFile
		if err := json.Unmarshal(raw, &f); err != nil || f.Year == 0 || len(f.Days) == 0 {
			continue
		}
		s.years[f.Year] = &f
	}
}

func (s *Store) Lookup(date string) (Day, bool) {
	year := parseYear(date)
	if year == 0 {
		return Day{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.years[year]
	if !ok {
		return Day{}, false
	}
	day, ok := f.Days[date]
	return day, ok
}

func (s *Store) Coverage() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]int, 0, len(s.years))
	for y := range s.years {
		out = append(out, y)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (s *Store) Has(year int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.years[year]
	return ok && len(f.Days) > 0
}

func (s *Store) Ensure(years ...int) bool {
	changed := false
	for _, year := range years {
		if year < 2000 || year > 2100 {
			continue
		}
		if !s.needFetch(year) {
			continue
		}
		if s.fetch(year) {
			changed = true
		}
	}
	return changed
}

func (s *Store) Refresh(years ...int) bool {
	changed := false
	for _, year := range years {
		if year < 2000 || year > 2100 {
			continue
		}
		if s.fetch(year) {
			changed = true
		}
	}
	return changed
}

func (s *Store) EnsureAsync(years ...int) {
	go func() {
		if s.Ensure(years...) {
			s.log("holiday data updated: %v", s.Coverage())
		}
	}()
}

func (s *Store) needFetch(year int) bool {
	s.mu.RLock()
	hasData := false
	var fetchedAt time.Time
	if f, ok := s.years[year]; ok && len(f.Days) > 0 {
		hasData = true
		fetchedAt = f.FetchedAt
	}
	fetching := s.fetching[year]
	lastAttempt := s.attempts[year]
	s.mu.RUnlock()

	if fetching {
		return false
	}
	now := s.now()
	if hasData && now.Sub(fetchedAt) < s.refresh {
		return false
	}
	if !lastAttempt.IsZero() && now.Sub(lastAttempt) < s.retry {
		return false
	}
	return true
}

func (s *Store) fetch(year int) bool {
	s.mu.Lock()
	if s.fetching[year] {
		s.mu.Unlock()
		return false
	}
	s.fetching[year] = true
	s.attempts[year] = s.now()
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.fetching[year] = false
		s.mu.Unlock()
	}()

	days, err := s.download(year)
	if err != nil {
		s.log("holiday fetch %d failed: %v", year, err)
		return false
	}
	if len(days) == 0 {
		s.log("holiday %d not published yet", year)
		return false
	}

	f := &yearFile{Year: year, FetchedAt: s.now(), Source: sourceRemote, Days: days}
	s.mu.Lock()
	s.years[year] = f
	s.mu.Unlock()

	if err := s.save(f); err != nil {
		s.log("holiday cache write failed: %v", err)
	}
	return true
}

func (s *Store) download(year int) (map[string]Day, error) {
	url := fmt.Sprintf(s.endpoint, year)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zhi-ai-assistant-holiday/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return parseTimor(body, year)
}

func parseTimor(body []byte, year int) (map[string]Day, error) {
	var payload struct {
		Code    int `json:"code"`
		Holiday map[string]struct {
			Holiday *bool  `json:"holiday"`
			Name    string `json:"name"`
			Date    string `json:"date"`
			Target  string `json:"target"`
			After   bool   `json:"after"`
		} `json:"holiday"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if payload.Code != 0 {
		return nil, fmt.Errorf("api code %d", payload.Code)
	}
	out := make(map[string]Day, len(payload.Holiday))
	for key, item := range payload.Holiday {
		date := item.Date
		if date == "" {
			date = fmt.Sprintf("%04d-%s", year, key)
		}
		if !strings.HasPrefix(date, fmt.Sprintf("%04d-", year)) {
			continue
		}
		if item.Holiday == nil {
			continue
		}
		name := item.Name
		if *item.Holiday {
			if item.Target != "" {
				name = item.Target
			}
		} else if item.Target != "" {
			name = item.Target
		}
		out[date] = Day{Name: strings.TrimSpace(name), Off: *item.Holiday}
	}
	return out, nil
}

func (s *Store) save(f *yearFile) error {
	if s.dir == "" {
		return nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, fmt.Sprintf("%d.json", f.Year))
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func parseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year := 0
	for i := 0; i < 4; i++ {
		c := date[i]
		if c < '0' || c > '9' {
			return 0
		}
		year = year*10 + int(c-'0')
	}
	return year
}
