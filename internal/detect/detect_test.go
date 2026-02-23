package detect

import (
	"os"
	"testing"
)

func TestIsTerminal_Pipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if IsTerminal(r) {
		t.Error("expected pipe read-end to not be a terminal")
	}
	if IsTerminal(w) {
		t.Error("expected pipe write-end to not be a terminal")
	}
}

func TestIsTerminal_RegularFile(t *testing.T) {
	f, err := os.CreateTemp("", "detect-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if IsTerminal(f) {
		t.Error("expected regular file to not be a terminal")
	}
}

func TestIsTerminal_Nil(t *testing.T) {
	if IsTerminal(nil) {
		t.Error("expected nil to not be a terminal")
	}
}

func TestIsTerminal_Stdin(t *testing.T) {
	// When running under go test, stdin is not a terminal.
	// This test just ensures the function doesn't panic.
	_ = IsTerminal(os.Stdin)
}
