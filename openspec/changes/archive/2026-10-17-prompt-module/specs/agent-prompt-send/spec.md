# Spec Delta

## MODIFIED Requirements

### Requirement: Send submits the prompt
Selecting a "Send to" entry SHALL read the prompt file at that moment and give its body to the selected agent as one prompt, then submit it. The body is the file text with a leading YAML frontmatter block removed and the placeholders filled in (spec: prompt-arguments, placeholder fill). A frontmatter block starts with a `---` first line and ends at the next `---` line; with no closing line, nothing is removed. The text SHALL keep its newlines. The text SHALL be delivered literally, whatever it starts with: a leading `-` SHALL NOT be read as an option. Edits saved to the file after the last dock repaint SHALL be included. A body that is empty after the frontmatter is removed and the placeholders are filled SHALL be refused as an empty prompt.

#### Scenario: Multi-line prompt
- **WHEN** the user sends a three-line prompt to an idle agent
- **THEN** the agent receives the three lines as one submitted prompt

#### Scenario: Prompt text starts with a dash
- **WHEN** the user sends a prompt whose body starts with a `- ` bullet
- **THEN** the agent receives that text unchanged, and no option error is raised

#### Scenario: Frontmatter is removed
- **WHEN** the user sends a file that starts with `---`, `description: x`, `---`, then `body line`
- **THEN** the agent receives `body line` and none of the frontmatter lines

#### Scenario: Unclosed frontmatter
- **WHEN** the user sends a file whose first line is `---` and that has no second `---` line
- **THEN** the agent receives the whole file text

#### Scenario: Only frontmatter
- **WHEN** the user sends a file that has a frontmatter block and no body
- **THEN** nothing is sent and a toast says the prompt is empty

#### Scenario: Body empty after the fill
- **WHEN** the body is only `$ARGUMENTS` and the value is empty
- **THEN** nothing is sent and a toast says the prompt is empty

#### Scenario: File edited after repaint
- **WHEN** the user saves new text to a prompt file and then sends it without a refresh
- **THEN** the agent receives the new text
