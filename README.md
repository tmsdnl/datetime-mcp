# datetime-mcp

A self-contained date/time provider for [Claude Desktop](https://claude.ai/download) and [Claude Code](https://github.com/anthropics/claude-code).

Compiled as a **zero-dependency static binary** that automatically detects its context:

- **Hook mode** (stdin is a terminal): prints the current date/time and exits. Use as a Claude Code `SessionStart` hook to inject time context.
- **MCP server mode** (stdin is a pipe): speaks JSON-RPC 2.0 over stdio, exposing a `get_current_datetime` tool per [MCP 2025-11-25](https://spec.modelcontextprotocol.io).

## Installation

### Homebrew (recommended — installs binary + format files)

```sh
brew install tmsdnl/mcp/datetime-mcp
```

### Go install (binary only)

```sh
go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest
```

Format files must be downloaded separately. See [Format Files](#format-files).

### Manual download

Download the binary and format files from [Releases](https://github.com/tmsdnl/datetime-mcp/releases).

## Usage

### Hook mode (terminal)

```sh
# Default output (from default.yaml):
datetime-mcp
# [CONTEXT] Current date/time: Monday, 2026-02-23 14:32:05 PST (America/Los_Angeles) | ISO: 2026-02-23T14:32:05-08:00

# Named format:
datetime-mcp --format iso8601
# 2026-02-23T14:32:05-08:00

# LDML tokens:
datetime-mcp --format "{yyyy-MM-dd} {HH:mm:ss}"
# 2026-02-23 14:32:05

# Mixed template:
datetime-mcp --format "{iso8601} ({timezone})"
# 2026-02-23T14:32:05-08:00 (America/Los_Angeles)

# Built-in keywords:
datetime-mcp --format "{unix}"
# 1771973525
datetime-mcp --format "{timezone}"
# America/Los_Angeles

# Go layout string (bare, no braces):
datetime-mcp --format "2006-01-02"
# 2026-02-23

# With timezone override:
datetime-mcp --format rfc2822 --tz UTC
# Mon, 23 Feb 2026 22:32:05 +0000
```

### MCP server mode

```sh
# Force MCP mode (e.g. for testing):
datetime-mcp --mcp

# With default timezone:
datetime-mcp --mcp --tz Europe/Vilnius
```

### Flags

```
--mcp               Force MCP server mode (override TTY auto-detection)
--tz string         Override timezone (IANA tz database identifier)
                    Falls back to TZ env var, then system local timezone.
--format string     Output format for hook mode. Accepts a named format,
                    a template string with {placeholders}, or a Go time
                    layout string. Default: "default" format file.
--formats-dir path  Format files directory. Default: {XDG_CONFIG_HOME}/datetime-mcp/formats/
--log               Enable diagnostic logging to stderr
--version           Print version, commit hash, and build date
--help              Print usage with all loaded format names
```

## Format Files

Each `.yaml` file in the formats directory defines one format. The filename (without extension) is the format name.

**Default location:** `~/.config/datetime-mcp/formats/` (XDG-compliant)

### Shipped formats

| Name | Description | Example output |
|------|-------------|----------------|
| `default` | Default hook output with context prefix | `[CONTEXT] Current date/time: Monday, 2026-02-23 14:32:05 PST (America/Los_Angeles) \| ISO: 2026-02-23T14:32:05-08:00` |
| `iso8601` | RFC 3339 with timezone offset | `2026-02-23T14:32:05-08:00` |
| `rfc2822` | Email/HTTP date format | `Mon, 23 Feb 2026 14:32:05 -0800` |

### Creating custom formats

```yaml
# ~/.config/datetime-mcp/formats/short.yaml
description: "Date and time without timezone"
template: "{yyyy-MM-dd} {HH:mm:ss}"
```

```yaml
# ~/.config/datetime-mcp/formats/deploy-stamp.yaml
description: "Deploy log line"
template: "Deployed at {iso8601} ({timezone}) by CI"
```

## Template Syntax

Templates use `{placeholder}` expressions resolved in this priority order:

### 1. Built-in keywords

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{unix}` | Unix timestamp | `1771973525` |
| `{timezone}` | IANA tz identifier | `America/Los_Angeles` |

### 2. Named formats

```
{iso8601}   → references iso8601.yaml
{rfc2822}   → references rfc2822.yaml
```

### 3. LDML tokens (Unicode UTS #35)

| Token | Description | Example |
|-------|-------------|---------|
| `yyyy` / `yy` | Four/two-digit year | `2026` / `26` |
| `MMMM` / `MMM` / `MM` | Full/abbreviated/numeric month | `February` / `Feb` / `02` |
| `dd` | Zero-padded day | `23` |
| `EEEE` / `EEE` | Full/abbreviated weekday | `Monday` / `Mon` |
| `HH` / `h` | 24-hour / 12-hour hour | `14` / `2` |
| `mm` | Zero-padded minute | `32` |
| `ss` | Zero-padded second | `05` |
| `a` | AM/PM marker | `PM` |
| `z` | Timezone abbreviation | `PST` |
| `ZZZZ` / `Z` | UTC offset with/without colon | `-08:00` / `-0800` |

Single quotes escape literal text: `{yyyy-MM-dd'T'HH:mm:ss}` → `2026-02-23T14:32:05`

### 4. Go time layouts (fallback)

```
{2006-01-02}   → 2026-02-23
{Monday}       → Monday
{MST}          → PST
```

### Escaping

```
{{  →  literal {
}}  →  literal }
```

## Configuration Examples

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "datetime": {
      "command": "/usr/local/bin/datetime-mcp"
    }
  }
}
```

With timezone override:

```json
{
  "mcpServers": {
    "datetime": {
      "command": "/usr/local/bin/datetime-mcp",
      "args": ["--tz", "America/Los_Angeles"]
    }
  }
}
```

### Claude Code MCP

```sh
claude mcp add datetime /usr/local/bin/datetime-mcp
claude mcp add datetime /usr/local/bin/datetime-mcp -- --tz America/Los_Angeles
```

### Claude Code SessionStart Hook (`~/.claude/settings.json`)

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/usr/local/bin/datetime-mcp"
          }
        ]
      }
    ]
  }
}
```

## MCP Tool

**Tool:** `get_current_datetime`

**Parameters:**
- `timezone` (optional): IANA tz database identifier (e.g. `America/Los_Angeles`). Default: effective timezone.
- `format` (optional): Named format or Go time layout string. Default: `iso8601`.

**Response:**
```json
{
  "content": [{ "type": "text", "text": "2026-02-23T14:32:05-08:00" }],
  "structuredContent": {
    "datetime": "2026-02-23T14:32:05-08:00",
    "timezone": "America/Los_Angeles",
    "utc_offset": "-08:00",
    "unix": 1771973525
  },
  "isError": false
}
```

## Building

```sh
# Build for current platform:
make build

# Cross-compile all targets:
make all

# Run tests:
make test
```

## License

MIT — see [LICENSE](LICENSE).
