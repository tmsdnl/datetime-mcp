package template

import (
	"fmt"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

func testTime(t *testing.T) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return time.Date(2026, 2, 23, 14, 32, 5, 0, loc)
}

func TestRender_BuiltinUnix(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	got := e.Render("{unix}", tm, nil)
	want := fmt.Sprintf("%d", tm.Unix())
	if got != want {
		t.Errorf("{unix}: got %q, want %q", got, want)
	}
}

func TestRender_BuiltinTimezone(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	got := e.Render("{timezone}", tm, nil)
	if got != "America/Los_Angeles" {
		t.Errorf("{timezone}: got %q, want America/Los_Angeles", got)
	}
}

func TestRender_NamedFormat(t *testing.T) {
	formats := map[string]string{
		"iso8601": "{yyyy-MM-dd'T'HH:mm:ssZZZZ}",
	}
	e := New(formats, nil)
	tm := testTime(t)
	got := e.Render("{iso8601}", tm, nil)
	want := "2026-02-23T14:32:05-08:00"
	if got != want {
		t.Errorf("{iso8601}: got %q, want %q", got, want)
	}
}

func TestRender_LDMLTokens(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)

	tests := []struct {
		tmpl string
		want string
	}{
		{"{yyyy-MM-dd}", "2026-02-23"},
		{"{HH:mm:ss}", "14:32:05"},
		{"{EEEE}", "Monday"},
		{"{EEE}", "Mon"},
		{"{MMMM}", "February"},
		{"{MMM}", "Feb"},
		{"{MM}", "02"},
		{"{dd}", "23"},
		{"{h:mm a}", "2:32 PM"},
		{"{z}", "PST"},
		{"{ZZZZ}", "-08:00"},
		{"{Z}", "-0800"},
	}
	for _, tc := range tests {
		t.Run(tc.tmpl, func(t *testing.T) {
			got := e.Render(tc.tmpl, tm, nil)
			if got != tc.want {
				t.Errorf("Render(%q) = %q, want %q", tc.tmpl, got, tc.want)
			}
		})
	}
}

func TestRender_GoLayoutFallback(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)

	tests := []struct {
		tmpl string
		want string
	}{
		{"{2006-01-02}", "2026-02-23"},
		{"{Monday}", "Monday"},
		{"{MST}", "PST"},
	}
	for _, tc := range tests {
		t.Run(tc.tmpl, func(t *testing.T) {
			got := e.Render(tc.tmpl, tm, nil)
			if got != tc.want {
				t.Errorf("Render(%q) = %q, want %q", tc.tmpl, got, tc.want)
			}
		})
	}
}

func TestRender_EscapedBraces(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	got := e.Render("Stamp: {{literal}}", tm, nil)
	if got != "Stamp: {literal}" {
		t.Errorf("got %q, want %q", got, "Stamp: {literal}")
	}
}

func TestRender_EscapedClosingBrace(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	// }} in isolation should produce a single literal }
	got := e.Render("text}}", tm, nil)
	want := "text}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_EscapedBracesMixed(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	// {{ produces {, B is a literal character, }} produces }
	// So A{{B}}C → A + { + B + } + C = "A{B}C"
	got := e.Render("A{{B}}C", tm, nil)
	want := "A{B}C"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRender_UnresolvablePlaceholder(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	// {unknown} is not a built-in, not in formats, not LDML, and not a Go layout
	// that produces a different result. Go's time.Format will pass "unknown"
	// through unchanged since it contains no layout tokens.
	got := e.Render("{unknown}", tm, nil)
	// "unknown" has no Go time layout tokens, so time.Format returns it unchanged.
	if got != "unknown" {
		t.Errorf("{unknown}: got %q, want %q", got, "unknown")
	}
}

func TestRender_MixedContent(t *testing.T) {
	formats := map[string]string{
		"iso8601": "{yyyy-MM-dd'T'HH:mm:ssZZZZ}",
	}
	e := New(formats, nil)
	tm := testTime(t)
	got := e.Render("Deploy: {iso8601} ({timezone})", tm, nil)
	want := "Deploy: 2026-02-23T14:32:05-08:00 (America/Los_Angeles)"
	if got != want {
		t.Errorf("mixed content: got %q, want %q", got, want)
	}
}

func TestRender_CircularReference(t *testing.T) {
	// Format A references B, B references A.
	formats := map[string]string{
		"a": "A:{b}",
		"b": "B:{a}",
	}
	var warnings []string
	e := New(formats, func(msg string) {
		warnings = append(warnings, msg)
	})
	tm := testTime(t)

	// Render template that starts with {a}.
	got := e.Render("{a}", tm, nil)

	// A renders A:{b} → b is not yet visited, so recurse into B.
	// B renders B:{a} → a IS visited, so leave {a} as-is.
	// Result: A:B:{a}
	want := "A:B:{a}"
	if got != want {
		t.Errorf("circular ref: got %q, want %q", got, want)
	}

	// A warning should have been logged.
	if len(warnings) == 0 {
		t.Error("expected at least one circular reference warning")
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "circular") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'circular' in warnings, got: %v", warnings)
	}
}

func TestRender_BuiltinKeywordWinsOverFormat(t *testing.T) {
	// A format named "unix" should NOT override the built-in {unix}.
	formats := map[string]string{
		"unix":     "SHOULD_NOT_APPEAR",
		"timezone": "SHOULD_NOT_APPEAR",
	}
	var warnings []string
	e := New(formats, func(msg string) {
		warnings = append(warnings, msg)
	})
	tm := testTime(t)

	// {unix} must return the timestamp, not "SHOULD_NOT_APPEAR".
	got := e.Render("{unix}", tm, nil)
	if got == "SHOULD_NOT_APPEAR" {
		t.Error("{unix} should use built-in, not the format file")
	}
	wantUnix := fmt.Sprintf("%d", tm.Unix())
	if got != wantUnix {
		t.Errorf("{unix}: got %q, want %q", got, wantUnix)
	}

	// {timezone} must return IANA string, not "SHOULD_NOT_APPEAR".
	got = e.Render("{timezone}", tm, nil)
	if got == "SHOULD_NOT_APPEAR" {
		t.Error("{timezone} should use built-in, not the format file")
	}
	if got != "America/Los_Angeles" {
		t.Errorf("{timezone}: got %q, want America/Los_Angeles", got)
	}

	// Warnings should have been emitted at engine creation.
	if len(warnings) < 2 {
		t.Errorf("expected at least 2 conflict warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestRender_LiteralText(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	got := e.Render("hello world", tm, nil)
	if got != "hello world" {
		t.Errorf("literal text: got %q, want %q", got, "hello world")
	}
}

func TestRender_DefaultFormatComposite(t *testing.T) {
	// Simulates the default.yaml template which references iso8601 and uses built-ins.
	formats := map[string]string{
		"iso8601": "{yyyy-MM-dd'T'HH:mm:ssZZZZ}",
	}
	e := New(formats, nil)
	tm := testTime(t)
	tmpl := "[CONTEXT] Current date/time: {EEEE}, {yyyy-MM-dd} {HH:mm:ss} {z} ({timezone}) | ISO: {iso8601}"
	got := e.Render(tmpl, tm, nil)
	want := "[CONTEXT] Current date/time: Monday, 2026-02-23 14:32:05 PST (America/Los_Angeles) | ISO: 2026-02-23T14:32:05-08:00"
	if got != want {
		t.Errorf("default template:\ngot  %q\nwant %q", got, want)
	}
}

func TestRender_UnixTimestamp(t *testing.T) {
	e := New(nil, nil)
	tm := testTime(t)
	got := e.Render("Build #{unix}", tm, nil)
	wantSuffix := fmt.Sprintf("Build #%d", tm.Unix())
	if got != wantSuffix {
		t.Errorf("got %q, want %q", got, wantSuffix)
	}
}
