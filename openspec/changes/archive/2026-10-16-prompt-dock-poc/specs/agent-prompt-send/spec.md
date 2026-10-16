# Spec Delta

## Purpose

Lets the user send a stored prompt to a live luvus agent from the prompt dock. The prompt text is pasted into the agent and submitted in one step.

## ADDED Requirements

### Requirement: Agent send targets
The right-click menu of each prompt row SHALL have one "Send to" entry for each live agent that luvus reports in the focused workspace. Agents in any other workspace SHALL NOT appear. Panes with no detected agent SHALL NOT show as targets. When no agent is live in the focused workspace, the menu SHALL have no "Send to" entries. The "Send to" entries SHALL come before the row's other menu entries (spec: prompt-dock, row menu order).

#### Scenario: Two live agents in the focused workspace
- **WHEN** luvus reports a `claude` agent and a `pi` agent in the focused workspace, and the user right-clicks a prompt row
- **THEN** the menu has two "Send to" entries, one for each agent

#### Scenario: An agent in another workspace
- **WHEN** an agent runs in a workspace that is not the focused one
- **THEN** no "Send to" entry names that agent

#### Scenario: Shell pane is not a target
- **WHEN** a pane in the focused workspace runs a plain shell with no detected agent
- **THEN** no "Send to" entry names that pane

### Requirement: Target labels
Each "Send to" entry label SHALL name the agent by its alias when it has one, else by its kind, and SHALL show the pane id. The label SHALL NOT show the agent's status or its workspace name.

#### Scenario: Named agent
- **WHEN** a `pi` agent with the alias `pipi` is in pane `9`
- **THEN** its entry label is `pipi (pi) · pane 9`

#### Scenario: Unnamed agent
- **WHEN** a `claude` agent in pane `118` has no alias
- **THEN** its entry label is `claude · pane 118`

### Requirement: Send submits the prompt
Selecting a "Send to" entry SHALL read the prompt file at that moment and give its full text to the selected agent as one prompt, then submit it. The text SHALL keep its newlines. The text SHALL be delivered literally, whatever it starts with: a leading `-` SHALL NOT be read as an option. Edits saved to the file after the last dock repaint SHALL be included.

#### Scenario: Multi-line prompt
- **WHEN** the user sends a three-line prompt to an idle agent
- **THEN** the agent receives the three lines as one submitted prompt

#### Scenario: Prompt text starts with a dash
- **WHEN** the user sends a prompt whose first line is `---` frontmatter, or that starts with a bullet
- **THEN** the agent receives that text unchanged, and no option error is raised

#### Scenario: File edited after repaint
- **WHEN** the user saves new text to a prompt file and then sends it without a refresh
- **THEN** the agent receives the new text

### Requirement: Stale target guard
A send SHALL go only to the terminal that the menu entry was built for. When that terminal no longer exists, including when a different agent now uses the same pane id, the send SHALL NOT deliver text to any pane.

#### Scenario: Pane id reused
- **WHEN** the target agent's pane closed and a new agent now has the same pane id
- **THEN** no text reaches the new agent and the user sees an error toast

### Requirement: Send failure feedback
When luvus rejects a send (for example, the agent is not ready for input, the target is gone, or the file cannot be read), the module SHALL show a toast that states the failure and the agent label. A successful send SHALL show a toast that names the prompt and the agent.

#### Scenario: Agent not ready
- **WHEN** the target agent is waiting for a permission answer and luvus refuses the prompt
- **THEN** a toast says that the send to that agent failed, and the prompt is not queued

#### Scenario: Success
- **WHEN** the send to `pipi` of prompt `plan` succeeds
- **THEN** a toast names `plan` and `pipi`

### Requirement: Target list stays current
The set of "Send to" entries SHALL be rebuilt when a workspace or pane is created or closed and when an agent's status changes, so an agent that starts in an existing pane appears without a manual refresh. The focused workspace SHALL be read again on each repaint.

#### Scenario: Agent starts in an existing shell
- **WHEN** the user starts `claude` in a shell pane that was already open
- **THEN** after luvus detects the agent, prompt row menus include that agent

#### Scenario: Workspace switch
- **WHEN** the user switches to another workspace and the dock repaints
- **THEN** the menu lists only the agents of the workspace the user is in
