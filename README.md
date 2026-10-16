# luvus-prompt-library

A luvus module: a sidebar dock for stored prompts kept as hierarchical
markdown files. Browse the library as a folder tree, click a prompt to select
it, then open it in the luvus "Open files with" editor (fallback `nvim`) or in
an external editor (zed, VS Code) from the right-click menu. That menu also
sends the prompt to a live agent in one click: the text is pasted into the
agent and submitted in one step.

Status: proof of concept. Planned follow-ups are tracked in
[issue #2](https://github.com/slatkin/luvus-prompt-library/issues/2).

## What the dock does

The **PROMPTS** dock (sidebar, left) shows the markdown files under your
prompt library root as a folder tree:

- Folders are collapsed by default. Clicking one expands or collapses it, and
  the expanded set survives restarts.
- The tree is drawn with box-drawing branches. A row with a parent starts with a
  `├─` or `└─` connector. A nested folder that still has a row after it adds a
  `│` guide column to its children, so the pipe shows only where a line really
  joins two rows. Root-level rows have no prefix, and their children draw no
  line above them.
- Every row also starts with a Nerd Font glyph: a folder shows `md-folder`, or
  `md-folder_open` when it is expanded, and a prompt shows `md-chat`. A
  terminal without a Nerd Font shows boxes instead.

```text
 claude
  ├─  mbv-orchestrator
  ├─  pi-orchestrator
  └─  shot
 pi
  ├─  nested
  │ └─  deep
  └─  simple
```
- Clicking a prompt selects it (accent colour); it does not open an editor.
- Only `.md` files show (any letter case); hidden entries (names starting with
  `.`) are skipped. Symlinks show as their targets.
- If the root folder does not exist, the dock shows a single muted row that
  names the missing folder.

**Selecting.** A left-click on a prompt row only selects it — it never opens
an editor. The selected row's glyph and name are drawn in the accent colour and
the row adds no marker of its own, since every prompt row already carries the
chat glyph. The branch prefix keeps the normal row colour, so selecting a row
never recolours the tree. The choice is kept in the module state directory and
survives a restart.

**Opening.** Right-click a prompt row and pick **Open** to open it in the
editor luvus uses for "Open files with" (`layout.file_open` in
`~/.luvus/config.json`). When that setting is empty, `readonly`, or
unreadable, the file opens in `nvim`. The editor opens in a new split pane.
The same menu offers **Edit in zed** and **Edit in VS Code**, each entry only
when the binary is on `PATH`.

**Sending.** The right-click menu lists one **Send to &lt;agent&gt;** entry per
live agent in the current workspace (from `luvus agent list`), above the Open
and Edit entries. Each label names the agent's alias or kind and its pane.
Plain shell panes do not show. Sending pastes the prompt text into the agent
and submits it in one atomic request. The file is read at send time, so edits
saved after the last dock repaint go out. If the agent is not ready (for
example it is waiting on a permission prompt), nothing is queued and a toast
tells you the send failed. A send target from a stale menu is rejected if the
pane now runs a different agent.

The dock repaints itself on startup, on workspace/pane lifecycle events, and
when an agent's status changes, so new agents appear without a manual
refresh. The row menu also has a **Refresh prompts** action if you want to
force one.

## Requirements

- **Go 1.22 or newer** — the module is one Go binary, `prompt-dock`, built
  with `go install` from this checkout. Go is needed only at install time; the
  binary itself has no runtime dependencies and uses the Go standard library
  only. There is no Node and no npm.
- **`GOBIN` on the `PATH` that luvus sees** — `go install` writes the binary to
  `GOBIN` (`~/go/bin` by default), and luvus starts the module by name from
  `PATH`. If luvus starts from a desktop launcher, its `PATH` may differ from
  your shell's; make sure the folder holds `prompt-dock` for that `PATH` too.
  Check with `command -v prompt-dock`. If it is missing, every dock action
  fails with a spawn error that names `prompt-dock`.
- **luvus >= 0.14.3** — needs `luvus agent list` with terminal ids and
  `luvus agent prompt --terminal-id`.
- **A Nerd Font in your terminal** — the dock prefixes each row with a Material
  Design glyph (`md-folder`, `md-folder_open`, `md-chat`). Luvus itself needs
  none, and its own UI avoids font-dependent glyphs, but these row prefixes do.
- Your agent must be running inside luvus so it appears in
  `luvus agent list`.

Linux is the tested platform. The code compiles for Windows, but Windows is
untested.

## Install

Point luvus at a checkout of this repo using the **absolute path**:

```sh
git clone https://github.com/slatkin/luvus-prompt-library
cd luvus-prompt-library
go install ./cmd/prompt-dock
command -v prompt-dock   # must print a path
luvus module link "$(pwd)"
```

> Use the absolute path. `luvus module link .` failed in earlier testing —
> always link with a full path.

Run `go install ./cmd/prompt-dock` again after each `git pull`: the linked
checkout and the installed binary can differ in version until you rebuild.

For development, set up the pre-commit hook once in the checkout:

```sh
git config core.hooksPath .githooks
```

Then restart luvus. The **PROMPTS** dock appears in the left sidebar (you
can move it in Settings → Layout).

To remove the module (the id comes from `luvus-module.toml`):

```sh
luvus module unlink slatkin.prompt-dock
```

Installing from GitHub also works once the module is published:

```sh
luvus module install slatkin/luvus-prompt-library
```

## Configure the prompt library root

The dock reads the `root` module setting (Settings → your module
configuration). Default: `~/.prompts`. A leading `~` is expanded to your home
folder, and an empty value falls back to the default. Use an absolute path or
one that starts with `~`.

Create the folder and add some markdown files:

```sh
mkdir -p ~/.prompts
cat > ~/.prompts/review.md <<'EOF'
Review the current diff. List bugs first, then style nits.
Keep it short.
EOF
```

Press **Refresh prompts** in the dock (or restart luvus) and the tree
appears.

## Known limits

- Very large prompts can exceed the argv size limit (about 32 K chars on
  Windows, 128 KB per argument on Linux). Same roadmap item.
- Inline editing quotes the file path for POSIX shells (sh, bash, zsh,
  fish). A path containing an apostrophe, a backslash, or a line break is
  refused with a toast. PowerShell needs different quoting — tracked in
  issue #2.
- The dock does not watch the filesystem; it repaints on luvus events and on
  the **Refresh prompts** action, not on file changes.
- Luvus fires no event when you switch workspaces, so the **Send to** entries
  follow the workspace on the next repaint: a pane opening or closing, an
  agent status change, or any click in the dock.

Roadmap (edit-before-submit, stdin send path, PowerShell quoting, Commander
`$prompt` exposure) is tracked in
[issue #2](https://github.com/slatkin/luvus-prompt-library/issues/2).
