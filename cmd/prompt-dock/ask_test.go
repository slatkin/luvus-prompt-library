// ask_test.go — the one panel test (design D8): the real bubbletea program
// runs on scripted input with no TTY (tea.WithInput via the ask streams,
// io.Discard for the output), and the fake luvus call log shows what the
// submit and the cancel did. The handoff cases that never reach the program
// are plain verb tests.
package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runAskInTest sets the environment up for one ask run, writes the handoff
// file when handoff is not empty, runs the panel on the scripted keys, and
// returns the fake luvus call log, the module state dir, and the verb error.
func runAskInTest(t *testing.T, handoff string, keys string) (string, string, error) {
	t.Helper()
	log := fakeLuvusLog(t)
	stateDir := t.TempDir()
	root := tempLibrary(t)
	if err := os.WriteFile(filepath.Join(root, "shot.md"), []byte("---\nargument-hint: \"[change-name]\"\n---\nbody $ARGUMENTS\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_STATE_DIR", stateDir)
	if handoff != "" {
		if err := os.WriteFile(filepath.Join(stateDir, "ask.json"), []byte(handoff), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	err := ask(strings.NewReader(keys), io.Discard)
	raw, readErr := os.ReadFile(log)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	return string(raw), stateDir, err
}

// Submit: the typed value is filled in and sent, then the focus goes back to
// the pane that had it (design D7), and the handoff file is gone.
func TestAskSubmitsTheFilledPromptAndReturnsTheFocus(t *testing.T) {
	skipOnWindows(t)
	calls, stateDir, err := runAskInTest(t,
		`{"value":"9|`+strings.Repeat("d", 32)+`|shot.md","return_pane":"3"}`,
		"my-change\r")
	if err != nil {
		t.Fatalf("ask(submit) = %v, want nil", err)
	}
	if want := "agent prompt 9 --terminal-id " + strings.Repeat("d", 32) + " -- body my-change"; !strings.Contains(calls, want) {
		t.Errorf("call log missing the filled prompt %q:\n%s", want, calls)
	}
	if want := "pane focus 3"; !strings.Contains(calls, want) {
		t.Errorf("call log missing the focus call %q:\n%s", want, calls)
	}
	if strings.Index(calls, "agent prompt") > strings.Index(calls, "pane focus") {
		t.Errorf("the focus call came before the send:\n%s", calls)
	}
	if _, statErr := os.Stat(filepath.Join(stateDir, "ask.json")); !os.IsNotExist(statErr) {
		t.Errorf("ask.json survived the run: %v", statErr)
	}
}

// Cancel: nothing reaches any agent, and the focus still goes back (spec:
// cancel sends nothing; design D7).
func TestAskCancelSendsNothingAndReturnsTheFocus(t *testing.T) {
	skipOnWindows(t)
	calls, stateDir, err := runAskInTest(t,
		`{"value":"9|`+strings.Repeat("d", 32)+`|shot.md","return_pane":"3"}`,
		"my-change\x1b")
	if err != nil {
		t.Fatalf("ask(cancel) = %v, want nil", err)
	}
	if strings.Contains(calls, "agent prompt") {
		t.Errorf("agent prompt was called on cancel:\n%s", calls)
	}
	if want := "pane focus 3"; !strings.Contains(calls, want) {
		t.Errorf("call log missing the focus call %q:\n%s", want, calls)
	}
	if _, statErr := os.Stat(filepath.Join(stateDir, "ask.json")); !os.IsNotExist(statErr) {
		t.Errorf("ask.json survived the run: %v", statErr)
	}
}

// A left-over panel (no handoff file) exits with no luvus call at all (spec:
// a left-over panel sends nothing; design D2).
func TestAskWithNoHandoffCallsNothing(t *testing.T) {
	skipOnWindows(t)
	calls, _, err := runAskInTest(t, "", "my-change\r")
	if err != nil {
		t.Fatalf("ask(no handoff) = %v, want nil", err)
	}
	if calls != "" {
		t.Errorf("a panel with no handoff made luvus calls:\n%s", calls)
	}
}

// A handoff file that is not valid JSON exits 0 with no luvus call (design
// D2).
func TestAskWithABadHandoffCallsNothing(t *testing.T) {
	skipOnWindows(t)
	calls, stateDir, err := runAskInTest(t, "{not json", "my-change\r")
	if err != nil {
		t.Fatalf("ask(bad handoff) = %v, want nil", err)
	}
	if calls != "" {
		t.Errorf("a panel with a bad handoff made luvus calls:\n%s", calls)
	}
	if _, statErr := os.Stat(filepath.Join(stateDir, "ask.json")); !os.IsNotExist(statErr) {
		t.Errorf("a bad handoff file survived the run: %v", statErr)
	}
}
