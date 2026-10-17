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

// fixedAgentReply is the `agent list` reply the golden rows were recorded
// with (task 3.2): one live agent in workspace 3.
const fixedAgentReply = `{"id":"1","result":{"agents":[{"pane":"9","agent":"pi","terminal_id":"cccccccccccccccccccccccccccccccc","name":"pipi","status":"idle","workspace_name":"pi-luvus","workspace":"3"}]}}`

// TestBuildRowsMatchesTheGoldenRowsFromTheTSRefresh replays the golden check
// of task 3.2: refresh.ts, run against a fixed library (the tempLibrary
// fixture), state (one expanded folder, plan.md selected), editor list [zed],
// and agent list, pushed the rows in testdata/refresh-golden.json. The Go
// buildRows must decode to the same rows. The check compares the decoded
// JSON, not the bytes, because Go escapes `<`, `>`, and `&` differently
// (design D16).
func TestBuildRowsMatchesTheGoldenRowsFromTheTSRefresh(t *testing.T) {
	root := tempLibrary(t)
	dir := t.TempDir()
	// The state files in the format the TS code writes (design D14).
	if err := os.WriteFile(filepath.Join(dir, "expanded.json"), []byte("[\n  \"review\"\n]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "selected.json"), []byte("\"plan.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rows := buildRows(root, readExpanded(dir), readSelected(dir),
		[]Editor{{Bin: "zed", Label: "zed"}}, parseAgents(fixedAgentReply))

	goldenRaw, err := os.ReadFile(filepath.Join("testdata", "refresh-golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var golden []any
	if err := json.Unmarshal(goldenRaw, &golden); err != nil {
		t.Fatalf("golden file does not decode: %v", err)
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("built rows do not decode: %v", err)
	}
	if !reflect.DeepEqual(golden, decoded) {
		t.Errorf("built rows differ from the golden rows\n built: %s\n golden: %s", encoded, goldenRaw)
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
	if want := "  └─ \U000F0B79 b"; plain.Text != want {
		t.Errorf("plain.Text = %q, want %q", plain.Text, want)
	}
	wantSpans := []DockSpan{{Text: "  └─ "}, {Text: "\U000F0B79 b"}}
	if !reflect.DeepEqual(plain.Spans, wantSpans) {
		t.Errorf("plain.Spans = %+v, want %+v", plain.Spans, wantSpans)
	}
	// Selected: the accent sits on the glyph and name, and the prefix keeps none.
	selected := toDockRow(prompt, "  └─ ", false, true, nil, nil)
	wantSelected := []DockSpan{{Text: "  └─ "}, {Text: "\U000F0B79 b", Tone: "accent"}}
	if !reflect.DeepEqual(selected.Spans, wantSelected) {
		t.Errorf("selected.Spans = %+v, want %+v", selected.Spans, wantSelected)
	}
	// A folder label is accent too, and a root row sends no prefix span at all.
	folder := TreeRow{Rel: "a", Name: "a", Kind: "folder"}
	collapsed := toDockRow(folder, "", false, false, nil, nil)
	wantFolder := []DockSpan{{Text: "\U000F024B a", Tone: "accent"}}
	if !reflect.DeepEqual(collapsed.Spans, wantFolder) {
		t.Errorf("collapsed folder spans = %+v, want %+v", collapsed.Spans, wantFolder)
	}
	open := toDockRow(folder, "", true, false, nil, nil)
	wantOpen := []DockSpan{{Text: "\U000F0770 a", Tone: "accent"}}
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

// fakeLuvusLog builds a fake `luvus` binary that logs each call's argv, one
// per line, to a log file, and exits 0 with no output. It returns the log
// path, so tests can count the calls they caused.
func fakeLuvusLog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	script := filepath.Join(dir, "fake-luvus")
	body := "#!/bin/sh\nPATH=/usr/bin:/bin\necho \"$@\" >> " + log + "\ncat > /dev/null\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return log
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
