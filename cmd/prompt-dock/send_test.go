// send_test.go — tests for send, ported from lib.test.ts (the send tests).
// They call the verb function in the test process with t.Setenv and a fake
// luvus that logs each call's argv, then check the returned error and the
// call log. They do not start a child process (design D16).
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runSendInTest sets the environment up for one send run, calls the verb
// function, and returns the returned error and the fake luvus call log.
func runSendInTest(t *testing.T, value string, root string) (error, string) {
	t.Helper()
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_ROW_VALUE", value)
	err := send(value)
	raw, err2 := os.ReadFile(log)
	if err2 != nil {
		if os.IsNotExist(err2) {
			return err, ""
		}
		t.Fatal(err2)
	}
	return err, string(raw)
}

func TestSendRefusesAnAgentWithNoTerminalIdBeforeAnyAgentPromptCall(t *testing.T) {
	skipOnWindows(t)
	root := tempLibrary(t)
	err, calls := runSendInTest(t, "9||plan.md", root)
	if err == nil {
		t.Error("send(9||plan.md) = nil, want an error")
	}
	if want := "ui toast Send to pane 9 failed: no terminal id"; !strings.Contains(calls, want) {
		t.Errorf("call log missing %q:\n%s", want, calls)
	}
	if strings.Contains(calls, "agent prompt") {
		t.Errorf("agent prompt was called before the terminal id guard:\n%s", calls)
	}
}

func TestSendRefusesAnEmptyPromptFileAndNamesTheAgent(t *testing.T) {
	skipOnWindows(t)
	root := tempLibrary(t)
	if err := os.WriteFile(filepath.Join(root, "empty.md"), []byte("  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err, calls := runSendInTest(t, "9|"+strings.Repeat("d", 32)+"|empty.md", root)
	if err == nil {
		t.Error("send(empty.md) = nil, want an error")
	}
	if want := `ui toast Send to pane 9 failed: "empty" is empty.`; !strings.Contains(calls, want) {
		t.Errorf("call log missing %q:\n%s", want, calls)
	}
	if strings.Contains(calls, "agent prompt") {
		t.Errorf("agent prompt was called for an empty prompt:\n%s", calls)
	}
}

func TestSendReadFailureToastNamesTheAgent(t *testing.T) {
	skipOnWindows(t)
	root := tempLibrary(t)
	err, calls := runSendInTest(t, "9|"+strings.Repeat("d", 32)+"|missing.md", root)
	if err == nil {
		t.Error("send(missing.md) = nil, want an error")
	}
	if want := `ui toast Send to pane 9 failed: couldn't read "missing".`; !strings.Contains(calls, want) {
		t.Errorf("call log missing %q:\n%s", want, calls)
	}
}

// Regression: a prompt starting with `-` (YAML frontmatter `---`, a leading
// bullet) was parsed by `luvus agent prompt` as an option and never submitted.
func TestSendPassesThePromptAfterDashDashSoDashLeadingTextIsNotAnOption(t *testing.T) {
	skipOnWindows(t)
	root := tempLibrary(t)
	body := "---\nallowed-tools: Read\n---\n\nbody line\n"
	if err := os.WriteFile(filepath.Join(root, "shot.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "9|"+strings.Repeat("d", 32)+"|shot.md")

	if err := send("9|" + strings.Repeat("d", 32) + "|shot.md"); err != nil {
		t.Fatalf("send(shot.md) = %v, want nil", err)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	calls := string(raw)
	if want := "agent prompt 9 --terminal-id " + strings.Repeat("d", 32) + " -- ---"; !strings.Contains(calls, want) {
		t.Errorf("call log missing the dash-dash separated prompt %q:\n%s", want, calls)
	}
	if !strings.Contains(calls, "body line") {
		t.Errorf("the whole text did not arrive as one element:\n%s", calls)
	}
	if want := `Sent "shot" to pane 9.`; !strings.Contains(calls, want) {
		t.Errorf("call log missing the success toast %q:\n%s", want, calls)
	}
}
