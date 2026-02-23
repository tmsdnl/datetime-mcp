package formats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testdataDir = "../../testdata/formats"

func TestLoad_ValidFiles(t *testing.T) {
	fmts, errs := Load(testdataDir)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(fmts) == 0 {
		t.Fatal("expected at least one format loaded")
	}

	// Check that the shipped formats are present.
	byName := make(map[string]Format)
	for _, f := range fmts {
		byName[f.Name] = f
	}

	for _, name := range []string{"iso8601", "rfc2822", "default", "short", "deploy-stamp", "weekday"} {
		f, ok := byName[name]
		if !ok {
			t.Errorf("expected format %q to be loaded", name)
			continue
		}
		if f.Template == "" {
			t.Errorf("format %q has empty template", name)
		}
	}
}

func TestLoad_NonExistentDir(t *testing.T) {
	fmts, errs := Load("/no/such/directory/datetime-mcp-test")
	if len(errs) != 0 {
		t.Errorf("expected no errors for missing dir, got: %v", errs)
	}
	if len(fmts) != 0 {
		t.Errorf("expected empty slice for missing dir, got %d formats", len(fmts))
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.yaml")
	if err := os.WriteFile(path, []byte(":\t:\t:bad yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	fmts, errs := Load(dir)
	if len(fmts) != 0 {
		t.Errorf("expected no formats from malformed file, got %d", len(fmts))
	}
	if len(errs) == 0 {
		t.Error("expected an error for malformed YAML")
	}
}

func TestLoad_MissingTemplateField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notemplate.yaml")
	if err := os.WriteFile(path, []byte("description: \"has no template field\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fmts, errs := Load(dir)
	if len(fmts) != 0 {
		t.Errorf("expected no formats from file missing template, got %d", len(fmts))
	}
	if len(errs) == 0 {
		t.Error("expected an error for missing template field")
	}
}

func TestLoad_InvalidFilename(t *testing.T) {
	dir := t.TempDir()
	// Uppercase letters not allowed in format names.
	path := filepath.Join(dir, "My-Format.yaml")
	if err := os.WriteFile(path, []byte("description: test\ntemplate: \"{yyyy}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fmts, errs := Load(dir)
	if len(fmts) != 0 {
		t.Errorf("expected no formats from invalid filename, got %d", len(fmts))
	}
	if len(errs) == 0 {
		t.Error("expected an error for invalid filename")
	}
}

func TestLoad_ReservedNames(t *testing.T) {
	dir := t.TempDir()
	// Files named unix.yaml and timezone.yaml should be loaded (with a warning)
	// since the template engine handles precedence, not the loader.
	for _, name := range []string{"unix", "timezone"} {
		path := filepath.Join(dir, name+".yaml")
		content := "description: \"should trigger warning\"\ntemplate: \"should-not-appear\"\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fmts, errs := Load(dir)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	// Both files should be loaded (precedence handled in template engine).
	if len(fmts) != 2 {
		t.Errorf("expected 2 formats loaded, got %d", len(fmts))
	}
	names := map[string]bool{}
	for _, f := range fmts {
		names[f.Name] = true
	}
	if !names["unix"] || !names["timezone"] {
		t.Errorf("expected both 'unix' and 'timezone' formats loaded; got: %v", names)
	}
}

func TestLoad_YMLExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myformat.yml")
	if err := os.WriteFile(path, []byte("description: test\ntemplate: \"{yyyy}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fmts, errs := Load(dir)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(fmts) != 1 || fmts[0].Name != "myformat" {
		t.Errorf("expected format 'myformat' from .yml file, got %v", fmts)
	}
}

func TestLoad_SkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"readme.md", "note.txt", "binary.bin"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not yaml"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fmts, errs := Load(dir)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(fmts) != 0 {
		t.Errorf("expected no formats from non-yaml files, got %d", len(fmts))
	}
}

func TestLoad_ErrorMessageContainsFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("template: "), 0o644); err != nil {
		t.Fatal(err)
	}
	// template: with no value means empty → missing template
	fmts, errs := Load(dir)
	if len(fmts) != 0 {
		t.Errorf("expected no formats, got %d", len(fmts))
	}
	if len(errs) == 0 {
		t.Fatal("expected at least one error")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "bad") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error message to reference filename, got: %v", errs)
	}
}
