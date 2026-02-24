package install

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Status describes the outcome of an install operation.
type Status int

const (
	StatusAdded    Status = iota // Successfully added (or would add in dry-run)
	StatusRemoved                // Successfully removed (or would remove in dry-run)
	StatusExisting               // Already present, no change
	StatusNotFound               // Parent directory / entry missing
	StatusError                  // Write failure
)

// Result holds the outcome of installing or removing one target.
type Result struct {
	Target  string
	Status  Status
	Path    string
	Err     error
	DryRun  bool
	Note    string // dry-run: command that would be run
	Snippet string // dry-run: config block that would be written
	Message string // override default status message
}

// Run is a deprecated alias for RunMCPAdd that accepts the legacy flag names
// (--claude-code-mcp, --codex-mcp). Kept for backwards compatibility.
func Run(args []string, exePath string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	claudeCodeHook := fs.Bool("claude-code-hook", false, "Register as Claude Code SessionStart hook (recommended)")
	claudeDesktop := fs.Bool("claude-desktop", false, "Register as Claude Desktop MCP server")
	claudeCodeMCP := fs.Bool("claude-code-mcp", false, "Register as Claude Code MCP server")
	codexMCP := fs.Bool("codex-mcp", false, "Register as Codex MCP server")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Register datetime-mcp with supported AI tool integrations.

Usage:
  datetime-mcp install [--claude-code-hook] [--claude-desktop] [--claude-code-mcp] [--codex-mcp] [--dry-run]

Flags:
  --claude-code-hook  Register as Claude Code SessionStart hook (recommended)
  --claude-desktop    Register as Claude Desktop MCP server
  --claude-code-mcp   Register as Claude Code MCP server (deprecated: use mcp add --claude-code)
  --codex-mcp         Register as Codex MCP server (deprecated: use mcp add --codex)
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	if !*claudeDesktop && !*claudeCodeMCP && !*claudeCodeHook && !*codexMCP {
		fs.Usage()
		os.Exit(1)
	}

	var results []Result
	if *claudeDesktop {
		results = append(results, installClaudeDesktop(exePath, *dryRun))
	}
	if *claudeCodeMCP {
		results = append(results, installClaudeCodeMCP(exePath, *dryRun))
	}
	if *claudeCodeHook {
		results = append(results, installClaudeCodeHook(exePath, *dryRun))
	}
	if *codexMCP {
		results = append(results, installCodexMCP(exePath, *dryRun))
	}

	printResults(results)
}

// RunMCPAdd parses args and registers datetime-mcp with the requested integrations.
func RunMCPAdd(args []string, exePath string) {
	fs := flag.NewFlagSet("mcp add", flag.ExitOnError)
	claudeCodeHook := fs.Bool("claude-code-hook", false, "Register as Claude Code SessionStart hook (recommended)")
	claudeDesktop := fs.Bool("claude-desktop", false, "Register as Claude Desktop MCP server")
	claudeCode := fs.Bool("claude-code", false, "Register as Claude Code MCP server")
	codex := fs.Bool("codex", false, "Register as Codex MCP server")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Register datetime-mcp with supported AI tool integrations.

Usage:
  datetime-mcp mcp add [--claude-code-hook] [--claude-desktop] [--claude-code] [--codex] [--dry-run]

Flags:
  --claude-code-hook  Register as Claude Code SessionStart hook (recommended)
  --claude-desktop    Register as Claude Desktop MCP server
  --claude-code       Register as Claude Code MCP server
  --codex             Register as Codex MCP server
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	if !*claudeDesktop && !*claudeCode && !*claudeCodeHook && !*codex {
		fs.Usage()
		os.Exit(1)
	}

	var results []Result
	if *claudeDesktop {
		results = append(results, installClaudeDesktop(exePath, *dryRun))
	}
	if *claudeCode {
		results = append(results, installClaudeCodeMCP(exePath, *dryRun))
	}
	if *claudeCodeHook {
		results = append(results, installClaudeCodeHook(exePath, *dryRun))
	}
	if *codex {
		results = append(results, installCodexMCP(exePath, *dryRun))
	}

	printResults(results)
}

// RunMCPRemove parses args and removes datetime-mcp from the requested integrations.
func RunMCPRemove(args []string, exePath string) {
	fs := flag.NewFlagSet("mcp remove", flag.ExitOnError)
	claudeCodeHook := fs.Bool("claude-code-hook", false, "Remove Claude Code SessionStart hook")
	claudeDesktop := fs.Bool("claude-desktop", false, "Remove Claude Desktop MCP server")
	claudeCode := fs.Bool("claude-code", false, "Remove Claude Code MCP server")
	codex := fs.Bool("codex", false, "Remove Codex MCP server")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Remove datetime-mcp from AI tool integrations.

Usage:
  datetime-mcp mcp remove [--claude-code-hook] [--claude-desktop] [--claude-code] [--codex] [--dry-run]

Flags:
  --claude-code-hook  Remove Claude Code SessionStart hook
  --claude-desktop    Remove Claude Desktop MCP server
  --claude-code       Remove Claude Code MCP server
  --codex             Remove Codex MCP server
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	if !*claudeDesktop && !*claudeCode && !*claudeCodeHook && !*codex {
		fs.Usage()
		os.Exit(1)
	}

	var results []Result
	if *claudeDesktop {
		results = append(results, removeClaudeDesktop(*dryRun))
	}
	if *claudeCode {
		results = append(results, removeClaudeCodeMCP(*dryRun))
	}
	if *claudeCodeHook {
		results = append(results, removeClaudeCodeHook(exePath, *dryRun))
	}
	if *codex {
		results = append(results, removeCodexMCP(*dryRun))
	}

	printResults(results)
}

func printResults(results []Result) {
	hasError := false
	for i, r := range results {
		if i > 0 {
			fmt.Println()
		}
		printResult(r)
		if r.Status == StatusError {
			hasError = true
		}
	}
	if hasError {
		os.Exit(1)
	}
}

func printResult(r Result) {
	switch r.Status {
	case StatusAdded:
		if r.DryRun {
			if r.Note != "" {
				fmt.Printf("%s: Run\n$ %s\n", r.Target, r.Note)
			} else {
				fmt.Printf("%s: Append to %s\n", r.Target, shortPath(r.Path))
				if r.Snippet != "" {
					fmt.Println()
					for _, line := range strings.Split(r.Snippet, "\n") {
						fmt.Println(line)
					}
				}
			}
		} else {
			fmt.Printf("%s: Added to %s\n", r.Target, shortPath(r.Path))
		}
	case StatusRemoved:
		if r.DryRun {
			if r.Note != "" {
				fmt.Printf("%s: Would remove — %s\n", r.Target, r.Note)
			} else {
				fmt.Printf("%s: Would remove from %s\n", r.Target, shortPath(r.Path))
			}
		} else {
			fmt.Printf("%s: Removed from %s\n", r.Target, shortPath(r.Path))
		}
	case StatusExisting:
		fmt.Printf("%s: Existing\n", r.Target)
	case StatusNotFound:
		if r.Message != "" {
			fmt.Printf("%s: Not found — %s\n", r.Target, r.Message)
		} else {
			fmt.Printf("%s: Not found — %s missing\n", r.Target, shortPath(filepath.Dir(r.Path)))
		}
	case StatusError:
		fmt.Printf("%s: Error — %v\n", r.Target, r.Err)
	}
}

func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// readJSONObject reads a JSON file as a map of raw messages (non-destructive).
// Returns an empty map if the file doesn't exist.
func readJSONObject(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return m, nil
}

// writeJSONObject writes a map as indented JSON with a trailing newline.
func writeJSONObject(path string, m map[string]json.RawMessage) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
