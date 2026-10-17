// edit_test.go — tests for editInline and editExternal, ported from
// lib.test.ts (the editing-helper tests) plus fake-luvus tests for the
// behaviour task 4.1 asks for: the `pane split` reply carrying `pane` as a
// string, as a number and in a flat object, and the pane close after a failed
// run. The external-mode tests sit in edit_external_test.go (task 4.2).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// splitReplyScript writes a fake luvus that logs each call's argv, and
// replies to `pane split` with the given JSON. When runStatus is non-zero,
// `pane run` fails with that status. It returns the log path.
func splitReplyScript(t *testing.T, reply string, runStatus int) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	script := filepath.Join(dir, "fake-luvus")
	body := "#!/bin/sh\nPATH=/usr/bin:/bin\necho \"$@\" >> " + log + "\ncat > /dev/null\n" +
		"if [ \"$1 $2\" = \"pane split\" ]; then printf '%s\\n' '" + reply + "'; exit 0; fi\n" +
		"if [ \"$1 $2\" = \"pane run\" ]; then exit " + fmt.Sprintf("%d", runStatus) + "; fi\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return log
}

// fakeHomeWithConfig writes a ~/.luvus/config.json with the given
// layout.file_open value in a temp HOME, and returns the home path.
func fakeHomeWithConfig(t *testing.T, fileOpen string) string {
	t.Helper()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".luvus"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`{"layout": {"file_open": %q}}`, fileOpen)
	if err := os.WriteFile(filepath.Join(home, ".luvus", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestResolveInlineEditorKeepsTheConfiguredEditor(t *testing.T) {
	if got, want := resolveInlineEditor("hx"), "hx"; got != want {
		t.Errorf("resolveInlineEditor(hx) = %q, want %q", got, want)
	}
	if got, want := resolveInlineEditor("emacs -nw"), "emacs -nw"; got != want {
		t.Errorf("resolveInlineEditor(emacs -nw) = %q, want %q", got, want)
	}
}

func TestResolveInlineEditorFallsBackToNvimWhenEmptyReadonlyOrMissing(t *testing.T) {
	for _, raw := range []string{"", "  ", "readonly"} {
		if got := resolveInlineEditor(raw); got != "nvim" {
			t.Errorf("resolveInlineEditor(%q) = %q, want nvim", raw, got)
		}
	}
}

// runEditInlineInTest sets the environment up for one edit-inline run.
func runEditInlineInTest(t *testing.T, log string, value string) error {
	t.Helper()
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", tempLibrary(t))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", value)
	t.Setenv("HOME", t.TempDir())
	return runEditInline()
}

func TestEditInlineAcceptsPaneAsStringNumberAndFlatObject(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reply string
	}{
		{"envelope with a string pane", `{"id":"1","result":{"pane":"42"}}`},
		{"envelope with a number pane", `{"id":"1","result":{"pane":42}}`},
		{"flat object", `{"pane":"42"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := splitReplyScript(t, tc.reply, 0)
			if err := runEditInlineInTest(t, log, "plan.md"); err != nil {
				t.Fatalf("runEditInline(%s) = %v, want nil", tc.reply, err)
			}
			raw, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"pane split --no-focus", "pane run 42"} {
				if !strings.Contains(string(raw), want) {
					t.Errorf("call log missing %q:\n%s", want, raw)
				}
			}
		})
	}
}

func TestEditInlineRunsTheConfiguredEditorOnTheFile(t *testing.T) {
	log := splitReplyScript(t, `{"result":{"pane":"7"}}`, 0)
	home := fakeHomeWithConfig(t, "hx")
	root := tempLibrary(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", root)
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "plan.md")
	t.Setenv("HOME", home)

	if err := runEditInline(); err != nil {
		t.Fatalf("runEditInline() = %v, want nil", err)
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if want := "pane run 7 hx '" + filepath.Join(root, "plan.md") + "'"; !strings.Contains(string(raw), want) {
		t.Errorf("call log missing the configured editor run %q:\n%s", want, raw)
	}
}

func TestEditInlineClosesThePaneAfterAFailedRun(t *testing.T) {
	log := splitReplyScript(t, `{"result":{"pane":"42"}}`, 3)
	if err := runEditInlineInTest(t, log, "plan.md"); err == nil {
		t.Fatal("runEditInline() = nil, want an error when pane run fails")
	}
	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pane close 42", "ui toast Couldn't start the editor in the new pane. Please try again."} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("call log missing %q:\n%s", want, raw)
		}
	}
}

func TestEditInlineRejectsQuoteBackslashAndLineBreak(t *testing.T) {
	for _, value := range []string{"it's.md", `back\slash.md`, "line\nbreak.md"} {
		log := splitReplyScript(t, `{"result":{"pane":"42"}}`, 0)
		if err := runEditInlineInTest(t, log, value); err == nil {
			t.Errorf("runEditInline(%q) = nil, want an error", value)
		}
		if got := countCalls(t, log, "pane split"); got != 0 {
			t.Errorf("pane split calls for %q = %d, want 0", value, got)
		}
	}
}
