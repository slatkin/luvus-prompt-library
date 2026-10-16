// edit_external_test.go — tests for editExternal, ported from lib.test.ts
// (the splitExternalValue test) plus the behaviour task 4.2 asks for: an
// unknown editor and a malformed value return errors before anything runs.
package main

import (
	"path/filepath"
	"testing"
)

func TestSplitExternalValueSplitsOnTheFirstPipeOnly(t *testing.T) {
	editor, rel := splitExternalValue("code|weird|name.md")
	if got, want := editor, "code"; got != want {
		t.Errorf("editor = %q, want %q", got, want)
	}
	if got, want := rel, "weird|name.md"; got != want {
		t.Errorf("rel = %q, want %q", got, want)
	}
}

func TestEditExternalRejectsAnUnknownEditor(t *testing.T) {
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", tempLibrary(t))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "notepad|plan.md")

	if err := runEditExternal(); err == nil {
		t.Error("runEditExternal(notepad|plan.md) = nil, want an error")
	}
	if got := countCalls(t, log, "luvus"); got != 0 {
		t.Errorf("luvus calls after an unknown editor = %d, want 0", got)
	}
}

func TestEditExternalRejectsAMalformedValue(t *testing.T) {
	log := fakeLuvusLog(t)
	t.Setenv("LUVUS_BIN_PATH", filepath.Join(filepath.Dir(log), "fake-luvus"))
	t.Setenv("LUVUS_SETTING_ROOT", tempLibrary(t))
	t.Setenv("LUVUS_MODULE_ROW_VALUE", "plan.md")

	if err := runEditExternal(); err == nil {
		t.Error("runEditExternal(plan.md) = nil, want an error")
	}
	if got := countCalls(t, log, "luvus"); got != 0 {
		t.Errorf("luvus calls after a malformed value = %d, want 0", got)
	}
}
