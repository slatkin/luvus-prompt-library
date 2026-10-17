// send.go — give a stored prompt to a live agent and submit it (design D4).
// Ported from send.ts. Row menu value: `<pane>|<terminal_id>|<relpath>`
// (design D3). The file is read at action time, so edits saved after the
// last dock repaint go out.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// askRequest is one pending input-panel request, written to `$LUVUS_MODULE_STATE_DIR/ask.json` (design D2). `ask` reads and deletes the file before it shows the field.
type askRequest struct {
	// Value is the row value `send` got: `<pane>|<terminal_id>|<relpath>`.
	Value string `json:"value"`
	// ReturnPane is send's own pane id (`LUVUS_PANE_ID`), the pane that had
	// the focus when the user clicked the dock. Empty means `ask` skips the
	// focus call.
	ReturnPane string `json:"return_pane"`
}

// askFile is the handoff file path under the module state dir (design D2).
func askFile(stateDir string) string {
	return filepath.Join(stateDir, "ask.json")
}

// summarize is one line, bounded, for a toast: stderr if luvus wrote any,
// else stdout. Whitespace runs collapse to single spaces, and the cut to 160
// is by rune, so it never splits a UTF-8 sequence (design D14).
func summarize(res LuvusResult) string {
	line := strings.Join(strings.Fields(res.detail()), " ")
	runes := []rune(line)
	if len(runes) > 160 {
		runes = runes[:160]
	}
	return string(runes)
}

// promptName is the prompt's display name: the file name without `.md`.
func promptName(rel string) string {
	name := filepath.Base(rel)
	if isPromptFile(name) {
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
		if a.Pane == pane && a.TerminalID == terminalId {
			return targetLabel(a)
		}
	}
	return fmt.Sprintf("pane %s", pane)
}

// deliver gives the prompt file named by the row value parts to the agent
// pane, with arg filled into the placeholders, and submits it (design D3).
// Both the direct send and the ask panel call it, so the stale-target guard,
// the toasts, and the read-at-send-time rule stay in one place.
func deliver(pane string, terminalId string, rel string, arg string) error {
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
	// (spec: send submits the prompt). The prompt module strips the BOM and the
	// frontmatter block (design D1); fill removes the argument placeholders —
	// with an empty value `${ARGUMENTS:-x}` becomes `x` and `$ARGUMENTS` is
	// removed (spec: prompt-arguments).
	emptyFail := func() error {
		toast(fmt.Sprintf("Send to %s failed: %q is empty.", label, promptName(rel)))
		return fmt.Errorf("send: prompt file is empty: %s", abs)
	}
	p, err := loadPrompt(abs)
	if errors.Is(err, errEmptyPrompt) {
		return emptyFail()
	}
	if err != nil {
		toast(fmt.Sprintf("Send to %s failed: couldn't read %q.", label, promptName(rel)))
		return fmt.Errorf("send: read failed for %s: %w", abs, err)
	}
	text, err := p.fill(arg)
	if err != nil {
		return emptyFail()
	}

	// The text is ONE argv element (no shell, design D4), so its newlines
	// survive. `agent prompt` submits it in one atomic request. `--` ends
	// option parsing, so dash-leading text (a leading bullet or YAML `---`) is
	// not rejected as an unknown option. The text must stay last, because
	// everything after `--` is positional.
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

// send gives the prompt file named by the row value to the agent pane the
// value names. A prompt whose frontmatter has an argument hint opens the
// input panel instead, with a handoff file carrying the request (design D2,
// D3); the panel's own deliver call sends the text.
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

	// Read at action time, so edits saved after the last repaint are included
	// (spec: send submits the prompt). The empty check comes before the hint
	// check, because send does not know the value yet (design D3).
	p, err := loadPrompt(abs)
	if errors.Is(err, errEmptyPrompt) {
		toast(fmt.Sprintf("Send to %s failed: %q is empty.", label, promptName(rel)))
		return fmt.Errorf("send: prompt file is empty: %s", abs)
	}
	if err != nil {
		toast(fmt.Sprintf("Send to %s failed: couldn't read %q.", label, promptName(rel)))
		return fmt.Errorf("send: read failed for %s: %w", abs, err)
	}
	if p.Hint == "" {
		// No hint: no value can be typed, so send at once with no value. The
		// fill removes `$ARGUMENTS` (spec: prompt-arguments, no hint sends at
		// once).
		return deliver(pane, terminalId, rel, "")
	}

	// The hint opens the input panel (design D2): write the handoff file
	// first, so the panel reads a complete request, then open the pane. No
	// text reaches the agent yet.
	stateDir, err := moduleStateDir()
	if err != nil {
		return err
	}
	handoff, err := json.Marshal(askRequest{Value: value, ReturnPane: os.Getenv("LUVUS_PANE_ID")})
	if err != nil {
		return fmt.Errorf("send: encode ask.json: %w", err)
	}
	if err := writeAtomic(askFile(stateDir), handoff); err != nil {
		return fmt.Errorf("send: write ask.json: %w", err)
	}
	res := luvus([]string{"module", "pane", "open", "slatkin.prompt-dock", "ask", "--placement", "split"}, "")
	if res.Status != 0 {
		// No panel can show the request, so no later panel may read it either
		// (design D2).
		if err := os.Remove(askFile(stateDir)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("send: remove ask.json after a failed pane open: %w", err)
		}
		toast(fmt.Sprintf("Couldn't open the input panel: %s", summarize(res)))
		return fmt.Errorf("send: pane open failed (status %d): %s", res.Status, summarize(res))
	}
	return nil
}
