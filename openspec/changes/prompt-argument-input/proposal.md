# Proposal

## Why

Some prompts in the library take an argument. Their frontmatter has an
`argument-hint`, and their body has `$ARGUMENTS` or `${ARGUMENTS:-default}`.
After the `prompt-module` change, the dock removes the frontmatter and fills
the placeholders, but always with an empty value, so the user cannot give a
value. luvus has no modal, so the dock needs its own short-lived panel to get
the value.

This change depends on `prompt-module`, which must be implemented first. That
change owns the frontmatter removal, the hint read, and the placeholder fill.

## What Changes

- When the user sends a prompt whose frontmatter has an `argument-hint`, the
  dock opens a temporary input panel in a split pane instead of sending at once.
  The panel shows the prompt name, the target agent, and the hint. Enter sends
  the prompt with the value filled in. Esc cancels. The panel closes in both
  cases. A prompt with no `argument-hint` is sent at once, as before.
- The module adds a `[[panes]]` entrypoint, `ask`, and a new `prompt-dock ask`
  verb.
- The module uses bubbletea for the panel. This is the first third-party Go
  dependency, and from now on bubbletea is the TUI library for module input and
  display elements.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `prompt-arguments` (created by `prompt-module`): adds the input panel that
  asks for a prompt's argument.

## Impact

- Code: `cmd/prompt-dock/send.go` (hint check, handoff, `deliver`), a new
  `cmd/prompt-dock/ask.go` (the panel), `main.go` (new verb). The text rules
  stay in `cmd/prompt-dock/prompt.go` (`loadPrompt`, `fill`) from
  `prompt-module`.
- Manifest: a new `[[panes]]` entry `ask`; version goes to `0.4.0`.
  `min_luvus_version` stays `0.14.3`.
- Dependencies: `charm.land/bubbletea/v2` and `charm.land/bubbles/v2`. The
  `go` directive in `go.mod` goes from `1.22` to the version that bubbles/v2
  needs (1.26 or later).
- Docs: the README no longer says "Go standard library only", and it documents
  the input panel.
