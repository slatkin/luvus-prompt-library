# Spec Delta

## Purpose

Lets the user send a stored prompt to a live luvus agent from the prompt dock. The prompt text is pasted into the agent and submitted in one step.

## ADDED Requirements

### Requirement: Agent send targets
The right-click menu of each prompt row SHALL have one "Send to" entry for each live agent that luvus reports, across all workspaces and tabs. Panes with no detected agent SHALL NOT show as targets. When no agent is live, the menu SHALL have no "Send to" entries.

#### Scenario: Two live agents
- **WHEN** luvus reports a `claude` agent and a `pi` agent, and the user right-clicks a prompt row
- **THEN** the menu has two "Send to" entries, one for each agent

#### Scenario: Shell pane is not a target
- **WHEN** a pane runs a plain shell with no detected agent
- **THEN** no "Send to" entry names that pane

### Requirement: Target labels
Each "Send to" entry label SHALL name the agent by its alias when it has one, else by its kind and pane id. The label SHALL also show the agent's status and its workspace name.

#### Scenario: Named agent
- **WHEN** a `pi` agent has the alias `pipi`, its status is `idle` and its workspace is `pi-luvus`
- **THEN** its entry label contains `pipi`, `pi`, `idle` and `pi-luvus`

#### Scenario: Unnamed agent
- **WHEN** a `claude` agent in pane `118` has no alias
- **THEN** its entry label contains `claude` and `118`

### Requirement: Send submits the prompt
Selecting a "Send to" entry SHALL read the prompt file at that moment and give its full text to the selected agent as one prompt, then submit it. The text SHALL keep its newlines. Edits saved to the file after the last dock repaint SHALL be included.

#### Scenario: Multi-line prompt
- **WHEN** the user sends a three-line prompt to an idle agent
- **THEN** the agent receives the three lines as one submitted prompt

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
The set of "Send to" entries SHALL be rebuilt when a workspace or pane is created or closed and when an agent's status changes, so an agent that starts in an existing pane appears without a manual refresh.

#### Scenario: Agent starts in an existing shell
- **WHEN** the user starts `claude` in a shell pane that was already open
- **THEN** after luvus detects the agent, prompt row menus include that agent
