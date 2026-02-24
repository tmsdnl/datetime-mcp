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
	StatusExisting               // Already present, no change
	StatusNotFound               // Parent directory missing
	StatusError                  // Write failure
)

// Result holds the outcome of installing one target.
type Result struct {
	Target  string
	Status  Status
	Path    string
	Err     error
	DryRun  bool
	Note    string // dry-run: command that would be run
	Snippet string // dry-run: config block that would be written
}

// Run parses args, runs installers for each requested target, prints results, and exits.
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
  --claude-code-mcp   Register as Claude Code MCP server
  --codex-mcp         Register as Codex MCP server
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
	case StatusExisting:
		fmt.Printf("%s: Existing\n", r.Target)
	case StatusNotFound:
		fmt.Printf("%s: Not found — %s missing\n", r.Target, shortPath(filepath.Dir(r.Path)))
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
