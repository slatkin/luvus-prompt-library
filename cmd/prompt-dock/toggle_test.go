// toggle_test.go — tests for runToggle. The TS code had no tests for
// toggle.ts; these cover the behaviour task 3.3 asks for: a fake luvus shows
// that toggle adds and then removes a folder in expanded.json and pushes rows
// each time, and that a value that escapes the root is refused with a
// non-nil error before any luvus call.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countCalls reads a fake-luvus log and counts the calls of one command.
func countCalls(t *testing.T, log string, command string) int {
	t.Helper()
	raw, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return strings.Count(string(raw), command)
}

func TestRunToggleAddsThenRemovesAFolderInExpandedJSONAndPushesRowsEachTime(t *testing.T) {
	root := tempLibrary(t)
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", filepath.Dir(log))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "review")

	// First toggle: the collapsed folder expands.
	if err := runToggle(); err != nil {
		t.Fatalf("runToggle(add) = %v, want nil", err)
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(log), "expanded.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "[\n  \"review\"\n]\n"; string(raw) != want {
		t.Errorf("expanded.json after add = %q, want %q", raw, want)
	}
	if got := countCalls(t, log, "ui dock push"); got != 1 {
		t.Errorf("dock pushes after add = %d, want 1", got)
	}

	// Second toggle: the expanded folder collapses again.
	if err := runToggle(); err != nil {
		t.Fatalf("runToggle(remove) = %v, want nil", err)
	}
	raw, err = os.ReadFile(filepath.Join(filepath.Dir(log), "expanded.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "[]\n"; string(raw) != want {
		t.Errorf("expanded.json after remove = %q, want %q", raw, want)
	}
	if got := countCalls(t, log, "ui dock push"); got != 2 {
		t.Errorf("dock pushes after add and remove = %d, want 2", got)
	}
}

func TestRunToggleRejectsAValueThatEscapesTheRoot(t *testing.T) {
	root := tempLibrary(t)
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", filepath.Dir(log))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "../x.md")

	if err := runToggle(); err == nil {
		t.Error("runToggle(../x.md) = nil, want an error")
	}
	if got := countCalls(t, log, "luvus"); got != 0 {
		t.Errorf("luvus calls after a refused toggle = %d, want 0", got)
	}
}
