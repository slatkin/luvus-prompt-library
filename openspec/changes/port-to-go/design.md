# Design

## Context

See proposal.md - Why. The current module is six TypeScript scripts run by `node`, one short-lived process per startup, event, or action. They share state through files in `LUVUS_MODULE_STATE_DIR`. The decisions D1-D11 in `openspec/changes/archive/2026-10-16-prompt-dock-poc/design.md` stay in force, and code comments still cite them. This design continues the numbering at D12, so a citation is never ambiguous.

Facts from the luvus source (0.14.3, `/home/slatkin/Dev/luvus`) that shape the approach:

- Luvus starts every module command with `Command::new(argv[0]).current_dir(module_root)` and no shell (`src/module/runtime.rs:287`). The child inherits luvus's `PATH`. Luvus does not add the module folder to `PATH` and does not substitute anything in argv (`src/module/runtime.rs:100-140`).
- A bare program name (no path separator) is searched for on `PATH`, on Linux and on Windows. On Windows, Rust adds `.exe` first.
- A relative program path such as `bin/prompt-dock` works on Linux, but on Windows Rust resolves it against the luvus process's own working directory, not the module root (`library/std/src/sys/process/windows.rs:487-503`). A bare name on `PATH` avoids this problem.
- `[[build]]` runs only at git-install time, in the module root, with `LUVUS_*` and `BOHAY_*` removed from the environment (`src/module/install.rs:338-360`). `HOME` and `PATH` pass through, so `go install` and the Go module cache work.

## Goals / Non-Goals

**Goals:**
- The Go binary pushes the same dock rows and calls the same luvus CLI commands as the TS scripts, for the same library and state.
- An upgrade keeps the user's expanded folders and selected prompt.

**Non-Goals:**
- Third-party Go modules. The menus change adds the UI libraries.
- Cleanups of behaviour, even small ones. If the TS code does something odd, the port does the same, and a later change fixes it.
- Testing on Windows, and building or distributing binaries. The Windows code paths are ported and must compile. Nothing more.

## Decisions

### D12: The binary is installed with `go install` and found on `PATH`

The program lives in `cmd/prompt-dock/` (`package main`), with `go.mod` at the repo root. `go install ./cmd/prompt-dock` writes `prompt-dock` (or `prompt-dock.exe`) to `GOBIN`. Go names the binary after the last element of the package path, so the folder name `prompt-dock` gives the binary its name.

Every manifest `command` is `["prompt-dock", "<verb>"]`. A single `[[build]]` entry, `["go", "install", "./cmd/prompt-dock"]`, installs the binary at git-install time. For a linked checkout, the user runs the same command after each pull. The manifest has no `platforms` field and no per-platform entries.

- Alternative: a relative path `bin/prompt-dock` built into the checkout. Rejected: on Windows it cannot start without a platform-split manifest, with `cmd /c` wrappers and duplicate action ids.
- Alternative: `["go", "run", "./cmd/prompt-dock", "<verb>"]`. Rejected: it needs Go at runtime, and every click pays the `go` start-up and stale check.
- Cost: the binary is shared by everything on the machine, so a linked checkout and the installed binary can differ in version until the user runs `go install` again.

### D13: One verb per action

`main.go` reads `os.Args[1]` and runs `refresh`, `toggle`, `select`, `edit-inline`, `edit-external`, or `send`. An unknown or missing verb prints a usage line to stderr and exits 1. The verb names equal the action ids, and the startup and event entries use `refresh`.

`edit.ts` chose its mode from `LUVUS_MODULE_ACTION_ID`. The binary uses the verb instead, so the mode is visible in the manifest.

The Go files mirror the TS files (`lib.go`, `refresh.go`, `toggle.go`, `select.go`, `edit.go`, `send.go`), so a reviewer can compare them side by side.

Each verb function returns an `error`. `main` prints it as `<context>: <detail>` to stderr and exits 1. This replaces `fail()` (which threw) and `isMain()` (which tests needed). Toasts stay in the same places as in the TS code, before the error returns.

### D14: Standard-library mapping, and where the port must take care

Most of the mapping is direct (`os`, `os/exec`, `path/filepath`, `encoding/json`). These points need care to keep the same behaviour:

- **`findOnPath`** becomes `exec.LookPath`. It does the same work: it scans `PATH`, uses `PATHEXT` on Windows, and checks the executable bit on Linux.
- **`luvus()`** becomes `exec.Command(LUVUS_BIN_PATH or "luvus", args...)` with the input on stdin. It never returns a Go error to the caller. A spawn failure gives status -1 and the error text as stderr, as in TS.
- **Lenient JSON.** `parseAgents`, `readExpanded`, `readSelected`, and the `pane split` reply decode into `any` and check each field's type. A wrong type drops the entry, as in TS. The `pane split` reply may carry `pane` as a string or a number, and both must work (TS used `String(...)`).
- **Pushed rows** keep their exact shape: a span with no tone has no `tone` key, a folder row has no `menu`, and a root row has one span. Struct tags use `omitempty`.
- **State files** keep their format: an indented JSON array (or string) plus a newline, written to `<file>.tmp` and renamed over the file. An empty set is written as `[]`, never `null`. The folder order in `expanded.json` becomes sorted instead of insertion order, and no reader depends on it.
- **Sorting.** TS sorted names by UTF-16 code unit. `sort.Strings` sorts by byte, which is code point order. The two differ only when a folder mixes names that contain characters above U+FFFF with names that contain U+E000-U+FFFF. This difference is accepted. The spec says only "sorted by name".
- **Toast text** that `send` cuts to 160 characters is cut by rune, so it never splits a UTF-8 sequence.
- **Path functions.** `path.resolve` becomes `filepath.Abs`, `realpathSync` becomes `filepath.EvalSymlinks`, and `statSync` (which follows links) becomes `os.Stat`. `resolveUnderRoot` stays lexical, as D3 says.

### D15: External editors start detached

`edit-external` calls `exec.Command(editor, abs).Start()`, does not wait, and leaves stdin, stdout, and stderr unset (the null device). A build-tagged helper sets the detach flags: `Setsid` on Linux (`detach_other.go`, `//go:build !windows`), and `DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP` on Windows (`detach_windows.go`). This matches Node's `detached: true`. The Windows file exists only so that the program compiles there. `GOOS=windows go vet ./...` on Linux checks that.

On Windows, `exec.Command("code")` finds `code.cmd` through `LookPath`, so the TS branch that used `shell: true` with manual quotes goes away. The code keeps a `ponytail:` comment that `cmd` still parses the arguments of a batch file, so a path with `%` or `&` can break.

### D16: Tests

`lib_test.go` ports every test in `lib.test.ts`, with the same names. Tests that run a fake luvus shell script or create symlinks keep their Windows skip, as they do today. The `send` tests call the verb function in the test process with `t.Setenv`, and check the returned error and the fake luvus call log. They do not start a child process.

Before the TS files are deleted, a golden check records the stdin that the TS `refresh.ts` gives to a fake luvus, for a fixed library, state, and agent list. The golden file is committed in the same commit as the Go test that reads it. The rows hold no absolute paths (every value is a relpath or carries one), so the golden file needs no placeholder. `buildRows` takes the editor list and the agent list as parameters, and `refreshDock` looks them up and passes them in, so the Go test calls `buildRows` directly with no fake binaries. The check compares the decoded JSON, not the bytes, because Go escapes `<`, `>`, and `&` differently.

### D17: Gates, hook, and CI

- The pre-commit hook keeps its quiet style and runs `gofmt -l .` (fail on any output), `go vet ./...`, and `go test ./...`.
- CI runs the same three gates on `ubuntu-latest` only. `actions/setup-go` is pinned by commit SHA with a version comment, like the current steps, and reads the Go version from `go.mod`. CI builds nothing for release.
- `go.mod` says `module github.com/slatkin/luvus-prompt-library` and `go 1.22`, with no `toolchain` line. A user with Go 1.22 or newer then builds without a toolchain download.
- Without `npm install`, nothing sets `core.hooksPath`. The README tells developers to run `git config core.hooksPath .githooks` once.

## Risks / Trade-offs

- [`GOBIN` is not on the `PATH` that luvus sees, for example when luvus starts from a desktop launcher] → Every command fails with a spawn error that names `prompt-dock`. The README says to put `GOBIN` on `PATH` and to check it with `command -v prompt-dock`.
- [The installed binary and a linked checkout differ in version] → The README says to run `go install ./cmd/prompt-dock` after each pull. `[[build]]` covers git installs.
- [The port changes behaviour by accident] → All ported tests, plus the golden row check (D16), plus a check by hand on Linux of every row menu entry.
- [The install fails when Go is not on `PATH`] → The `[[build]]` spawn error names `go`. The README lists Go as a requirement.
- [Windows is untested] → Accepted. The README says so.

## Migration Plan

1. Build and test the Go binary next to the TS files, then switch the manifest in one commit, and delete the TS files and npm tooling in the same change.
2. A linked install: `git pull`, `go install ./cmd/prompt-dock`, then restart luvus.
3. A git install: reinstall the module, and `[[build]]` installs the binary.
4. Rollback: revert the commits. The state files have the same format in both versions, so no state is lost in either direction.
