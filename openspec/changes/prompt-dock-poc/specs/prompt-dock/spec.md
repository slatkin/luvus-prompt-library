# Spec Delta

## Purpose

Shows a folder tree of markdown prompt files in a luvus sidebar dock and lets the user open each prompt in an inline or external editor.

## ADDED Requirements

### Requirement: Configurable library root
The module SHALL read the prompt library from the folder in the module setting `root`. The default is `~/.prompts`. A leading `~` SHALL expand to the user's home folder.

#### Scenario: Default root
- **WHEN** the user has not set `root`
- **THEN** the dock shows the contents of `~/.prompts`

#### Scenario: Missing root
- **WHEN** the `root` folder does not exist
- **THEN** the dock shows one muted row that names the missing folder, and no other rows

### Requirement: Tree rows
The dock SHALL show one row per folder and one row per `.md` file under the root, in depth-first order, indented by depth. Folders SHALL come before files at each level, and each group SHALL be sorted by name. Prompt rows SHALL show the file name without `.md`. Hidden entries (names that start with `.`) and non-`.md` files SHALL NOT show.

#### Scenario: Nested library
- **WHEN** the root has `review/rust.md`, `review/ts.md` and `plan.md`, and `review` is expanded
- **THEN** the dock shows `review` (folder), then `rust` and `ts` indented one level, then `plan`

#### Scenario: Ignored files
- **WHEN** the root has `notes.txt` and `.git/`
- **THEN** neither shows in the dock

### Requirement: Expand and collapse folders
Clicking a folder row SHALL toggle it between expanded and collapsed. The children of a collapsed folder SHALL NOT show. The expanded set SHALL persist across luvus restarts. Folders SHALL be collapsed by default.

#### Scenario: Toggle a folder
- **WHEN** the user clicks the collapsed folder row `review`
- **THEN** the dock repaints with `review` expanded and its children visible

#### Scenario: State survives restart
- **WHEN** `review` is expanded and luvus restarts
- **THEN** the dock repaints with `review` still expanded

### Requirement: Inline edit
Clicking a prompt row SHALL open that file in a new split pane, without moving focus, in the editor set in luvus "Open files with" (`layout.file_open`). When that setting is empty, `readonly` or not readable, the module SHALL use `nvim`.

#### Scenario: Configured editor
- **WHEN** `layout.file_open` is `hx` and the user clicks the prompt row `plan`
- **THEN** a new pane opens and runs `hx` on `<root>/plan.md`

#### Scenario: Fallback editor
- **WHEN** `layout.file_open` is `readonly` and the user clicks a prompt row
- **THEN** the new pane runs `nvim` on that file

### Requirement: External editors
The right-click menu of a prompt row SHALL have "Open in zed" and "Open in VS Code" entries only when `zed` and `code` are on PATH. Selecting one SHALL start that editor on the file, detached, and the action SHALL exit without waiting for the editor.

#### Scenario: Editor not installed
- **WHEN** `code` is not on PATH
- **THEN** the prompt row menu has no "Open in VS Code" entry

#### Scenario: Detached launch
- **WHEN** the user selects "Open in zed"
- **THEN** zed opens the file and the module action finishes at once

### Requirement: Dock refresh
The dock SHALL repaint on luvus startup and when the user runs the `refresh` action. A repaint SHALL show files that were added or removed since the last repaint.

#### Scenario: Startup repaint
- **WHEN** luvus starts with the module installed
- **THEN** the dock shows the current library tree

#### Scenario: Manual refresh
- **WHEN** the user adds `new.md` to the root and runs `refresh`
- **THEN** the dock shows `new`
