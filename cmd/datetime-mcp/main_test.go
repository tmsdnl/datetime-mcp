package main_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
	_ "time/tzdata"
)

// binaryPath is the path to the compiled binary, built once in TestMain.
var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "datetime-mcp-integ-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	binaryPath = filepath.Join(dir, "datetime-mcp"+ext)

	// Build from the package directory (two levels up from this file).
	_, thisFile, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(thisFile)
	cmd := exec.Command("go", "build", "-o", binaryPath, pkgDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// run executes the binary with the given args and stdin, returning stdout and stderr.
func run(t *testing.T, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestVersion(t *testing.T) {
	stdout, _, code := run(t, "", "--version")
	if code != 0 {
		t.Fatalf("--version exited with %d", code)
	}
	// Output format: "<version> (<commit>) <date>"
	// With dev build: "dev (unknown) unknown"
	if !strings.Contains(stdout, "dev") && !regexp.MustCompile(`\d+\.\d+\.\d+`).MatchString(stdout) {
		t.Errorf("--version output unexpected: %q", stdout)
	}
}

func TestHelp_ContainsFlags(t *testing.T) {
	stdout, _, code := run(t, "", "--help")
	if code != 0 {
		t.Fatalf("--help exited with %d", code)
	}
	for _, want := range []string{"--mcp", "--tz", "--format", "--formats-dir", "--log", "--version", "--help"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--help missing flag %q", want)
		}
	}
}

func TestHelp_ContainsTemplateSyntax(t *testing.T) {
	stdout, _, _ := run(t, "", "--help")
	if !strings.Contains(stdout, "{unix}") {
		t.Errorf("--help missing template syntax reference: %q", stdout)
	}
	if !strings.Contains(stdout, "{timezone}") {
		t.Errorf("--help missing {timezone} reference: %q", stdout)
	}
}

func TestHelp_ListsFormats(t *testing.T) {
	// Use testdata formats directory.
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "formats")

	stdout, _, _ := run(t, "", "--help", "--formats-dir", testdataDir)
	// testdata has iso8601, rfc2822, default at minimum.
	for _, name := range []string{"iso8601", "rfc2822", "default"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("--help missing format name %q in output:\n%s", name, stdout)
		}
	}
	for _, desc := range []string{"RFC 3339 with timezone offset", "Email/HTTP date format (RFC 2822)", "Default hook output with context prefix"} {
		if !strings.Contains(stdout, desc) {
			t.Errorf("--help missing format description %q", desc)
		}
	}
}

func TestMCPMode_Initialize(t *testing.T) {
	// Pipe a single initialize message then EOF → expect initialize response.
	msg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	stdout, _, _ := run(t, msg, "--mcp")

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	if !scanner.Scan() {
		t.Fatal("no output from MCP server")
	}
	var resp map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v\n%s", err, scanner.Text())
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result in response: %v", resp)
	}
	if result["protocolVersion"] != "2025-11-25" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
}

func TestMCPMode_ToolCall(t *testing.T) {
	// Full lifecycle: initialize → initialized → tools/call → EOF.
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "formats")

	messages := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_current_datetime","arguments":{"format":"iso8601"}}}`,
	}, "\n") + "\n"

	stdout, _, code := run(t, messages, "--mcp", "--formats-dir", testdataDir)
	if code != 0 {
		t.Fatalf("binary exited with %d", code)
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	var responses []map[string]any
	for scanner.Scan() {
		var resp map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, scanner.Text())
		}
		responses = append(responses, resp)
	}

	// 2 responses: initialize + tools/call (initialized has no response).
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	// tools/call response.
	callResult := responses[1]["result"].(map[string]any)
	if callResult["isError"] == true {
		t.Errorf("tools/call returned isError=true: %v", callResult)
	}
	sc := callResult["structuredContent"].(map[string]any)
	if sc["datetime"] == nil || sc["timezone"] == nil || sc["utc_offset"] == nil || sc["unix"] == nil {
		t.Errorf("structuredContent missing fields: %v", sc)
	}

	// datetime must look like ISO 8601.
	dt := sc["datetime"].(string)
	matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`, dt)
	if !matched {
		t.Errorf("structuredContent.datetime %q doesn't match ISO 8601", dt)
	}
}

func TestMCPMode_PipeAutoDetect(t *testing.T) {
	// Without --mcp, piping stdin should auto-detect pipe mode and enter MCP mode.
	msg := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	stdout, _, code := run(t, msg) // no --mcp flag, but stdin is a pipe via cmd.Stdin
	if code != 0 {
		t.Fatalf("exited with %d", code)
	}
	// Should have ping response.
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	if !scanner.Scan() {
		t.Fatal("no response for ping")
	}
	var resp map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["result"] == nil {
		t.Errorf("ping response missing result: %v", resp)
	}
}

func TestMCPMode_SIGTERMShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM not available on Windows")
	}
	// Create a pipe for stdin — keep write end open so server doesn't see EOF.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	cmd := exec.Command(binaryPath, "--mcp")
	cmd.Stdin = r
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Give the server a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Send SIGTERM.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal: %v", err)
	}

	// Wait for exit with timeout.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				t.Errorf("process exited with non-zero code %d on SIGTERM", ee.ExitCode())
			} else {
				t.Errorf("wait error: %v", err)
			}
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("process did not exit within 5s after SIGTERM")
	}
}
