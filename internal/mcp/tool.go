package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/tmsdnl/datetime-mcp/internal/datetime"
)

// toolDefinition returns the get_current_datetime tool definition per MCP 2025-11-25.
func toolDefinition() map[string]any {
	return map[string]any{
		"name":  "get_current_datetime",
		"title": "Current Date and Time",
		"description": "Returns the current date and time. " +
			"Call this proactively before any query where recency matters — " +
			"e.g. \"latest\", \"recent\", \"today\", \"now\", or web searches for current information. " +
			"The format parameter accepts named formats (iso8601, rfc2822, etc.) or Go time layout strings. " +
			"The timezone parameter accepts IANA tz database identifiers (e.g. America/Los_Angeles, Europe/Vilnius, UTC).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA tz database identifier (e.g. America/Los_Angeles). Default: server timezone.",
				},
				"format": map[string]any{
					"type":        "string",
					"description": "Named format (iso8601, rfc2822) or Go time layout string. Default: iso8601.",
				},
			},
		},
		"outputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"datetime": map[string]any{
					"type":        "string",
					"description": "Formatted datetime string.",
				},
				"timezone": map[string]any{
					"type":        "string",
					"description": "IANA tz database identifier used.",
				},
				"utc_offset": map[string]any{
					"type":        "string",
					"description": "UTC offset in [+-]HH:MM format.",
				},
				"unix": map[string]any{
					"type":        "integer",
					"description": "Unix timestamp (seconds since epoch).",
				},
			},
			"required": []string{"datetime", "timezone", "utc_offset", "unix"},
		},
	}
}

// handleToolsList responds to tools/list (F-025).
func (s *server) handleToolsList(req jsonrpcRequest) *jsonrpcResponse {
	return resultResponse(req.ID, map[string]any{
		"tools": []any{toolDefinition()},
	})
}

// handleToolsCall responds to tools/call (F-026, F-030).
func (s *server) handleToolsCall(req jsonrpcRequest) *jsonrpcResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32600, "invalid request: "+err.Error())
	}

	if params.Name != "get_current_datetime" {
		// Unknown tool → JSON-RPC error -32602 (F-029)
		return errorResponse(req.ID, -32602, "unknown tool: "+params.Name)
	}

	var args struct {
		Timezone string `json:"timezone"`
		Format   string `json:"format"`
	}
	if params.Arguments != nil {
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return errorResponse(req.ID, -32600, "invalid arguments: "+err.Error())
		}
	}

	return s.callGetCurrentDatetime(req.ID, args.Timezone, args.Format)
}

// callGetCurrentDatetime executes the get_current_datetime tool.
func (s *server) callGetCurrentDatetime(id json.RawMessage, tzOverride, formatStr string) *jsonrpcResponse {
	// Resolve timezone: tool parameter > server default.
	loc := s.defaultLoc
	if tzOverride != "" {
		l, err := time.LoadLocation(tzOverride)
		if err != nil {
			// Return tool error (not JSON-RPC error) per MCP convention (F-030).
			return toolErrorResponse(id, fmt.Sprintf(
				"Invalid timezone: %q. Must be a valid IANA tz database identifier (e.g., America/Los_Angeles, Europe/Vilnius, UTC).",
				tzOverride,
			))
		}
		loc = l
	}

	t := time.Now().In(loc)

	// Default format for MCP tool is iso8601.
	if formatStr == "" {
		formatStr = "iso8601"
	}

	if formatStr == "iso8601" {
		if _, ok := s.reg.Get("iso8601"); !ok {
			formatted := datetime.ISO8601Fallback(t)
			sc := datetime.NewStructuredContent(t, formatted)
			result := map[string]any{
				"content":           []any{map[string]any{"type": "text", "text": formatted}},
				"structuredContent": map[string]any{"datetime": sc.Datetime, "timezone": sc.Timezone, "utc_offset": sc.UTCOffset, "unix": sc.Unix},
				"isError":           false,
			}
			return resultResponse(id, result)
		}
	}

	// In MCP strict mode, a format string that is not a template (no '{') and is
	// not found in the registry is treated as an unknown named format.
	if !strings.Contains(formatStr, "{") {
		if _, ok := s.reg.Get(formatStr); !ok {
			return toolErrorResponse(id, fmt.Sprintf("unknown format: %q", formatStr))
		}
	}

	f := datetime.New(s.reg, func(msg string) { s.logger("%s", msg) })
	formatted, err := f.FormatAuto(t, formatStr)
	if err != nil {
		return toolErrorResponse(id, err.Error())
	}

	sc := datetime.NewStructuredContent(t, formatted)

	result := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": formatted},
		},
		"structuredContent": map[string]any{
			"datetime":   sc.Datetime,
			"timezone":   sc.Timezone,
			"utc_offset": sc.UTCOffset,
			"unix":       sc.Unix,
		},
		"isError": false,
	}
	return resultResponse(id, result)
}

// toolErrorResponse wraps a tool error as a successful JSON-RPC response with
// isError:true per MCP convention (SEP-1303, F-030, F-115).
func toolErrorResponse(id json.RawMessage, message string) *jsonrpcResponse {
	result := map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": message},
		},
		"isError": true,
	}
	return resultResponse(id, result)
}
