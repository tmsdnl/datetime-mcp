package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func installCodexMCP(exePath string, dryRun bool) Result {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".codex")
	path := filepath.Join(dir, "config.toml")

	// Check directory exists (absent if Codex has not been configured).
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return Result{Target: "Codex MCP", Status: StatusNotFound, Path: path}
	}

	// Check if codex CLI is available.
	if _, err := exec.LookPath("codex"); err != nil {
		if dryRun {
			return Result{Target: "Codex MCP", Status: StatusAdded, Path: path, DryRun: true,
				Note: "codex mcp add datetime-mcp -- " + exePath + " --mcp"}
		}
		return Result{Target: "Codex MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("codex CLI not found in PATH — run: codex mcp add datetime-mcp -- %s --mcp", exePath)}
	}

	// Idempotency check via codex mcp get.
	if exec.Command("codex", "mcp", "get", "datetime-mcp").Run() == nil {
		return Result{Target: "Codex MCP", Status: StatusExisting, Path: path}
	}

	if dryRun {
		return Result{Target: "Codex MCP", Status: StatusAdded, Path: path, DryRun: true,
			Note: "codex mcp add datetime-mcp -- " + exePath + " --mcp"}
	}

	// Delegate to codex CLI.
	cmd := exec.Command("codex", "mcp", "add", "datetime-mcp", "--", exePath, "--mcp")
	if out, err := cmd.CombinedOutput(); err != nil {
		return Result{Target: "Codex MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("codex mcp add: %w: %s", err, out)}
	}
	return Result{Target: "Codex MCP", Status: StatusAdded, Path: path}
}
