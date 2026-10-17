# Tasks

Start only after `prompt-module` is implemented: these tasks use `loadPrompt`,
`fill`, and `errEmptyPrompt` from `cmd/prompt-dock/prompt.go`.

## 1. Dependencies

- [ ] 1.1 Run `go get charm.land/bubbletea/v2 charm.land/bubbles/v2` and `go mod tidy`, so the `go` directive in `go.mod` moves to the version that bubbles/v2 needs (design D6). Verify that `go build ./...` and `go vet ./...` pass.

## 2. Send path

- [x] 2.1 Move the delivery part of `send` into `deliver(pane, terminalId, rel, arg)` (design D3): it calls `loadPrompt`, then `fill(arg)`, then `agent prompt`, with the same toasts. Verify with `go test ./...`: the existing `send` tests pass unchanged.
- [x] 2.2 Make `send` read the hint: `errEmptyPrompt` from `loadPrompt` gets the "is empty" toast and an error, before the hint check; no hint calls `deliver(..., "")`; a hint writes `ask.json` with `value` and `return_pane` (from `LUVUS_PANE_ID`) through `writeAtomic`, then runs `module pane open slatkin.prompt-dock ask --placement split`; a failed open deletes `ask.json`, shows a toast, and returns an error (design D2, D3). Verify with fake-luvus tests: a hinted prompt logs the pane-open call, writes the expected `ask.json`, and logs no `agent prompt`; a hinted file with only frontmatter logs the "is empty" toast and no pane-open call; `shot`-style `$ARGUMENTS` with no hint is sent at once, with no pane-open call; a failed open logs the toast and leaves no `ask.json`.

## 3. Input panel

- [ ] 3.1 Add `ask.go` with the bubbletea model (design D6): a focused `textinput` with the hint as the placeholder text, a view with the prompt name and the target label, Enter submits, Esc and Ctrl+C cancel. Verify that `go build ./...` passes; task 3.2 tests the behavior (design D8).
- [ ] 3.2 Add the `ask` verb: read and delete `ask.json` (missing or bad JSON exits 0 with no luvus call), run the program, call `deliver` with the value on submit, then `pane focus <return_pane>` when it is set, on submit and on cancel (design D2, D7). Register `ask` in `main.go` (verbs map and usage). Verify with the one panel test (design D8), which runs the program on scripted input (`tea.WithInput`, `tea.WithOutput(io.Discard)`); use the `Update` fallback only if scripted input does not work: `my-change` + Enter logs `agent prompt` with the value filled in and then `pane focus 3`; Esc logs only `pane focus 3`; a missing handoff logs nothing; `ask.json` is gone after each run.
- [ ] 3.3 Add the `[[panes]]` entry `ask` to `luvus-module.toml` (design D1) and set `version = "0.4.0"`. Verify that the entry matches design D1 (id, placement, command); task 5.1 checks it live in luvus.

## 4. Docs

- [ ] 4.1 Update the README: Requirements (bubbletea, Go 1.26 or later, no "standard library only"), a new section for the input panel (hint opens a split, Enter sends, Esc cancels, focus returns), and the "Sending" paragraph, which after `prompt-module` says that no value can be given yet. Verify that each documented behavior matches the specs in this change and in `prompt-module`.

## 5. Integration check

- [ ] 5.1 In luvus, with the rebuilt binary (`go install ./cmd/prompt-dock`): send `pi/simple` to a live agent, type a ref, press Enter, and check that the agent gets the body with no frontmatter and with the ref filled in, and that the focus is back on the pane that had it. Send it again and press Esc, and check that the agent gets nothing. Send `claude/shot`, and check that no panel opens and that `$ARGUMENTS` is gone. Verify that `luvus module log` shows no failed run.
