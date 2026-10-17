# Design

## Context

See proposal.md for the reason. Current state:

- `send(value)` in `cmd/prompt-dock/send.go` does all steps inline: split the
  row value, get the target label, resolve the relpath under the root, refuse
  an empty terminal id, read the file, remove a UTF-8 BOM, refuse a body that
  is only whitespace, call `luvus agent prompt <pane> --terminal-id <id> --
  <text>`, and toast.
- The `send` tests use the fake-luvus call log and skip on Windows
  (`skipOnWindows`). The dash test sends a file that starts with frontmatter
  and expects `-- ---` in the log. After this change the frontmatter is gone,
  so that test must change.
- `prompt-argument-input` (not started) planned the frontmatter parser (its
  D4) and the fill (its D5) inside `send.go`. This design moves both here,
  with the same rules.

## Goals / Non-Goals

**Goals:**

- One module that holds every prompt text rule: BOM, frontmatter, hint, empty
  check, and placeholder fill. `send` and the later input panel call only its
  two operations.
- Table tests for the module that need no fake luvus and do not skip on
  Windows.

**Non-Goals:**

- Agent-specific placeholder rules (pi `$1`, `${@:N}`; Claude `$ARGUMENTS[N]`,
  `$N`, `$name`). One rule set serves every agent.
- Running `` !`cmd` `` shell injections. That text goes out unchanged.
- A full YAML parser. The module reads only `argument-hint`.
- A way for the user to give a value. `prompt-argument-input` adds that.
- Changes to the row value codec or the luvus CLI seam (issue #7, findings B
  and C).

## Decisions

### D1. The module interface: `loadPrompt` and `fill`

```go
// prompt.go
type Prompt struct {
	Body string // file text after the BOM and the frontmatter block
	Hint string // argument-hint value; "" means none
}

var errEmptyPrompt = errors.New("prompt is empty")

func loadPrompt(abs string) (Prompt, error)
func (p Prompt) fill(arg string) (string, error)
```

- `loadPrompt` reads the file, removes the BOM, splits the frontmatter (D2),
  and returns `errEmptyPrompt` when the body is only whitespace. A read
  failure returns the wrapped `os.ReadFile` error.
- `fill` replaces the placeholders (D3) and returns `errEmptyPrompt` when the
  result is only whitespace.

`send` becomes:

```
split value -> label -> resolve under root -> terminal id guard
p, err := loadPrompt(abs)   errEmptyPrompt -> "is empty" toast, else "couldn't read" toast
text, err := p.fill("")     errEmptyPrompt -> "is empty" toast
agent prompt ... -- text -> toast
```

The toast texts do not change. `send` tells the two load failures apart with
`errors.Is(err, errEmptyPrompt)`.

Both operations check for an empty body. The load check lets the input panel
(next change) refuse a frontmatter-only file before it opens. The fill check
refuses a body that the fill makes empty (`$ARGUMENTS` only, empty value).

`Hint` has no reader in this change. The input panel in
`prompt-argument-input` reads it. It is here because the frontmatter parse
finds it, and splitting the parse over two changes gives two parsers.

Alternatives:
- A pure `parsePrompt(text)` next to `loadPrompt(abs)`: rejected. It adds a
  second entry point only for tests. The table tests write each case to a
  `t.TempDir()` file, which works on every OS.
- `fill` returns only a string and `send` checks for empty: rejected. Every
  caller would then repeat the check, which is the leak that finding A
  removes.

### D2. Frontmatter: a small line parser, no YAML library

From `prompt-argument-input` D4, unchanged:

- Split the text into lines. A trailing `\r` on a line is ignored for the
  match.
- A block exists only when line 1 is `---` and a later line is `---`. The body
  is the text after the closing line. With no closing line, there is no block
  and the body is the full text.
- The hint is the first block line that starts with `argument-hint:`. The rest
  of that line is trimmed, and one matching pair of outer `"` or `'` is
  removed. An empty result means no hint.

The body keeps its own leading blank line, if it has one. `agent prompt` gets
it as it is. Nothing trims the body except the empty checks.

Alternative: a YAML library. Rejected: the module reads one scalar key, and a
library is a new dependency for that.

### D3. Fill: one regex, one pass

From `prompt-argument-input` D5, unchanged:

```go
var placeholder = regexp.MustCompile(`\$\{ARGUMENTS:-([^}]*)\}|\$ARGUMENTS`)
```

`ReplaceAllStringFunc` replaces each match once: with the value, or with the
`:-` default when the value is empty. Text that the function inserts is not
scanned again, so a value that contains `$ARGUMENTS` stays literal. This is the
same method pi uses (`substituteArgs`), limited to the two forms the library
uses.

### D4. Testing

- `prompt_test.go`: table tests for `loadPrompt` (BOM, no block, closed block,
  unclosed block, CRLF lines, hint with `"`/`'`/no quotes, empty hint, hint
  missing, block with no body, whitespace-only file, missing file) and for
  `fill` (every scenario of the "Placeholder fill" requirement, plus a result
  that is empty). No fake luvus, no Windows skip.
- `send_test.go` keeps the fake-luvus tests for what `send` still owns: the
  guards, the toasts, and the `--` argv shape. The dash test changes to a body
  that starts with `- `. New send tests: frontmatter removed, frontmatter-only
  file refused, `$ARGUMENTS`-only body refused.

## Risks / Trade-offs

- [The frontmatter is now removed for every prompt] → a user who wanted the
  agent to see it loses that. Marked **BREAKING** in the proposal. The README
  says so.
- [`$ARGUMENTS` disappears from every send until the panel ships] → a prompt
  that used it as a visible marker for the user to edit now loses it. The
  library prompts use it as a placeholder only. The README says so.
- [Hand-written frontmatter parse] → a multi-line YAML value, or a hint with
  escaped quotes, is not read correctly. The library uses one-line hints only.

## Migration Plan

Users run `go install ./cmd/prompt-dock` again after the pull, as the README
says for every update. Rollback: check out the previous version and run
`go install` again.
