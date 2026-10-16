# Tasks

## 1. Scaffold

- [x] 1.1 Add `luvus-module.toml` at the repo root (id, dock `prompts` on `sidebar.left`, `root` string setting with default `~/.prompts`, actions `refresh`/`toggle`/`select`/`edit-inline`/`edit-external`/`send`, startup + events `workspace.created`/`workspace.closed`/`pane.created`/`pane.closed`/`pane.agent_status_changed`, all running `node <script>.ts`). Verify: `luvus module link .` succeeds and the PROMPTS dock appears in Settings → Layout.
- [x] 1.2 Add `lib.ts` with a `luvus()` runner (execFileSync on `LUVUS_BIN_PATH`, falling back to `luvus`, no shell), root resolution with `~` expansion, and a relpath guard that rejects paths outside root (design D3). Verify: `node --test` passes `lib.test.ts` cases for `~` expansion and for rejection of `../x.md`.

## 2. Tree dock (spec: prompt-dock)

- [x] 2.1 Implement the tree scan in `lib.ts`: depth-first, folders before files, sorted by name, `.md` only, hidden entries skipped, collapsed folders' children left out. Verify: `lib.test.ts` builds a temp tree that matches the "Nested library" and "Ignored files" scenarios and checks the row order and indentation.
- [x] 2.2 Implement `refresh.ts` to build rows (folder `▸/▾`, prompt names without `.md`, the missing-root muted row) and push them with `ui dock push` through stdin. Verify: run `luvus module run <id> refresh` with a sample `~/.prompts`. The dock shows the tree, and a non-existent root shows the single muted row.
- [x] 2.3 Implement `toggle.ts` to keep the expanded set in `LUVUS_MODULE_STATE_DIR/expanded.json` (treat a corrupt or missing file as empty) and then run refresh. Verify: clicking a folder expands and collapses it, and the state survives a luvus restart.

## 3. Editing (spec: prompt-dock)

- [x] 3.1 Implement `edit.ts` inline mode: read `layout.file_open` (fall back to `nvim` when empty, `readonly` or unreadable), `pane split --no-focus`, then `pane run <pane> <editor> '<abs>'`. Refuse a path that contains `'` with a toast. Verify: clicking a prompt row opens it in the configured editor in a new pane, and with `readonly` set it opens in `nvim`.
- [x] 3.2 Implement `edit.ts` external mode (detached spawn, `stdio: "ignore"`, `unref`) and the stdlib PATH probe that adds the zed/VS Code menu entries only when the binary is found. Verify: the entry for a missing binary does not show, "Open in zed" opens the file, and `luvus module log` shows no hang note.

## 4. Send to agent (spec: agent-prompt-send)

- [x] 4.1 In `lib.ts`, add value encode/decode for `<pane>|<terminal_id>|<relpath>` (split on the first two `|` only) and the target label builder (design D8). Verify: `lib.test.ts` covers a relpath that contains `|`, plus the named and unnamed label scenarios.
- [x] 4.2 In `refresh.ts`, read `agent list` and add one "Send to" menu entry per agent to each prompt row. Verify: with a claude pane and a pi pane open, the menu lists both, and a plain shell pane is not listed.
- [x] 4.3 Implement `send.ts`: read the file at action time, run `agent prompt <pane> <text> --terminal-id <id>` with the text as one argv element, and show a toast on success and on failure. Verify: a multi-line prompt arrives in an idle agent as one submitted prompt. A send to an agent that waits on a permission prompt shows a failure toast and nothing is typed. A send from a menu built before its pane closed shows a failure toast.
- [ ] 4.4 Verify that the target list follows agents: start `claude` in an existing shell pane and check that it appears in the prompt row menus with no manual refresh.

## 5. Docs and integration

- [ ] 5.1 Update `README.md` with link/install steps, the `root` setting, Node ≥ 23.6 and luvus ≥ 0.14.3 requirements, and a screenshot of the dock with an open send menu. Verify: following the README on a clean `~/.prompts` gives a working dock.
- [ ] 5.2 Run the end-to-end check: restart luvus, the dock repaints from startup, edit a prompt inline, send it to an agent, and confirm that the agent receives the edited text. Verify: all steps pass with no errors in `luvus module log`.

## 6. Interaction rework (spec: prompt-dock, agent-prompt-send)

- [x] 6.1 Left-click selects instead of opening (supersedes the 3.1/3.2 entry point): prompt rows carry `action: "select"`, `select.ts` keeps the selection in `LUVUS_MODULE_STATE_DIR/selected.json`, and the selected row is drawn with `spans` (accent tone + `▌` marker) so its `text` stays the plain label. Verify: a left-click on a prompt creates no pane, shows the marker, and survives a restart.
- [x] 6.2 Put opening in the row menu, in this order: "Send to" entries, `Open` (inline editor pane), `Edit in zed` / `Edit in VS Code`. Verify: the menu lists the send targets first, and `Open` starts the configured editor in a new pane.
- [x] 6.3 Row colours and header: folders take `tone: "accent"`, prompts keep the default, and every push carries `--title PROMPTS`. Verify: the header reads `PROMPTS` in upper case.
- [x] 6.4 Shorten the "Send to" labels to name, kind and pane (design D8), and drop the now-unused `status`/`workspace_name` from the parsed agent. Verify: a menu entry reads `bruv-orch (pi) · pane 12`.
- [x] 6.5 Limit the "Send to" entries to the focused workspace (design D11): keep the agents whose `workspace` id equals `LUVUS_WORKSPACE_ID`. Verify: with agents in two workspaces the menu lists only the focused one's, the id was confirmed live with a throwaway probe module, and `lib.test.ts` covers the filter and the unknown-id fallback.

## Workflow follow-up

- Roadmap items (edit-before-submit, stdin send path, PowerShell quoting, Commander `$prompt`) are tracked in roadmap issue #2, not in this change.
- Archive the change after review.
