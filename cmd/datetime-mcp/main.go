package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	_ "time/tzdata"

	"github.com/tmsdnl/datetime-mcp/internal/detect"
	"github.com/tmsdnl/datetime-mcp/internal/formats"
	"github.com/tmsdnl/datetime-mcp/internal/hook"
	"github.com/tmsdnl/datetime-mcp/internal/install"
	"github.com/tmsdnl/datetime-mcp/internal/mcp"
)

// Injected at build time via -ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "install" {
		exe, _ := os.Executable()
		exe, _ = filepath.EvalSymlinks(exe)
		install.Run(os.Args[2:], exe)
		return
	}

	var (
		forceMCP   = flag.Bool("mcp", false, "Force MCP server mode (override TTY auto-detection)")
		tz         = flag.String("tz", "", "Override timezone (IANA tz database identifier)")
		format     = flag.String("format", "", "Output format")
		formatsDir = flag.String("formats-dir", "", "Format files directory (default: {XDG_CONFIG_HOME}/datetime-mcp/formats/)")
		logFlag    = flag.Bool("log", false, "Enable diagnostic logging to stderr")
		showVer    = flag.Bool("version", false, "Print version, commit hash, and build date")
		showHelp   = flag.Bool("help", false, "Print this help message with all loaded formats")
	)

	flag.Usage = func() {
		printHelp("")
	}

	flag.Parse()

	if *showVer {
		fmt.Printf("%s (%s) %s\n", version, commit, date)
		os.Exit(0)
	}

	if *showHelp {
		printHelp(*formatsDir)
		os.Exit(0)
	}

	// Mode selection: --mcp flag or non-TTY stdin → MCP server mode.
	if *forceMCP || !detect.IsTerminal(os.Stdin) {
		if err := mcp.Run(mcp.Config{
			Timezone:   *tz,
			FormatsDir: *formatsDir,
			Log:        *logFlag,
			Version:    version,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Hook mode.
	if err := hook.Run(hook.Config{
		Format:     *format,
		Timezone:   *tz,
		FormatsDir: *formatsDir,
		Log:        *logFlag,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func defaultFormatsDir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "datetime-mcp", "formats")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "datetime-mcp", "formats")
}

func printHelp(formatsDir string) {
	dir := formatsDir
	if dir == "" {
		dir = defaultFormatsDir()
	}

	fmt.Print(`A date/time provider for Claude Desktop, Claude Code, and Codex.
Prints the current date/time when run from a terminal, or starts an MCP
server when stdin is a pipe.

Usage:
  datetime-mcp [flags]
  datetime-mcp install [--claude-code-hook] [--claude-desktop] [--claude-code-mcp] [--codex-mcp] [--dry-run]

Subcommands:
  install             Register with supported AI tool integrations

Flags:
  --mcp               Force MCP server mode (override TTY auto-detection)
  --tz string         Override timezone (IANA tz database identifier).
                      Falls back to TZ env var, then system local timezone.
                      Examples: America/Los_Angeles, Europe/Vilnius, UTC
  --format string     Output format. Accepts a named format,
                      a template string with {placeholders}, or a Go time
                      layout string. Default: "default" format file.
  --formats-dir path  Format files directory. Overrides the default XDG path.
`)
	fmt.Printf("                      Default: %s\n", dir)
	fmt.Print(`  --log               Enable diagnostic logging to stderr
  --version           Print version, commit hash, and build date
  --help              Print this help message with all loaded formats

Template Syntax:
  {unix}              Unix timestamp
  {timezone}          IANA tz identifier
  {iso8601}           Reference named format iso8601
  {yyyy-MM-dd}        LDML tokens: yyyy yy MMMM MMM MM dd EEEE EEE HH mm ss h a z Z ZZZZ
  2006-01-02          Go time layout (bare string, no braces)
  {{literal}}         Escaped brace: outputs {literal}

`)

	fmts, _ := formats.Load(dir)
	if len(fmts) == 0 {
		fmt.Printf("Loaded Formats (from %s):\n  (none)\n\n", dir)
		return
	}
	fmt.Printf("Loaded Formats (from %s):\n", dir)
	for _, f := range fmts {
		if f.Description != "" {
			fmt.Printf("  %-14s  %s\n", f.Name, f.Description)
		} else {
			fmt.Printf("  %s\n", f.Name)
		}
	}
	fmt.Println()
}
