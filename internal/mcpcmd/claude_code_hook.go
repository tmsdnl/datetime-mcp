package mcpcmd

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

	// Parse SessionStart array (treat missing or null as empty).
	var sessionStart []json.RawMessage
	if raw, ok := hooksMap["SessionStart"]; ok && string(raw) != "null" {
		if err := json.Unmarshal(raw, &sessionStart); err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing SessionStart: %w", err)}
		}
	}

	// Idempotency check: search for any hook whose command is our binary
	// (matched by binary name, to catch versioned/cached paths from old installs).
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
			if filepath.Base(hook["command"]) == filepath.Base(exePath) {
				return Result{Target: "Hook", Status: StatusExisting, Path: path}
			}
		}
	}

	if dryRun {
		return Result{Target: "Hook", Status: StatusAdded, Path: path, DryRun: true}
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

func removeClaudeCodeHook(exePath string, dryRun bool) Result {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "settings.json")

	top, err := readJSONObject(path)
	if err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}

	hooksMap := map[string]json.RawMessage{}
	if raw, ok := top["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooksMap); err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing hooks: %w", err)}
		}
	}

	var sessionStart []json.RawMessage
	if raw, ok := hooksMap["SessionStart"]; ok {
		if err := json.Unmarshal(raw, &sessionStart); err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path,
				Err: fmt.Errorf("parsing SessionStart: %w", err)}
		}
	}

	// Filter out any entries whose hooks contain a command matching exePath.
	var filtered []json.RawMessage
	removed := false
	for _, elem := range sessionStart {
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(elem, &entry); err != nil {
			filtered = append(filtered, elem)
			continue
		}
		var hooks []json.RawMessage
		if raw, ok := entry["hooks"]; ok {
			_ = json.Unmarshal(raw, &hooks)
		}
		var keptHooks []json.RawMessage
		for _, h := range hooks {
			var hook map[string]string
			if err := json.Unmarshal(h, &hook); err != nil {
				keptHooks = append(keptHooks, h)
				continue
			}
			// Remove any hook whose command is our binary name, regardless of
			// path — this cleans up versioned Cellar paths, go-build cache
			// paths, and other stale entries from previous installs.
			if filepath.Base(hook["command"]) == filepath.Base(exePath) {
				removed = true
			} else {
				keptHooks = append(keptHooks, h)
			}
		}
		if len(keptHooks) > 0 {
			hooksRaw, _ := json.Marshal(keptHooks)
			entry["hooks"] = json.RawMessage(hooksRaw)
			entryRaw, _ := json.Marshal(entry)
			filtered = append(filtered, json.RawMessage(entryRaw))
		}
	}

	if !removed {
		return Result{Target: "Hook", Status: StatusNotFound, Path: path,
			Message: "no matching hook entry found"}
	}

	if dryRun {
		return Result{Target: "Hook", Status: StatusRemoved, Path: path, DryRun: true}
	}

	if len(filtered) == 0 {
		delete(hooksMap, "SessionStart")
	} else {
		ssRaw, err := json.Marshal(filtered)
		if err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
		}
		hooksMap["SessionStart"] = json.RawMessage(ssRaw)
	}

	if len(hooksMap) == 0 {
		delete(top, "hooks")
	} else {
		hooksRaw, err := json.Marshal(hooksMap)
		if err != nil {
			return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
		}
		top["hooks"] = json.RawMessage(hooksRaw)
	}

	if err := writeJSONObject(path, top); err != nil {
		return Result{Target: "Hook", Status: StatusError, Path: path, Err: err}
	}
	return Result{Target: "Hook", Status: StatusRemoved, Path: path}
}
