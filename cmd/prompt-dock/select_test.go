// select_test.go — tests for runSelect. The TS code had no tests for
// select.ts; these cover the behaviour task 3.3 asks for: a fake luvus shows
// that select writes selected.json and pushes rows, that selecting the
// already selected prompt pushes nothing, and that a value that escapes the
// root is refused with a non-nil error before any luvus call.
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSelectWritesSelectedJSONAndPushesRows(t *testing.T) {
	root := tempLibrary(t)
	log := fakeLuvusLog(t)
	stateDir := filepath.Dir(log)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(stateDir, "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", stateDir)
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "plan.md")

	if err := runSelect(); err != nil {
		t.Fatalf("runSelect() = %v, want nil", err)
	}
	raw, err := os.ReadFile(filepath.Join(stateDir, "selected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "\"plan.md\"\n"; string(raw) != want {
		t.Errorf("selected.json = %q, want %q", raw, want)
	}
	if got := countCalls(t, log, "ui dock push"); got != 1 {
		t.Errorf("dock pushes after select = %d, want 1", got)
	}
}

func TestRunSelectOfTheAlreadySelectedPromptPushesNothing(t *testing.T) {
	root := tempLibrary(t)
	log := fakeLuvusLog(t)
	stateDir := filepath.Dir(log)
	if err := os.WriteFile(filepath.Join(stateDir, "selected.json"), []byte("\"plan.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(stateDir, "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", stateDir)
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "plan.md")

	if err := runSelect(); err != nil {
		t.Fatalf("runSelect(already selected) = %v, want nil", err)
	}
	if got := countCalls(t, log, "ui dock push"); got != 0 {
		t.Errorf("dock pushes after re-select = %d, want 0", got)
	}
}

func TestRunSelectRejectsAValueThatEscapesTheRoot(t *testing.T) {
	root := tempLibrary(t)
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", filepath.Dir(log))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "../x.md")

	if err := runSelect(); err == nil {
		t.Error("runSelect(../x.md) = nil, want an error")
	}
	if got := countCalls(t, log, "luvus"); got != 0 {
		t.Errorf("luvus calls after a refused select = %d, want 0", got)
	}
}
