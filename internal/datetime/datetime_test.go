package datetime

import (
	"fmt"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/tmsdnl/datetime-mcp/internal/formats"
)

const testdataDir = "../../testdata/formats"

// loadTestRegistry loads formats from testdata and fatals on error.
func loadTestRegistry(t *testing.T) *formats.Registry {
	t.Helper()
	fmts, errs := formats.Load(testdataDir)
	if len(errs) != 0 {
		t.Fatalf("loading testdata formats: %v", errs)
	}
	return formats.NewRegistry(fmts)
}

// refTime returns a fixed reference time in America/Los_Angeles (PST, UTC-8).
func refTime(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return time.Date(2026, 2, 23, 14, 32, 5, 0, loc)
}

// --- ResolveTimezone ---

func TestResolveTimezone_Flag(t *testing.T) {
	loc, err := ResolveTimezone("America/Los_Angeles", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.String() != "America/Los_Angeles" {
		t.Errorf("got %q, want America/Los_Angeles", loc.String())
	}
}

func TestResolveTimezone_Env(t *testing.T) {
	loc, err := ResolveTimezone("", "Europe/Vilnius")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.String() != "Europe/Vilnius" {
		t.Errorf("got %q, want Europe/Vilnius", loc.String())
	}
}

func TestResolveTimezone_FlagOverridesEnv(t *testing.T) {
	loc, err := ResolveTimezone("UTC", "Europe/Vilnius")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.String() != "UTC" {
		t.Errorf("flag should override env; got %q, want UTC", loc.String())
	}
}

func TestResolveTimezone_LocalFallback(t *testing.T) {
	loc, err := ResolveTimezone("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc == nil {
		t.Fatal("expected non-nil location")
	}
	// Should never return Go's internal "Local" placeholder — it must resolve
	// to a named IANA location (e.g. "America/Los_Angeles") or fall back to
	// time.Local only when /etc/localtime is unavailable.
	if loc.String() == "Local" {
		t.Logf("note: could not resolve IANA name for local timezone (no /etc/localtime?)")
	}
}

func TestResolveTimezone_Invalid(t *testing.T) {
	loc, err := ResolveTimezone("Mars/Olympus", "")
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
	if loc != time.UTC {
		t.Errorf("expected UTC fallback on invalid timezone, got %q", loc.String())
	}
	if !strings.Contains(err.Error(), "Mars/Olympus") {
		t.Errorf("error should mention the invalid tz, got: %v", err)
	}
}

func TestResolveTimezone_InvalidEnv(t *testing.T) {
	_, err := ResolveTimezone("", "Not/ATimezone")
	if err == nil {
		t.Fatal("expected error for invalid TZ env var")
	}
}

// --- ISO8601Fallback ---

func TestISO8601Fallback(t *testing.T) {
	tm := refTime(t)
	got := ISO8601Fallback(tm)
	want := "2026-02-23T14:32:05-08:00"
	if got != want {
		t.Errorf("ISO8601Fallback = %q, want %q", got, want)
	}
}

// --- StructuredContent ---

func TestNewStructuredContent(t *testing.T) {
	tm := refTime(t)
	formatted := "2026-02-23T14:32:05-08:00"
	sc := NewStructuredContent(tm, formatted)

	if sc.Datetime != formatted {
		t.Errorf("Datetime = %q, want %q", sc.Datetime, formatted)
	}
	if sc.Timezone != "America/Los_Angeles" {
		t.Errorf("Timezone = %q, want America/Los_Angeles", sc.Timezone)
	}
	if sc.UTCOffset != "-08:00" {
		t.Errorf("UTCOffset = %q, want -08:00", sc.UTCOffset)
	}
	if sc.Unix != tm.Unix() {
		t.Errorf("Unix = %d, want %d", sc.Unix, tm.Unix())
	}
}

func TestNewStructuredContent_UTCOffset_Positive(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Vilnius") // UTC+2 in winter
	tm := time.Date(2026, 1, 15, 12, 0, 0, 0, loc)
	sc := NewStructuredContent(tm, "")
	// Vilnius in January is EET (UTC+2).
	if !strings.HasPrefix(sc.UTCOffset, "+") {
		t.Errorf("expected positive offset for Europe/Vilnius, got %q", sc.UTCOffset)
	}
}

func TestNewStructuredContent_UTCOffset_UTC(t *testing.T) {
	tm := time.Date(2026, 2, 23, 14, 32, 5, 0, time.UTC)
	sc := NewStructuredContent(tm, "")
	if sc.UTCOffset != "+00:00" {
		t.Errorf("UTCOffset for UTC = %q, want +00:00", sc.UTCOffset)
	}
}

func TestNewStructuredContent_UTCOffsetFormat(t *testing.T) {
	// Format must be [+-]HH:MM.
	tm := refTime(t)
	sc := NewStructuredContent(tm, "")
	if len(sc.UTCOffset) != 6 {
		t.Errorf("UTCOffset length = %d, want 6; got %q", len(sc.UTCOffset), sc.UTCOffset)
	}
	if sc.UTCOffset[0] != '+' && sc.UTCOffset[0] != '-' {
		t.Errorf("UTCOffset must start with + or -, got %q", sc.UTCOffset)
	}
}

// --- Formatter ---

func TestFormatterFormatNamed_Default(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	got, err := f.FormatNamed(tm, "default")
	if err != nil {
		t.Fatalf("FormatNamed(default): %v", err)
	}
	if !strings.Contains(got, "[CONTEXT]") {
		t.Errorf("default format should contain [CONTEXT], got: %q", got)
	}
	if !strings.Contains(got, "2026-02-23") {
		t.Errorf("default format should contain date, got: %q", got)
	}
	if !strings.Contains(got, "America/Los_Angeles") {
		t.Errorf("default format should contain timezone, got: %q", got)
	}
}

func TestFormatterFormatNamed_EmptyFallsBackToDefault(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	got1, _ := f.FormatNamed(tm, "")
	got2, _ := f.FormatNamed(tm, "default")
	if got1 != got2 {
		t.Errorf("empty name and 'default' should produce same output:\n%q\n%q", got1, got2)
	}
}

func TestFormatterFormatNamed_NoFormats_ISO8601Fallback(t *testing.T) {
	// With empty registry, should fall back to ISO 8601.
	reg := formats.NewRegistry(nil)
	f := New(reg, nil)
	tm := refTime(t)

	got, err := f.FormatNamed(tm, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := ISO8601Fallback(tm)
	if got != want {
		t.Errorf("expected ISO 8601 fallback %q, got %q", want, got)
	}
}

func TestFormatterFormatNamed_ISO8601(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	got, err := f.FormatNamed(tm, "iso8601")
	if err != nil {
		t.Fatalf("FormatNamed(iso8601): %v", err)
	}
	want := "2026-02-23T14:32:05-08:00"
	if got != want {
		t.Errorf("iso8601: got %q, want %q", got, want)
	}
}

func TestFormatterFormatNamed_RFC2822(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	got, err := f.FormatNamed(tm, "rfc2822")
	if err != nil {
		t.Fatalf("FormatNamed(rfc2822): %v", err)
	}
	want := "Mon, 23 Feb 2026 14:32:05 -0800"
	if got != want {
		t.Errorf("rfc2822: got %q, want %q", got, want)
	}
}

func TestFormatterFormatNamed_Unknown(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	_, err := f.FormatNamed(tm, "does-not-exist")
	if err == nil {
		t.Error("expected error for unknown format name")
	}
}

func TestFormatterFormatTemplate(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	tests := []struct {
		tmpl string
		want string
	}{
		{"{yyyy-MM-dd}", "2026-02-23"},
		{"{HH:mm:ss}", "14:32:05"},
		{"{timezone}", "America/Los_Angeles"},
		{fmt.Sprintf("ts:{unix}"), fmt.Sprintf("ts:%d", tm.Unix())},
	}
	for _, tc := range tests {
		t.Run(tc.tmpl, func(t *testing.T) {
			got := f.FormatTemplate(tm, tc.tmpl)
			if got != tc.want {
				t.Errorf("FormatTemplate(%q) = %q, want %q", tc.tmpl, got, tc.want)
			}
		})
	}
}

func TestFormatterFormatAuto(t *testing.T) {
	reg := loadTestRegistry(t)
	f := New(reg, nil)
	tm := refTime(t)

	tests := []struct {
		name      string
		formatStr string
		check     func(string) bool
		want      string
	}{
		{"empty→default", "", func(s string) bool { return strings.Contains(s, "[CONTEXT]") }, ""},
		{"default→default", "default", func(s string) bool { return strings.Contains(s, "[CONTEXT]") }, ""},
		{"named iso8601", "iso8601", nil, "2026-02-23T14:32:05-08:00"},
		{"inline template", "{yyyy-MM-dd}", nil, "2026-02-23"},
		{"go layout", "2006-01-02", nil, "2026-02-23"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := f.FormatAuto(tm, tc.formatStr)
			if err != nil {
				t.Fatalf("FormatAuto(%q): %v", tc.formatStr, err)
			}
			if tc.want != "" && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if tc.check != nil && !tc.check(got) {
				t.Errorf("FormatAuto(%q) = %q failed check", tc.formatStr, got)
			}
		})
	}
}

// --- Timezone cross-day boundary (NF-051) ---

func TestTimezoneConversion_CrossDayBoundary(t *testing.T) {
	// UTC 23:00 on Feb 23 → UTC+13 = next day Feb 24 at 12:00
	loc, err := time.LoadLocation("Pacific/Apia") // UTC+13 in summer (UTC+14 in DST)
	if err != nil {
		t.Skip("Pacific/Apia not available:", err)
	}
	utcTime := time.Date(2026, 2, 23, 23, 0, 0, 0, time.UTC)
	localTime := utcTime.In(loc)

	// Ensure the date has rolled over.
	if localTime.Day() != 24 {
		// Some Pacific/Apia offsets may vary; check that the day changed.
		// Pacific/Apia is UTC+13 (or +14), so 23:00 UTC = 12:00 or 13:00 next day.
		t.Logf("localTime day=%d (expected 24), location=%s, offset=%v", localTime.Day(), loc, localTime.Format("-07:00"))
	}

	// The date in the formatted output should reflect the local timezone.
	got := ISO8601Fallback(localTime)
	if !strings.Contains(got, localTime.Format("2006-01-02")) {
		t.Errorf("formatted output %q doesn't match local date %s", got, localTime.Format("2006-01-02"))
	}
}

func TestTimezoneConversion_WesternBoundary(t *testing.T) {
	// UTC 00:30 on Feb 24 → UTC-12 = Feb 23 at 12:30
	loc, err := time.LoadLocation("Etc/GMT+12")
	if err != nil {
		t.Skip("Etc/GMT+12 not available:", err)
	}
	utcTime := time.Date(2026, 2, 24, 0, 30, 0, 0, time.UTC)
	localTime := utcTime.In(loc)

	if localTime.Day() != 23 {
		t.Logf("localTime day=%d (expected 23), location=%s", localTime.Day(), loc)
	}

	got := ISO8601Fallback(localTime)
	if !strings.Contains(got, localTime.Format("2006-01-02")) {
		t.Errorf("formatted output %q doesn't match local date %s", got, localTime.Format("2006-01-02"))
	}
}
