package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	_ "time/tzdata"

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
	if len(os.Args) > 1 {
		exe, _ := os.Executable()
		exe, _ = filepath.EvalSymlinks(exe)
		switch os.Args[1] {
		case "install":
			install.Run(os.Args[2:], exe)
			return
		case "mcp":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "usage: datetime-mcp mcp <add|remove> [flags]\n")
				os.Exit(1)
			}
			switch os.Args[2] {
			case "add":
				install.RunMCPAdd(os.Args[3:], exe)
			case "remove":
				install.RunMCPRemove(os.Args[3:], exe)
			default:
				fmt.Fprintf(os.Stderr, "unknown mcp subcommand %q — use add or remove\n", os.Args[2])
				os.Exit(1)
			}
			return
		}
	}

	var (
		mcpMode    = flag.Bool("mcp", false, "Run as MCP server (stdio JSON-RPC)")
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

	resolvedDir := resolveFormatsDir(*formatsDir)

	// Mode selection: --mcp flag → MCP server mode, otherwise hook mode.
	if *mcpMode {
		if err := mcp.Run(mcp.Config{
			Timezone:   *tz,
			FormatsDir: resolvedDir,
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
		FormatsDir: resolvedDir,
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

// resolveFormatsDir returns the directory to load formats from.
// Prefers the XDG path if it contains yaml files, then falls back to the
// share directory adjacent to the binary (Homebrew pkgshare layout).
func resolveFormatsDir(override string) string {
	if override != "" {
		return override
	}
	xdg := defaultFormatsDir()
	if hasYAML(xdg) {
		return xdg
	}
	// Homebrew installs formats to {prefix}/share/datetime-mcp/ alongside the
	// binary at {prefix}/bin/datetime-mcp.
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			dir := filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "share", "datetime-mcp"))
			if hasYAML(dir) {
				return dir
			}
		}
	}
	return xdg
}

func hasYAML(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".yaml") || strings.HasSuffix(n, ".yml") {
			return true
		}
	}
	return false
}

func printHelp(formatsDir string) {
	dir := formatsDir
	if dir == "" {
		dir = defaultFormatsDir()
	}

	fmt.Print(`A date/time provider for Claude Desktop, Claude Code, and Codex.
Prints the current date/time by default, or starts an MCP server with --mcp.

Usage:
  datetime-mcp [flags]
  datetime-mcp mcp add    [--claude-code-hook] [--claude-desktop] [--claude-code] [--codex] [--dry-run]
  datetime-mcp mcp remove [--claude-code-hook] [--claude-desktop] [--claude-code] [--codex] [--dry-run]
  datetime-mcp install    [--claude-code-hook] [--claude-desktop] [--claude-code-mcp] [--codex-mcp] [--dry-run]

Subcommands:
  mcp add             Register with AI tool integrations
  mcp remove          Remove from AI tool integrations
  install             Alias for mcp add (deprecated)

Flags:
  --mcp               Run as MCP server (stdio JSON-RPC)
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
