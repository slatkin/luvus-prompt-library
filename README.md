# luvus-prompt-library

A luvus module: a sidebar dock for stored prompts kept as hierarchical
markdown files. Browse the library as a folder tree, edit a prompt inline
(the luvus "Open files with" editor, fallback `nvim`) or in an external
editor (zed, VS Code), and send it to any live agent with one menu click.
Sending pastes the prompt into the agent and submits it in one step.

Status: proof of concept. Planned follow-ups are tracked in
[issue #2](https://github.com/slatkin/luvus-prompt-library/issues/2).

<!-- TODO screenshot: dock with open send menu -->

## What the dock does

The **PROMPTS** dock (sidebar, left) shows the markdown files under your
prompt library root as a folder tree:

- Folders expand and collapse on click; the expanded set survives restarts.
- Only `.md` files show (any letter case); hidden entries (names starting with
  `.`) are skipped. Symlinks show as their targets.
- If the root folder does not exist, the dock shows a single muted row that
  names the missing folder.

**Editing.** Click a prompt row to open it in the editor luvus uses for
"Open files with" (`layout.file_open` in `~/.luvus/config.json`). When that
setting is empty, `readonly`, or unreadable, the file opens in `nvim`. The
editor opens in a new split pane. Right-clicking a prompt row also offers
"Open in zed" and "Open in VS Code" — each entry appears only when the
binary is on `PATH`.

**Sending.** Right-click a prompt and pick **Send to &lt;agent&gt;**. One
entry is listed per live agent (from `luvus agent list`), labelled with its name
or agent and pane, status, and workspace. Plain shell panes
do not show. Sending pastes the prompt text into the agent and submits it in
one atomic request. The file is read at send time, so edits saved after the
last dock repaint go out. If the agent is not ready (for example it is
waiting on a permission prompt), nothing is queued and a toast tells you the
send failed. A send target from a stale menu is rejected if the pane now
runs a different agent.

The dock repaints itself on startup, on workspace/pane lifecycle events, and
when an agent's status changes, so new agents appear without a manual
refresh. The row menu also has a **Refresh prompts** action if you want to
force one.

## Requirements

- **Node >= 23.6** — the module scripts are TypeScript run directly by
  `node` (type stripping); the module needs no npm dependencies to run.
  (The dev tooling for lint/format/CI gates uses devDependencies only.)
- **luvus >= 0.14.3** — needs `luvus agent list` with terminal ids and
  `luvus agent prompt --terminal-id`.
- Your agent must be running inside luvus so it appears in
  `luvus agent list`.

## Install

Point luvus at a checkout of this repo using the **absolute path**:

```sh
git clone https://github.com/slatkin/luvus-prompt-library
cd luvus-prompt-library
luvus module link "$(pwd)"
```

> Use the absolute path. `luvus module link .` failed in earlier testing —
> always link with a full path.

For development, run `npm install` once in the checkout. Its `prepare` script
points git at the pre-commit hook in `.githooks`.

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

- A prompt that **starts with `--`** is read as an option by
  `luvus agent prompt` and the send fails with a toast. Rename the file's
  content to start with text, or wait for the stdin send path (issue #2).
- Very large prompts can exceed the argv size limit (about 32 K chars on
  Windows, 128 KB per argument on Linux). Same roadmap item.
- Inline editing quotes the file path for POSIX shells (sh, bash, zsh,
  fish). A path containing an apostrophe, a backslash, or a line break is
  refused with a toast. PowerShell needs different quoting — tracked in
  issue #2.
- The dock does not watch the filesystem; it repaints on luvus events and on
  the **Refresh prompts** action, not on file changes.

Roadmap (edit-before-submit, stdin send path, PowerShell quoting, Commander
`$prompt` exposure) is tracked in
[issue #2](https://github.com/slatkin/luvus-prompt-library/issues/2).
