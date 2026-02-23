package hook

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func captureRun(t *testing.T, cfg Config) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runErr := Run(cfg)

	w.Close()
	os.Stdout = old

	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(buf.String())
}

func TestRunDefaultFormat(t *testing.T) {
	// Load from testdata, verify contains [CONTEXT] and date
	output := captureRun(t, Config{FormatsDir: "../../testdata/formats"})
	if !strings.Contains(output, "[CONTEXT]") {
		t.Errorf("default output missing [CONTEXT]: %q", output)
	}
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(output, today) {
		t.Errorf("default output missing today's date %s: %q", today, output)
	}
}

func TestRunNamedFormatISO8601(t *testing.T) {
	output := captureRun(t, Config{
		FormatsDir: "../../testdata/formats",
		Format:     "iso8601",
	})
	// ISO 8601: YYYY-MM-DDTHH:MM:SS±HH:MM
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`, output)
	if !matched {
		t.Errorf("iso8601 output %q doesn't match expected pattern", output)
	}
}

func TestRunInlineTemplate(t *testing.T) {
	output := captureRun(t, Config{
		FormatsDir: "../../testdata/formats",
		Format:     "{yyyy-MM-dd}",
	})
	today := time.Now().Format("2006-01-02")
	if output != today {
		t.Errorf("template output %q, want %q", output, today)
	}
}

func TestRunGoLayout(t *testing.T) {
	output := captureRun(t, Config{
		FormatsDir: "../../testdata/formats",
		Format:     "2006-01-02",
	})
	today := time.Now().Format("2006-01-02")
	if output != today {
		t.Errorf("go layout output %q, want %q", output, today)
	}
}

func TestRunNoFormats_ISO8601Fallback(t *testing.T) {
	output := captureRun(t, Config{FormatsDir: t.TempDir()})
	// Should produce ISO 8601 output
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, output)
	if !matched {
		t.Errorf("no-formats fallback output %q doesn't look like ISO 8601", output)
	}
}

func TestRunInvalidTimezone(t *testing.T) {
	// Should not panic, should produce output with UTC
	output := captureRun(t, Config{
		FormatsDir: "../../testdata/formats",
		Timezone:   "Mars/Olympus",
		Format:     "{timezone}",
	})
	if output != "UTC" {
		t.Errorf("invalid tz should fall back to UTC, got %q", output)
	}
}

func TestRunTimezoneOverride(t *testing.T) {
	output := captureRun(t, Config{
		FormatsDir: "../../testdata/formats",
		Timezone:   "UTC",
		Format:     "{timezone}",
	})
	if output != "UTC" {
		t.Errorf("timezone override: got %q, want UTC", output)
	}
}
