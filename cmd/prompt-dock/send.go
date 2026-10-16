// send.go — give a stored prompt to a live agent and submit it (design D4).
// Ported from send.ts. Row menu value: `<pane>|<terminal_id>|<relpath>`
// (design D3). The file is read at action time, so edits saved after the
// last dock repaint go out.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// summarize is one line, bounded, for a toast: stderr if luvus wrote any,
// else stdout. Whitespace runs collapse to single spaces, and the cut to 160
// is by rune, so it never splits a UTF-8 sequence (design D14).
func summarize(res LuvusResult) string {
	line := strings.TrimSpace(res.Stderr)
	if line == "" {
		line = strings.TrimSpace(res.Stdout)
	}
	line = strings.Join(strings.Fields(line), " ")
	runes := []rune(line)
	if len(runes) > 160 {
		runes = runes[:160]
	}
	return string(runes)
}

// promptName is the prompt's display name: the file name without `.md`.
func promptName(rel string) string {
	name := filepath.Base(rel)
	if len(name) >= 3 && strings.EqualFold(name[len(name)-3:], ".md") {
		return name[:len(name)-3]
	}
	return name
}

// labelForTarget is the agent label for toasts: the agent now at `pane` with
// that exact terminal id (design D4 — the id is the guard; an empty id
// matches an agent with none), or `pane <id>` when luvus no longer reports
// that pair.
func labelForTarget(pane string, terminalId string) string {
	for _, a := range listAgents() {
		id := ""
		if a.TerminalID != nil {
			id = *a.TerminalID
		}
		if a.Pane == pane && id == terminalId {
			return targetLabel(a)
		}
	}
	return fmt.Sprintf("pane %s", pane)
}

// send gives the prompt file named by the row value to the agent pane the
// value names, and submits it.
func send(value string) error {
	pane, terminalId, rel := splitSendValue(value)
	if pane == "" {
		return fmt.Errorf("send: malformed value (expected <pane>|<terminal_id>|<relpath>): %s", value)
	}

	label := labelForTarget(pane, terminalId)
	abs := resolveUnderRoot(settingRoot(), rel)
	if abs == "" {
		toast(fmt.Sprintf("Send to %s failed: that file is outside your prompt library root.", label))
		return fmt.Errorf("send: relpath escapes root: %s", rel)
	}

	if terminalId == "" {
		// Without a terminal id the send can't be guarded against a reused
		// pane id (design D4).
		toast(fmt.Sprintf("Send to %s failed: no terminal id", label))
		return fmt.Errorf("send: no terminal id for pane %s", pane)
	}

	// Read at action time, so edits saved after the last repaint are included
	// (spec: send submits the prompt).
	raw, err := os.ReadFile(abs)
	if err != nil {
		toast(fmt.Sprintf("Send to %s failed: couldn't read %q.", label, promptName(rel)))
		return fmt.Errorf("send: read failed for %s: %v", abs, err)
	}
	// A UTF-8 byte-order mark is not part of the prompt; drop it.
	text := strings.TrimPrefix(string(raw), "\ufeff")
	if strings.TrimSpace(text) == "" {
		toast(fmt.Sprintf("Send to %s failed: %q is empty.", label, promptName(rel)))
		return fmt.Errorf("send: prompt file is empty: %s", abs)
	}

	// The text is ONE argv element (no shell, design D4), so its newlines
	// survive; `agent prompt` submits it in one atomic request. `--` ends
	// option parsing: a prompt that starts with `-` (YAML frontmatter `---`,
	// a leading bullet) is otherwise rejected as an unknown option. The text
	// must stay last, because everything after `--` is positional.
	res := luvus([]string{"agent", "prompt", pane, "--terminal-id", terminalId, "--", text}, "")
	if res.Status != 0 {
		// `agent_not_ready` (permission prompt up), a stale target, or any
		// other refusal: nothing was queued — say so, with the agent label.
		toast(fmt.Sprintf("Send to %s failed: %s", label, summarize(res)))
		return fmt.Errorf("send: agent prompt failed (status %d): %s", res.Status, summarize(res))
	}
	toast(fmt.Sprintf("Sent %q to %s.", promptName(rel), label))
	return nil
}
