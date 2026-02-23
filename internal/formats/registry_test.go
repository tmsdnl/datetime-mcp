package formats

import (
	"testing"
)

func makeTestFormats() []Format {
	return []Format{
		{Name: "iso8601", Description: "RFC 3339", Template: "{yyyy-MM-dd'T'HH:mm:ssZZZZ}"},
		{Name: "rfc2822", Description: "Email date", Template: "{EEE, dd MMM yyyy HH:mm:ss Z}"},
		{Name: "short", Description: "Short date", Template: "{yyyy-MM-dd} {HH:mm:ss}"},
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry(makeTestFormats())
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry(makeTestFormats())

	tmpl, ok := r.Get("iso8601")
	if !ok {
		t.Fatal("expected iso8601 to be found")
	}
	if tmpl != "{yyyy-MM-dd'T'HH:mm:ssZZZZ}" {
		t.Errorf("unexpected template: %q", tmpl)
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent format to not be found")
	}
}

func TestRegistry_All(t *testing.T) {
	fmts := makeTestFormats()
	r := NewRegistry(fmts)

	all := r.All()
	if len(all) != len(fmts) {
		t.Errorf("All() returned %d formats, want %d", len(all), len(fmts))
	}
	// Verify order preserved.
	for i, want := range fmts {
		if all[i].Name != want.Name {
			t.Errorf("All()[%d].Name = %q, want %q", i, all[i].Name, want.Name)
		}
	}
}

func TestRegistry_Map(t *testing.T) {
	r := NewRegistry(makeTestFormats())
	m := r.Map()

	if len(m) != 3 {
		t.Errorf("Map() returned %d entries, want 3", len(m))
	}
	if m["iso8601"] != "{yyyy-MM-dd'T'HH:mm:ssZZZZ}" {
		t.Errorf("Map()[iso8601] = %q", m["iso8601"])
	}
	if m["rfc2822"] != "{EEE, dd MMM yyyy HH:mm:ss Z}" {
		t.Errorf("Map()[rfc2822] = %q", m["rfc2822"])
	}
}

func TestRegistry_IsEmpty(t *testing.T) {
	empty := NewRegistry(nil)
	if !empty.IsEmpty() {
		t.Error("expected empty registry to be empty")
	}

	r := NewRegistry(makeTestFormats())
	if r.IsEmpty() {
		t.Error("expected non-empty registry to not be empty")
	}
}

func TestRegistry_NilSafe(t *testing.T) {
	var r *Registry

	if _, ok := r.Get("iso8601"); ok {
		t.Error("nil registry Get should return false")
	}
	if r.All() != nil {
		t.Error("nil registry All should return nil")
	}
	if r.Map() != nil {
		t.Error("nil registry Map should return nil")
	}
	if !r.IsEmpty() {
		t.Error("nil registry IsEmpty should return true")
	}
}

func TestRegistry_DuplicateNames(t *testing.T) {
	// Last value wins for a given name, but order is based on first occurrence.
	fmts := []Format{
		{Name: "iso8601", Template: "first"},
		{Name: "rfc2822", Template: "second"},
		{Name: "iso8601", Template: "third"}, // duplicate
	}
	r := NewRegistry(fmts)

	tmpl, ok := r.Get("iso8601")
	if !ok {
		t.Fatal("iso8601 not found")
	}
	if tmpl != "third" {
		t.Errorf("expected last value to win in Get(), got %q", tmpl)
	}

	// Map() must agree with Get() for the same name (BUG-02).
	m := r.Map()
	if m["iso8601"] != "third" {
		t.Errorf("expected Map()[\"iso8601\"] = \"third\" (last value wins), got %q", m["iso8601"])
	}
	if m["iso8601"] != tmpl {
		t.Errorf("Map()[\"iso8601\"] = %q, Get(\"iso8601\") = %q: they must agree", m["iso8601"], tmpl)
	}

	all := r.All()
	if len(all) != 2 {
		t.Errorf("expected 2 unique formats, got %d", len(all))
	}
}

func TestRegistry_LoadIntegration(t *testing.T) {
	fmts, errs := Load("../../testdata/formats")
	if len(errs) != 0 {
		t.Fatalf("Load errors: %v", errs)
	}

	r := NewRegistry(fmts)

	// Check specific formats from testdata.
	if tmpl, ok := r.Get("iso8601"); !ok || tmpl == "" {
		t.Error("iso8601 not in registry")
	}
	if tmpl, ok := r.Get("rfc2822"); !ok || tmpl == "" {
		t.Error("rfc2822 not in registry")
	}
	if tmpl, ok := r.Get("default"); !ok || tmpl == "" {
		t.Error("default not in registry")
	}

	m := r.Map()
	if len(m) != len(r.All()) {
		t.Errorf("Map() and All() disagree on count: %d vs %d", len(m), len(r.All()))
	}
}
