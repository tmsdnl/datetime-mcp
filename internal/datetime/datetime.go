package datetime

import (
	"fmt"
	"strings"
	"time"

	"github.com/tmsdnl/datetime-mcp/internal/formats"
	"github.com/tmsdnl/datetime-mcp/internal/template"
)

// ResolveTimezone resolves the effective timezone location.
//
// Precedence: tzFlag > tzEnv > system local timezone (F-103).
// If a non-empty timezone string is invalid, it returns time.UTC and an error
// so callers can surface a clear message (F-105).
func ResolveTimezone(tzFlag, tzEnv string) (*time.Location, error) {
	for _, tz := range []string{tzFlag, tzEnv} {
		if tz == "" {
			continue
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return time.UTC, fmt.Errorf(
				"invalid timezone: %q. Must be a valid IANA tz database identifier (e.g., America/Los_Angeles, Europe/Vilnius, UTC)",
				tz,
			)
		}
		return loc, nil
	}
	return time.Local, nil
}

// ISO8601Fallback formats t in RFC 3339 / ISO 8601 format.
// Used when no format files are loaded and no "default" format exists.
func ISO8601Fallback(t time.Time) string {
	return t.Format(time.RFC3339)
}

// StructuredContent is the structured output included in MCP tool responses (F-113).
type StructuredContent struct {
	Datetime  string `json:"datetime"`
	Timezone  string `json:"timezone"`
	UTCOffset string `json:"utc_offset"`
	Unix      int64  `json:"unix"`
}

// NewStructuredContent builds a StructuredContent for the given time and pre-formatted string.
func NewStructuredContent(t time.Time, formatted string) StructuredContent {
	_, offsetSecs := t.Zone()
	sign := "+"
	if offsetSecs < 0 {
		sign = "-"
		offsetSecs = -offsetSecs
	}
	h := offsetSecs / 3600
	m := (offsetSecs % 3600) / 60
	utcOffset := fmt.Sprintf("%s%02d:%02d", sign, h, m)

	return StructuredContent{
		Datetime:  formatted,
		Timezone:  t.Location().String(),
		UTCOffset: utcOffset,
		Unix:      t.Unix(),
	}
}

// Formatter formats datetime values using a format registry and template engine.
type Formatter struct {
	registry *formats.Registry
	engine   *template.Engine
}

// New creates a Formatter backed by the given registry and optional diagnostic logger.
func New(registry *formats.Registry, logger func(string)) *Formatter {
	var fmtMap map[string]string
	if registry != nil {
		fmtMap = registry.Map()
	}
	return &Formatter{
		registry: registry,
		engine:   template.New(fmtMap, logger),
	}
}

// FormatNamed renders a named format for the given time.
//
//   - If name is "" or "default", the "default" format is used when loaded;
//     otherwise ISO 8601 fallback is returned.
//   - If name is set to any other value and is not found in the registry,
//     an error is returned.
func (f *Formatter) FormatNamed(t time.Time, name string) (string, error) {
	if name == "" || name == "default" {
		if tmpl, ok := f.registry.Get("default"); ok {
			return f.engine.Render(tmpl, t, nil), nil
		}
		return ISO8601Fallback(t), nil
	}
	tmpl, ok := f.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown format: %q", name)
	}
	return f.engine.Render(tmpl, t, nil), nil
}

// FormatTemplate renders an inline template string (containing {placeholders})
// for the given time.
func (f *Formatter) FormatTemplate(t time.Time, tmpl string) string {
	return f.engine.Render(tmpl, t, nil)
}

// FormatAuto selects a formatting strategy based on formatStr (F-011 to F-014):
//
//  1. Empty string or "default"  → FormatNamed("default")
//  2. Name found in registry     → FormatNamed(name)
//  3. Contains '{'               → FormatTemplate (inline template)
//  4. Otherwise                  → bare Go time layout string
func (f *Formatter) FormatAuto(t time.Time, formatStr string) (string, error) {
	// Step 1: empty or "default".
	if formatStr == "" || formatStr == "default" {
		return f.FormatNamed(t, "default")
	}

	// Step 2: named format in registry.
	if _, ok := f.registry.Get(formatStr); ok {
		return f.FormatNamed(t, formatStr)
	}

	// Step 3: inline template with {placeholders}.
	if strings.Contains(formatStr, "{") {
		return f.FormatTemplate(t, formatStr), nil
	}

	// Step 4: bare Go time layout string.
	return t.Format(formatStr), nil
}
