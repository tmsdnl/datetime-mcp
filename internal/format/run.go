package format

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Run dispatches to the appropriate format subcommand.
func Run(subcmd string, args []string, version string) {
	switch subcmd {
	case "install":
		runInstall(args, version)
	case "update":
		runUpdate(args, version)
	case "uninstall":
		runUninstall(args, version)
	default:
		fmt.Fprintf(os.Stderr, "unknown format subcommand %q — use install, update, or uninstall\n", subcmd)
		os.Exit(1)
	}
}

func runInstall(args []string, version string) {
	fs := flag.NewFlagSet("format install", flag.ExitOnError)
	dir := fs.String("formats-dir", "", "Target directory (default: XDG config dir)")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Download and install built-in format files to the XDG config dir.
Skips files that already exist.

Usage:
  datetime-mcp format install [--formats-dir path] [--dry-run]

Flags:
  --formats-dir path  Target directory (default: ~/.config/datetime-mcp/formats/)
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	printResults(Install(Config{Dir: *dir, Version: version, DryRun: *dryRun}))
}

func runUpdate(args []string, version string) {
	fs := flag.NewFlagSet("format update", flag.ExitOnError)
	dir := fs.String("formats-dir", "", "Target directory (default: XDG config dir)")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Re-download and overwrite managed format files.
User-created format files (not present in the upstream repo) are not touched.

Usage:
  datetime-mcp format update [--formats-dir path] [--dry-run]

Flags:
  --formats-dir path  Target directory (default: ~/.config/datetime-mcp/formats/)
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	printResults(Update(Config{Dir: *dir, Version: version, DryRun: *dryRun}))
}

func runUninstall(args []string, version string) {
	fs := flag.NewFlagSet("format uninstall", flag.ExitOnError)
	dir := fs.String("formats-dir", "", "Target directory (default: XDG config dir)")
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	fs.Usage = func() {
		fmt.Print(`Remove managed format files from the XDG config dir.
User-created format files (not present in the upstream repo) are not touched.

Usage:
  datetime-mcp format uninstall [--formats-dir path] [--dry-run]

Flags:
  --formats-dir path  Target directory (default: ~/.config/datetime-mcp/formats/)
  --dry-run           Preview changes without writing
`)
	}
	fs.Parse(args)

	printResults(Uninstall(Config{Dir: *dir, Version: version, DryRun: *dryRun}))
}

func printResults(results []Result) {
	hasError := false
	for _, r := range results {
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
	home, _ := os.UserHomeDir()
	path := r.Path
	if strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}

	switch r.Status {
	case StatusInstalled:
		if r.DryRun {
			fmt.Printf("%s: Would install to %s\n", r.Name, path)
		} else {
			fmt.Printf("%s: Installed to %s\n", r.Name, path)
		}
	case StatusUpdated:
		if r.DryRun {
			fmt.Printf("%s: Would update %s\n", r.Name, path)
		} else {
			fmt.Printf("%s: Updated %s\n", r.Name, path)
		}
	case StatusSkipped:
		fmt.Printf("%s: Skipped (already exists)\n", r.Name)
	case StatusRemoved:
		if r.DryRun {
			fmt.Printf("%s: Would remove %s\n", r.Name, path)
		} else {
			fmt.Printf("%s: Removed %s\n", r.Name, path)
		}
	case StatusNotFound:
		fmt.Printf("%s: Not found\n", r.Name)
	case StatusError:
		fmt.Printf("%s: Error — %v\n", r.Name, r.Err)
	}
}
