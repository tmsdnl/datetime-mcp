package formats

// Registry holds all loaded formats indexed by name.
type Registry struct {
	ordered []Format
	byName  map[string]Format
}

// NewRegistry creates a Registry from a slice of Format values.
// If multiple formats share the same name, the last one wins.
func NewRegistry(fmts []Format) *Registry {
	r := &Registry{
		ordered: make([]Format, 0, len(fmts)),
		byName:  make(map[string]Format, len(fmts)),
	}
	for _, f := range fmts {
		if _, exists := r.byName[f.Name]; !exists {
			r.ordered = append(r.ordered, f)
		}
		r.byName[f.Name] = f
	}
	return r
}

// Get returns the template string for the named format.
// The second return value reports whether the name was found.
func (r *Registry) Get(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	f, ok := r.byName[name]
	if !ok {
		return "", false
	}
	return f.Template, true
}

// All returns all loaded formats in load order.
func (r *Registry) All() []Format {
	if r == nil {
		return nil
	}
	return r.ordered
}

// Map returns a map of format name → template string, suitable for passing
// to template.New.
func (r *Registry) Map() map[string]string {
	if r == nil {
		return nil
	}
	m := make(map[string]string, len(r.ordered))
	for _, f := range r.ordered {
		m[f.Name] = f.Template
	}
	return m
}

// IsEmpty reports whether no formats are loaded.
func (r *Registry) IsEmpty() bool {
	if r == nil {
		return true
	}
	return len(r.ordered) == 0
}
