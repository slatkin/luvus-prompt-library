# Proposal

Source issue: [#1 Prompt Dock module: plan seed](https://github.com/slatkin/luvus-prompt-library/issues/1). Roadmap: [#2](https://github.com/slatkin/luvus-prompt-library/issues/2).

## Why

Reusable prompts are kept as loose markdown files. To use one, you open the file, copy the text, find the agent pane and paste it. A luvus sidebar dock that shows the prompt library and sends a prompt to a live agent in one click removes these steps. This change is a proof of concept (PoC) that proves the full loop: browse, edit, send.

## What Changes

- New luvus module at the root of this repo (`luvus-module.toml` + TypeScript scripts run by Node, zero npm dependencies).
- A sidebar dock `PROMPTS` that shows a configurable folder tree of `.md` prompt files. Folders expand and collapse, and the state persists.
- Click a prompt row to select it; the row menu opens it in the luvus "Open files with" editor (fallback `nvim`) in a split pane, or in a detached external editor (zed, VS Code) when that binary is on PATH.
- The row menu lists one "Send to &lt;agent&gt;" entry for each live agent in the focused workspace, from `luvus agent list`, above the Open and Edit entries. Non-agent panes do not show.
- Send means **paste and submit**: `luvus agent prompt <pane> --terminal-id <id> -- <text>`, with the text after `--` so a prompt that starts with `-` is not read as an option. The terminal-id guard makes sure that a stale menu cannot send the prompt to a different agent that reuses the pane id.
- The dock refreshes on startup, on workspace/pane lifecycle events and on `pane.agent_status_changed`, so new agents appear in the menu.

Out of scope (tracked in roadmap issue #2):
- Edit before submit (load the prompt into an editor or modal, edit it, then submit).
- A stdin send path (`luvus uhp proxy`) for prompts larger than the argv limit. (A leading `-` is handled by the `--` separator.)
- Commander `$prompt @agent` exposure.

## Capabilities

### New Capabilities
- `prompt-dock`: shows the prompt library tree in a luvus dock, expands and collapses folders, and opens prompts in inline or external editors.
- `agent-prompt-send`: lists live agents as send targets for a prompt and submits the prompt text to the selected agent.

### Modified Capabilities
- None.

## Impact

- New files at the repo root: `luvus-module.toml`, `lib.ts`, `refresh.ts`, `toggle.ts`, `select.ts`, `edit.ts`, `send.ts`, plus README updates. (PLAN.md said `~/Dev/prompt-dock/`. This repo is the module, so the files go here.)
- Runtime dependencies: Node >= 23.6 (type stripping), luvus >= the version that has `agent list` with `terminal_id` and `agent prompt --terminal-id` (0.14.3 is verified).
- Luvus CLI surfaces used: `ui dock push`, `agent list`, `agent prompt`, `pane split`, `pane run`. Config read: `~/.luvus/config.json` `layout.file_open`.
- No change to luvus itself.
