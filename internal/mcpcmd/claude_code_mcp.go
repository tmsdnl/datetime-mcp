package mcpcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func installClaudeCodeMCP(exePath string, dryRun bool) Result {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude.json")

	// Idempotency check via ~/.claude.json.
	top, err := readJSONObject(path)
	if err != nil {
		return Result{Target: "Claude Code MCP", Status: StatusError, Path: path, Err: err}
	}
	mcpServers := map[string]json.RawMessage{}
	if raw, ok := top["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &mcpServers); err != nil {
			return Result{Target: "Claude Code MCP", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing mcpServers: %w", err)}
		}
	}
	if _, ok := mcpServers["datetime"]; ok {
		return Result{Target: "Claude Code MCP", Status: StatusExisting, Path: path}
	}

	if dryRun {
		return Result{Target: "Claude Code MCP", Status: StatusAdded, Path: path, DryRun: true,
			Note: "claude mcp add --scope user datetime -- " + exePath + " --mcp"}
	}

	// Delegate to claude CLI.
	if _, err := exec.LookPath("claude"); err != nil {
		return Result{Target: "Claude Code MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("claude CLI not found in PATH — run: claude mcp add --scope user datetime -- %s --mcp", exePath)}
	}
	cmd := exec.Command("claude", "mcp", "add", "--scope", "user", "datetime", "--", exePath, "--mcp")
	if out, err := cmd.CombinedOutput(); err != nil {
		return Result{Target: "Claude Code MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("claude mcp add: %w: %s", err, out)}
	}
	return Result{Target: "Claude Code MCP", Status: StatusAdded, Path: path}
}

func removeClaudeCodeMCP(dryRun bool) Result {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude.json")

	if dryRun {
		return Result{Target: "Claude Code MCP", Status: StatusRemoved, Path: path, DryRun: true,
			Note: "claude mcp remove datetime"}
	}

	if _, err := exec.LookPath("claude"); err != nil {
		return Result{Target: "Claude Code MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("claude CLI not found in PATH — run: claude mcp remove datetime")}
	}
	cmd := exec.Command("claude", "mcp", "remove", "datetime")
	if out, err := cmd.CombinedOutput(); err != nil {
		return Result{Target: "Claude Code MCP", Status: StatusError, Path: path,
			Err: fmt.Errorf("claude mcp remove: %w: %s", err, out)}
	}
	return Result{Target: "Claude Code MCP", Status: StatusRemoved, Path: path}
}
