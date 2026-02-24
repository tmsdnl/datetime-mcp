# datetime-mcp

A self-contained date/time provider for [Claude Desktop](https://claude.ai/download), [Claude Code](https://github.com/anthropics/claude-code), and [Codex](https://github.com/openai/codex).

Zero-dependency static binary. Prints the current date/time when run from a terminal; starts an MCP server when stdin is a pipe.

## Installation

### Homebrew (recommended)

```sh
brew install tmsdnl/mcp/datetime-mcp
```

### Other

```sh
# Via Go:
go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest

# Or download the binary from Releases:
# https://github.com/tmsdnl/datetime-mcp/releases
```

Then install the format files from the release archive:

```sh
mkdir -p ~/.config/datetime-mcp/formats
cp formats/*.yaml ~/.config/datetime-mcp/formats/
```

## Setup

`datetime-mcp install` registers the binary with your AI tool:

```sh
datetime-mcp install --claude-code-hook    # Claude Code hook (recommended)
datetime-mcp install --claude-desktop      # MCP for Claude Desktop
datetime-mcp install --claude-code-mcp     # MCP for Claude Code
datetime-mcp install --codex-mcp           # MCP for Codex
```

The hook option is recommended for Claude Code — it injects the current date/time at session start automatically, so the AI always has it without you having to ask.

Add `--dry-run` to preview changes before writing.

## Usage

**With the hook:** datetime is injected at session start — the AI has it in context automatically, no prompting needed.

**With MCP:** ask the AI naturally, e.g. *"What time is it?"* or *"What's today's date?"* and it will call the `get_current_datetime` tool.

When run from a terminal, `datetime-mcp` prints and exits:

```sh
datetime-mcp                          # default output
datetime-mcp --format iso8601         # named format
datetime-mcp --format "{yyyy-MM-dd}"  # template
datetime-mcp --tz Europe/Vilnius      # timezone override
datetime-mcp --mcp                    # force MCP server mode
```

See [docs/reference.md](docs/reference.md) for format files, template syntax, and manual configuration.

## Flags

```
--mcp               Force MCP server mode
--tz string         Override timezone (IANA identifier)
--format string     Named format, {template}, or Go time layout
--formats-dir path  Format files directory (default: {XDG_CONFIG_HOME}/datetime-mcp/formats/)
--version           Print version
--help              Print usage
```

## License

MIT — see [LICENSE](LICENSE).
