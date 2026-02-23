package hook

import (
	"fmt"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/adrg/xdg"
	"github.com/tmsdnl/datetime-mcp/internal/datetime"
	"github.com/tmsdnl/datetime-mcp/internal/formats"
)

// Config holds hook mode configuration.
type Config struct {
	Format     string // named format, inline template, or Go layout
	Timezone   string // IANA tz identifier; empty = use TZ env or local
	FormatsDir string // override XDG formats directory
	Log        bool   // enable diagnostic logging to stderr
}

// Run executes hook mode: loads formats, resolves timezone, formats current
// time, prints to stdout, and exits with code 0 (F-010, F-016).
func Run(cfg Config) error {
	logger := makeLogger(cfg.Log)

	// Resolve formats directory.
	dir := cfg.FormatsDir
	if dir == "" {
		dir = xdgFormatsDir()
	}
	logger("formats dir: %s", dir)

	// Load format files.
	fmts, errs := formats.Load(dir)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "warning: %v\n", e)
	}

	// Detect empty/missing formats dir and warn (F-069).
	if len(fmts) == 0 {
		fmt.Fprintf(os.Stderr, "note: no format files found in %s\n"+
			"  For default output, download format files from:\n"+
			"  https://github.com/tmsdnl/datetime-mcp/tree/main/formats\n", dir)
	}

	reg := formats.NewRegistry(fmts)

	// Resolve timezone (flag > TZ env > local).
	tzEnv := os.Getenv("TZ")
	loc, err := datetime.ResolveTimezone(cfg.Timezone, tzEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; using UTC\n", err)
	}
	logger("effective timezone: %s", loc.String())

	t := time.Now().In(loc)

	var logFn func(string)
	if cfg.Log {
		logFn = func(msg string) {
			fmt.Fprintf(os.Stderr, "[datetime-mcp] %s\n", msg)
		}
	}
	f := datetime.New(reg, logFn)

	output, err := f.FormatAuto(t, cfg.Format)
	if err != nil {
		return fmt.Errorf("formatting: %w", err)
	}

	fmt.Println(output)
	return nil
}

func xdgFormatsDir() string {
	return xdg.ConfigHome + "/datetime-mcp/formats"
}

func makeLogger(enabled bool) func(string, ...any) {
	if !enabled {
		return func(string, ...any) {}
	}
	return func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "[datetime-mcp] "+format+"\n", args...)
	}
}
