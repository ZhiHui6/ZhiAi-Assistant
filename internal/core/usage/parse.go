package usage

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func normKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func lookup(m map[string]any, names ...string) (any, bool) {
	if m == nil {
		return nil, false
	}
	for key, value := range m {
		key = normKey(key)
		for _, name := range names {
			if key == normKey(name) {
				return value, true
			}
		}
	}
	return nil, false
}

func numberValue(v any) (float64, bool) {
	switch value := v.(type) {
	case json.Number:
		n, err := value.Float64()
		return n, err == nil
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case string:
		cleaned := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "%"))
		cleaned = strings.ReplaceAll(cleaned, ",", "")
		if cleaned == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(cleaned, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func num(m map[string]any, names ...string) (float64, bool) {
	v, ok := lookup(m, names...)
	if !ok {
		return 0, false
	}
	return numberValue(v)
}

func text(m map[string]any, names ...string) string {
	v, ok := lookup(m, names...)
	if !ok {
		return ""
	}
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

func boolean(m map[string]any, names ...string) (bool, bool) {
	v, ok := lookup(m, names...)
	if !ok {
		return false, false
	}
	switch value := v.(type) {
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	}
	if n, ok := numberValue(v); ok {
		return n != 0, true
	}
	return false, false
}

func walkMaps(v any, fn func(map[string]any)) {
	switch value := v.(type) {
	case map[string]any:
		fn(value)
		for _, child := range value {
			walkMaps(child, fn)
		}
	case []any:
		for _, child := range value {
			walkMaps(child, fn)
		}
	}
}

func findArray(v any, names ...string) []any {
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[normKey(name)] = true
	}
	var result []any
	walkMaps(v, func(m map[string]any) {
		if result != nil {
			return
		}
		for key, value := range m {
			if !wanted[normKey(key)] {
				continue
			}
			if list, ok := value.([]any); ok {
				result = list
				return
			}
		}
	})
	return result
}

func unwrapMap(v any, names ...string) map[string]any {
	if m, ok := v.(map[string]any); ok {
		if found, ok := lookup(m, names...); ok {
			if child, ok := found.(map[string]any); ok {
				return child
			}
		}
	}
	return nil
}

func maskAccount(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 2 {
		return string(runes[0]) + "***"
	}
	if len(runes) <= 6 {
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	}
	return string(runes[:2]) + "***" + string(runes[len(runes)-2:])
}

func normalizedPercent(used, total, percentage float64) float64 {
	if percentage > 1 {
		percentage = percentage / 100
	}
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 1 {
		percentage = 1
	}
	if percentage == 0 && total > 0 {
		percentage = used / total
	}
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 1 {
		percentage = 1
	}
	return percentage
}

func timeValue(v any) time.Time {
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return time.Time{}
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return epochTime(n)
		}
		return time.Time{}
	}
	if n, ok := numberValue(v); ok {
		return epochTime(n)
	}
	return time.Time{}
}

func epochTime(n float64) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	if n > 1e12 {
		return time.UnixMilli(int64(n))
	}
	return time.Unix(int64(n), 0)
}
