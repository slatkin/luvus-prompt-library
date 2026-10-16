# Spec Delta

## Purpose

Shows a folder tree of markdown prompt files in a luvus sidebar dock. A left-click selects a prompt; opening, editing and sending are choices in the row's right-click menu.

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
The dock SHALL show one row per folder and one row per `.md` file under the root, in depth-first order. A child row SHALL start one space to the right of its parent's name, at every level of the tree. Folders SHALL come before files at each level, and each group SHALL be sorted by name. Prompt rows SHALL show the file name without `.md`. Hidden entries (names that start with `.`) and non-`.md` files SHALL NOT show.

#### Scenario: Nested library
- **WHEN** the root has `review/rust.md`, `review/ts.md` and `plan.md`, and `review` is expanded
- **THEN** the dock shows `review` (folder), then `rust` and `ts` indented one level, then `plan`

#### Scenario: Ignored files
- **WHEN** the root has `notes.txt` and `.git/`
- **THEN** neither shows in the dock

### Requirement: Row glyphs
Each folder row SHALL start with a Nerd Font glyph: `md-folder` when the folder is collapsed, `md-folder_open` when it is expanded. Each prompt row SHALL start with the `md-chat` glyph. The terminal SHALL have a Nerd Font, or these glyphs show as boxes.

#### Scenario: Collapsed and expanded folders
- **WHEN** `review` is collapsed and `claude` is expanded
- **THEN** `review` shows `md-folder` and `claude` shows `md-folder_open`

#### Scenario: Prompt glyph
- **WHEN** the dock shows the prompt `plan`
- **THEN** `plan` shows the `md-chat` glyph

### Requirement: Expand and collapse folders
Clicking a folder row SHALL toggle it between expanded and collapsed. The children of a collapsed folder SHALL NOT show. The expanded set SHALL persist across luvus restarts. Folders SHALL be collapsed by default.

#### Scenario: Toggle a folder
- **WHEN** the user clicks the collapsed folder row `review`
- **THEN** the dock repaints with `review` expanded and its children visible

#### Scenario: State survives restart
- **WHEN** `review` is expanded and luvus restarts
- **THEN** the dock repaints with `review` still expanded

### Requirement: Open in the editor pane
The right-click menu of a prompt row SHALL have an `Open` entry. Selecting it SHALL open that file in a new split pane, without moving focus, in the editor set in luvus "Open files with" (`layout.file_open`). When that setting is empty, `readonly` or not readable, the module SHALL use `nvim`. A left-click on a prompt row SHALL NOT open the file.

#### Scenario: Configured editor
- **WHEN** `layout.file_open` is `hx` and the user picks `Open` on the prompt row `plan`
- **THEN** a new pane opens and runs `hx` on `<root>/plan.md`

#### Scenario: Fallback editor
- **WHEN** `layout.file_open` is `readonly` and the user picks `Open` on a prompt row
- **THEN** the new pane runs `nvim` on that file

### Requirement: External editors
The right-click menu of a prompt row SHALL have an "Edit in zed" entry only when `zed` is on PATH, and an "Edit in VS Code" entry only when `code` is on PATH. Selecting one SHALL start that editor on the file, detached, and the action SHALL exit without waiting for the editor.

#### Scenario: Editor not installed
- **WHEN** `code` is not on PATH
- **THEN** the prompt row menu has no "Edit in VS Code" entry

#### Scenario: Detached launch
- **WHEN** the user selects "Edit in zed"
- **THEN** zed opens the file and the module action finishes at once

### Requirement: Select a prompt
A left-click on a prompt row SHALL select that prompt and SHALL NOT open it. The selected row SHALL be drawn in the accent tone and SHALL NOT add a marker glyph of its own. Luvus gives a module dock row no bold and no row background, so the accent tone is the closest available match to the focused row of the built-in docks. The selection SHALL persist across luvus restarts. A prompt that is no longer in the library SHALL NOT draw as selected.

#### Scenario: Left-click selects
- **WHEN** the user left-clicks the prompt row `plan`
- **THEN** `plan` is drawn in the accent tone, no pane is created, and no editor starts

#### Scenario: Selection survives restart
- **WHEN** `plan` is selected and luvus restarts
- **THEN** the dock repaints with `plan` still drawn as selected

### Requirement: Row colours
Folder rows SHALL use the accent tone. Prompt rows SHALL use the default row colour, except a selected prompt row, which SHALL use the accent tone.

#### Scenario: Folders are distinct from prompts
- **WHEN** the dock shows a folder row and a prompt row
- **THEN** the folder is drawn in the accent tone and the prompt in the default colour

### Requirement: Row menu order
The right-click menu of a prompt row SHALL list the "Send to" entries first, then the `Open` entry, then the external-editor entries.

#### Scenario: Menu order
- **WHEN** one agent is live and `zed` and `code` are on PATH
- **THEN** the menu lists the "Send to" entry, then `Open`, then "Edit in zed", then "Edit in VS Code"

### Requirement: Dock header
The dock header SHALL read `PROMPTS`.

#### Scenario: Header case
- **WHEN** the dock paints after a restart or a refresh
- **THEN** its header reads `PROMPTS`

### Requirement: Dock refresh
The dock SHALL repaint on luvus startup and when the user runs the `refresh` action. A repaint SHALL show files that were added or removed since the last repaint.

#### Scenario: Startup repaint
- **WHEN** luvus starts with the module installed
- **THEN** the dock shows the current library tree

#### Scenario: Manual refresh
- **WHEN** the user adds `new.md` to the root and runs `refresh`
- **THEN** the dock shows `new`
