package festival

import (
	"container/list"
	"fmt"
	"strings"
	"time"

	"github.com/6tail/lunar-go/HolidayUtil"
	"github.com/6tail/lunar-go/calendar"
)

type Day struct {
	Date    string `json:"date"`
	Label   string `json:"label"`
	Detail  string `json:"detail"`
	Lunar   string `json:"lunar"`
	Kind    string `json:"kind"`
	Holiday bool   `json:"holiday"`
	Workday bool   `json:"workday"`
}

type Provider interface {
	Lookup(date string) (name string, off bool, ok bool)
}

type libraryProvider struct{}

func (libraryProvider) Lookup(date string) (string, bool, bool) {
	holiday := HolidayUtil.GetHoliday(strings.ReplaceAll(date, "-", ""))
	if holiday == nil {
		return "", false, false
	}
	return holiday.GetName(), !holiday.IsWork(), true
}

var provider Provider = libraryProvider{}

func SetHolidayProvider(p Provider) {
	if p == nil {
		provider = libraryProvider{}
		return
	}
	provider = p
}

const (
	KindHoliday  = "holiday"
	KindFestival = "festival"
	KindJieQi    = "jieqi"
	KindRest     = "rest"
	KindWorkday  = "workday"
	KindLunar    = "lunar"
)

func Of(t time.Time) Day {
	return OfYmd(t.Year(), int(t.Month()), t.Day())
}

func OfYmd(year, month, day int) Day {
	solar := calendar.NewSolarFromYmd(year, month, day)
	lunar := solar.GetLunar()

	lunarText := lunarDayText(lunar)
	out := Day{
		Date:   fmt.Sprintf("%04d-%02d-%02d", year, month, day),
		Label:  lunarText,
		Lunar:  lunarText,
		Kind:   KindLunar,
		Detail: "农历" + lunarText,
	}

	lunarFestival := first(lunar.GetFestivals())
	solarFestival := first(solar.GetFestivals())
	jieqi := lunar.GetJieQi()

	restDay := false
	workday := false
	holidayName, off, hasHoliday := provider.Lookup(out.Date)
	if hasHoliday {
		if off {
			restDay = true
		} else {
			workday = true
		}
	}

	out.Holiday = restDay
	out.Workday = workday

	switch {
	case lunarFestival != "":
		out.Label, out.Kind = shortLabel(lunarFestival), KindFestival
	case workday:
		out.Label, out.Kind = "班", KindWorkday
	case solarFestival != "":
		out.Label, out.Kind = shortLabel(solarFestival), KindFestival
	case jieqi != "":
		out.Label, out.Kind = jieqi, KindJieQi
	case restDay:
		out.Label, out.Kind = "休", KindRest
	}

	var parts []string
	if lunarFestival != "" {
		parts = append(parts, lunarFestival)
	}
	if solarFestival != "" && solarFestival != lunarFestival {
		parts = append(parts, solarFestival)
	}
	if jieqi != "" {
		parts = append(parts, jieqi+"节气")
	}
	if out.Holiday && lunarFestival == "" && solarFestival == "" && jieqi == "" {
		parts = append(parts, "法定节假日")
	}
	if out.Workday {
		if holidayName != "" {
			parts = append(parts, holidayName+"调休上班")
		} else {
			parts = append(parts, "调休上班")
		}
	}
	parts = append(parts, "农历"+lunarText)
	out.Detail = strings.Join(parts, " · ")

	return out
}

func Month(year, month int) map[string]Day {
	days := time.Date(year, time.Month(month)+1, 0, 12, 0, 0, 0, time.Local).Day()
	out := make(map[string]Day, days)
	for d := 1; d <= days; d++ {
		day := OfYmd(year, month, d)
		out[day.Date] = day
	}
	return out
}

func lunarDayText(lunar *calendar.Lunar) string {
	if lunar.GetDay() == 1 {
		return lunar.GetMonthInChinese() + "月"
	}
	return lunar.GetDayInChinese()
}

func first(l *list.List) string {
	if l == nil || l.Len() == 0 {
		return ""
	}
	if s, ok := l.Front().Value.(string); ok {
		return s
	}
	return ""
}

var preferred = map[string]string{
	"元旦节":  "元旦",
	"春节":   "春节",
	"清明节":  "清明",
	"劳动节":  "劳动",
	"端午节":  "端午",
	"中秋节":  "中秋",
	"国庆节":  "国庆",
	"国庆中秋": "国庆中秋",
}

func shortLabel(name string) string {
	if short, ok := preferred[name]; ok {
		return short
	}
	runes := []rune(name)
	if len(runes) > 4 {
		trimmed := strings.TrimSuffix(strings.TrimSuffix(name, "节"), "日")
		runes = []rune(trimmed)
	}
	if len(runes) > 4 {
		runes = runes[:4]
	}
	return string(runes)
}
