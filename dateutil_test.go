package main

import (
	"testing"
	"time"
)

// TestFormat and Translate functions using a known date.
func TestFormatFunctions(t *testing.T) {
	// Use a fixed time for testing.
	// January 2, 2006 is a Monday.
	testTime := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)

	t.Run("FormatDate", func(t *testing.T) {
		// DefaultDateFormat is "YYYY-MM-DD" which translates to "2006-01-02"
		expected := "2006-01-02"
		result := FormatDate(testTime)
		if result != expected {
			t.Errorf("FormatDate: expected %q, got %q", expected, result)
		}
	})

	t.Run("FormatTime", func(t *testing.T) {
		// DefaultTimeFormat is "hh:mm:ss" which translates to "15:04:05"
		expected := "15:04:05"
		result := FormatTime(testTime)
		if result != expected {
			t.Errorf("FormatTime: expected %q, got %q", expected, result)
		}
	})

	t.Run("FormatDateTime", func(t *testing.T) {
		// DefaultDateTimeFormat is "DDDD, DD MMM YYYY hh:mm:ss pm"
		// After replacement, it should yield "Monday, 02 Jan 2006 15:04:05 PM"
		expected := "Monday, 02 Jan 2006 15:04:05 PM"
		result := FormatDateTime(testTime)
		if result != expected {
			t.Errorf("FormatDateTime: expected %q, got %q", expected, result)
		}
	})
}

// TestParse functions by formatting a time then parsing it back.
func TestParseFunctions(t *testing.T) {
	// Use a fixed time for testing.
	// Note: When parsing a date or time that does not contain a full timestamp,
	// time.Parse returns a time with default date values.
	original := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)

	t.Run("ParseDate", func(t *testing.T) {
		formatted := FormatDate(original) // "2006-01-02"
		parsed, err := ParseDate(formatted)
		if err != nil {
			t.Fatalf("ParseDate returned error: %v", err)
		}
		// Since the parsed date has no time info, compare the year, month, and day.
		if parsed.Year() != original.Year() || parsed.Month() != original.Month() || parsed.Day() != original.Day() {
			t.Errorf("ParseDate: expected %v, got %v", original, parsed)
		}
	})

	t.Run("ParseTime", func(t *testing.T) {
		formatted := FormatTime(original) // "15:04:05"
		parsed, err := ParseTime(formatted)
		if err != nil {
			t.Fatalf("ParseTime returned error: %v", err)
		}
		// When parsing only a time, Go sets the date to January 1, year 0 in UTC.
		if parsed.Hour() != original.Hour() || parsed.Minute() != original.Minute() || parsed.Second() != original.Second() {
			t.Errorf("ParseTime: expected %02d:%02d:%02d, got %02d:%02d:%02d",
				original.Hour(), original.Minute(), original.Second(),
				parsed.Hour(), parsed.Minute(), parsed.Second())
		}
	})

	t.Run("ParseDateTime", func(t *testing.T) {
		formatted := FormatDateTime(original) // "Monday, 02 Jan 2006 15:04:05 PM"
		parsed, err := ParseDateTime(formatted)
		if err != nil {
			t.Fatalf("ParseDateTime returned error: %v", err)
		}
		// Compare year, month, day and time components.
		if parsed.Year() != original.Year() || parsed.Month() != original.Month() || parsed.Day() != original.Day() ||
			parsed.Hour() != original.Hour() || parsed.Minute() != original.Minute() || parsed.Second() != original.Second() {
			t.Errorf("ParseDateTime: expected %v, got %v", original, parsed)
		}
	})
}

// Test Translate to ensure custom formats are properly converted.
func TestTranslate(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"YYYY-MM-DD", "2006-01-02"},
		{"hh:mm:ss", "15:04:05"},
		{"DDDD, DD MMM YYYY hh:mm:ss pm", "Monday, 02 Jan 2006 15:04:05 PM"},
	}
	for _, c := range cases {
		result := Translate(c.input)
		if result != c.expected {
			t.Errorf("Translate(%q): expected %q, got %q", c.input, c.expected, result)
		}
	}
}

// Test DaysToWeekend computes the correct number of days remaining until Saturday.
func TestDaysToWeekend(t *testing.T) {
	now := time.Now()
	weekday := now.Weekday()
	var expected int
	if weekday == time.Saturday || weekday == time.Sunday {
		expected = 0
	} else {
		expected = int(time.Saturday - weekday)
	}
	result := DaysToWeekend()
	if result != expected {
		t.Errorf("DaysToWeekend: expected %d, got %d", expected, result)
	}
}
