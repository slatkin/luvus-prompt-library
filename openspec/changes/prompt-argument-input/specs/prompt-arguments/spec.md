# Spec Delta

## ADDED Requirements

### Requirement: Argument hint opens the input panel
When the user selects a "Send to" entry for a prompt whose frontmatter has a non-empty `argument-hint`, the module SHALL open a temporary input panel as a split pane and SHALL NOT send the prompt yet. The file SHALL be read at that moment, so a hint added after the last repaint counts. A prompt whose body is empty after the frontmatter is removed SHALL be refused before the panel opens.

#### Scenario: Prompt with a hint
- **WHEN** the user sends `mbv-orchestrator`, whose frontmatter has `argument-hint: "[change-name]"`, to an agent
- **THEN** an input panel opens in a split pane and no text reaches the agent yet

#### Scenario: Hint with no body
- **WHEN** the file has a frontmatter block with an `argument-hint` and no body
- **THEN** no panel opens, nothing is sent, and a toast says the prompt is empty

#### Scenario: Empty hint
- **WHEN** the frontmatter has `argument-hint: ""`
- **THEN** no panel opens and the prompt is sent at once

### Requirement: Input panel content
The input panel SHALL show the prompt name, the target agent label (spec: agent-prompt-send, target labels), and a one-line text field that shows the hint text while the field is empty. The text field SHALL have the keyboard focus when the panel opens.

#### Scenario: Panel shows its context
- **WHEN** the panel opens for prompt `mbv-orchestrator` and target `claude · pane 7` with hint `[change-name]`
- **THEN** the panel shows `mbv-orchestrator`, `claude · pane 7`, and an empty field that shows `[change-name]`

#### Scenario: Typing goes to the field
- **WHEN** the panel opens and the user types `my-change`
- **THEN** the field shows `my-change`

### Requirement: Submit sends the filled prompt
When the user presses Enter in the input panel, the module SHALL send the prompt to the target agent with the typed value filled in (spec: prompt-arguments, placeholder fill), and the panel SHALL close. An empty value SHALL be allowed. The send SHALL obey the stale target guard and the failure feedback of agent-prompt-send.

#### Scenario: Value given
- **WHEN** the user types `my-change` and presses Enter
- **THEN** the agent gets the prompt with `my-change` filled in, a success toast shows, and the panel closes

#### Scenario: No value given
- **WHEN** the user presses Enter with an empty field
- **THEN** the agent gets the prompt with the empty value filled in, and the panel closes

#### Scenario: Target gone while the panel is open
- **WHEN** the target agent's pane closes before the user presses Enter
- **THEN** no text reaches any pane, an error toast shows, and the panel closes

### Requirement: Cancel sends nothing
When the user presses Esc or Ctrl+C in the input panel, the panel SHALL close and no text SHALL reach any agent.

#### Scenario: Esc
- **WHEN** the user types `my-change` and presses Esc
- **THEN** the panel closes and the agent gets no text

### Requirement: No hint sends at once
When a prompt has no `argument-hint`, the module SHALL send it at once, with no panel and with no value (spec: prompt-arguments, placeholder fill).

#### Scenario: Placeholder without a hint
- **WHEN** the user sends `shot`, which has `$ARGUMENTS` in the body and no `argument-hint`
- **THEN** no panel opens, and the agent gets the body with `$ARGUMENTS` removed

### Requirement: Focus returns when the panel closes
When the input panel closes, by submit or by cancel, the module SHALL give the focus back to the pane that had the focus before the panel opened, when that pane still exists.

#### Scenario: Focus back after a send
- **WHEN** pane `3` has the focus, the user sends a hinted prompt from the dock, and then presses Enter in the panel
- **THEN** after the panel closes, pane `3` has the focus

### Requirement: A left-over panel sends nothing
An input panel that luvus opens again after a restart, or that opens with no pending request, SHALL close at once and SHALL NOT send text to any agent.

#### Scenario: Restart with the panel open
- **WHEN** the input panel is open, the user quits luvus, and luvus opens the panel again on the next launch
- **THEN** the panel closes at once and no agent gets text
