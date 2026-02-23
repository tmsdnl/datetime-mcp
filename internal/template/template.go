package template

import (
	"fmt"
	"time"
)

// Engine resolves {placeholder} expressions in templates.
//
// Placeholder resolution priority (F-070):
//  1. Built-in keywords: {unix}, {timezone}
//  2. Named format reference: exact match in formats map
//  3. LDML expression: contains recognized LDML tokens → translate and format
//  4. Go layout string: pass directly to time.Format()
//
// Unresolvable placeholders are left as-is (F-071).
// {{ and }} produce literal { and } (F-072).
type Engine struct {
	formats map[string]string // name → raw template string
	logger  func(string)      // diagnostic logger; nil = no-op
}

// New creates a new Engine with the given named formats map.
// logger is called with diagnostic/warning messages (nil = no-op).
func New(formats map[string]string, logger func(string)) *Engine {
	e := &Engine{
		formats: formats,
		logger:  logger,
	}
	// Warn if any format name conflicts with a built-in keyword (F-075).
	for _, kw := range []string{"unix", "timezone"} {
		if _, ok := formats[kw]; ok {
			e.log("warning: format file named %q conflicts with built-in keyword; built-in takes precedence", kw)
		}
	}
	return e
}

func (e *Engine) log(format string, args ...any) {
	if e.logger != nil {
		e.logger(fmt.Sprintf(format, args...))
	}
}

// Render resolves all {placeholder} expressions in tmpl for the given time t.
//
// visited tracks which named format references are currently being resolved
// (for circular reference detection). Pass nil for a top-level call; the
// engine will initialise the set. The same map is shared across all recursive
// calls within one top-level render so cycles are caught correctly (F-083/F-084).
func (e *Engine) Render(tmpl string, t time.Time, visited map[string]bool) string {
	if visited == nil {
		visited = make(map[string]bool)
	}

	var out []byte
	i := 0
	for i < len(tmpl) {
		switch {
		case tmpl[i] == '{' && i+1 < len(tmpl) && tmpl[i+1] == '{':
			// Escaped {{ → literal {
			out = append(out, '{')
			i += 2

		case tmpl[i] == '}' && i+1 < len(tmpl) && tmpl[i+1] == '}':
			// Escaped }} → literal }
			out = append(out, '}')
			i += 2

		case tmpl[i] == '{':
			// Find matching closing brace.
			j := i + 1
			for j < len(tmpl) && tmpl[j] != '}' {
				j++
			}
			if j >= len(tmpl) {
				// Unclosed brace — treat rest of string as literal.
				out = append(out, tmpl[i:]...)
				i = len(tmpl)
				break
			}
			placeholder := tmpl[i+1 : j]
			out = append(out, e.resolvePlaceholder(placeholder, t, visited)...)
			i = j + 1

		default:
			out = append(out, tmpl[i])
			i++
		}
	}
	return string(out)
}

// resolvePlaceholder resolves a single placeholder expression and returns
// the rendered string.
func (e *Engine) resolvePlaceholder(expr string, t time.Time, visited map[string]bool) string {
	// Priority 1: built-in keywords (F-073, F-074, F-075).
	switch expr {
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "timezone":
		return t.Location().String()
	}

	// Priority 2: named format reference (F-083/F-084).
	if tmpl, ok := e.formats[expr]; ok {
		if visited[expr] {
			// Circular reference detected — leave placeholder as-is and warn.
			e.log("warning: circular reference detected for format %q; leaving placeholder as-is", expr)
			return "{" + expr + "}"
		}
		visited[expr] = true
		result := e.Render(tmpl, t, visited)
		delete(visited, expr) // pop from stack after resolving
		return result
	}

	// Priority 3: LDML expression.
	goLayout, isLDML := ldmlToGoLayout(expr)
	if isLDML {
		return t.Format(goLayout)
	}

	// Priority 4: Go layout string (fallback).
	formatted := t.Format(expr)
	// If the format produced something that looks unchanged (not a layout),
	// we still return it — Go's time.Format passes unknown sequences through.
	// Only return original placeholder if the expression is empty.
	if expr == "" {
		return "{}"
	}
	return formatted
}
