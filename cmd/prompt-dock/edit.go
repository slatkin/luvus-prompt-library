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
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const FALLBACK_EDITOR = "nvim"

// externalEditors are the external editors the dock offers, in menu order.
// edit-external refuses any editor not listed here.
var externalEditors = []Editor{{Bin: "zed", Label: "zed"}, {Bin: "code", Label: "VS Code"}}

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
	paneId := paneIdFromSplit(split.Stdout)
	if split.Status != 0 || paneId == "" {
		toast("Couldn't open a pane for editing. Please try again.")
		return fmt.Errorf("edit-inline: pane split failed (status %d): %s", split.Status, split.detail())
	}

	run := luvus([]string{"pane", "run", paneId, editor + " '" + abs + "'"}, "")
	if run.Status != 0 {
		// Close the pane we just split, so a failed run leaves no empty pane behind.
		luvus([]string{"pane", "close", paneId}, "")
		toast("Couldn't start the editor in the new pane. Please try again.")
		return fmt.Errorf("edit-inline: pane run failed (status %d): %s", run.Status, run.detail())
	}
	return nil
}

// paneIdFromSplit decodes the `pane split` reply and returns the new pane id,
// or "" when the reply has none. luvus prints the RPC envelope; a flat object
// is accepted too, and `pane` may be a string or a number (the TS code used
// String(...)).
func paneIdFromSplit(stdout string) string {
	var parsed map[string]any
	if json.Unmarshal([]byte(stdout), &parsed) != nil {
		return ""
	}
	node := parsed
	if result, ok := parsed["result"].(map[string]any); ok {
		node = result
	}
	switch v := node["pane"].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		return ""
	}
}

// splitExternalValue splits an edit-external row value on the FIRST `|` only
// (design D3), so a `|` in a file name still works. Returns [editor, relpath].
func splitExternalValue(value string) (string, string) {
	editor, rel, ok := strings.Cut(value, "|")
	if !ok {
		return "", value
	}
	return editor, rel
}

// editExternal starts the editor detached and finishes at once (design D7).
func editExternal(value string) error {
	editor, rel := splitExternalValue(value)
	if editor == "" {
		return fmt.Errorf("edit-external: malformed value (expected <editor>|<relpath>): %s", value)
	}
	if !slices.ContainsFunc(externalEditors, func(e Editor) bool { return e.Bin == editor }) {
		return fmt.Errorf("edit-external: unknown editor: %s", editor)
	}

	abs := resolveUnderRoot(settingRoot(), rel)
	if abs == "" {
		toast("That file is outside your prompt library root, so it can't be opened.")
		return fmt.Errorf("edit-external: relpath escapes root: %s", rel)
	}

	// The TS code quoted the path and used shell:true on Windows because spawn
	// could not run the `code.cmd` shim directly. exec.LookPath finds the .cmd
	// shim on Windows, so that branch goes away (design D15).
	// ponytail: cmd still parses the arguments of a batch file, so a path
	// containing % or & can break.
	cmd := exec.Command(editor, abs)
	setDetach(cmd)
	// A failed launch (e.g. binary vanished) must not crash the action.
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "edit-external: spawn error: %v\n", err)
		toast(fmt.Sprintf("Couldn't start %s. Please try again.", editor))
		return nil
	}
	return nil
}
