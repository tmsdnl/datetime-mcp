package template

import (
	"testing"
	"time"
	_ "time/tzdata"
)

// refTime is the canonical reference time used across LDML tests.
// 2026-02-23 14:32:05 America/Los_Angeles (PST, UTC-8).
func refTime(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return time.Date(2026, 2, 23, 14, 32, 5, 0, loc)
}

func TestLDMLAllTokens(t *testing.T) {
	tm := refTime(t)

	tests := []struct {
		name      string
		expr      string
		wantOK    bool
		wantFmt   string // expected output of time.Format(goLayout)
	}{
		{"yyyy", "yyyy", true, "2026"},
		{"yy", "yy", true, "26"},
		{"MMMM", "MMMM", true, "February"},
		{"MMM", "MMM", true, "Feb"},
		{"MM", "MM", true, "02"},
		{"dd", "dd", true, "23"},
		{"EEEE", "EEEE", true, "Monday"},
		{"EEE", "EEE", true, "Mon"},
		{"HH", "HH", true, "14"},
		{"h", "h", true, "2"},
		{"mm", "mm", true, "32"},
		{"ss", "ss", true, "05"},
		{"a", "a", true, "PM"},
		{"z", "z", true, "PST"},
		{"ZZZZ", "ZZZZ", true, "-08:00"},
		{"Z", "Z", true, "-0800"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goLayout, ok := ldmlToGoLayout(tc.expr)
			if ok != tc.wantOK {
				t.Fatalf("ldmlToGoLayout(%q) ok=%v, want %v", tc.expr, ok, tc.wantOK)
			}
			got := tm.Format(goLayout)
			if got != tc.wantFmt {
				t.Errorf("time.Format(%q) = %q, want %q", goLayout, got, tc.wantFmt)
			}
		})
	}
}

func TestLDMLCaseSensitivity(t *testing.T) {
	tm := refTime(t)

	// MM (month) vs mm (minute) are distinct.
	goMM, _ := ldmlToGoLayout("MM")
	gomm, _ := ldmlToGoLayout("mm")
	if goMM == gomm {
		t.Errorf("MM and mm should produce different Go layouts")
	}
	if tm.Format(goMM) != "02" {
		t.Errorf("MM → month, got %q", tm.Format(goMM))
	}
	if tm.Format(gomm) != "32" {
		t.Errorf("mm → minute, got %q", tm.Format(gomm))
	}

	// HH (24-hour) vs h (12-hour) are distinct.
	goHH, _ := ldmlToGoLayout("HH")
	goh, _ := ldmlToGoLayout("h")
	if goHH == goh {
		t.Errorf("HH and h should produce different Go layouts")
	}
	if tm.Format(goHH) != "14" {
		t.Errorf("HH → 24-hour, got %q", tm.Format(goHH))
	}
	if tm.Format(goh) != "2" {
		t.Errorf("h → 12-hour, got %q", tm.Format(goh))
	}
}

func TestLDMLLongestFirst(t *testing.T) {
	tm := refTime(t)

	// MMMM before MMM before MM.
	t.Run("MMMM", func(t *testing.T) {
		got, ok := ldmlToGoLayout("MMMM")
		if !ok {
			t.Fatal("expected ok=true for MMMM")
		}
		if tm.Format(got) != "February" {
			t.Errorf("MMMM: got %q, want February", tm.Format(got))
		}
	})
	t.Run("MMM", func(t *testing.T) {
		got, ok := ldmlToGoLayout("MMM")
		if !ok {
			t.Fatal("expected ok=true for MMM")
		}
		if tm.Format(got) != "Feb" {
			t.Errorf("MMM: got %q, want Feb", tm.Format(got))
		}
	})
	t.Run("MM", func(t *testing.T) {
		got, ok := ldmlToGoLayout("MM")
		if !ok {
			t.Fatal("expected ok=true for MM")
		}
		if tm.Format(got) != "02" {
			t.Errorf("MM: got %q, want 02", tm.Format(got))
		}
	})

	// EEEE before EEE.
	t.Run("EEEE", func(t *testing.T) {
		got, ok := ldmlToGoLayout("EEEE")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "Monday" {
			t.Errorf("EEEE: got %q, want Monday", tm.Format(got))
		}
	})
	t.Run("EEE", func(t *testing.T) {
		got, ok := ldmlToGoLayout("EEE")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "Mon" {
			t.Errorf("EEE: got %q, want Mon", tm.Format(got))
		}
	})

	// yyyy before yy.
	t.Run("yyyy", func(t *testing.T) {
		got, ok := ldmlToGoLayout("yyyy")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "2026" {
			t.Errorf("yyyy: got %q, want 2026", tm.Format(got))
		}
	})
	t.Run("yy", func(t *testing.T) {
		got, ok := ldmlToGoLayout("yy")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "26" {
			t.Errorf("yy: got %q, want 26", tm.Format(got))
		}
	})

	// ZZZZ before Z.
	t.Run("ZZZZ", func(t *testing.T) {
		got, ok := ldmlToGoLayout("ZZZZ")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "-08:00" {
			t.Errorf("ZZZZ: got %q, want -08:00", tm.Format(got))
		}
	})
	t.Run("Z", func(t *testing.T) {
		got, ok := ldmlToGoLayout("Z")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if tm.Format(got) != "-0800" {
			t.Errorf("Z: got %q, want -0800", tm.Format(got))
		}
	})
}

func TestLDMLNonTokenLiterals(t *testing.T) {
	tm := refTime(t)
	// Hyphens, colons, slashes, spaces are preserved.
	goLayout, ok := ldmlToGoLayout("yyyy-MM-dd HH:mm:ss")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "2026-02-23 14:32:05" {
		t.Errorf("got %q, want 2026-02-23 14:32:05", got)
	}
}

func TestLDMLSingleQuoteEscaping(t *testing.T) {
	tm := refTime(t)

	// T literal inside single quotes.
	goLayout, ok := ldmlToGoLayout("yyyy-MM-dd'T'HH:mm:ssZZZZ")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	want := "2026-02-23T14:32:05-08:00"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLDMLDoubledSingleQuote(t *testing.T) {
	tm := refTime(t)
	// h 'o''clock' should produce layout "3 o'clock"
	goLayout, ok := ldmlToGoLayout("h 'o''clock'")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	// hour 14 in 12-hour format = 2
	want := "2 o'clock"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLDMLNoTokens(t *testing.T) {
	// Strings with no LDML tokens → returns false.
	// Note: single-char tokens (h, a, z, Z) only trigger LDML when they appear
	// at a valid token boundary (start of string, after separator, after another
	// token). Keywords like "Monday" contain 'a' but it is preceded by a letter
	// that was not matched as an LDML token, so no LDML is detected.
	cases := []string{
		"2006-01-02",
		"Monday",  // 'a' is at an invalid boundary (preceded by 'd')
		"MST",
		"January", // 'a' is at invalid boundaries
		"",
	}
	for _, s := range cases {
		got, ok := ldmlToGoLayout(s)
		if ok {
			t.Errorf("ldmlToGoLayout(%q): expected ok=false, got layout %q", s, got)
		}
		if got != s {
			t.Errorf("ldmlToGoLayout(%q): expected original string back, got %q", s, got)
		}
	}
}

func TestLDMLISO8601Expression(t *testing.T) {
	// Full iso8601 expression from the shipped format file.
	tm := refTime(t)
	goLayout, ok := ldmlToGoLayout("yyyy-MM-dd'T'HH:mm:ssZZZZ")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "2026-02-23T14:32:05-08:00" {
		t.Errorf("iso8601 expression: got %q, want 2026-02-23T14:32:05-08:00", got)
	}
}

func TestLDMLRFC2822Expression(t *testing.T) {
	// Full rfc2822 expression from the shipped format file.
	tm := refTime(t)
	goLayout, ok := ldmlToGoLayout("EEE, dd MMM yyyy HH:mm:ss Z")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "Mon, 23 Feb 2026 14:32:05 -0800" {
		t.Errorf("rfc2822 expression: got %q, want Mon, 23 Feb 2026 14:32:05 -0800", got)
	}
}

func TestLDML12HourFormat(t *testing.T) {
	tm := refTime(t)
	goLayout, ok := ldmlToGoLayout("h:mm a")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "2:32 PM" {
		t.Errorf("h:mm a: got %q, want 2:32 PM", got)
	}
}

func TestLDMLEUDate(t *testing.T) {
	tm := refTime(t)
	goLayout, ok := ldmlToGoLayout("dd/MM/yyyy")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "23/02/2026" {
		t.Errorf("dd/MM/yyyy: got %q, want 23/02/2026", got)
	}
}

func TestLDMLFullWeekdayMonth(t *testing.T) {
	tm := refTime(t)
	goLayout, ok := ldmlToGoLayout("EEEE, MMMM dd, yyyy")
	if !ok {
		t.Fatal("expected ok=true")
	}
	got := tm.Format(goLayout)
	if got != "Monday, February 23, 2026" {
		t.Errorf("EEEE, MMMM dd, yyyy: got %q, want Monday, February 23, 2026", got)
	}
}
