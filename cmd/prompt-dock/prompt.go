// prompt.go — the prompt text rules in one place: read a prompt file and
// strip the BOM and the frontmatter block, read the `argument-hint`, and
// fill the argument placeholders (design D1). `send` calls only these two
// operations.
package main

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

// Prompt is one prompt file: the text the agent gets and the hint for the
// argument input panel.
type Prompt struct {
	Body string // file text after the BOM and the frontmatter block
	Hint string // argument-hint value; "" means none
}

// errEmptyPrompt says the body is only whitespace, before or after the fill.
var errEmptyPrompt = errors.New("prompt is empty")

// placeholder matches the two argument forms: the default form
// `${ARGUMENTS:-x}` and the plain `$ARGUMENTS` (design D3).
var placeholder = regexp.MustCompile(`\$\{ARGUMENTS:-[^}]*\}|\$ARGUMENTS`)

// loadPrompt reads the prompt file at abs, removes the BOM, splits the
// frontmatter (design D2), and refuses a body that is only whitespace. A
// read failure returns the os.ReadFile error unchanged.
func loadPrompt(abs string) (Prompt, error) {
	raw, err := os.ReadFile(abs)
	if err != nil {
		return Prompt{}, err
	}
	p := parsePrompt(strings.TrimPrefix(string(raw), "\ufeff"))
	if strings.TrimSpace(p.Body) == "" {
		return p, errEmptyPrompt
	}
	return p, nil
}

// parsePrompt splits the frontmatter block off the text and reads the
// `argument-hint` (design D2). The body is the text after the closing line,
// keeping its own leading blank line; with no closing line the body is the
// full text. A trailing `\r` on a line is ignored for the match.
func parsePrompt(text string) Prompt {
	lines := strings.Split(text, "\n")
	start := 0 // one past the closing `---`, or 0 when there is no block
	var hint string
	if len(lines) > 0 && strings.TrimSuffix(lines[0], "\r") == "---" {
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSuffix(lines[i], "\r") == "---" {
				end = i
				break
			}
		}
		if end >= 0 {
			start = end + 1
			hint = hintFromBlock(lines[1:end])
		}
	}
	body := strings.Join(lines[start:], "\n")
	return Prompt{Body: body, Hint: hint}
}

// hintFromBlock returns the `argument-hint` value from the block lines: the
// rest of the first line that starts with `argument-hint:`, trimmed, with
// one matching pair of outer `"` or `'` removed (design D2). An empty result
// means no hint.
func hintFromBlock(lines []string) string {
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		value, ok := strings.CutPrefix(line, "argument-hint:")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		return value
	}
	return ""
}

// fill replaces each `${ARGUMENTS:-x}` with the value, or with the default
// `x` when the value is empty, and each `$ARGUMENTS` with the value, in one
// pass: text the replacement function inserts is not scanned again, so a
// value that contains `$ARGUMENTS` stays literal (design D3). It refuses a
// result that is only whitespace.
func (p Prompt) fill(arg string) (string, error) {
	text := placeholder.ReplaceAllStringFunc(p.Body, func(match string) string {
		if def, ok := strings.CutPrefix(match, "${ARGUMENTS:-"); ok {
			if arg == "" {
				return strings.TrimSuffix(def, "}")
			}
			return arg
		}
		return arg
	})
	if strings.TrimSpace(text) == "" {
		return text, errEmptyPrompt
	}
	return text, nil
}
