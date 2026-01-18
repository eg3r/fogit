package common

import (
	"fmt"
	"time"
)

// FormatTimeAgo formats a time as a human-readable "time ago" string
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// FormatDuration formats a duration as a short human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatDurationLong formats a duration as a long human-readable string (e.g., "2 days, 3 hours")
func FormatDurationLong(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%d minutes", minutes)
}

// FormatDate formats a time as a date string (YYYY-MM-DD)
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateTime formats a time as a datetime string (YYYY-MM-DD HH:MM:SS)
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDateTimeShort formats a time as a short datetime string (MM-DD HH:MM)
func FormatDateTimeShort(t time.Time) string {
	return t.Format("01-02 15:04")
}

// StartOfDay returns the start of the day for the given time
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day for the given time
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// IsToday returns true if the given time is today
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.YearDay() == now.YearDay()
}

// IsYesterday returns true if the given time is yesterday
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay()
}

// IsThisWeek returns true if the given time is in the current week
func IsThisWeek(t time.Time) bool {
	now := time.Now()
	_, nowWeek := now.ISOWeek()
	_, tWeek := t.ISOWeek()
	return now.Year() == t.Year() && nowWeek == tWeek
}

// DaysBetween returns the number of days between two times
func DaysBetween(a, b time.Time) int {
	diff := b.Sub(a)
	return int(diff.Hours() / 24)
}

// ParseDuration parses a duration string with common shortcuts
// Supports: "1h", "30m", "2d" (days), "1w" (weeks)
func ParseDuration(s string) (time.Duration, error) {
	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle custom formats
	var value int
	var unit string
	_, err := fmt.Sscanf(s, "%d%s", &value, &unit)
	if err != nil {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	switch unit {
	case "d", "day", "days":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w", "week", "weeks":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
