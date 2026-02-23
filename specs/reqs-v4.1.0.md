# datetime-mcp: Requirements Specification

**Version:** 4.1.0
**Date:** 2026-02-23
**Author:** Tomas (with Claude)
**MCP Protocol Version:** 2025-11-25
**License:** MIT

---

## 1. Problem Statement

Claude Desktop injects the current date into its system prompt but omits the current time. Claude Code has had well-documented issues with both date and time awareness, often defaulting to training-data-era dates when performing web searches or generating time-sensitive content. Existing community solutions (MCP servers for datetime) all require external runtimes such as Node.js, Python, or Docker, adding unnecessary dependency overhead for what is fundamentally a trivial operation.

There is no single, self-contained tool that provides accurate date and time context to both Claude Desktop and Claude Code without requiring a runtime dependency.

## 2. Solution Overview

A single, statically compiled binary (`datetime-mcp`) that operates in two modes, detected automatically:

- **Hook mode** (stdin is a TTY): Prints the current date and time to stdout and exits immediately. Used by Claude Code's SessionStart hook and direct CLI invocation.
- **MCP server mode** (stdin is a pipe): Speaks JSON-RPC 2.0 over stdio, exposing a single `get_current_datetime` tool per the Model Context Protocol specification (version 2025-11-25). Used by Claude Desktop and Claude Code's MCP integration.

An explicit `--mcp` flag is available to force MCP server mode in environments where TTY detection may be unreliable (e.g., CI pipelines, unusual terminal configurations).

Date formats are configuration-driven using a one-file-per-format design. Each format is a single YAML file with a `template` field that defines its output. Templates can contain literal text, references to other named formats, inline format expressions using LDML tokens (e.g., `yyyy-MM-dd` per Unicode UTS #35) or Go time layouts (e.g., `2006-01-02`), and two built-in keywords (`{unix}` and `{timezone}`). All formats live on disk and are loaded at runtime from a configuration directory. The binary ships a set of default format files that are installed to the configuration directory.

The binary has zero runtime dependencies. It is distributed as a precompiled static binary for all major platforms via automated GitHub releases and Homebrew.

## 3. Functional Requirements

### 3.1 Mode Detection and Operation

| ID | Requirement |
|----|-------------|
| F-001 | The binary MUST auto-detect its operating mode based on whether stdin is a TTY. TTY = hook mode. Pipe = MCP server mode. |
| F-002 | The `--mcp` flag MUST force MCP server mode regardless of TTY detection. |
| F-003 | The `--version` flag MUST print version string, Git commit hash (short), and build date (ISO 8601), then exit. |
| F-004 | The `--help` flag MUST print usage information including all flags, all currently loaded format names with descriptions, template syntax reference, and timezone usage with examples. |
| F-005 | The `--log` flag MUST enable diagnostic logging to stderr. In MCP server mode, this aids debugging without polluting the stdio transport. In hook mode, this logs format loading and timezone resolution details. Default: off. |

### 3.2 Hook Mode

Hook mode activates when stdin is a TTY (and `--mcp` is not set).

| ID | Requirement |
|----|-------------|
| F-010 | In hook mode, the binary MUST print datetime information to stdout and exit with code 0. |
| F-011 | The default output (no `--format` flag) MUST be determined by the `default` format file. If no `default` format file is loaded, the binary MUST fall back to ISO 8601 output. |
| F-012 | If `--format` is set to a named format (matching a loaded format file), the output MUST be the rendered template from that format file. |
| F-013 | If `--format` is set to a template string containing `{` placeholders, the output MUST be the template with all placeholders resolved (see 3.5). |
| F-014 | If `--format` is set to a value that is neither a known named format nor contains `{` placeholder syntax, it MUST be treated as a Go time layout string and passed directly to `time.Format()`. |
| F-015 | No JSON-RPC initialization or stdin reading MUST occur in hook mode. |
| F-016 | The process MUST exit with code 0 after printing. |

### 3.3 MCP Server Mode

MCP server mode activates when stdin is a pipe, or when `--mcp` is explicitly set.

#### 3.3.1 Protocol Compliance (MCP 2025-11-25)

| ID | Requirement |
|----|-------------|
| F-020 | The server MUST implement the MCP protocol over stdio transport using JSON-RPC 2.0 message format, with messages delimited by newlines and no embedded newlines. |
| F-021 | The server MUST respond to `initialize` with its capabilities and server info, negotiating protocol version per the MCP lifecycle specification. The server MUST support protocol version `2025-11-25`. |
| F-022 | The server MUST NOT send requests (other than pings and logging) before receiving the `initialized` notification from the client. |
| F-023 | The server MUST declare the `tools` capability in its `initialize` response. The `listChanged` field SHOULD be `false` (the tool set is static for the lifetime of the server). |
| F-024 | The server MUST include `serverInfo` in the `initialize` response with `name` (`"datetime-mcp"`), `version` (semantic version), and `description` (`"Self-contained date/time provider for Claude Desktop and Claude Code"`). |
| F-025 | The server MUST respond to `tools/list` with the single tool definition (see 3.4). |
| F-026 | The server MUST respond to `tools/call` for the registered tool. |
| F-027 | The server MUST handle `ping` requests by responding with an empty result. |
| F-028 | The server MUST gracefully shut down on EOF from stdin or SIGTERM/SIGINT, per the MCP stdio transport shutdown specification. |
| F-029 | The server MUST return JSON-RPC error responses for unknown tool names (error code `-32602`). |
| F-030 | The server MUST return tool execution errors (invalid timezone, invalid format) as successful JSON-RPC responses with `isError: true` in the result, per MCP convention (SEP-1303), to enable model self-correction. |
| F-031 | When `--log` is enabled, the server MAY write UTF-8 diagnostic messages to stderr, per the MCP stdio transport specification. |

### 3.4 MCP Tool

A single tool MUST be exposed. The tool definition MUST include `name`, `title`, `description`, `inputSchema`, and `outputSchema` per the MCP 2025-11-25 tool specification.

**Tool: `get_current_datetime`**

| ID | Requirement |
|----|-------------|
| F-040 | MUST return the current date and time. |
| F-041 | MUST accept an optional `timezone` parameter (IANA tz database identifier, e.g., `America/Los_Angeles`). Default: effective timezone (see 3.6). |
| F-042 | MUST accept an optional `format` parameter. Values: any loaded named format or a Go time layout string. Default: `iso8601`. |
| F-043 | `title` MUST be `"Current Date and Time"`. |
| F-044 | `description` MUST clearly state that the format parameter accepts named formats and Go time layout strings, and that the timezone parameter accepts IANA tz database identifiers. |

### 3.5 Format System

#### 3.5.1 Format Files

Each format is defined in its own YAML file. The filename (without extension) is the format name. Every format file contains a `template` field that defines the format's output.

**Format file schema:**

```yaml
description: "Human-readable description of this format"
template: "the template string"
```

| ID | Requirement |
|----|-------------|
| F-050 | Each format MUST be defined in its own `.yaml` or `.yml` file. The filename (without extension) is the format name. |
| F-051 | The format name (derived from filename) MUST match the pattern `[a-z0-9_-]+` (lowercase alphanumeric, underscore, hyphen). Files not matching this pattern MUST be skipped with a warning logged to stderr. |
| F-052 | Each format file MUST contain a `template` field. `description` is optional. |
| F-053 | If a format file is malformed or missing the `template` field, the binary MUST log an error to stderr and skip it, continuing to load other files. |

#### 3.5.2 Format Loading

All formats are loaded from disk at runtime. There is no embedding.

| ID | Requirement |
|----|-------------|
| F-060 | The binary MUST load format files from the formats directory at startup. |
| F-061 | The default formats directory MUST follow XDG conventions using the `github.com/adrg/xdg` library for cross-platform path resolution. The path MUST be `{xdg.ConfigHome}/datetime-mcp/formats/`. |
| F-062 | A `--formats-dir` flag MUST allow overriding the formats directory. |
| F-063 | All `.yaml` and `.yml` files in the formats directory MUST be loaded. |
| F-064 | If the formats directory does not exist, the binary MUST proceed with no loaded formats (falling back to ISO 8601 for default output and MCP tool responses). |
| F-065 | The `--help` output MUST list all currently loaded format names with their descriptions. |

#### 3.5.3 Shipped Formats

The binary ships with a set of default format files. These are not embedded in the binary. They are distributed alongside the binary and installed to the formats directory.

| ID | Requirement |
|----|-------------|
| F-066 | The repository MUST include a `formats/` directory containing the shipped format files. |
| F-067 | The Homebrew formula and release archives MUST include the shipped format files. |
| F-068 | Installation methods (Homebrew, manual download) MUST place the shipped format files into the XDG formats directory, or document how to do so. |
| F-069 | `go install` does not support installing data files. The binary MUST detect on first run if the formats directory is empty or missing, and print a message directing the user to download the format files from the repository. |

The following format files MUST be shipped:

**`iso8601.yaml`**
```yaml
description: "RFC 3339 with timezone offset"
template: "{yyyy-MM-dd'T'HH:mm:ssZZZZ}"
```

**`rfc2822.yaml`**
```yaml
description: "Email/HTTP date format (RFC 2822)"
template: "{EEE, dd MMM yyyy HH:mm:ss Z}"
```

**`default.yaml`**
```yaml
description: "Default hook output with context prefix"
template: "[CONTEXT] Current date/time: {EEEE}, {yyyy-MM-dd} {HH:mm:ss} {z} ({timezone}) | ISO: {iso8601}"
```

#### 3.5.4 Template Resolution

The `template` field in a format file is a string that is rendered by the template engine. The engine resolves `{placeholder}` expressions in the following priority order:

| Priority | Type | Example | Description |
|----------|------|---------|-------------|
| 1 | Built-in keyword | `{unix}`, `{timezone}` | Values that cannot be expressed through any format string system. |
| 2 | Named format | `{iso8601}`, `{rfc2822}` | Exact match against a loaded format file name. Renders that format's template. |
| 3 | LDML format expression | `{yyyy-MM-dd}`, `{HH:mm:ss}` | String containing recognized LDML date field symbols (per Unicode UTS #35). Translated to a Go layout and rendered via `time.Format()`. |
| 4 | Go layout string | `{2006-01-02}`, `{MST}`, `{Monday}` | Fallback. Passed directly to `time.Format()`. |

If a placeholder does not produce output through any of these steps, it is left as-is in the output.

| ID | Requirement |
|----|-------------|
| F-070 | The template engine MUST resolve placeholders in the priority order defined above. |
| F-071 | Unresolvable placeholders MUST be left as-is in the output (no error, no substitution). |
| F-072 | Literal `{` and `}` MUST be escaped as `{{` and `}}`. |

#### 3.5.5 Built-in Keywords

Two keywords are built into the template engine. These represent values that cannot be produced by Go's `time.Format()`, LDML tokens, or any standard datetime format system.

| Keyword | Value | Example | Rationale |
|---------|-------|---------|-----------|
| `{unix}` | Unix timestamp as integer string | `1771973525` | No standard format token exists for Unix timestamps. Requires `t.Unix()`. |
| `{timezone}` | IANA tz database identifier | `America/Los_Angeles` | Go's `time.Format()` has no layout for the IANA identifier. Requires `t.Location().String()`. LDML defines `VV` for this purpose, but Go does not implement it. |

| ID | Requirement |
|----|-------------|
| F-073 | The template engine MUST recognize `{unix}` and render it as the integer Unix timestamp (seconds since 1970-01-01T00:00:00Z) as a decimal string. |
| F-074 | The template engine MUST recognize `{timezone}` and render it as the IANA tz database identifier of the effective timezone (e.g., `America/Los_Angeles`, `Europe/Vilnius`, `UTC`). |
| F-075 | Built-in keywords MUST take precedence over named format files. If a user creates a format file named `unix.yaml` or `timezone.yaml`, the built-in keyword MUST win and a warning MUST be logged to stderr. |

#### 3.5.6 LDML Format Tokens

The template engine supports a subset of LDML date field symbols (Unicode Technical Standard #35) within inline format expressions. When recognized LDML tokens are present, they are translated to Go time layout equivalents.

| LDML Token | Go Equivalent | Example Output | Description |
|------------|---------------|----------------|-------------|
| `yyyy` | `2006` | `2026` | Four-digit year |
| `yy` | `06` | `26` | Two-digit year |
| `MMMM` | `January` | `February` | Full month name |
| `MMM` | `Jan` | `Feb` | Abbreviated month name |
| `MM` | `01` | `02` | Zero-padded month (01-12) |
| `dd` | `02` | `23` | Zero-padded day (01-31) |
| `EEEE` | `Monday` | `Monday` | Full weekday name |
| `EEE` | `Mon` | `Mon` | Abbreviated weekday name |
| `HH` | `15` | `14` | 24-hour hour (00-23) |
| `h` | `3` | `2` | 12-hour hour (no leading zero) |
| `mm` | `04` | `32` | Zero-padded minute (00-59) |
| `ss` | `05` | `05` | Zero-padded second (00-59) |
| `a` | `PM` | `PM` | AM/PM marker |
| `z` | `MST` | `PST` | Timezone abbreviation (short specific non-location) |
| `ZZZZ` | `-07:00` | `-08:00` | UTC offset with colon (long localized GMT) |
| `Z` | `-0700` | `-0800` | UTC offset without colon (ISO 8601 basic) |

**Quoting literal text within LDML expressions:** Single quotes (`'`) within a `{}` expression escape literal text, per LDML conventions. For example, `{yyyy-MM-dd'T'HH:mm:ss}` treats the `T` as a literal character, not a token.

| ID | Requirement |
|----|-------------|
| F-076 | The template engine MUST detect whether an inline format expression contains any recognized LDML token. If at least one token is found, the entire expression MUST be treated as an LDML format string. |
| F-077 | LDML tokens MUST be replaced with their Go layout equivalents, and the resulting string MUST be passed to `time.Format()`. |
| F-078 | Token detection MUST be case-sensitive. `MM` (month) and `mm` (minute) are distinct. `HH` (24-hour) and `h` (12-hour) are distinct. |
| F-079 | Tokens MUST be replaced longest-first to avoid partial matches (e.g., `MMMM` before `MMM` before `MM`, `EEEE` before `EEE`, `yyyy` before `yy`, `ZZZZ` before `Z`). |
| F-080 | If no LDML tokens are detected in an inline expression, it MUST be treated as a Go time layout string (priority 4). |
| F-081 | Non-token characters within an LDML format expression (e.g., `-`, `/`, `:`, spaces) MUST be preserved as-is. |
| F-082 | Text enclosed in single quotes within an LDML expression MUST be treated as literal text and passed through to the Go layout unchanged. The quotes themselves MUST be stripped. A doubled single quote (`''`) MUST produce a literal single quote. |

#### 3.5.7 Circular Reference Handling

Format templates can reference other named formats. Circular references must be handled gracefully.

| ID | Requirement |
|----|-------------|
| F-083 | The template engine MUST maintain a visited set during placeholder resolution for each top-level render call. |
| F-084 | If a named format placeholder is encountered that is already in the visited set, the engine MUST leave the placeholder as-is in the output and log a warning to stderr (when `--log` is enabled). |
| F-085 | The visited set MUST be scoped to a single render call and not persist between calls. |

#### 3.5.8 Template Examples

**Using the shipped format files (`iso8601.yaml`, `rfc2822.yaml`, `default.yaml`):**

| Input | Output |
|-------|--------|
| `datetime-mcp` | `[CONTEXT] Current date/time: Monday, 2026-02-23 14:32:05 PST (America/Los_Angeles) \| ISO: 2026-02-23T14:32:05-08:00` |
| `--format iso8601` | `2026-02-23T14:32:05-08:00` |
| `--format rfc2822` | `Mon, 23 Feb 2026 14:32:05 -0800` |
| `--format "{iso8601}"` | `2026-02-23T14:32:05-08:00` |

**Using LDML format tokens:**

| Input | Output |
|-------|--------|
| `--format "{yyyy-MM-dd}"` | `2026-02-23` |
| `--format "{HH:mm:ss}"` | `14:32:05` |
| `--format "{dd/MM/yyyy}"` | `23/02/2026` |
| `--format "{EEEE, MMMM dd, yyyy}"` | `Monday, February 23, 2026` |
| `--format "{h:mm a}"` | `2:32 PM` |
| `--format "{yyyy-MM-dd'T'HH:mm:ssZZZZ}"` | `2026-02-23T14:32:05-08:00` |
| `--format "{EEE, dd MMM yyyy HH:mm:ss Z}"` | `Mon, 23 Feb 2026 14:32:05 -0800` |
| `--format "{z}"` | `PST` |
| `--format "{ZZZZ}"` | `-08:00` |

**Using Go time layouts (fallback):**

| Input | Output |
|-------|--------|
| `--format "{2006-01-02}"` | `2026-02-23` |
| `--format "{Monday}"` | `Monday` |
| `--format "{MST}"` | `PST` |
| `--format "2006-01-02"` | `2026-02-23` _(bare string, no braces, Go layout fallback)_ |

**Using built-in keywords:**

| Input | Output |
|-------|--------|
| `--format "{unix}"` | `1771973525` |
| `--format "{timezone}"` | `America/Los_Angeles` |
| `--format "Build #{unix}"` | `Build #1771973525` |

**Mixed examples:**

| Input | Output |
|-------|--------|
| `--format "Deploy: {iso8601} ({timezone})"` | `Deploy: 2026-02-23T14:32:05-08:00 (America/Los_Angeles)` |
| `--format "{yyyy-MM-dd} {HH:mm:ss} {timezone}"` | `2026-02-23 14:32:05 America/Los_Angeles` |
| `--format "Stamp: {{literal}}"` | `Stamp: {literal}` |

**User-created format files:**

```yaml
# weekday.yaml — user creates this if they want {weekday}
description: "Full weekday name"
template: "{EEEE}"
```

```yaml
# tz_abbr.yaml — user creates this if they want {tz_abbr}
description: "Timezone abbreviation"
template: "{z}"
```

```yaml
# eu_date.yaml
description: "European date format dd/MM/yyyy"
template: "{dd/MM/yyyy}"
```

```yaml
# deploy-stamp.yaml — referencing other formats
description: "Deploy log line"
template: "Deployed at {iso8601} ({timezone}) by CI"
```

### 3.6 Timezone Handling

| ID | Requirement |
|----|-------------|
| F-100 | The binary MUST read the current time from the system clock via `time.Now()` (UTC internally). |
| F-101 | The binary MUST convert the system time to the effective timezone before formatting. This is a full timezone conversion -- the displayed hours, minutes, and date MUST reflect the target timezone. |
| F-102 | A `--tz` flag MUST override the default timezone for both modes. Value: IANA tz database identifier (e.g., `America/Los_Angeles`). |
| F-103 | Timezone precedence MUST be: `--tz` flag > `TZ` environment variable > system local timezone. |
| F-104 | In MCP tool calls, the `timezone` parameter MUST override the effective default for that single call. |
| F-105 | If an invalid timezone string is provided (via flag, env var, or tool parameter), the binary MUST return a clear error message and fall back to UTC. |

### 3.7 MCP Tool Response Format

| ID | Requirement |
|----|-------------|
| F-110 | Tool responses MUST return unstructured content as a text content block: `{ "type": "text", "text": "..." }`. |
| F-111 | The text content MUST contain the fully rendered format output. |
| F-112 | Tool responses MUST also return `structuredContent` alongside the text content block, per MCP 2025-11-25. |
| F-113 | The `structuredContent` object MUST include: `datetime` (the formatted string matching the text content), `timezone` (IANA tz database identifier used), `utc_offset` (UTC offset string in `[+-]HH:MM` format), and `unix` (integer Unix timestamp). |
| F-114 | The tool MUST declare an `outputSchema` in its definition that describes the `structuredContent` shape. |
| F-115 | Tool error responses (invalid timezone, invalid format) MUST use `isError: true` with a descriptive error message in the text content block. |

**Example tool response:**

```json
{
  "content": [
    { "type": "text", "text": "2026-02-23T14:32:05-08:00" }
  ],
  "structuredContent": {
    "datetime": "2026-02-23T14:32:05-08:00",
    "timezone": "America/Los_Angeles",
    "utc_offset": "-08:00",
    "unix": 1771973525
  },
  "isError": false
}
```

**Example error response:**

```json
{
  "content": [
    { "type": "text", "text": "Invalid timezone: 'Mars/Olympus'. Must be a valid IANA tz database identifier (e.g., America/Los_Angeles, Europe/Vilnius, UTC)." }
  ],
  "isError": true
}
```

## 4. Non-Functional Requirements

### 4.1 Build and Distribution

| ID | Requirement |
|----|-------------|
| NF-001 | The binary MUST be written in Go. |
| NF-002 | The binary MUST compile as a statically linked executable with `CGO_ENABLED=0`. |
| NF-003 | The binary MUST have zero runtime dependencies. |
| NF-004 | The binary MUST use `github.com/adrg/xdg` for cross-platform XDG Base Directory resolution. |
| NF-005 | The `--version` output MUST include: semantic version, Git commit hash (short), and build date (ISO 8601). Injected via `-ldflags` at build time. |
| NF-006 | A Makefile MUST support cross-compilation for: macOS (arm64, amd64), Linux (arm64, amd64), Windows (arm64, amd64). |
| NF-007 | A `.goreleaser.yml` MUST be included for automated release builds. |
| NF-008 | Goreleaser MUST produce archives named `datetime-mcp_{version}_{os}_{arch}.{ext}` (`.tar.gz` for macOS/Linux, `.zip` for Windows). |
| NF-009 | Goreleaser MUST generate a SHA256 checksums file. |
| NF-010 | Goreleaser MUST include a Homebrew tap formula targeting the `tmsdnl/mcp` tap repository. |
| NF-011 | Release archives MUST include the shipped format files alongside the binary. |
| NF-012 | The Homebrew formula MUST install format files to the XDG formats directory. |
| NF-013 | The project MUST be installable via `go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest`. Note: `go install` only installs the binary; format files must be obtained separately. |
| NF-014 | The project MUST be licensed under MIT. A `LICENSE` file MUST be included in the repository root. |

### 4.2 Automated Releases

| ID | Requirement |
|----|-------------|
| NF-020 | A GitHub Actions workflow (`.github/workflows/release.yml`) MUST trigger on version tag push (pattern: `v*`). |
| NF-021 | The workflow MUST run `go test ./...` before building to prevent releasing broken code. |
| NF-022 | The workflow MUST run Goreleaser to build binaries for all targets, publish a GitHub release with attached assets, and update the Homebrew tap formula. |
| NF-023 | The workflow MUST use the free/open-source edition of Goreleaser. |
| NF-024 | A GitHub Actions workflow (`.github/workflows/test.yml`) MUST run `go test ./...` on push and pull request to the main branch. |

### 4.3 Performance

| ID | Requirement |
|----|-------------|
| NF-030 | Hook mode MUST complete (print and exit) in under 50ms on commodity hardware, including format loading from disk. |
| NF-031 | MCP server mode MUST respond to `tools/call` requests in under 10ms (wall clock, excluding I/O). |
| NF-032 | The compiled binary size SHOULD be under 10MB. |

### 4.4 Reliability

| ID | Requirement |
|----|-------------|
| NF-040 | The MCP server MUST NOT crash on malformed JSON-RPC input. It MUST return a JSON-RPC error response. |
| NF-041 | The MCP server MUST NOT leak goroutines on shutdown. |
| NF-042 | The binary MUST handle SIGTERM and SIGINT gracefully in MCP server mode. |
| NF-043 | Malformed or missing format files MUST NOT prevent the binary from starting. Errors MUST be logged to stderr. |

### 4.5 Testing

| ID | Requirement |
|----|-------------|
| NF-050 | Unit tests MUST cover all shipped format file outputs. |
| NF-051 | Unit tests MUST cover timezone conversion including cross-day boundary scenarios (e.g., UTC to UTC+13 where the date changes). |
| NF-052 | Unit tests MUST cover invalid timezone input and fallback behavior. |
| NF-053 | Unit tests MUST cover hook mode output for: default format, named format, template string with placeholders, LDML format tokens, and Go layout string. |
| NF-054 | Unit tests MUST cover template placeholder resolution including: built-in keywords (`{unix}`, `{timezone}`), named format references, LDML token formats, Go layout fallback, escaped braces, unresolvable placeholders left as-is, and mixed content. |
| NF-055 | Unit tests MUST cover LDML token parsing: case sensitivity (`MM` vs `mm`, `HH` vs `h`), longest-first matching (`MMMM` before `MMM` before `MM`, `EEEE` before `EEE`, `ZZZZ` before `Z`), non-token literal preservation, single-quote escaping, and all defined tokens. |
| NF-056 | Unit tests MUST cover auto-detection logic (TTY vs. pipe). |
| NF-057 | Unit tests MUST cover format loading: valid files, malformed files, missing `template` field, invalid filenames, and duplicate format names. |
| NF-058 | Unit tests MUST cover circular reference detection and graceful handling. |
| NF-059 | Unit tests MUST verify that built-in keywords take precedence over format files with conflicting names. |
| NF-060 | Integration tests MUST verify JSON-RPC message exchange for the full MCP lifecycle: `initialize` (with protocol version negotiation), `initialized` notification, `tools/list`, `tools/call` (success and error cases), and shutdown via stdin EOF. |
| NF-061 | Integration tests MUST verify `structuredContent` is present and valid alongside text content in tool responses. |
| NF-062 | Tests MUST be runnable via `go test ./...` with no external dependencies. Tests that require shipped format files MUST include test fixture copies in a `testdata/` directory. |

## 5. CLI Reference

```
datetime-mcp [flags]

A self-contained date/time provider for Claude Desktop and Claude Code.
Automatically detects mode: prints datetime and exits when run from a
terminal, or starts an MCP server when stdin is a pipe.

Flags:
  --mcp               Force MCP server mode (override TTY auto-detection)
  --tz string         Override timezone (IANA tz database identifier). Falls
                      back to TZ env var, then system local timezone.
                      Converts the system clock time to the specified timezone.
                      Examples: America/Los_Angeles, Europe/Vilnius, UTC
  --format string     Output format for hook mode. Accepts a named format,
                      a template string with {placeholders}, or a Go time
                      layout string. Default: "default" format file.
  --formats-dir path  Format files directory. Overrides the default XDG path.
                      Default: {XDG_CONFIG_HOME}/datetime-mcp/formats/
  --log               Enable diagnostic logging to stderr
  --version           Print version, commit hash, and build date
  --help              Print this help message with all loaded formats

Format Files:
  Each .yaml file in the formats directory defines one format.
  Filename = format name. Each file has a template field.

  Example (eu_date.yaml):
    description: "European date dd/MM/yyyy"
    template: "{dd/MM/yyyy}"

  Shipped formats (installed to formats directory):
    iso8601     RFC 3339 with timezone offset
    rfc2822     Email/HTTP date format
    default     Hook mode context line

Template Syntax:
  Templates can contain literal text and {placeholder} expressions.
  Placeholders are resolved in this order:

  1. Built-in keywords:
     {unix}                                Unix timestamp (1771973525)
     {timezone}                            IANA tz identifier
                                           (America/Los_Angeles)

  2. Named formats:
     {iso8601}                             References iso8601.yaml
     {rfc2822}                             References rfc2822.yaml

  3. LDML tokens (Unicode UTS #35):
     yyyy  four-digit year    yy    two-digit year
     MMMM  full month name    MMM   abbreviated month
     MM    zero-padded month  dd    zero-padded day
     EEEE  full weekday       EEE   abbreviated weekday
     HH    24-hour hour       h     12-hour hour
     mm    minute             ss    second
     a     AM/PM marker
     z     timezone abbreviation (PST)
     ZZZZ  UTC offset with colon (-08:00)
     Z     UTC offset without colon (-0800)

     {yyyy-MM-dd}                          2026-02-23
     {HH:mm:ss}                            14:32:05
     {dd/MM/yyyy}                          23/02/2026
     {EEEE, MMMM dd, yyyy}                Monday, February 23, 2026
     {h:mm a}                              2:32 PM
     {z}                                   PST
     {ZZZZ}                                -08:00

     Single quotes escape literal text: {yyyy-MM-dd'T'HH:mm:ss}
     Doubled single quote for literal quote: {h 'o''clock'}

  4. Go time layouts (fallback):
     {2006-01-02}                          2026-02-23
     {Monday}                              Monday
     {MST}                                 PST

  Escaping: {{ and }} produce literal { and }
  Go layout reference: https://pkg.go.dev/time#pkg-constants

Timezone Examples:
  America/New_York    America/Chicago     America/Denver
  America/Los_Angeles Europe/London       Europe/Berlin
  Europe/Vilnius      Asia/Tokyo          Asia/Shanghai
  Australia/Sydney    Pacific/Auckland    UTC

Environment Variables:
  TZ                  Fallback timezone if --tz is not set
  XDG_CONFIG_HOME     Base config directory (platform-dependent default)

Examples:
  # Terminal (hook mode):
  datetime-mcp                                          # Default context line
  datetime-mcp --format iso8601                         # ISO 8601 timestamp
  datetime-mcp --format rfc2822 --tz UTC                # RFC 2822 in UTC
  datetime-mcp --format "{yyyy-MM-dd} {HH:mm:ss}"      # LDML tokens
  datetime-mcp --format "{iso8601} ({timezone})"        # Mixed template
  datetime-mcp --format "2006-01-02"                    # Go layout string
  datetime-mcp --format "{unix}"                        # Unix timestamp

  # MCP server:
  datetime-mcp --mcp                                    # Explicit MCP mode
  datetime-mcp --mcp --tz Europe/Vilnius                # MCP with TZ default

Install:
  # Homebrew (installs binary + format files)
  brew install tmsdnl/mcp/datetime-mcp

  # Go (binary only, format files must be added manually)
  go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest

  # Binary + format files
  Download from https://github.com/tmsdnl/datetime-mcp/releases
```

## 6. Configuration Examples

### 6.1 User-Created Format Files

```yaml
# ~/.config/datetime-mcp/formats/short.yaml
description: "Date and time without timezone"
template: "{yyyy-MM-dd} {HH:mm:ss}"
```

```yaml
# ~/.config/datetime-mcp/formats/human.yaml
description: "Human-readable, long form"
template: "{EEEE, MMMM dd, yyyy h:mm:ss a z}"
```

```yaml
# ~/.config/datetime-mcp/formats/lt_date.yaml
description: "Lithuanian date format"
template: "{2006 m. sausio 2 d.}"
```

```yaml
# ~/.config/datetime-mcp/formats/weekday.yaml
description: "Full weekday name"
template: "{EEEE}"
```

```yaml
# ~/.config/datetime-mcp/formats/deploy-stamp.yaml
description: "Deploy log line"
template: "Deployed at {iso8601} ({timezone}) by CI"
```

### 6.2 Claude Desktop (`claude_desktop_config.json`)

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

### 6.3 Claude Code MCP

```bash
# Add with defaults
claude mcp add datetime /usr/local/bin/datetime-mcp

# Add with timezone override
claude mcp add datetime /usr/local/bin/datetime-mcp -- --tz America/Los_Angeles

# Verify
claude mcp list
```

### 6.4 Claude Code SessionStart Hook (`~/.claude/settings.json`)

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

With timezone and custom template:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/usr/local/bin/datetime-mcp --tz America/Los_Angeles --format \"{EEEE}, {MMMM} {dd}, {yyyy} at {HH:mm} ({timezone})\""
          }
        ]
      }
    ]
  }
}
```

## 7. Project Structure

```
datetime-mcp/
  .github/
    workflows/
      release.yml              # GitHub Actions: test + Goreleaser on tag push
      test.yml                 # GitHub Actions: test on push/PR
  cmd/
    datetime-mcp/
      main.go                  # Entry point, flag parsing, mode detection
  formats/
    iso8601.yaml               # Shipped: RFC 3339
    rfc2822.yaml               # Shipped: Email/HTTP format
    default.yaml               # Shipped: Default hook output
  internal/
    hook/
      hook.go                  # Hook mode output logic
      hook_test.go
    mcp/
      server.go                # JSON-RPC stdio server, MCP lifecycle
      server_test.go
      tool.go                  # Tool definition and handler
      tool_test.go
    datetime/
      datetime.go              # Core datetime formatting (shared)
      datetime_test.go
    formats/
      loader.go                # YAML format file loader (one-file-per-format)
      loader_test.go
      registry.go              # Format registry
      registry_test.go
    template/
      template.go              # Template engine (placeholder resolution)
      template_test.go
      ldml.go                  # LDML token detection and Go layout translation
      ldml_test.go
    detect/
      detect.go                # TTY detection logic
      detect_test.go
  testdata/
    formats/                   # Test fixture format files
  .goreleaser.yml
  Makefile
  go.mod
  go.sum
  README.md
  LICENSE
```

## 8. Acceptance Criteria

1. Running `datetime-mcp` in a terminal with shipped format files installed prints the default context line (from `default.yaml`) and exits.
2. Running `datetime-mcp` with no format files loaded falls back to ISO 8601 output.
3. Running `echo '{}' | datetime-mcp` enters MCP server mode (auto-detected via pipe).
4. Running `datetime-mcp --mcp` in a terminal forces MCP server mode despite TTY.
5. Running `datetime-mcp --format iso8601` prints the ISO 8601 timestamp (from `iso8601.yaml`).
6. Running `datetime-mcp --format "{yyyy-MM-dd} {HH:mm:ss}"` prints `2026-02-23 14:32:05` (LDML tokens).
7. Running `datetime-mcp --format "{dd/MM/yyyy}"` prints `23/02/2026` (LDML tokens, EU style).
8. Running `datetime-mcp --format "{EEEE, MMMM dd, yyyy}"` prints `Monday, February 23, 2026`.
9. Running `datetime-mcp --format "{h:mm a}"` prints `2:32 PM`.
10. Running `datetime-mcp --format "{unix}"` prints `1771973525`.
11. Running `datetime-mcp --format "{timezone}"` prints `America/Los_Angeles`.
12. Running `datetime-mcp --format "{iso8601} ({timezone})"` prints `2026-02-23T14:32:05-08:00 (America/Los_Angeles)`.
13. Running `datetime-mcp --format "2006-01-02"` (bare Go layout, no braces) prints `2026-02-23`.
14. Running `datetime-mcp --format "{z}"` prints `PST` (LDML timezone abbreviation token).
15. Running `datetime-mcp --format "{ZZZZ}"` prints `-08:00` (LDML UTC offset token).
16. Running `datetime-mcp --format "{yyyy-MM-dd'T'HH:mm:ssZZZZ}"` prints `2026-02-23T14:32:05-08:00` (LDML with single-quote literal escaping).
17. Running `datetime-mcp --version` prints version, commit, and build date.
18. Running `datetime-mcp --log` enables diagnostic output to stderr.
19. MCP `initialize` response includes protocol version `2025-11-25`, `tools` capability, and `serverInfo`.
20. MCP `tools/list` returns exactly one tool (`get_current_datetime`) with `name`, `title`, `description`, `inputSchema`, and `outputSchema`.
21. MCP `tools/call` for `get_current_datetime` returns both text content and `structuredContent` with `datetime`, `timezone`, `utc_offset`, and `unix` fields.
22. MCP `tools/call` with `timezone: "Europe/Vilnius"` returns the time converted to EET/EEST.
23. MCP `tools/call` with an invalid timezone returns `isError: true` with a descriptive message.
24. Dropping a file `short.yaml` with `template: "{yyyy-MM-dd} {HH:mm:ss}"` into the formats directory makes `short` usable as `--format short` and as `{short}` in templates.
25. A format file `deploy-stamp.yaml` with `template: "Deployed at {iso8601} ({timezone})"` correctly resolves `{iso8601}` by rendering `iso8601.yaml`'s template.
26. A circular reference (format A references format B which references format A) leaves the re-encountered placeholder as-is and logs a warning.
27. Creating a file `unix.yaml` in the formats directory triggers a warning and the built-in `{unix}` keyword takes precedence.
28. Deleting a format file removes that format from the next invocation.
29. `--help` lists all loaded format names with descriptions.
30. On first run with no format files, the binary prints a message about obtaining format files.
31. XDG paths resolve correctly on macOS, Linux, and Windows (via `adrg/xdg`).
32. All cross-compilation targets build via `make all`.
33. `go test ./...` passes with no external dependencies.
34. Pushing a `v*` tag triggers GitHub Actions, runs tests, and publishes a GitHub release with binaries and shipped format files for all six platform targets, checksums file, and Homebrew formula update to `tmsdnl/mcp`.
35. `brew install tmsdnl/mcp/datetime-mcp` installs the binary and format files.
36. `go install github.com/tmsdnl/datetime-mcp/cmd/datetime-mcp@latest` builds and installs the binary.