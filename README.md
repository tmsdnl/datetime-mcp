# datetime-mcp

A self-contained date/time provider for [Claude Desktop](https://claude.ai/download), [Claude Code](https://github.com/anthropics/claude-code), and [Codex](https://github.com/openai/codex).

Zero-dependency static binary. Prints the current date/time by default (hook mode); use `--mcp` to run as an MCP server.

## Installation

### Homebrew (recommended)

```sh
brew install tmsdnl/mcp/datetime-mcp
datetime-mcp format install   # download built-in format files
```

### Other

```sh
# Via Go:
go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest

# Or download the binary from Releases:
# https://github.com/tmsdnl/datetime-mcp/releases
```

Then install the built-in format files:

```sh
datetime-mcp format install
```

## Setup

`datetime-mcp mcp add` registers the binary with your AI tool:

```sh
datetime-mcp mcp add --claude-code         # MCP for Claude Code
datetime-mcp mcp add --claude-desktop      # MCP for Claude Desktop
datetime-mcp mcp add --codex               # MCP for Codex
datetime-mcp mcp add --claude-code-hook    # Claude Code hook (timestamp set once at session start)
```

Add `--dry-run` to preview changes before writing.

To unregister, use `datetime-mcp mcp remove` with the same flags.

## Usage

**With the hook:** datetime is injected at session start — the AI has it in context automatically, no prompting needed.

**With MCP:** ask the AI naturally, e.g. *"What time is it?"* or *"What's today's date?"* and it will call the `get_current_datetime` tool.

When run from a terminal, `datetime-mcp` prints and exits:

```sh
datetime-mcp                          # default output
datetime-mcp --format iso8601         # named format
datetime-mcp --format "{yyyy-MM-dd}"  # template
datetime-mcp --tz Europe/Vilnius      # timezone override
datetime-mcp --mcp                    # MCP server mode
```

See [docs/reference.md](docs/reference.md) for format files, template syntax, and manual configuration.

## Flags

```
--mcp               Run as MCP server (stdio JSON-RPC)
--tz string         Override timezone (IANA identifier)
--format string     Named format, {template}, or Go time layout
--formats-dir path  Format files directory (default: {XDG_CONFIG_HOME}/datetime-mcp/formats/)
--version           Print version
--help              Print usage
```

## License

MIT — see [LICENSE](LICENSE).
