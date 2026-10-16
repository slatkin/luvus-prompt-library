# Prompt Dock — a luvus module plan

> Historical plan. The shipped design is in `openspec/changes/prompt-dock-poc` and
> the README. Names and versions in this file are out of date.

## Context

A luvus module that adds a sidebar dock for stored prompts:

- Prompts live as **markdown files in a hierarchical folder tree** (folders = categories).
- The dock lists the tree; click a prompt row to edit it inline.
- **Right-click a prompt row** → menu: edit inline (nvim / the luvus "Open files with" editor), open in an external editor (zed, VS Code), and **"Send to <pane>"** — one entry per live pane in the active workspace(s), which pastes the prompt text into that pane via `luvus pane send`.

### Decisions (user-confirmed)

| Question | Decision |
|---|---|
| Language | **TypeScript** on Node (user accepts the Node dependency; no bash — Windows). Node ≥ 23.6 runs `.ts` directly via type stripping; this machine has v24.20.0. Zero npm dependencies (node stdlib only, erasable-only syntax: no `enum`/namespaces). |
| Prompt library location | **Configurable module setting** (`root`), default `~/.prompts`. |
| Send UX | **Menu lists each live pane** — one entry per pane, built at refresh time from `luvus pane list --all-tabs`. |
| Editors | **Honour luvus's "Open files with" setting** (`layout.file_open` in `~/.luvus/config.json`) for inline editing, with a **fallback to `nvim`** when unset/read-only. Plus detached launches for external GUI editors (zed, VS Code). |

## Approach

Modeled on `examples/modules/branch-dock`, in TypeScript.

### Manifest (`luvus-module.toml`)

```toml
id = "you.prompt-dock"
name = "Prompt Dock"
version = "0.1.0"
min_luvus_version = "0.8.3"

[[docks]]
id = "prompts"
title = "PROMPTS"
placement = "sidebar.left"

[[startup]]
command = ["node", "refresh.ts"]

[[events]]   # keep pane send-targets fresh
on = "workspace.created" / "workspace.closed" / "pane.created" / "pane.closed"
command = ["node", "refresh.ts"]

[[actions]]  # refresh (contexts: workspace), toggle (folder click),
             # edit-inline, edit-external (editor in value), send (pane id in value)
[[settings]]
key = "root"          # type string, default "~/.prompts"
```

### Dock rows

- **Flat list, indented by depth** (docks are flat rows). Folder rows `▸/▾ name`, prompt rows the filename (`.md` stripped).
- Folder row click → `toggle` action; expanded set persisted in `LUVUS_MODULE_STATE_DIR/expanded.json`.
- Prompt row click → `edit-inline` (open in editor).
- Prompt row **`menu`** (per-row right-click, writing-modules.mdx "A row's own right-click menu" ~line 534):
  - `Edit inline (nvim / <configured editor>)`
  - `Open in zed`, `Open in VS Code` (entries only when the binary is on PATH)
  - `Send to <pane>` — one entry per pane from `luvus pane list --all-tabs` JSON (id, agent, status, workspace); action `send`, value `"<paneId>|<promptPath>"`.
- Row `value` carries the relative prompt path so one action per kind serves all rows.

### Editing

- **Inline (honours "Open files with")**: read `~/.luvus/config.json` → `layout.file_open`. If it names a detected terminal editor (non-empty, not `"readonly"`), split a pane and run it: `luvus pane split --no-focus` then `luvus pane run <id> <editor-words...> <file>` — mirrors luvus's own editor-as-PTY-pane behavior. Fallback: `nvim <file>`.
- **External**: `zed <file>` / `code <file>` spawned detached with stdio ignored (`unref`), so the action exits immediately (per the module guide's background-process note).

### Send

- `send.ts` reads the prompt file from `value`, calls `luvus pane send <paneId> <text>`. Verify argv limits / whether a stdin mode exists (`luvus help pane send`) during implementation; large prompts may need stdin.

## Files to create — `~/Dev/prompt-dock/`

- `luvus-module.toml`
- `lib.ts` — shared: tree scan, rows JSON, config/setting reads, `luvus()` helper
- `refresh.ts` — scan tree, read pane list, build rows + menus, `ui dock push`
- `edit.ts` — inline (configured editor / nvim) or external (zed/vscode) from `value`
- `send.ts` — send prompt text to the pane in `value`
- `toggle.ts` — expand/collapse a folder row
- `README.md` — screenshot + install notes (module guide convention)

## Reuse

- `~/Dev/luvus/examples/modules/branch-dock/{luvus-module.toml,refresh.sh}` — dock + rows + settings + startup pattern
- `~/Dev/luvus/website/src/content/docs/docs/extend/writing-modules.mdx` — row `menu` (~534), `[[build]]` (not needed), background-process note, env vars
- `luvus pane list --all-tabs` (JSON; `src/cli.rs:3569`, `src/app/dispatch/topology.rs:67`)
- `luvus pane send [<id>] <text>`, `luvus pane split --no-focus`, `luvus pane run`
- `~/.luvus/config.json` `layout.file_open` (`src/config.rs:305`) + `file_open_editor` semantics (`src/app/files.rs:410`: empty/`"readonly"` = no editor)

## Steps

- [ ] 1. Scaffold `~/Dev/prompt-dock/`: manifest + `lib.ts` (scan tree, JSON rows, `luvus()` runner, setting/config reads)
- [ ] 2. `refresh.ts`: render hierarchy (indent, expand/collapse from state), push dock
- [ ] 3. `toggle.ts`: folder expand/collapse persisted in `LUVUS_MODULE_STATE_DIR`
- [ ] 4. `edit.ts`: inline via `layout.file_open` (fallback nvim) in a split pane; detached zed/vscode
- [ ] 5. `send.ts` + per-pane menu entries in refresh
- [ ] 6. Settings wiring (`root`), events for pane/workspace changes
- [ ] 7. README with screenshot

## Verification

```
luvus module link ~/Dev/prompt-dock     # then restart luvus or `module run you.prompt-dock refresh`
luvus module log                        # tail output/errors while iterating
```

- Nested folders + `.md` files under the configured root render indented; expand/collapse survives restarts.
- Clicking a prompt opens nvim (or the configured "Open files with" editor) in a pane, on that file.
- Right-click a prompt: menu shows live panes; sending lands the text in the target pane.
- `zed`/`code` entries launch externally and the action exits cleanly (`module log` shows no 2 s hang note).
- Restart luvus → startup hook repaints the dock.
