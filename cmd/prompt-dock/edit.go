// edit.go — open a prompt file from a dock row (design D6, D7). Ported from
// edit.ts. Two modes, chosen by the verb (which replaces the TS
// LUVUS_MODULE_ACTION_ID check, design D13):
//
//	edit-inline    — split a pane without taking focus and type the editor
//	                 command (from luvus "Open files with", layout.file_open)
//	                 into its shell; falls back to nvim.
//	edit-external  — spawn zed or VS Code detached on the file and exit at
//	                 once (row value is `<editor>|<relpath>`, design D3).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FALLBACK_EDITOR = "nvim"

// readRawFileOpen reads luvus "Open files with" from ~/.luvus/config.json
// (design D6). Returns the raw setting value, or "" when the file is missing,
// unreadable or not valid JSON.
func readRawFileOpen() string {
	home := homeDir()
	if home == "" {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".luvus", "config.json"))
	if err != nil {
		return ""
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return ""
	}
	layout, _ := config["layout"].(map[string]any)
	value, _ := layout["file_open"].(string)
	return value
}

// resolveInlineEditor returns the editor to open the file with: luvus "Open
// files with" when it is set and readable, otherwise nvim (spec: inline edit
// fallback).
func resolveInlineEditor(raw string) string {
	value := strings.TrimSpace(raw)
	// `readonly` is luvus's built-in viewer, which can't edit — use nvim instead.
	if value == "" || value == "readonly" {
		return FALLBACK_EDITOR
	}
	return value
}

// editInline splits a pane without focusing it, then types the editor command.
func editInline(value string) error {
	root := settingRoot()
	abs := resolveUnderRoot(root, value)
	if abs == "" {
		toast("That file is outside your prompt library root, so it can't be opened.")
		return fmt.Errorf("edit-inline: relpath escapes root: %s", value)
	}
	if strings.ContainsAny(abs, "'\\\r\n") {
		// `pane run` types the command into a shell, so the path must be
		// single-quoted; an apostrophe, a backslash (fish reads \\ as one \),
		// or a line break would break it.
		toast("Files with an apostrophe, a backslash, or a line break in the name can't be opened inline. Please rename the file.")
		return fmt.Errorf("edit-inline: path contains a quote, backslash, or line break: %s", abs)
	}

	editor := resolveInlineEditor(readRawFileOpen())
	split := luvus([]string{"pane", "split", "--no-focus"}, "")
	if split.Status != 0 {
		toast("Couldn't open a pane for editing. Please try again.")
		detail := strings.TrimSpace(split.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(split.Stdout)
		}
		return fmt.Errorf("edit-inline: pane split failed (status %d): %s", split.Status, detail)
	}
	paneId, ok := paneIdFromSplit(split.Stdout)
	if !ok {
		toast("Couldn't open a pane for editing. Please try again.")
		return fmt.Errorf("edit-inline: unexpected pane split output: %s", strings.TrimSpace(split.Stdout))
	}
	if paneId == "" {
		toast("Couldn't open a pane for editing. Please try again.")
		return fmt.Errorf("edit-inline: no pane id in pane split output: %s", strings.TrimSpace(split.Stdout))
	}

	run := luvus([]string{"pane", "run", paneId, editor + " '" + abs + "'"}, "")
	if run.Status != 0 {
		// Close the pane we just split, so a failed run leaves no empty pane behind.
		luvus([]string{"pane", "close", paneId}, "")
		toast("Couldn't start the editor in the new pane. Please try again.")
		detail := strings.TrimSpace(run.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(run.Stdout)
		}
		return fmt.Errorf("edit-inline: pane run failed (status %d): %s", run.Status, detail)
	}
	return nil
}

// paneIdFromSplit decodes the `pane split` reply and returns the new pane id.
// luvus prints the RPC envelope; a flat object is accepted too, and `pane`
// may be a string or a number (the TS code used String(...)).
func paneIdFromSplit(stdout string) (string, bool) {
	var parsed map[string]any
	if json.Unmarshal([]byte(stdout), &parsed) != nil {
		return "", false
	}
	node := parsed
	if result, ok := parsed["result"].(map[string]any); ok {
		node = result
	}
	switch v := node["pane"].(type) {
	case string:
		return v, true
	case float64:
		return fmt.Sprintf("%v", v), true
	default:
		return "", true // parsed, but no usable pane id
	}
}
