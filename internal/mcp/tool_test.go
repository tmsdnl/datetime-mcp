package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	_ "time/tzdata"
)

func TestToolDefinition(t *testing.T) {
	tool := toolDefinition()

	if tool["name"] != "get_current_datetime" {
		t.Errorf("name = %v", tool["name"])
	}
	if tool["title"] != "Current Date and Time" {
		t.Errorf("title = %v", tool["title"])
	}
	desc, _ := tool["description"].(string)
	if !strings.Contains(desc, "format") || !strings.Contains(desc, "timezone") {
		t.Errorf("description should mention format and timezone: %q", desc)
	}
	if tool["inputSchema"] == nil {
		t.Error("inputSchema missing")
	}
	if tool["outputSchema"] == nil {
		t.Error("outputSchema missing")
	}

	// Verify outputSchema has required fields.
	outputSchema := tool["outputSchema"].(map[string]any)
	required := outputSchema["required"].([]string)
	requiredSet := map[string]bool{}
	for _, r := range required {
		requiredSet[r] = true
	}
	for _, field := range []string{"datetime", "timezone", "utc_offset", "unix"} {
		if !requiredSet[field] {
			t.Errorf("outputSchema required missing field: %q", field)
		}
	}
}

func TestCallGetCurrentDatetime_StructuredContent(t *testing.T) {
	s := testServer(t)

	resp := s.callGetCurrentDatetime(json.RawMessage(`1`), "", "iso8601")
	if resp == nil {
		t.Fatal("nil response")
	}

	data, _ := json.Marshal(resp.Result)
	var result map[string]any
	json.Unmarshal(data, &result)

	sc := result["structuredContent"].(map[string]any)

	// utc_offset must match [+-]HH:MM format.
	offset := sc["utc_offset"].(string)
	if len(offset) != 6 || (offset[0] != '+' && offset[0] != '-') {
		t.Errorf("utc_offset format invalid: %q", offset)
	}

	// unix must be a number > 0.
	unix := sc["unix"].(float64)
	if unix <= 0 {
		t.Errorf("unix timestamp should be positive, got %v", unix)
	}

	// datetime and text content must match.
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if sc["datetime"] != text {
		t.Errorf("structuredContent.datetime %q != text %q", sc["datetime"], text)
	}
}
