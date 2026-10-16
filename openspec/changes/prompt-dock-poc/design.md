# Design

## Context

See proposal.md for motivation and specs/ for the behavior contract. Facts verified against luvus 0.14.3 (source at `~/Dev/luvus`):

- Modules run argv commands from `luvus-module.toml`. Docks are flat row lists pushed with `ui dock push` (JSON on stdin). A row has `text`, `dot`, `tone`, `action`, `value` and a per-row `menu` of `{title, action, value}`. Clicked rows get `LUVUS_MODULE_ROW_VALUE`, and menu entries override it with their own `value`. `LUVUS_BIN_PATH` names the luvus binary.
- `luvus agent list` returns only agents. Each has `pane`, `terminal_id` (32 hex chars), `agent`, `name`, `status`, `workspace_name`, `project`.
- `luvus agent prompt <target> <text> --terminal-id <id>` pastes and submits in one atomic request (`agent.prompt`). It refuses with `agent_not_ready` and queues nothing when the agent has no prompt-ready composer (`src/app/dispatch/agent_workflow.rs:137`).
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
- Prompt row: `action: "edit-inline"`, `value: "<relpath>"`.
- Menu entry "Open in zed/VS Code": `action: "edit-external"`, `value: "zed|<relpath>"` / `"code|<relpath>"`.
- Menu entry "Send to …": `action: "send"`, `value: "<pane>|<terminal_id>|<relpath>"`.

Split each value on the first one or two `|` only, so a `|` in a file name still works. Paths are relative to `root` and resolved again at action time, so a change to the `root` setting cannot point an old row at a file outside the new root. Reject any relpath that resolves outside `root`.

### D4: Send uses `agent prompt` with `--terminal-id`
`send.ts` reads the file, then runs `luvus agent prompt <pane> <text> --terminal-id <terminal_id>`, with the text as **one** argv element (`execFileSync`, no shell). The terminal-id makes luvus reject a reused pane id, which covers the stale-target requirement with no module state. On a non-zero exit, show a toast with the stderr summary. On success, show a toast with the prompt and agent labels.
Alternative: `pane send` + `agent keys enter`. Rejected: two requests are not atomic, and `pane send` has no readiness check.

### D5: Expanded state in `LUVUS_MODULE_STATE_DIR/expanded.json`
A JSON array of folder relpaths. `toggle.ts` adds or removes the value and then runs `refresh.ts`. A missing or corrupt file means "all collapsed".

### D6: Inline edit = split pane + typed command
Read `~/.luvus/config.json` → `layout.file_open`. If it is empty, `readonly` or unreadable, use `nvim`. Run `pane split --no-focus`, then `pane run <new> <editor> '<abs path>'`. The editor value is used as the shell words that luvus stores (for example `emacs -nw`).
Alternative: `files open`. Rejected, it is read-only (Context).

### D7: External editors detached
Use `spawn(editor, [absPath], {detached: true, stdio: "ignore"}).unref()`. Probe PATH at refresh time with a stdlib PATH scan (split `PATH`, check that the file exists, add `PATHEXT` on Windows), not `which`, to keep zero dependencies and Windows support.

### D8: Target labels
`<name ?? agent+" · pane "+pane> (<agent>) · <status> · <workspace_name>`. When `name` is null, the kind is already in the label, so leave out the `(<agent>)` part.

## Risks / Trade-offs

- [`pane run` types into a shell, so paths need quoting] → single-quote the path, and refuse (with a toast) paths that contain `'`. This works in sh/bash/zsh/fish. PowerShell needs `''` escaping, which is a known gap and is noted in the roadmap issue.
- [The editor pane stays as a shell after the editor quits] → acceptable for the PoC.
- [A prompt that begins with `--` is read as an option by `agent prompt`] → shows as a send-failure toast. The roadmap issue tracks the stdin path through `uhp proxy`.
- [argv size limit, about 32 K chars on Windows and 128 KB per argument on Linux] → the same roadmap item.
- [`pane.agent_status_changed` fires often (working ↔ idle), and each one rescans the tree] → acceptable for small libraries. If it gets slow, cache the tree scan in the state dir.
- [Menus built at refresh show stale status text] → cosmetic only. Delivery safety comes from D4.

## Migration Plan

New module, nothing to migrate. Install with `luvus module link <repo>`. Roll back with `luvus module unlink`.
