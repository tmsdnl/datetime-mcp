package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func claudeDesktopConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Claude", "claude_desktop_config.json")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	default: // linux
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
}

func installClaudeDesktop(exePath string, dryRun bool) Result {
	path := claudeDesktopConfigPath()

	// Check parent directory exists (absent if Claude Desktop is not installed).
	if _, err := os.Stat(filepath.Dir(path)); errors.Is(err, os.ErrNotExist) {
		return Result{Target: "Claude Desktop", Status: StatusNotFound, Path: path}
	}

	top, err := readJSONObject(path)
	if err != nil {
		return Result{Target: "Claude Desktop", Status: StatusError, Path: path, Err: err}
	}

	// Parse or create mcpServers.
	mcpServers := map[string]json.RawMessage{}
	if raw, ok := top["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &mcpServers); err != nil {
			return Result{Target: "Claude Desktop", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing mcpServers: %w", err)}
		}
	}

	// Idempotency check.
	if _, ok := mcpServers["datetime"]; ok {
		return Result{Target: "Claude Desktop", Status: StatusExisting, Path: path}
	}

	if dryRun {
		b, _ := json.MarshalIndent(map[string]map[string]string{"datetime": {"command": exePath}}, "", "  ")
		return Result{Target: "Claude Desktop", Status: StatusAdded, Path: path, DryRun: true, Snippet: string(b)}
	}

	// Add the entry.
	entry, _ := json.Marshal(map[string]string{"command": exePath})
	mcpServers["datetime"] = json.RawMessage(entry)

	serialized, err := json.Marshal(mcpServers)
	if err != nil {
		return Result{Target: "Claude Desktop", Status: StatusError, Path: path, Err: err}
	}
	top["mcpServers"] = json.RawMessage(serialized)

	if err := writeJSONObject(path, top); err != nil {
		return Result{Target: "Claude Desktop", Status: StatusError, Path: path, Err: err}
	}
	return Result{Target: "Claude Desktop", Status: StatusAdded, Path: path}
}
