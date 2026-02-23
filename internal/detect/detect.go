package detect

import "os"

// IsTerminal reports whether f refers to a terminal (TTY).
// Uses os.ModeCharDevice which works cross-platform without cgo.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
