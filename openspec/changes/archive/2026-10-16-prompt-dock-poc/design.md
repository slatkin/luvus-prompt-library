# Design

## Context

See proposal.md for motivation and specs/ for the behavior contract. Facts verified against luvus 0.14.3 (source at `~/Dev/luvus`):

- Modules run argv commands from `luvus-module.toml`. Docks are flat row lists pushed with `ui dock push` (JSON on stdin). A row has `text`, `dot`, `tone`, `spans`, `action`, `value` and a per-row `menu` of `{title, action, value}`. A row does at most one thing on a left-click, and **a row has no selection state of its own**; `spans` is drawn instead of `text`, while `text` stays the plain label the action receives. Clicked rows get `LUVUS_MODULE_ROW_VALUE`, and menu entries override it with their own `value`. `LUVUS_BIN_PATH` names the luvus binary. A dock can be placed left, right or off and resized, but **not reordered**, so a module cannot put its own dock at the top of a sidebar.
- `luvus agent list` returns only agents. Each has `pane`, `terminal_id` (32 hex chars), `agent`, `name`, `workspace` (the numeric focused-workspace id), `workspace_name`, `project`.
- `luvus agent prompt <target> --terminal-id <id> -- <text>` pastes and submits in one atomic request (`agent.prompt`). The text must come after `--`, because without it `agent prompt` reads text that starts with `-` as an unknown option and refuses the send. It refuses with `agent_not_ready` and queues nothing when the agent has no prompt-ready composer (`src/app/dispatch/agent_workflow.rs:137`).
- `luvus pane split --no-focus` returns `{"pane": "<id>", ...}`. `luvus pane run <id> <cmd>` *types* `cmd` + Enter into the pane's shell (`src/app/dispatch/topology.rs:258`). It is not an exec.
- `luvus files open` opens a read-only file view, not the configured editor, so it does not meet the inline-edit requirement.
- Events available: `workspace.created/closed`, `pane.created/closed`, `pane.agent_status_changed`.
- `ui toast <text>` shows a one-line message.

## Goals / Non-Goals

**Goals:**
- Smallest module that proves browse → edit → send-and-submit.
- Node stdlib only. Run `.ts` directly (type stripping, erasable syntax only).

**Non-Goals:**
- File watching on the prompt root. Repaint comes from events and the `refresh` action.
- Prompt templating or variables.
- Windows-specific shell quoting beyond what is noted in Risks.

## Decisions

### D1: The module lives at the repo root
`luvus module install <owner>/luvus-prompt-library` (README) installs the repo, so the manifest goes at the root. PLAN.md's `~/Dev/prompt-dock/` path is superseded.

### D2: One `refresh.ts` builds everything
`refresh.ts` scans the tree, reads `agent list`, builds rows with menus and pushes the dock. Startup, every event hook, the `refresh` action and `toggle.ts` (after it writes state) all run it. Alternative: separate refreshes for tree and targets. Rejected, because a dock push replaces all rows anyway.

### D3: Row and menu `value` encoding
- Folder row: `action: "toggle"`, `value: "<relpath>"`.
- Prompt row: `action: "select"`, `value: "<relpath>"` (design D9).
- Menu entry `Open`: `action: "edit-inline"`, `value: "<relpath>"`.
- Menu entry "Edit in zed/VS Code": `action: "edit-external"`, `value: "zed|<relpath>"` / `"code|<relpath>"`.
- Menu entry "Send to …": `action: "send"`, `value: "<pane>|<terminal_id>|<relpath>"`.

Split each value on the first one or two `|` only, so a `|` in a file name still works. Paths are relative to `root` and resolved again at action time, so a change to the `root` setting cannot point an old row at a file outside the new root. Reject any relpath that resolves outside `root`.

### D4: Send uses `agent prompt` with `--terminal-id`
`send.ts` reads the file, then runs `luvus agent prompt <pane> --terminal-id <terminal_id> -- <text>`, with the text as **one** argv element (`execFileSync`, no shell). The `--` ends option parsing: without it a prompt that starts with `-` (a `---` frontmatter block, a leading bullet) is read as an unknown option and the send fails. The terminal-id makes luvus reject a reused pane id, which covers the stale-target requirement with no module state. On a non-zero exit, show a toast with the stderr summary. On success, show a toast with the prompt and agent labels.
Alternative: `pane send` + `agent keys enter`. Rejected: two requests are not atomic, and `pane send` has no readiness check.

### D5: Expanded state in `LUVUS_MODULE_STATE_DIR/expanded.json`
A JSON array of folder relpaths. `toggle.ts` adds or removes the value and then runs `refresh.ts`. A missing or corrupt file means "all collapsed".
The state dir is per luvus session (`session_dir()/modules/state/<id>`, `src/module/paths.rs:25`), while settings live in the global config dir — so a named session starts with every folder collapsed but still honours the `root` setting. Nothing deletes the state dir: unlink and relink keep it, because the path is keyed by module id.

### D6: Inline edit = split pane + typed command
Read `~/.luvus/config.json` → `layout.file_open`. If it is empty, `readonly` or unreadable, use `nvim`. Run `pane split --no-focus`, then `pane run <new> <editor> '<abs path>'`. The editor value is used as the shell words that luvus stores (for example `emacs -nw`).
Alternative: `files open`. Rejected, it is read-only (Context).

### D7: External editors detached
Use `spawn(editor, [absPath], {detached: true, stdio: "ignore"}).unref()`. Probe PATH at refresh time with a stdlib PATH scan (split `PATH`, check that the file exists, add `PATHEXT` on Windows), not `which`, to keep zero dependencies and Windows support.

### D8: Target labels
`<name> (<agent>) · pane <pane>`, or `<agent> · pane <pane>` when `name` is null. Short by decision: the menu holds one entry per live agent, so status and workspace_name are left out and the parsed agent no longer carries them.

### D9: Left-click selects; opening is a menu choice
The dock has no selection state, so `select.ts` keeps it in `LUVUS_MODULE_STATE_DIR/selected.json` (a JSON string, or absent for no selection) and then repaints.
Alternative: keep `edit-inline` on the left-click and add selection later. Rejected: a stray click pulls focus into a new editor pane, which is the behaviour this rework removes.

### D10: Row glyphs, tones, the selected row, and the pushed title
Each row starts with a Nerd Font glyph (Material Design set) and a space: `md-folder` (`U+F024B`) for a collapsed folder, `md-folder_open` (`U+F0770`) for an expanded one, `md-chat` (`U+F0B79`) for a prompt. Both folders and prompts therefore put their name two columns past the row's indent, so names line up per level however the rows are mixed. The glyphs are written as `\u{...}` escapes in `refresh.ts` so the codepoints stay exact and reviewable.
Folders take `tone: "accent"`; prompts keep the default row colour; the selected prompt takes the accent tone and adds no marker of its own, because every prompt row already carries a glyph. The terminal must have a Nerd Font — the same assumption luvus's own web client makes in its font stack (`src/web/assets/app.css`), and the reason a plain monospace font shows boxes here.
The built-in docks mark their focused row with accent text **and bold, plus a row background** (`src/ui/workspace_row.rs`, `src/ui/sidebar.rs:809`). A module dock row cannot draw either: `DockRow` has no such field (`src/app/mod.rs:182`), and `draw_module_dock` styles each row with `Style::new().fg(...)` only (`src/ui/sidebar.rs:1020`). `tone: "accent"` is exactly `t.accent` (`src/bar/render.rs:164`), so the colour matches the native row. A true mirror needs a Luvus change.
Double-click is unavailable for the same reason: a module row gets a left-click (its `action`) and a right-click (its `menu`) only, and the 400 ms double-click detector serves pane content and ORCH rows (`src/app/input.rs`), so a double-click just runs the action twice.
`refresh.ts` pushes `--title PROMPTS` on every push, because a push without a title leaves luvus titling the dock from its id.

### D11: Send targets come from the focused workspace
The context maps `LUVUS_WORKSPACE_ID` to `workspace.id` (`src/module/context.rs:157`), which is the focused workspace for CLI, socket, startup and event invocations — the same numeric id `agent list` reports as `workspace`. `refresh.ts` therefore keeps the agents whose id matches. Verified live: a `module run` probe reported `LUVUS_WORKSPACE_ID=3` for pane 12 in workspace 3.
An unset or empty id keeps every agent: the module then cannot tell which workspace is focused, and hiding every send target is the worse failure of the two.

## Risks / Trade-offs

- [`pane run` types into a shell, so paths need quoting] → single-quote the path, and refuse (with a toast) paths that contain `'`. This works in sh/bash/zsh/fish. PowerShell needs `''` escaping, which is a known gap and is noted in the roadmap issue.
- [The editor pane stays as a shell after the editor quits] → acceptable for the PoC.
- [A prompt that begins with `-` is read as an option by `agent prompt`] → fixed: `send.ts` passes the text after a `--` separator, so leading `---` frontmatter and bullet lines arrive as literal text. The roadmap issue still covers the stdin path through `uhp proxy`.
- [argv size limit, about 32 K chars on Windows and 128 KB per argument on Linux] → the same roadmap item.
- [`pane.agent_status_changed` fires often (working ↔ idle), and each one rescans the tree] → acceptable for small libraries. If it gets slow, cache the tree scan in the state dir.
- [Luvus fires no event on a workspace switch] → the send list is built at repaint time, so after a switch the menu keeps the previous workspace's agents until the next repaint: a pane opened or closed, an agent status change, or any dock click (toggle, select and refresh all repaint).
- [The focused workspace cannot be determined] → every agent is listed (D11), rather than no agent.
- [Menus built at refresh can go stale] → cosmetic only, and the labels no longer carry status (D8). Delivery safety comes from D4.

## Migration Plan

New module, nothing to migrate. Install with `luvus module link <repo>`. Roll back with `luvus module unlink`.
