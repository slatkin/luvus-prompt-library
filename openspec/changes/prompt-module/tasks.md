# Tasks

## 1. Prompt module

- [ ] 1.1 Add `cmd/prompt-dock/prompt.go` with `Prompt{Body, Hint}`, `errEmptyPrompt`, and `loadPrompt(abs)` (design D1, D2): read the file, remove the BOM, split the frontmatter, read the `argument-hint`, return `errEmptyPrompt` for a whitespace-only body, and return the wrapped read error when the read fails. Verify with a table test in `prompt_test.go` that writes each case to a `t.TempDir()` file and has no Windows skip: BOM, no block, closed block, unclosed block, CRLF lines, hint with `"`/`'`/no quotes, empty hint, hint missing, block with no body (`errEmptyPrompt`), whitespace-only file (`errEmptyPrompt`), missing file (not `errEmptyPrompt`).
- [ ] 1.2 Add `Prompt.fill(arg)` (design D1, D3) with one regex in one pass, returning `errEmptyPrompt` when the result is only whitespace. Verify with a table test that covers every scenario of the "Placeholder fill" requirement (spec: prompt-arguments): plain, default used, default not used, value not filled again, `$HOME` and `` !`cmd` `` unchanged, no value removes `$ARGUMENTS`; plus a `$ARGUMENTS`-only body with an empty value (`errEmptyPrompt`).

## 2. Send uses the module

- [ ] 2.1 Change `send` in `send.go` to call `loadPrompt` and then `fill("")` in place of the inline read, BOM trim, and empty check (design D1). Keep the toast texts: `errEmptyPrompt` from either call gives the "is empty" toast, any other load error gives the "couldn't read" toast. Update the `--` comment, which names YAML frontmatter as a dash-leading case. Verify with `go test ./...`: the existing guard, empty-file, and read-failure tests pass unchanged.
- [ ] 2.2 Update `send_test.go` for the "Send submits the prompt" requirement (spec: agent-prompt-send): change the dash test to a body that starts with `- ` and check `-- - ` in the call log; add tests that a frontmatter file sends `body line` and none of the frontmatter lines, a frontmatter-only file logs the "is empty" toast and no `agent prompt`, and a `$ARGUMENTS`-only body logs the "is empty" toast and no `agent prompt`. Verify with `go test ./...` and `go vet ./...`.
- [ ] 2.3 Set `version = "0.3.0"` in `luvus-module.toml`, and update the README "Sending" paragraph: every send removes the frontmatter block, `$ARGUMENTS` is removed and `${ARGUMENTS:-x}` becomes `x` (no value can be given yet), and a body that is empty after that is refused. Verify that each sentence matches the specs in this change.

## 3. Integration check

- [ ] 3.1 In luvus, with the rebuilt binary (`go install ./cmd/prompt-dock`), send `claude/shot` and a prompt with frontmatter to a live agent. Check that the agent gets no frontmatter lines and no literal `$ARGUMENTS`, and that the success toast shows. Verify that `luvus module log` shows no failed run.

## Workflow follow-up

- Implement `prompt-argument-input` after this change; its tasks use `loadPrompt` and `fill`.
