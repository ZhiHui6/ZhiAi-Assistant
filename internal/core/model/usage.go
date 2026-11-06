package model

import "time"

type UsagePack struct {
	Name      string  `json:"name,omitempty"`
	Total     float64 `json:"total"`
	Remaining float64 `json:"remaining"`
	Unit      string  `json:"unit,omitempty"`
	ExpiresAt string  `json:"expiresAt,omitempty"`
	URL       string  `json:"url,omitempty"`
}

type UsageSnapshot struct {
	Account          string      `json:"account,omitempty"`
	Plan             string      `json:"plan,omitempty"`
	Unit             string      `json:"unit,omitempty"`
	Total            float64     `json:"total"`
	Used             float64     `json:"used"`
	Remaining        float64     `json:"remaining"`
	Percent          float64     `json:"percent"`
	ResetAt          string      `json:"resetAt,omitempty"`
	DailyCredit      float64     `json:"dailyCredit,omitempty"`
	CheckedIn        bool        `json:"checkedIn"`
	CheckinAvailable bool        `json:"checkinAvailable"`
	Message          string      `json:"message,omitempty"`
	Packs            []UsagePack `json:"packs,omitempty"`
	AccountRaw       string      `json:"-"`
	UpdatedAt        time.Time   `json:"updatedAt"`
}

type UsagePlatformState struct {
	Snapshot        UsageSnapshot `json:"snapshot"`
	Source          string        `json:"source"`
	AuthState       string        `json:"authState"`
	LastError       string        `json:"lastError,omitempty"`
	LastFetchAt     time.Time     `json:"lastFetchAt,omitempty"`
	LastCheckinDate string        `json:"lastCheckinDate,omitempty"`
	AttemptDate     string        `json:"attemptDate,omitempty"`
	RetryAt         time.Time     `json:"retryAt,omitempty"`
	Attempts        int           `json:"attempts,omitempty"`
	SignedOut       bool          `json:"signedOut,omitempty"`
	Preferred       string        `json:"preferred,omitempty"`
}

type UsageState struct {
	Platforms   map[string]UsagePlatformState `json:"platforms"`
	AutoCheckin map[string]bool               `json:"autoCheckin"`
	NoticeAcked bool                          `json:"noticeAcked"`
}

type UsageCredential struct {
	Token        string    `json:"token,omitempty"`
	Account      string    `json:"account,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	UserID       string    `json:"userId,omitempty"`
	Domain       string    `json:"domain,omitempty"`
	DeviceID     string    `json:"deviceId,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}
