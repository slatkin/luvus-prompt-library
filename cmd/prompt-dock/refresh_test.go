// refresh_test.go — ported from lib.test.ts (run there with `node --test`).
// The tests here cover refresh.ts helpers (tasks 3.1); the lib.ts tests live
// in lib_test.go.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// skipOnWindows skips tests that run a fake luvus shell script (not on
// Windows), as in lib.test.ts.
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake luvus is a shell script")
	}
}

func TestBranchPrefixesChildrenOfATopLevelFolderDrawNoPipe(t *testing.T) {
	root := tempLibrary(t)
	rows := scanTree(root, map[string]bool{"review": true})
	prefixes := branchPrefixes(rows)
	type relPrefix struct {
		rel    string
		prefix string
	}
	want := []relPrefix{
		{"review", ""},              // root rows carry no prefix
		{"review/rust.md", "  ├─ "}, // blank guide: `review` is a root row, so it has no line to join
		{"review/ts.md", "  └─ "},
		{"plan.md", ""},
	}
	got := make([]relPrefix, len(rows))
	for i, r := range rows {
		got[i] = relPrefix{r.Rel, prefixes[i]}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("branchPrefixes = %+v, want %+v", got, want)
	}
}

func TestBranchPrefixesOneGuideCellPerAncestorWithAPipeBelowDepth0(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"a/b/deep.md": "deep\n",
		"a/c.md":      "c\n",
		"z.md":        "z\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rows := scanTree(root, map[string]bool{"a": true, "a/b": true})
	prefixes := branchPrefixes(rows)
	type relPrefix struct {
		rel    string
		prefix string
	}
	want := []relPrefix{
		{"a", ""},
		{"a/b", "  ├─ "},           // `a` has no connector, so its guide cell stays blank
		{"a/b/deep.md", "  │ └─ "}, // `b` is joined to `c.md`, so its column keeps the pipe
		{"a/c.md", "  └─ "},
		{"z.md", ""},
	}
	got := make([]relPrefix, len(rows))
	for i, r := range rows {
		got[i] = relPrefix{r.Rel, prefixes[i]}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("branchPrefixes = %+v, want %+v", got, want)
	}
}

func TestToDockRowOnlyTheGlyphAndNameTakeAToneNeverTheBranchPrefix(t *testing.T) {
	prompt := TreeRow{Rel: "a/b.md", Name: "b", Depth: 1, Kind: "file", IsLast: true}
	// Not selected: no span has a tone, so the whole row takes the default colour.
	plain := toDockRow(prompt, "  └─ ", false, false, nil, nil)
	if want := "  └─ \uF0B79 b"; plain.Text != want {
		t.Errorf("plain.Text = %q, want %q", plain.Text, want)
	}
	wantSpans := []DockSpan{{Text: "  └─ "}, {Text: "\uF0B79 b"}}
	if !reflect.DeepEqual(plain.Spans, wantSpans) {
		t.Errorf("plain.Spans = %+v, want %+v", plain.Spans, wantSpans)
	}
	// Selected: the accent sits on the glyph and name, and the prefix keeps none.
	selected := toDockRow(prompt, "  └─ ", false, true, nil, nil)
	wantSelected := []DockSpan{{Text: "  └─ "}, {Text: "\uF0B79 b", Tone: "accent"}}
	if !reflect.DeepEqual(selected.Spans, wantSelected) {
		t.Errorf("selected.Spans = %+v, want %+v", selected.Spans, wantSelected)
	}
	// A folder label is accent too, and a root row sends no prefix span at all.
	folder := TreeRow{Rel: "a", Name: "a", Kind: "folder"}
	collapsed := toDockRow(folder, "", false, false, nil, nil)
	wantFolder := []DockSpan{{Text: "\uF024B a", Tone: "accent"}}
	if !reflect.DeepEqual(collapsed.Spans, wantFolder) {
		t.Errorf("collapsed folder spans = %+v, want %+v", collapsed.Spans, wantFolder)
	}
	open := toDockRow(folder, "", true, false, nil, nil)
	wantOpen := []DockSpan{{Text: "\uF0770 a", Tone: "accent"}}
	if !reflect.DeepEqual(open.Spans, wantOpen) {
		t.Errorf("open folder spans = %+v, want %+v", open.Spans, wantOpen)
	}
}

func TestEditMenuEntriesUseTheEditorRelpathEncoding(t *testing.T) {
	// Editors are passed in, so the test does not depend on what is on PATH.
	entries := editMenuEntries("review/plan.md", []Editor{
		{Bin: "zed", Label: "zed"},
		{Bin: "code", Label: "VS Code"},
	})
	type triple struct{ title, action, value string }
	want := []triple{
		{"Edit in zed", "edit-external", "zed|review/plan.md"},
		{"Edit in VS Code", "edit-external", "code|review/plan.md"},
	}
	got := make([]triple, len(entries))
	for i, e := range entries {
		got[i] = triple{e.Title, e.Action, e.Value}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("editMenuEntries = %+v, want %+v", got, want)
	}
}

// fakeLuvus builds a fake `luvus` binary whose `agent list` prints the given
// JSON. It returns the script path.
func fakeLuvus(t *testing.T, stdout string) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-luvus")
	body := "#!/bin/sh\ncat <<'JSON'\n" + stdout + "\nJSON\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func TestSendMenuEntriesOneSendToEntryPerAgentNoTerminalIdEncodesAsEmpty(t *testing.T) {
	skipOnWindows(t)
	hex := strings.Repeat("c", 32)
	reply, err := json.Marshal(map[string]any{
		"id": "1",
		"result": map[string]any{
			"agents": []map[string]any{
				{
					"pane": "9", "agent": "pi", "terminal_id": hex, "name": "pipi",
					"status": "idle", "workspace_name": "pi-luvus",
				},
				{
					"pane": "10", "agent": "claude", "terminal_id": nil, "name": nil,
					"status": "working", "workspace_name": "mbv",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUVUS_BIN_PATH", fakeLuvus(t, string(reply)))
	entries := sendMenuEntries("plan.md", listAgents())
	type pair struct{ action, value string }
	want := []pair{
		{"send", "9|" + hex + "|plan.md"},
		{"send", "10||plan.md"},
	}
	got := make([]pair, len(entries))
	for i, e := range entries {
		got[i] = pair{e.Action, e.Value}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sendMenuEntries = %+v, want %+v", got, want)
	}
	if !strings.Contains(entries[0].Title, "pipi") {
		t.Errorf("entries[0].Title = %q, want it to name the alias", entries[0].Title)
	}
	if !strings.HasPrefix(entries[1].Title, "Send to ") {
		t.Errorf("entries[1].Title = %q, want the Send to prefix", entries[1].Title)
	}
}
