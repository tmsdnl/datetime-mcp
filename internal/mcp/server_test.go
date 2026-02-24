package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/tmsdnl/datetime-mcp/internal/formats"
)

// testServer creates a server with testdata formats loaded.
func testServer(t *testing.T) *server {
	t.Helper()
	fmts, errs := formats.Load("../../testdata/formats")
	if len(errs) != 0 {
		t.Fatalf("loading formats: %v", errs)
	}
	reg := formats.NewRegistry(fmts)
	loc, _ := time.LoadLocation("America/Los_Angeles")
	return &server{
		reg:        reg,
		defaultLoc: loc,
		logger:     func(string, ...any) {},
	}
}

// exchange sends a JSON-RPC message to the server and reads the response.
func (s *server) exchange(t *testing.T, req map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp := s.handleMessage(data)
	if resp == nil {
		return nil
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestHandleInitialize(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "test", "version": "1.0"},
		},
	})

	result := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2025-11-25" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
	caps := result["capabilities"].(map[string]any)
	if _, ok := caps["tools"]; !ok {
		t.Error("capabilities missing 'tools'")
	}
	serverInfo := result["serverInfo"].(map[string]any)
	if serverInfo["name"] != "datetime-mcp" {
		t.Errorf("serverInfo.name = %v", serverInfo["name"])
	}
	// F-024: description must match exactly.
	if serverInfo["description"] != "Self-contained date/time provider for Claude Desktop and Claude Code" {
		t.Errorf("serverInfo.description = %v", serverInfo["description"])
	}
	// version must be a string (empty in test since no version set).
	if _, ok := serverInfo["version"].(string); !ok {
		t.Errorf("serverInfo.version is not a string: %T", serverInfo["version"])
	}
	// F-023: listChanged must be false.
	toolsCap, ok := caps["tools"].(map[string]any)
	if !ok {
		t.Fatal("capabilities.tools is not a map")
	}
	if toolsCap["listChanged"] != false {
		t.Errorf("capabilities.tools.listChanged = %v, want false", toolsCap["listChanged"])
	}
}

func TestHandleInitialized_NoResponse(t *testing.T) {
	s := testServer(t)
	resp := s.handleMessage([]byte(`{"jsonrpc":"2.0","method":"initialized"}`))
	if resp != nil {
		t.Error("initialized notification should return nil")
	}
}

func TestHandlePing(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	})
	if resp["result"] == nil {
		t.Error("ping should return empty result")
	}
	if resp["error"] != nil {
		t.Errorf("ping should not return error: %v", resp["error"])
	}
}

func TestHandleToolsList(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	})

	result := resp["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0].(map[string]any)
	if tool["name"] != "get_current_datetime" {
		t.Errorf("tool name = %v", tool["name"])
	}
	if tool["title"] != "Current Date and Time" {
		t.Errorf("tool title = %v", tool["title"])
	}
	if tool["description"] == nil || tool["description"] == "" {
		t.Error("tool description is empty")
	}
	if tool["inputSchema"] == nil {
		t.Error("tool inputSchema is missing")
	}
	if tool["outputSchema"] == nil {
		t.Error("tool outputSchema is missing")
	}
}

func TestHandleToolsCall_Success(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_current_datetime",
			"arguments": map[string]any{},
		},
	})

	result := resp["result"].(map[string]any)

	// isError must be false (F-030).
	if result["isError"] != false {
		t.Errorf("isError = %v, want false", result["isError"])
	}

	// content must be a text block (F-110).
	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("content is empty")
	}
	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Errorf("content[0].type = %v, want text", block["type"])
	}
	text, ok := block["text"].(string)
	if !ok || text == "" {
		t.Errorf("content[0].text is empty or not string: %v", block["text"])
	}

	// structuredContent must be present (F-112, F-113).
	sc := result["structuredContent"].(map[string]any)
	if sc["datetime"] == nil {
		t.Error("structuredContent.datetime is missing")
	}
	if sc["timezone"] == nil {
		t.Error("structuredContent.timezone is missing")
	}
	if sc["utc_offset"] == nil {
		t.Error("structuredContent.utc_offset is missing")
	}
	if sc["unix"] == nil {
		t.Error("structuredContent.unix is missing")
	}

	// datetime in structuredContent must match text content (F-113).
	if sc["datetime"] != text {
		t.Errorf("structuredContent.datetime %q != text content %q", sc["datetime"], text)
	}
}

func TestHandleToolsCall_WithTimezone(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_current_datetime",
			"arguments": map[string]any{
				"timezone": "Europe/Vilnius",
				"format":   "iso8601",
			},
		},
	})

	result := resp["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("unexpected error: %v", result)
	}

	sc := result["structuredContent"].(map[string]any)
	if sc["timezone"] != "Europe/Vilnius" {
		t.Errorf("structuredContent.timezone = %v, want Europe/Vilnius", sc["timezone"])
	}
}

func TestHandleToolsCall_InvalidTimezone(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "get_current_datetime",
			"arguments": map[string]any{
				"timezone": "Mars/Olympus",
			},
		},
	})

	result := resp["result"].(map[string]any)
	// Must be isError:true (not a JSON-RPC error) (F-030, F-115).
	if result["isError"] != true {
		t.Errorf("expected isError=true for invalid timezone, got: %v", result)
	}
	if resp["error"] != nil {
		t.Errorf("invalid timezone should be tool error not JSON-RPC error: %v", resp["error"])
	}
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "Mars/Olympus") {
		t.Errorf("error message should contain timezone name: %q", text)
	}
}

func TestHandleToolsCall_UnknownTool(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "nonexistent_tool",
			"arguments": map[string]any{},
		},
	})

	// Must be JSON-RPC error -32602 (F-029).
	if resp["error"] == nil {
		t.Fatalf("expected JSON-RPC error for unknown tool, got result: %v", resp["result"])
	}
	errObj := resp["error"].(map[string]any)
	code := errObj["code"].(float64)
	if code != -32602 {
		t.Errorf("error code = %v, want -32602", code)
	}
}

func TestHandleMalformedJSON(t *testing.T) {
	s := testServer(t)
	// Must not crash, must return parse error (NF-040).
	resp := s.handleMessage([]byte(`{not valid json}`))
	if resp == nil {
		t.Fatal("expected a response for malformed JSON")
	}
	if resp.Error == nil {
		t.Error("expected JSON-RPC error for malformed input")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := testServer(t)
	resp := s.exchange(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "unknown/method",
	})
	if resp["error"] == nil {
		t.Error("expected error for unknown method")
	}
}

func TestServeLoop_StdinEOF(t *testing.T) {
	// Run the serve loop with stdin that immediately closes.
	s := testServer(t)

	var outBuf bytes.Buffer
	s.out = &outBuf
	s.in = strings.NewReader("") // empty → immediate EOF

	err := s.serve()
	if err != nil {
		t.Errorf("serve() returned error on stdin EOF: %v", err)
	}
}

func TestFullLifecycle(t *testing.T) {
	// Simulate a complete MCP session: initialize → initialized → tools/list → tools/call → EOF.
	s := testServer(t)

	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_current_datetime","arguments":{}}}`,
	}

	input := strings.Join(messages, "\n") + "\n"
	var outBuf bytes.Buffer
	s.in = strings.NewReader(input)
	s.out = &outBuf

	if err := s.serve(); err != nil {
		t.Fatalf("serve() error: %v", err)
	}

	// Parse output lines.
	scanner := bufio.NewScanner(&outBuf)
	var responses []map[string]any
	for scanner.Scan() {
		var resp map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("invalid JSON in output: %v\n%s", err, scanner.Text())
		}
		responses = append(responses, resp)
	}

	// Should have 3 responses (initialize, tools/list, tools/call).
	// The "initialized" notification has no response.
	if len(responses) != 3 {
		t.Fatalf("expected 3 responses, got %d: %v", len(responses), responses)
	}

	// Response 1: initialize
	initResult := responses[0]["result"].(map[string]any)
	if initResult["protocolVersion"] != "2025-11-25" {
		t.Errorf("initialize: protocolVersion = %v", initResult["protocolVersion"])
	}

	// Response 2: tools/list
	listResult := responses[1]["result"].(map[string]any)
	tools := listResult["tools"].([]any)
	if len(tools) != 1 {
		t.Errorf("tools/list: expected 1 tool, got %d", len(tools))
	}

	// Response 3: tools/call
	callResult := responses[2]["result"].(map[string]any)
	if callResult["isError"] == true {
		t.Errorf("tools/call returned isError=true: %v", callResult)
	}
	sc := callResult["structuredContent"].(map[string]any)
	if sc["datetime"] == nil || sc["timezone"] == nil || sc["utc_offset"] == nil || sc["unix"] == nil {
		t.Errorf("structuredContent missing fields: %v", sc)
	}
}

func TestServeLoop_OversizedInput(t *testing.T) {
	// Build a line that exceeds the 1 MB scanner buffer limit.
	// The scanner will return an error via scanner.Err() instead of delivering the line.
	var sb strings.Builder
	sb.WriteString(`{"jsonrpc":"2.0","id":1,"method":"ping","params":"`)
	for i := 0; i < 1024*1024+1; i++ {
		sb.WriteByte('x')
	}
	sb.WriteString("\"}\n")

	s := testServer(t)
	var outBuf bytes.Buffer
	s.out = &outBuf
	s.in = strings.NewReader(sb.String())

	if err := s.serve(); err != nil {
		t.Fatalf("serve() returned error: %v", err)
	}

	// Parse all output lines and expect at least one error response with code -32700.
	scanner := bufio.NewScanner(&outBuf)
	var responses []map[string]any
	for scanner.Scan() {
		var resp map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("invalid JSON in output: %v\n%s", err, scanner.Text())
		}
		responses = append(responses, resp)
	}

	if len(responses) == 0 {
		t.Fatal("expected at least one error response for oversized input, got none")
	}
	errObj, ok := responses[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected first response to have an 'error' field, got: %v", responses[0])
	}
	code, ok := errObj["code"].(float64)
	if !ok || int(code) != -32700 {
		t.Errorf("error code = %v, want -32700", errObj["code"])
	}
}
