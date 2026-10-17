# Spec Delta

## Purpose

Lets a stored prompt take an argument: the rules that fill an argument value into the prompt text before the dock sends it to an agent.

## ADDED Requirements

### Requirement: Placeholder fill
Before every send, the module SHALL replace each `${ARGUMENTS:-x}` with the value, or with `x` when the value is empty, and each `$ARGUMENTS` with the value. A send that has no value SHALL use an empty value. All other text SHALL stay unchanged. Placeholders inside the value or inside `x` SHALL NOT be replaced again.

#### Scenario: Plain placeholder
- **WHEN** the body is `Review $ARGUMENTS now.` and the value is `HEAD~1`
- **THEN** the agent gets `Review HEAD~1 now.`

#### Scenario: Default used
- **WHEN** the body is `Target: ${ARGUMENTS:-none}.` and the value is empty
- **THEN** the agent gets `Target: none.`

#### Scenario: Default not used
- **WHEN** the body is `Target: ${ARGUMENTS:-none}.` and the value is `my-change`
- **THEN** the agent gets `Target: my-change.`

#### Scenario: Value is not filled again
- **WHEN** the body is `Run $ARGUMENTS.` and the value is `$ARGUMENTS`
- **THEN** the agent gets `Run $ARGUMENTS.`

#### Scenario: Other dollar text
- **WHEN** the body has `$HOME` and `` !`clip-shot.sh` ``
- **THEN** the agent gets both unchanged

#### Scenario: Send with no value
- **WHEN** the user sends `shot`, which has `$ARGUMENTS` in the body, and no value is given
- **THEN** the agent gets the body with `$ARGUMENTS` removed
