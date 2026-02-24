# datetime-mcp Reference

## Format Files

Each `.yaml` file in the formats directory defines one format. The filename (without extension) is the format name.

**Default location:** `~/.config/datetime-mcp/formats/` (XDG-compliant)

### Shipped formats

| Name | Description | Example output |
|------|-------------|----------------|
| `default` | Default output with context prefix | `[CONTEXT] Current date/time: Monday, 2026-02-23 14:32:05 PST (America/Los_Angeles) \| ISO: 2026-02-23T14:32:05-08:00` |
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

## Manual Configuration

`datetime-mcp mcp add --dry-run` shows the exact changes each flag would make. The snippets below are for reference if you prefer editing config files by hand.

### Claude Code hook (`~/.claude/settings.json`)

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          { "type": "command", "command": "/usr/local/bin/datetime-mcp" }
        ]
      }
    ]
  }
}
```

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "datetime": {
      "command": "/usr/local/bin/datetime-mcp",
      "args": ["--mcp"]
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
      "args": ["--mcp", "--tz", "America/Los_Angeles"]
    }
  }
}
```

### Claude Code MCP

```sh
claude mcp add --scope user datetime /usr/local/bin/datetime-mcp --mcp
```

### Codex (`~/.codex/config.toml`)

```toml
[mcp_servers.datetime-mcp]
command = "/usr/local/bin/datetime-mcp"
args = ["--mcp"]
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
make build   # build for current platform
make all     # cross-compile all targets
make test    # run tests
```
