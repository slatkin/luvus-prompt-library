// lib_test.go — ported from lib.test.ts (run there with `node --test`). The
// tests that cover edit.ts, refresh.ts, and send.ts helpers are ported with
// those files (tasks 3.1, 4.1, 5.1); this file holds the lib.ts tests.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// posixOnly skips tests that create symlinks (not on Windows).
func posixOnly(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a POSIX filesystem")
	}
}

func TestExpandHomeExpandsALeadingTildeToTheHomeFolder(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got, want := expandHome("~/.prompts"), filepath.Join(home, ".prompts"); got != want {
		t.Errorf("expandHome(~/.prompts) = %q, want %q", got, want)
	}
	if got, want := expandHome("~"), home; got != want {
		t.Errorf("expandHome(~) = %q, want %q", got, want)
	}
	if got, want := expandHome(`~\prompts`), filepath.Join(home, "prompts"); got != want {
		t.Errorf(`expandHome(~\prompts) = %q, want %q`, got, want)
	}
}

func TestExpandHomeLeavesPlainAndAbsolutePathsAlone(t *testing.T) {
	if got, want := expandHome("prompts"), "prompts"; got != want {
		t.Errorf("expandHome(prompts) = %q, want %q", got, want)
	}
	if got, want := expandHome("/tmp/prompts"), "/tmp/prompts"; got != want {
		t.Errorf("expandHome(/tmp/prompts) = %q, want %q", got, want)
	}
}

func TestResolveRootDefaultsAnEmptySettingToHomePrompts(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got, want := resolveRoot(""), filepath.Join(home, ".prompts"); got != want {
		t.Errorf("resolveRoot() = %q, want %q", got, want)
	}
}

func TestResolveRootReadsTheRootSettingAndExpandsTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got, want := resolveRoot("~/notes"), filepath.Join(home, "notes"); got != want {
		t.Errorf("resolveRoot(~/notes) = %q, want %q", got, want)
	}
	if got, want := resolveRoot("  ~/notes  "), filepath.Join(home, "notes"); got != want {
		t.Errorf("resolveRoot(  ~/notes  ) = %q, want %q", got, want)
	}
	if got, want := resolveRoot("/tmp/prompts"), filepath.Clean("/tmp/prompts"); got != want {
		t.Errorf("resolveRoot(/tmp/prompts) = %q, want %q", got, want)
	}
}

func TestResolveUnderRootResolvesRelpathsInsideTheRoot(t *testing.T) {
	root := filepath.Clean("/tmp/library")
	if got, want := resolveUnderRoot(root, "plan.md"), filepath.Join(root, "plan.md"); got != want {
		t.Errorf("resolveUnderRoot(plan.md) = %q, want %q", got, want)
	}
	if got, want := resolveUnderRoot(root, "review/rust.md"), filepath.Join(root, "review", "rust.md"); got != want {
		t.Errorf("resolveUnderRoot(review/rust.md) = %q, want %q", got, want)
	}
}

func TestResolveUnderRootRejectsARelpathThatEscapesTheRoot(t *testing.T) {
	for _, rel := range []string{"../x.md", "review/../../x.md", "/etc/passwd"} {
		if got := resolveUnderRoot("/tmp/library", rel); got != "" {
			t.Errorf("resolveUnderRoot(%q) = %q, want empty", rel, got)
		}
	}
}

// tempLibrary builds a temp prompt library matching the spec scenarios,
// removed when the test ends.
func tempLibrary(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"plan.md":        "plan\n",
		"review/rust.md": "rust\n",
		"review/ts.md":   "ts\n",
		"notes.txt":      "not a prompt\n",
		".hidden.md":     "hidden\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestScanTreeNestedLibraryFoldersBeforeFilesSortedIndented(t *testing.T) {
	root := tempLibrary(t)
	rows := scanTree(root, map[string]bool{"review": true})
	want := []TreeRow{
		{Rel: "review", Name: "review", Kind: "folder"},
		{Rel: "review/rust.md", Name: "rust", Depth: 1, Kind: "file"},
		{Rel: "review/ts.md", Name: "ts", Depth: 1, Kind: "file"},
		{Rel: "plan.md", Name: "plan", Kind: "file"},
	}
	got := make([]TreeRow, len(rows))
	for i, r := range rows {
		got[i] = TreeRow{Rel: r.Rel, Name: r.Name, Depth: r.Depth, Kind: r.Kind}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("scanTree rows = %+v, want %+v", got, want)
	}
}

func TestScanTreeACollapsedFolderLeavesItsChildrenOut(t *testing.T) {
	root := tempLibrary(t)
	rows := scanTree(root, map[string]bool{})
	want := []TreeRow{
		{Rel: "review", Kind: "folder"},
		{Rel: "plan.md", Kind: "file"},
	}
	got := make([]TreeRow, len(rows))
	for i, r := range rows {
		got[i] = TreeRow{Rel: r.Rel, Depth: r.Depth, Kind: r.Kind}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("scanTree rows = %+v, want %+v", got, want)
	}
}

func TestScanTreeSymlinksShowAsTheirTargetsAndAFolderLoopStops(t *testing.T) {
	posixOnly(t)
	root := tempLibrary(t)
	if err := os.Symlink(filepath.Join(root, "plan.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "review"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "review", "up")); err != nil {
		t.Fatal(err)
	}
	rows := scanTree(root, map[string]bool{"alias": true, "review": true, "review/up": true})
	want := []TreeRow{
		{Rel: "alias", Depth: 0, Kind: "folder"},
		{Rel: "alias/up", Depth: 1, Kind: "folder"},
		{Rel: "alias/rust.md", Depth: 1, Kind: "file"},
		{Rel: "alias/ts.md", Depth: 1, Kind: "file"},
		{Rel: "review", Depth: 0, Kind: "folder"},
		{Rel: "review/up", Depth: 1, Kind: "folder"},
		{Rel: "review/rust.md", Depth: 1, Kind: "file"},
		{Rel: "review/ts.md", Depth: 1, Kind: "file"},
		{Rel: "linked.md", Depth: 0, Kind: "file"},
		{Rel: "plan.md", Depth: 0, Kind: "file"},
	}
	got := make([]TreeRow, len(rows))
	for i, r := range rows {
		got[i] = TreeRow{Rel: r.Rel, Depth: r.Depth, Kind: r.Kind}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("scanTree rows = %+v, want %+v", got, want)
	}
}

func TestScanTreeIsLastMarksTheLastRowOfEachSiblingGroup(t *testing.T) {
	root := tempLibrary(t)
	rows := scanTree(root, map[string]bool{"review": true})
	type relIsLast struct {
		rel    string
		isLast bool
	}
	want := []relIsLast{
		{"review", false}, // plan.md follows it at the root
		{"review/rust.md", false},
		{"review/ts.md", true},
		{"plan.md", true},
	}
	got := make([]relIsLast, len(rows))
	for i, r := range rows {
		got[i] = relIsLast{r.Rel, r.IsLast}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("isLast rows = %+v, want %+v", got, want)
	}
}

func TestReadExpandedTreatsAMissingOrCorruptFileAsEmpty(t *testing.T) {
	empty := t.TempDir()
	if got := readExpanded(empty); len(got) != 0 {
		t.Errorf("readExpanded(missing) = %v, want empty", got)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "expanded.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readExpanded(dir); len(got) != 0 {
		t.Errorf("readExpanded(corrupt) = %v, want empty", got)
	}
}

func TestWriteExpandedThenReadExpandedRoundTripsTheFolderSet(t *testing.T) {
	dir := t.TempDir()
	if err := writeExpanded(dir, map[string]bool{"review": true, "a/b": true}); err != nil {
		t.Fatal(err)
	}
	if got := readExpanded(dir); !got["review"] || !got["a/b"] {
		t.Errorf("readExpanded(round-trip) = %v, want review and a/b", got)
	}
	// A corrupt file after a good write still reads as empty (design D5).
	if err := os.WriteFile(filepath.Join(dir, "expanded.json"), []byte("null"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readExpanded(dir); len(got) != 0 {
		t.Errorf("readExpanded(null) = %v, want empty", got)
	}
}

func TestWriteSelectedThenReadSelectedRoundTripsAndJunkReadsAsNoSelection(t *testing.T) {
	dir := t.TempDir()
	if got := readSelected(dir); got != "" {
		t.Errorf("readSelected(missing) = %q, want empty", got)
	}
	if err := writeSelected(dir, "review/plan.md"); err != nil {
		t.Fatal(err)
	}
	if got, want := readSelected(dir), "review/plan.md"; got != want {
		t.Errorf("readSelected(round-trip) = %q, want %q", got, want)
	}
	// A corrupt file after a good write still reads as no selection (design D9).
	if err := os.WriteFile(filepath.Join(dir, "selected.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readSelected(dir); got != "" {
		t.Errorf("readSelected(corrupt) = %q, want empty", got)
	}
}

// writeExpanded writes an empty set as `[]`, never `null`, so the TS code and
// older states keep reading the file (design D14: state files keep their
// format).
func TestWriteExpandedWritesAnEmptySetAsEmptyArray(t *testing.T) {
	dir := t.TempDir()
	if err := writeExpanded(dir, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "expanded.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(raw), "[]\n"; got != want {
		t.Errorf("empty expanded.json = %q, want %q", got, want)
	}
}

// The state files are written by the TS code today; the fixture text below is
// what `JSON.stringify(value, null, 2) + "\n"` produces. The Go code must
// read those files back the same (design D14: state files keep their format).
func TestStateFilesWrittenByTheTSCodeReadBackTheSame(t *testing.T) {
	dir := t.TempDir()
	expandedFixture := "[\n  \"review\",\n  \"a/b\"\n]\n"
	if err := os.WriteFile(filepath.Join(dir, "expanded.json"), []byte(expandedFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readExpanded(dir)
	want := map[string]bool{"review": true, "a/b": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readExpanded(TS fixture) = %v, want %v", got, want)
	}
	selectedFixture := "\"review/plan.md\"\n"
	if err := os.WriteFile(filepath.Join(dir, "selected.json"), []byte(selectedFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := readSelected(dir), "review/plan.md"; got != want {
		t.Errorf("readSelected(TS fixture) = %q, want %q", got, want)
	}
}

func TestEncodeSendValueThenSplitSendValueRoundTripsASendValue(t *testing.T) {
	value := encodeSendValue("118", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "plan.md")
	pane, terminalId, rel := splitSendValue(value)
	if pane != "118" || terminalId != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || rel != "plan.md" {
		t.Errorf("splitSendValue(%q) = (%q, %q, %q)", value, pane, terminalId, rel)
	}
}

func TestSplitSendValueSplitsOnTheFirstTwoPipeOnlySoPipeInARelpathWorks(t *testing.T) {
	cases := []struct{ value, pane, terminalId, rel string }{
		{"12|abc|review|plan.md", "12", "abc", "review|plan.md"},
		{"12|abc|plan.md", "12", "abc", "plan.md"},
		{"no-separators.md", "", "", "no-separators.md"},
	}
	for _, c := range cases {
		pane, terminalId, rel := splitSendValue(c.value)
		if pane != c.pane || terminalId != c.terminalId || rel != c.rel {
			t.Errorf("splitSendValue(%q) = (%q, %q, %q), want (%q, %q, %q)",
				c.value, pane, terminalId, rel, c.pane, c.terminalId, c.rel)
		}
	}
}

func TestTargetLabelANamedAgentShowsAliasKindAndPaneOnly(t *testing.T) {
	label := targetLabel(AgentInfo{
		Pane:       "9",
		Agent:      "pi",
		TerminalID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Name:       "pipi",
		Workspace:  "3",
	})
	if want := "pipi (pi) · pane 9"; label != want {
		t.Errorf("targetLabel() = %q, want %q", label, want)
	}
}

func TestTargetLabelAnUnnamedAgentShowsKindAndPaneWithoutTheParenthetical(t *testing.T) {
	label := targetLabel(AgentInfo{
		Pane:       "118",
		Agent:      "claude",
		TerminalID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Name:       "",
		Workspace:  "2",
	})
	if want := "claude · pane 118"; label != want {
		t.Errorf("targetLabel() = %q, want %q", label, want)
	}
}

func TestParseAgentsReadsTheAgentArrayOutOfTheCLIEnvelope(t *testing.T) {
	stdout, err := json.Marshal(map[string]any{
		"id": "1",
		"result": map[string]any{
			"agents": []map[string]any{
				{
					"pane":        "9",
					"agent":       "pi",
					"terminal_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"name":        "pipi",
					"workspace":   "3",
				},
				{
					"pane":        "10",
					"agent":       "claude",
					"terminal_id": nil,
					"name":        nil,
					"workspace":   "2",
				},
				{"pane": "11"}, // no agent kind: dropped
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	agents := parseAgents(string(stdout))
	if len(agents) != 2 {
		t.Fatalf("parseAgents kept %d agents, want 2: %+v", len(agents), agents)
	}
	hex := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if got, want := agents[0].Pane, "9"; got != want {
		t.Errorf("pane = %q, want %q", got, want)
	}
	if got, want := agents[0].Agent, "pi"; got != want {
		t.Errorf("agent = %q, want %q", got, want)
	}
	if agents[0].TerminalID != hex {
		t.Errorf("terminal_id = %q, want %q", agents[0].TerminalID, hex)
	}
	if agents[0].Name != "pipi" {
		t.Errorf("name = %v, want pipi", agents[0].Name)
	}
	if got, want := agents[0].Workspace, "3"; got != want {
		t.Errorf("workspace = %q, want %q", got, want)
	}
	if got, want := agents[1].Pane, "10"; got != want {
		t.Errorf("pane = %q, want %q", got, want)
	}
	if got, want := agents[1].Agent, "claude"; got != want {
		t.Errorf("agent = %q, want %q", got, want)
	}
	if agents[1].TerminalID != "" {
		t.Errorf("terminal_id = %q, want empty", agents[1].TerminalID)
	}
	if agents[1].Name != "" {
		t.Errorf("name = %q, want empty", agents[1].Name)
	}
	if got, want := agents[1].Workspace, "2"; got != want {
		t.Errorf("workspace = %q, want %q", got, want)
	}
}

func TestParseAgentsReturnsNoAgentsForOtherShapesNullsAndGarbage(t *testing.T) {
	agent := map[string]any{
		"pane": "7", "agent": "codex",
		"terminal_id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"name":        nil, "status": "idle", "workspace_name": "w",
	}
	encoded := func(v any) string {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for name, stdout := range map[string]string{
		"agents outside result": encoded(map[string]any{"agents": []any{agent}}),
		"array instead of env":  encoded([]any{agent}),
		"garbage":               "not json",
		"null":                  "null",
		"null and number rows":  encoded(map[string]any{"result": map[string]any{"agents": []any{nil, 5}}}),
		"other envelope":        encoded(map[string]any{"other": true}),
	} {
		if got := parseAgents(stdout); len(got) != 0 {
			t.Errorf("parseAgents(%s) = %+v, want none", name, got)
		}
	}
}

// twoWorkspaceAgents returns two agents in different workspaces, for the
// workspace filter (design D11).
func twoWorkspaceAgents() []AgentInfo {
	return []AgentInfo{
		{Pane: "9", Agent: "pi", Workspace: "3"},
		{Pane: "6", Agent: "claude", Workspace: "2"},
	}
}

func TestAgentsInWorkspaceKeepsOnlyAgentsInTheFocusedWorkspace(t *testing.T) {
	kept := agentsInWorkspace(twoWorkspaceAgents(), "3")
	if len(kept) != 1 || kept[0].Pane != "9" {
		t.Errorf("agentsInWorkspace(3) = %+v, want only pane 9", kept)
	}
}

func TestAgentsInWorkspaceKeepsEveryAgentWhenTheWorkspaceIsUnknown(t *testing.T) {
	kept := agentsInWorkspace(twoWorkspaceAgents(), "")
	if len(kept) != 2 || kept[0].Pane != "9" || kept[1].Pane != "6" {
		t.Errorf("agentsInWorkspace(empty) = %+v, want both agents", kept)
	}
}
