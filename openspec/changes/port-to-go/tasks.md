# Tasks

## 1. Scaffold (design D12, D13)

- [x] 1.1 Add `go.mod` (`module github.com/slatkin/luvus-prompt-library`, `go 1.22`, no `toolchain` line) and `cmd/prompt-dock/main.go` with the verb dispatch. Each verb that is not ported yet returns a "not implemented" error. Verify: `go vet ./...` succeeds, `go install ./cmd/prompt-dock` puts `prompt-dock` on `PATH` (`command -v prompt-dock`), and `prompt-dock nope` exits 1 with a usage line.

## 2. Shared library (design D14)

- [x] 2.1 Port `lib.ts` to `cmd/prompt-dock/lib.go`: `luvus()`, `expandHome`, `resolveRoot`, `settingRoot`, `resolveUnderRoot`, `moduleStateDir`, the expanded and selected state files, `parseAgents`, `listAgents`, `currentWorkspaceId`, `agentsInWorkspace`, `encodeSendValue`, `splitSendValue`, `targetLabel`, `scanTree`, and `toast`. Use `exec.LookPath` instead of `findOnPath`. Keep the D-citations in the comments. Verify: the matching tests from `lib.test.ts`, ported to `lib_test.go` with the same names, pass with `go test ./...`. Add a test that an empty expanded set is written as `[]`, and a test that `expanded.json` and `selected.json` written by the TS code (fixture text) read back the same.

## 3. Dock rows (design D14, D16)

- [ ] 3.1 Port `refresh.ts` to `refresh.go`: the row and span shapes with `omitempty`, `branchPrefixes`, `toDockRow`, the send and edit menu entries, the missing-root row, `buildRows` (which takes the editor list and the agent list as parameters), and `refreshDock` (which looks them up and passes them in). Verify: the ported `branchPrefixes`, `toDockRow`, `editMenuEntries`, and `sendMenuEntries` tests pass.
- [ ] 3.2 While the TS files still exist, record the golden rows: run the TS `refresh.ts` with a fake luvus that saves its stdin and returns a fixed `agent list` reply, on a fixed library with one expanded folder, one selected prompt, and a `PATH` that holds a fake `zed`. Save the saved stdin as `cmd/prompt-dock/testdata/refresh-golden.json`. Add a Go test that calls `buildRows` with the same library and state, the editor list `[zed]`, and the same agents, and compares the decoded JSON with the golden file. Commit the golden file and the test together. Verify: the test passes, and it still passes after the TS files are deleted in group 7.
- [ ] 3.3 Port `toggle.ts` and `select.ts` to `toggle.go` and `select.go`. Verify: tests with a fake luvus show that `toggle` adds and then removes a folder in `expanded.json` and pushes rows each time, that `select` writes `selected.json` and pushes rows, that `select` of the already selected prompt pushes nothing, and that both reject a value that escapes the root with a non-nil error.

## 4. Editing (design D14, D15)

- [ ] 4.1 Port the inline mode of `edit.ts` to the `edit-inline` verb in `edit.go`: `resolveInlineEditor`, the `~/.luvus/config.json` read, the path check for `'`, `\`, and line breaks, `pane split --no-focus`, `pane run`, and the pane close after a failed run. Verify: the ported `resolveInlineEditor` tests pass, plus fake-luvus tests where the `pane split` reply carries `pane` as a string, as a number, and in a flat object.
- [ ] 4.2 Port the external mode to the `edit-external` verb, with `detach_other.go` (`//go:build !windows`, `Setsid`) and `detach_windows.go` (`DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP`). Remove the `shell: true` branch, and keep a `ponytail:` comment about `%` and `&` in Windows batch arguments. Verify: the ported `splitExternalValue` test passes, a test shows that an unknown editor and a malformed value return errors, and `GOOS=windows go vet ./...` succeeds on Linux.

## 5. Send (design D14, D16)

- [ ] 5.1 Port `send.ts` to the `send` verb in `send.go`, with the BOM strip, the empty-prompt check, `--` before the text, and the toast text cut to 160 runes. Verify: the four ported `send` tests pass, calling the verb function in the test process with `t.Setenv` and a fake luvus call log.

## 6. Manifest switch (design D12)

- [ ] 6.1 Rewrite `luvus-module.toml`: every `command` becomes `["prompt-dock", "<verb>"]`, add `[[build]]` with `command = ["go", "install", "./cmd/prompt-dock"]`, and set `version = "0.2.0"`. Remove the comment about `edit.ts` reading `LUVUS_MODULE_ACTION_ID`. Verify: run `go install ./cmd/prompt-dock`, relink the module, restart luvus, and check by hand that the dock paints, a folder toggles, a prompt selects, **Open** and **Edit in zed** open the file, and **Send to** submits a prompt to a live agent. `luvus module log` shows only `prompt-dock` runs with zero exits.
- [ ] 6.2 Update `README.md`: the requirements (Go 1.22 or newer, `GOBIN` on the `PATH` that luvus sees, no Node, luvus 0.14.3, Nerd Font), the install steps for a linked checkout (`go install ./cmd/prompt-dock`, then `command -v prompt-dock`), "run `go install` again after each pull", the hook setup with `git config core.hooksPath .githooks`, and a note that Windows is untested. Verify: on a clean checkout, following the README on Linux gives a working dock.

## 7. Remove the TS code and npm tooling (design D17)

- [ ] 7.1 Delete `lib.ts`, `refresh.ts`, `toggle.ts`, `select.ts`, `edit.ts`, `send.ts`, `lib.test.ts`, `package.json`, `package-lock.json`, `tsconfig.json`, `eslint.config.js`, `.prettierrc.json`, and `.prettierignore`, and remove `node_modules/` from `.gitignore`. Verify: `git ls-files '*.ts' package.json` prints nothing, and `go test ./...` still passes.
- [ ] 7.2 Rewrite `.githooks/pre-commit` to run `gofmt -l .` (fail on any output), `go vet ./...`, and `go test ./...` in the same quiet style. Verify: a commit with a badly formatted Go file is refused, and a clean commit passes.
- [ ] 7.3 Rewrite `.github/workflows/ci.yml` for `ubuntu-latest` only, with `actions/setup-go` pinned by commit SHA with a version comment, `go-version-file: go.mod`, and the three gates. Verify: CI is green for the pushed branch.

## Workflow follow-up

- Run `/opsx:archive` after the review passes.
- Close issue #5 with a link to the archived change.
