package timeutil

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const layoutNumericOffset = "2006-01-02T15:04:05-0700"
const layoutNaiveISO = "2006-01-02T15:04:05"

var durationRe = regexp.MustCompile(`^(?:(\d+)\s*h)?\s*(?:(\d+)\s*m)?$`)

// ParseTimestamp parses RFC3339, numeric-offset (+0700), or naive ISO datetimes
// combined with an IANA or WIB/WITA/WIT timezone name.
func ParseTimestamp(raw, tzName string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(layoutNumericOffset, s); err == nil {
		return t, nil
	}

	loc, err := LoadLocation(tzName)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse naive timestamp %q: %w", raw, err)
	}
	t, err := time.ParseInLocation(layoutNaiveISO, s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q in %q: %w", raw, tzName, err)
	}
	return t, nil
}

// ParseDurationMinutes parses human duration strings such as "1h 45m", "3h 5m", "55m", or "2h".
func ParseDurationMinutes(raw string) (int, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	matches := durationRe.FindStringSubmatch(s)
	if matches == nil || (matches[1] == "" && matches[2] == "") {
		return 0, fmt.Errorf("invalid duration format %q", raw)
	}

	var total int
	if matches[1] != "" {
		h, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid hours in %q: %w", raw, err)
		}
		total += h * 60
	}
	if matches[2] != "" {
		m, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes in %q: %w", raw, err)
		}
		total += m
	}
	return total, nil
}

// HoursToMinutes converts fractional hours (e.g., 4.33) to rounded total minutes (260).
func HoursToMinutes(hours float64) int {
	return int(math.Round(hours * 60))
}

// FormatDuration formats total minutes into a human-readable string (e.g., 260 -> "4h 20m").
func FormatDuration(totalMinutes int) string {
	if totalMinutes <= 0 {
		return "0m"
	}
	h := totalMinutes / 60
	m := totalMinutes % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}
