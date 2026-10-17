# Design

## Context

See proposal.md for the reason. The facts below come from the luvus 0.14.3
source and docs, and they shape the approach:

- A module has only these UI surfaces: dock rows, toasts, notifications, bar
  widgets, and real panes. Only a pane gives a program a TTY, so only a pane
  can show a text field.
- `luvus module pane open <id> <entrypoint> [--placement split|overlay|tab]`
  runs a `[[panes]]` command in a new pane (`src/app/modules.rs`,
  `module_open_pane`). It takes **no per-call arguments** and adds no per-call
  environment. The command gets the normal `LUVUS_*` environment and a context
  snapshot.
- The new pane splits the **focused** pane and takes the focus
  (`layout.split_focused` sets `focus = new_id`).
- `overlay` is a split plus `zoomed = true`. `close_pane` does not clear
  `zoomed`, so luvus can stay zoomed on another pane after the overlay closes.
- When the pane's process exits, luvus closes the pane. When a closed pane had
  the focus, the focus goes to the first leaf of the layout, which is not
  always the pane the user had before.
- A module pane that is open when the user detaches is opened again on the next
  launch.
- After the `prompt-module` change, `send` calls `loadPrompt(abs)` (BOM,
  frontmatter, `argument-hint`, empty check) and `fill("")` from
  `cmd/prompt-dock/prompt.go`, then calls
  `luvus agent prompt <pane> --terminal-id <id> -- <text>` (design D4 of the
  archived `prompt-dock-poc` change). Both operations return `errEmptyPrompt`
  for an empty body. The row value is `<pane>|<terminal_id>|<relpath>`.

## Goals / Non-Goals

**Goals:**

- One delivery function that both the direct send and the panel use, so the
  stale-target guard, the toasts, and the read-at-send-time rule stay in one
  place.
- A panel test that runs the real bubbletea program on scripted input.

**Non-Goals:**

- Changes to the text rules (frontmatter, hint, fill). `prompt-module` owns
  them.
- More than one pending panel request at a time.
- A multi-line input field.

## Decisions

### D1. The panel is a `[[panes]]` entrypoint with `split` placement

```toml
[[panes]]
id = "ask"
title = "Prompt argument"
placement = "split"
command = ["prompt-dock", "ask"]
```

`send` opens it with `luvus module pane open slatkin.prompt-dock ask
--placement split`. The pane closes by itself when `prompt-dock ask` exits.

Alternatives:
- `overlay`: rejected. luvus stays zoomed after the pane closes.
- `pane split` + `pane run "prompt-dock ask ..."` (the edit-inline method):
  rejected. It types a command into a shell, so the arguments need shell
  quoting, and the shell pane stays open after the program exits.
- Commander `$prompt @agent <text>` (`commander = { input = "text" }`): not a
  panel, and the user must type the prompt name. It can come later as an extra
  entry point.

### D2. A handoff file carries the request to the panel

`module pane open` takes no arguments, so `send` writes
`$LUVUS_MODULE_STATE_DIR/ask.json` (with `writeAtomic`) before it opens the
pane:

```json
{"value": "<pane>|<terminal_id>|<relpath>", "return_pane": "3"}
```

`value` is the row value that `send` got. `return_pane` is `send`'s own
`LUVUS_PANE_ID`, which is the focused pane when the user clicked the dock.
`ask` reads the file and **deletes it at once**, before it shows the field. If
the file is missing or not valid JSON, `ask` exits 0 at once and sends nothing.
This covers the panel that luvus opens again after a restart. When
`module pane open` fails, `send` deletes `ask.json`, so no later panel can read
a request that the user never saw.

Alternative: one file per request. Rejected: `send` does not know the new pane
id before the pane opens, and the user cannot fill two panels at the same time.
`// ponytail: one handoff slot; the last send wins if two open at once.`

### D3. One delivery function for both paths

Today's body of `send` after the value parse becomes
`deliver(pane, terminalId, rel, arg string) error`: resolve under the root,
apply the terminal-id guard, `loadPrompt`, `fill(arg)`, call `agent prompt`,
and toast. `fill` refuses a body that is empty after the fill, so a body that
is only `$ARGUMENTS`, sent with an empty value, never reaches `agent prompt`.

`send(value)` becomes:

```
parse value -> resolve -> loadPrompt(abs)
   errEmptyPrompt -> "is empty" toast, error (no panel)
   Hint == ""     -> deliver(..., "")
   Hint != ""     -> write ask.json -> module pane open ... ask
```

The early check is the `loadPrompt` check, before the fill, because `send`
does not know the value yet. A hinted body that is only `$ARGUMENTS` therefore
opens the panel; `deliver` refuses it after Enter if the value is empty.

`ask` calls `deliver(..., typedValue)` after Enter. `deliver` reads the file
again, so edits saved while the panel is open go out. If `send` cannot read the
file, it keeps today's "couldn't read" toast. If the pane does not open, `send`
deletes `ask.json` (D2), shows a toast ("Couldn't open the input panel ..."),
and returns an error.

### D4. Frontmatter (moved)

Moved to `prompt-module` design D2. The number is kept so that references
stay valid.

### D5. Fill (moved)

Moved to `prompt-module` design D3.

### D6. bubbletea v2 and bubbles/v2 `textinput`

Module paths: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2/textinput`.
The model holds a focused `textinput.Model` with `Placeholder = hint`, plus the
prompt name and target label for the view. Keys:

| Key | Result |
|---|---|
| Enter | quit with `submitted = true` |
| Esc, Ctrl+C | quit with `submitted = false` |
| all other keys | go to the text input |

No alt screen: the view is three short lines in a small pane. The program
returns the final model, and `ask` reads `submitted` and the value from it.

From now on, bubbletea is the TUI library for module input and display
elements. bubbles/v2 needs Go 1.26 or later, so the `go` directive moves up.
The binary is still built with `go install` at install time, so users need
the newer Go only to build.

Alternative: a stdlib-only line read (`bufio.Scanner` on the pane TTY). It is
smaller, but it gives no placeholder text, no cursor editing, and no Esc. You
asked for bubbletea.

### D7. Focus goes back before exit

After the program ends, by submit or by cancel, `ask` runs
`luvus pane focus <return_pane>` when `return_pane` is not empty, and it
ignores a failure (the pane can be gone). It does this **before** it exits.
The panel pane then no longer has the focus, so `close_pane` does not move the
focus to the first leaf. On submit, `ask` calls `deliver` first and focus
second.

### D8. Testing

- Frontmatter and fill: covered by `prompt-module` (`prompt_test.go`).
- `send` and `deliver`: the existing fake-luvus call log (`fakeLuvusLog`). The
  hint path asserts `module pane open slatkin.prompt-dock ask --placement
  split`, the `ask.json` content, and no `agent prompt` call.
- `ask`: the real program runs with `tea.WithInput(strings.NewReader(...))` and
  `tea.WithOutput(io.Discard)`, so scripted keys (`my-change` then Enter, or
  Esc) drive it with no TTY. The test then checks the call log and that
  `ask.json` is gone. This is the one panel test; there are no separate
  `Update` tests, and no test of typing into the field (that is the
  `textinput` dependency's job). Only if scripted input does not work in v2,
  the fallback is to call the model's `Update` with `tea.KeyPressMsg` values,
  and to test `ask`'s handoff and delivery with the value passed in.

## Risks / Trade-offs

- [The panel splits the focused pane, which may be small] → luvus picks the
  split axis by size; the view needs three lines only.
- [Two hinted sends in quick succession share one handoff slot] → the second
  overwrites the first before the first panel reads it. Accepted (D2). The
  user sees a panel for the second prompt.
- [New dependency tree and a higher Go version] → the README states the Go
  requirement. `go vet` and `go test` in the pre-commit hook cover the build.
- [`LUVUS_PANE_ID` empty for some invocation sources] → `return_pane` is empty,
  and `ask` skips the focus call. luvus then uses its own focus rule.

## Migration Plan

Users run `go install ./cmd/prompt-dock` again after the pull (as the README
says for every update) and restart luvus, so luvus reads the new `[[panes]]`
entry. Rollback: check out the previous version and run `go install` again.
