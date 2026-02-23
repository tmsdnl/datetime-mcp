package formats

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents a single loaded format file.
type Format struct {
	Name        string
	Description string
	Template    string
}

// validNameRe matches valid format names: lowercase alphanumeric, hyphens, underscores.
var validNameRe = regexp.MustCompile(`^[a-z0-9_-]+$`)

// reservedNames are built-in keywords that take precedence over format files.
var reservedNames = map[string]bool{"unix": true, "timezone": true}

// formatFile is the YAML structure of a format file.
type formatFile struct {
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// Load loads all .yaml / .yml format files from dir.
//
// Returns the loaded formats (in filesystem order) and a slice of non-fatal
// errors for individual files that were skipped. If dir does not exist, Load
// returns an empty slice and no error (F-064). If dir exists but individual
// files are malformed, those files are skipped and their errors are returned
// (F-053).
func Load(dir string) ([]Format, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // F-064: missing directory is not an error
		}
		return nil, []error{fmt.Errorf("reading formats dir %q: %w", dir, err)}
	}

	var formats []Format
	var errs []error

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		// Derive format name from filename (without extension, F-050).
		fmtName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")

		// Validate format name pattern (F-051).
		if !validNameRe.MatchString(fmtName) {
			errs = append(errs, fmt.Errorf("skipping %q: format name %q does not match [a-z0-9_-]+", name, fmtName))
			continue
		}

		// Warn if name conflicts with a built-in keyword (F-075).
		if reservedNames[fmtName] {
			_, _ = fmt.Fprintf(os.Stderr, "warning: format file %q has a name that conflicts with built-in keyword %q; built-in takes precedence\n", name, fmtName)
		}

		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("reading %q: %w", path, err))
			continue
		}

		var ff formatFile
		if err := yaml.Unmarshal(data, &ff); err != nil {
			errs = append(errs, fmt.Errorf("parsing %q: %w", path, err))
			continue
		}

		// Template field is required (F-052/F-053).
		if ff.Template == "" {
			errs = append(errs, fmt.Errorf("skipping %q: missing or empty 'template' field", path))
			continue
		}

		formats = append(formats, Format{
			Name:        fmtName,
			Description: ff.Description,
			Template:    ff.Template,
		})
	}

	return formats, errs
}
