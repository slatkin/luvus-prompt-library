// refresh.go — scan the prompt library and push the dock rows (design D2).
// Runs on startup, on every event hook, for the `refresh` action, and from
// toggle.go after it flips the expanded set. Sends one JSON row array to
// `ui dock push` on stdin.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	dockID    = "prompts"
	dockTitle = "PROMPTS"
)

// The tree is drawn with box-drawing branches. Each row with a parent starts
// with one guide cell per ancestor, then a connector; root rows carry no prefix,
// so their glyph sits at the dock's left edge.
const (
	guidePipe  = "│ "
	guideBlank = "  "
	branchMid  = "├─ "
	branchLast = "└─ "
)

// Row prefixes, written as escapes so the codepoints stay exact and reviewable.
// These are Nerd Font glyphs (Material Design set), so the terminal needs a Nerd
// Font — the same assumption luvus's own web client makes in its font stack.
const (
	promptGlyph     = "\U000F0B79" // md-chat
	folderGlyph     = "\U000F024B" // md-folder
	folderOpenGlyph = "\U000F0770"
)

// DockRow is one row as `ui dock push` accepts it (JSON on stdin).
type DockRow struct {
	Text   string          `json:"text"`
	Spans  []DockSpan      `json:"spans,omitempty"`
	Action string          `json:"action,omitempty"`
	Value  string          `json:"value,omitempty"`
	Menu   []DockMenuEntry `json:"menu,omitempty"`
}

// DockSpan is one drawn segment of a row. A span with no tone takes the row's
// tone, and a row with no tone takes the default row colour; `text` is drawn
// only when `spans` is empty.
type DockSpan struct {
	Text string `json:"text"`
	Tone string `json:"tone,omitempty"`
}

// DockMenuEntry is one per-row right-click menu entry as `ui dock push`
// accepts it.
type DockMenuEntry struct {
	Title  string `json:"title"`
	Action string `json:"action"`
	Value  string `json:"value"`
}

// Editor is an external editor found on PATH.
type Editor struct {
	Bin string
	// Label is the editor's name, for the menu entry label.
	Label string
}

// availableEditors returns the external editors found on PATH; a refresh
// checks this once.
func availableEditors() []Editor {
	editors := []Editor{}
	for _, e := range externalEditors {
		if _, err := exec.LookPath(e.Bin); err == nil {
			editors = append(editors, e)
		}
	}
	return editors
}

// editMenuEntries returns the "Edit" entries for a prompt row: one per
// external editor on PATH (design D7, spec: external editors). "Open" is the
// inline editor and is built in toDockRow; these entries are the detached
// applications.
func editMenuEntries(rel string, editors []Editor) []DockMenuEntry {
	entries := []DockMenuEntry{}
	for _, e := range editors {
		entries = append(entries, DockMenuEntry{
			Title:  fmt.Sprintf("Edit in %s", e.Label),
			Action: "edit-external",
			Value:  e.Bin + "|" + rel,
		})
	}
	return entries
}

// sendMenuEntries returns one "Send to" entry per live agent for a prompt row
// (spec: agent send targets). An agent without a terminal id still gets an
// entry: its value carries an empty id, and send.go refuses it with a toast
// (design D4).
func sendMenuEntries(rel string, agents []AgentInfo) []DockMenuEntry {
	entries := []DockMenuEntry{}
	for _, a := range agents {
		entries = append(entries, DockMenuEntry{
			Title:  fmt.Sprintf("Send to %s", targetLabel(a)),
			Action: "send",
			Value:  encodeSendValue(a.Pane, a.TerminalID, rel),
		})
	}
	return entries
}

// branchPrefixes returns the branch prefix for every row, in row order (spec:
// tree rows). A row draws one two-column guide cell per ancestor, then a
// connector: `├─` when a sibling follows it, `└─` when it is the last. Root
// rows get no prefix.
//
// A guide cell is a pipe only when its ancestor has a later sibling AND is not a
// root row: a pipe continues the ancestor's own connector down to the next
// sibling, and a root row has no connector. So the children of a top-level
// folder keep the two columns but draw no line back up to the folder.
func branchPrefixes(rows []TreeRow) []string {
	// hasLater[d] is the last row seen at depth d. It is the ancestor of every
	// deeper row that follows it, since scanTree emits rows depth-first.
	hasLater := []bool{}
	prefixes := make([]string, len(rows))
	for i, row := range rows {
		var guides strings.Builder
		for depth := 0; depth < row.Depth; depth++ {
			if depth > 0 && depth < len(hasLater) && hasLater[depth] {
				guides.WriteString(guidePipe)
			} else {
				guides.WriteString(guideBlank)
			}
		}
		for len(hasLater) <= row.Depth {
			hasLater = append(hasLater, false)
		}
		hasLater[row.Depth] = !row.IsLast
		if row.Depth == 0 {
			prefixes[i] = ""
			continue
		}
		branch := branchMid
		if row.IsLast {
			branch = branchLast
		}
		prefixes[i] = guides.String() + branch
	}
	return prefixes
}

// rowSpans returns the drawn segments of a row: the branch prefix, then the
// glyph and name. The prefix never carries a tone, so only the label takes the
// accent tone — a row tone would paint its branch glyphs and its text the same
// colour (spec: row colours). The row's text stays the plain label a click
// hands to the action.
func rowSpans(prefix string, label string, labelTone string) []DockSpan {
	labelSpan := DockSpan{Text: label}
	if labelTone != "" {
		labelSpan.Tone = labelTone
	}
	// luvus drops a span with empty text, so a root row sends the label alone.
	if prefix == "" {
		return []DockSpan{labelSpan}
	}
	return []DockSpan{{Text: prefix}, labelSpan}
}

// isFolder reports whether p is a directory.
func isFolder(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false // missing, or not readable: show the missing-root row
	}
	return info.IsDir()
}

// buildRows returns the dock rows for root: the tree, or the missing-root row.
// The editors and agents are parameters so tests can pass fixed lists; the
// refreshDock caller looks them up (design D16).
func buildRows(root string, expanded map[string]bool, selected string, editors []Editor, agents []AgentInfo) []DockRow {
	if !isFolder(root) {
		// Spec: missing root — one muted row that names the folder, no other rows.
		label := fmt.Sprintf("root not found: %s", root)
		return []DockRow{{Text: label, Spans: rowSpans("", label, "muted")}}
	}
	tree := scanTree(root, expanded)
	prefixes := branchPrefixes(tree)
	rows := make([]DockRow, len(tree))
	for i, row := range tree {
		rows[i] = toDockRow(row, prefixes[i], expanded[row.Rel], row.Rel == selected, editors, agents)
	}
	return rows
}

// toDockRow builds one dock row: its branch prefix, then the glyph and name
// for its kind.
func toDockRow(row TreeRow, prefix string, isExpanded bool, isSelected bool, editors []Editor, agents []AgentInfo) DockRow {
	if row.Kind == "folder" {
		glyph := folderGlyph
		if isExpanded {
			glyph = folderOpenGlyph
		}
		label := fmt.Sprintf("%s %s", glyph, row.Name)
		return DockRow{
			Text: prefix + label,
			// Folders take the accent tone, so they read differently from prompts.
			Spans:  rowSpans(prefix, label, "accent"),
			Action: "toggle",
			Value:  row.Rel,
		}
	}
	// A left-click only selects (design D9): opening a file is a menu choice, so a
	// stray click cannot pull focus into a new editor pane.
	label := fmt.Sprintf("%s %s", promptGlyph, row.Name)
	return DockRow{
		Text: prefix + label,
		// Mirrors the built-in docks' focused row as far as the dock API allows
		// (design D10): accent text, the exact colour they use, and no marker glyph
		// of its own — every prompt row already carries the prompt glyph.
		Spans:  rowSpans(prefix, label, tone(isSelected, "accent")),
		Action: "select",
		Value:  row.Rel,
		// Send targets first: they are the common action, and the menu reads top-down.
		Menu: append(append(sendMenuEntries(row.Rel, agents),
			DockMenuEntry{Title: "Open", Action: "edit-inline", Value: row.Rel}),
			editMenuEntries(row.Rel, editors)...),
	}
}

// tone returns tone when flag holds, else "".
func tone(flag bool, tone string) string {
	if flag {
		return tone
	}
	return ""
}

// refreshDock scans the configured root and pushes the rows into the `prompts`
// dock. A failed push is reported on stderr, not fatal: the next event repaints.
func refreshDock() error {
	root := settingRoot()
	dir, err := moduleStateDir()
	if err != nil {
		return err
	}
	rows := buildRows(root, readExpanded(dir), readSelected(dir), availableEditors(),
		agentsInWorkspace(listAgents(), currentWorkspaceId()))
	input, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	// Push the title too: without one, luvus titles the dock from its id.
	res := luvus([]string{"ui", "dock", "push", "--id", dockID, "--title", dockTitle}, string(input))
	if res.Status != 0 {
		fmt.Fprintf(os.Stderr, "ui dock push failed (status %d): %s\n", res.Status, res.detail())
	}
	return nil
}
