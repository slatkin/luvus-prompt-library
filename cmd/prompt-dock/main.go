// main.go — prompt-dock, the Prompt Dock module binary (design D12, D13).
// One binary, one verb per action (design D13): main reads os.Args[1] and runs
// refresh, toggle, select, edit-inline, edit-external, or send. The verb names
// equal the action ids in luvus-module.toml, and the startup and event entries
// use refresh. Every verb function returns an error; main prints it to stderr
// and exits 1, which replaces the TS fail() (which threw) and isMain() (which
// tests needed). Verbs that are not ported yet return a "not implemented"
// error; tasks 2-5 of the port-to-go change fill them in.
package main

import (
	"fmt"
	"os"
)

const usage = "usage: prompt-dock <refresh|toggle|select|edit-inline|edit-external|send>"

// notImplemented is the placeholder error of a verb that has no port yet.
func notImplemented(verb string) error {
	return fmt.Errorf("%s: not implemented", verb)
}

// runEditInline opens a prompt in a split pane (task 4.1). It replaces the
// LUVUS_MODULE_ACTION_ID branch of edit.ts.
func runEditInline() error { return notImplemented("edit-inline") }

// runEditExternal opens a prompt in a detached editor window (task 4.2). It
// replaces the other LUVUS_MODULE_ACTION_ID branch of edit.ts.
func runEditExternal() error { return notImplemented("edit-external") }

// runSend sends a prompt to an agent pane (task 5.1).
func runSend() error { return notImplemented("send") }

var verbs = map[string]func() error{
	"refresh":       runRefresh,
	"toggle":        runToggle,
	"select":        runSelect,
	"edit-inline":   runEditInline,
	"edit-external": runEditExternal,
	"send":          runSend,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	run, ok := verbs[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "%s\n%s\n", os.Args[1], usage)
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
