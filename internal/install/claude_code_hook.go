package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func installClaudeCodeHook(exePath string, dryRun bool) Result {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".claude")
	path := filepath.Join(dir, "settings.json")

	// Check directory exists (absent if Claude Code has not been configured).
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return Result{Target: "Hook", Status: StatusNotFound, Path: path}
	}

	top, err := readJSONObject(path)
	if err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}

	// Parse hooks object.
	hooksMap := map[string]json.RawMessage{}
	if raw, ok := top["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooksMap); err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing hooks: %w", err)}
		}
	}

	// Parse SessionStart array.
	var sessionStart []json.RawMessage
	if raw, ok := hooksMap["SessionStart"]; ok {
		if err := json.Unmarshal(raw, &sessionStart); err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing SessionStart: %w", err)}
		}
	}

	// Idempotency check: search for exePath in any SessionStart hook command.
	for _, elem := range sessionStart {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(elem, &entry); err != nil {
			continue
		}
		var hooks []json.RawMessage
		if raw, ok := entry["hooks"]; ok {
			_ = json.Unmarshal(raw, &hooks)
		}
		for _, h := range hooks {
			var hook map[string]string
			if err := json.Unmarshal(h, &hook); err != nil {
				continue
			}
			cmd := hook["command"]
			if resolved, err := filepath.EvalSymlinks(cmd); err == nil {
				cmd = resolved
			}
			if cmd == exePath {
				return Result{Target: "Hook", Status: StatusExisting, Path: path}
			}
		}
	}

	if dryRun {
		b, _ := json.MarshalIndent(map[string]any{
			"hooks": []map[string]string{{"type": "command", "command": exePath}},
		}, "", "  ")
		return Result{Target: "Hook", Status: StatusAdded, Path: path, DryRun: true, Snippet: string(b)}
	}

	// Build new hook entry.
	hookEntry := map[string]any{
		"hooks": []map[string]string{
			{"type": "command", "command": exePath},
		},
	}
	hookRaw, err := json.Marshal(hookEntry)
	if err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}
	sessionStart = append(sessionStart, json.RawMessage(hookRaw))

	ssRaw, err := json.Marshal(sessionStart)
	if err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}
	hooksMap["SessionStart"] = json.RawMessage(ssRaw)

	hooksRaw, err := json.Marshal(hooksMap)
	if err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}
	top["hooks"] = json.RawMessage(hooksRaw)

	if err := writeJSONObject(path, top); err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}
	return Result{Target: "Hook", Status: StatusAdded, Path: path}
}
