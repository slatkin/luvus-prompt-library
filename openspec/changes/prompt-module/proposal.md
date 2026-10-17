# Proposal

## Why

`send()` reads the prompt file, removes a BOM, and checks for an empty body
inline. The in-flight change `prompt-argument-input` adds three more steps to
that same function (frontmatter removal, `argument-hint` read, `$ARGUMENTS`
fill), and it adds a second caller (the input panel), which would have to know
the same steps. Issue #7, finding A: put the prompt text steps in one Prompt
module with a small interface, before the panel work starts, so that the panel
change has a small surface and the text rules have pure tests.

## What Changes

- A new Prompt module, `cmd/prompt-dock/prompt.go`, with two operations:
  load a prompt file (read, remove the BOM, split off the frontmatter, read the
  `argument-hint`, refuse an empty body) and fill a value into the body.
  `send()` keeps only the value parse, the root and target guards, the
  `agent prompt` call, and the toasts.
- **BREAKING** Every send removes the YAML frontmatter block (`---` ... `---`)
  before it gives the text to the agent. Before this change, the agent got the
  file unchanged.
- Every send fills in the placeholders: `$ARGUMENTS` becomes the value, and
  `${ARGUMENTS:-x}` becomes the value, or `x` when the value is empty. This
  change has no way to give a value, so every send uses an empty value:
  `$ARGUMENTS` goes away and `${ARGUMENTS:-x}` becomes `x`.
- A body that is empty after the frontmatter removal or after the fill is
  refused with the existing "is empty" toast.
- These items move here from `prompt-argument-input`. That change is edited in
  the same step: it keeps the input panel, the handoff, the `ask` verb, and the
  bubbletea dependency, and it uses this module. It must be implemented after
  this change.

## Capabilities

### New Capabilities

- `prompt-arguments`: the rules that fill an argument value into the prompt
  text. `prompt-argument-input` later adds the input panel requirements to this
  capability.

### Modified Capabilities

- `agent-prompt-send`: "Send submits the prompt" changes. The agent now gets
  the prompt body with the frontmatter removed and the placeholders filled in,
  not the file unchanged.

## Impact

- Code: new `cmd/prompt-dock/prompt.go` and `prompt_test.go`;
  `cmd/prompt-dock/send.go` (uses the module); `send_test.go` (the dash test
  uses frontmatter, which is now removed).
- Manifest: `version` goes from `0.2.0` to `0.3.0` (behavior change).
  `min_luvus_version` stays `0.14.3`.
- Dependencies: none. Standard library only.
- Docs: the README "Sending" paragraph says that the frontmatter is removed
  and the placeholders are filled.
- Other changes: `prompt-argument-input` proposal, specs, design, and tasks are
  edited to remove the moved parts and to use this module.
