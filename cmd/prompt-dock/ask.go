// ask.go — the input panel (design D6): one bubbletea program with a focused
// text field for the argument value. The ask verb reads and deletes the
// handoff file, runs the program, and on submit delivers the filled prompt,
// then gives the focus back to the pane that had it (design D2, D7).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// askModel is the input panel model (design D6): the prompt name, the target
// label, and the text field the value is typed into.
type askModel struct {
	input  textinput.Model
	name   string // prompt display name
	target string // target agent label
	// submitted is true when the user quit with Enter, false on Esc and
	// Ctrl+C.
	submitted bool
}

// newAskModel builds the panel model for one request: the hint is the
// placeholder text, and the field starts focused. Focus is set here, not in
// Init: Init runs on a copy of the model, so a focus set there is lost.
func newAskModel(name string, target string, hint string) askModel {
	input := textinput.New()
	input.Placeholder = hint
	input.Focus()
	return askModel{input: input, name: name, target: target}
}

// Init returns no command: the field is focused already, and the panel quits
// on its own keys, so nothing else is pending.
func (m askModel) Init() tea.Cmd {
	return nil
}

// Update quits on Enter (submit) and on Esc and Ctrl+C (cancel); every other
// key goes to the text field (design D6).
func (m askModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "enter":
			m.submitted = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View is three short lines in a small pane: the prompt name, the target
// label, and the field (design D6). No alt screen.
func (m askModel) View() tea.View {
	return tea.NewView(m.name + "\n" + m.target + "\n" + m.input.View())
}

// runAskPanel runs the panel program on the given streams and returns the
// final model (design D6). runAsk passes the process streams; the test passes
// scripted input and a discard writer (design D8).
func runAskPanel(model askModel, in io.Reader, out io.Writer) (askModel, error) {
	final, err := tea.NewProgram(model, tea.WithInput(in), tea.WithOutput(out)).Run()
	if err != nil {
		return askModel{}, err
	}
	result, ok := final.(askModel)
	if !ok {
		return askModel{}, fmt.Errorf("ask: unexpected final model type %T", final)
	}
	return result, nil
}

// runAsk is the ask verb (task 3.2).
func runAsk() error {
	return ask(os.Stdin, os.Stdout)
}

// ask does the ask verb's work on the given streams.
func ask(in io.Reader, out io.Writer) error {
	dir, err := moduleStateDir()
	if err != nil {
		return err
	}

	// Read the handoff file and delete it at once, before the field shows
	// (design D2). A missing or unreadable file means a left-over panel from
	// a restart: exit 0 with no luvus call, so luvus closes the pane.
	raw, err := os.ReadFile(askFile(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ask: read handoff: %w", err)
	}
	if err := os.Remove(askFile(dir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ask: remove handoff: %w", err)
	}
	var req askRequest
	if json.Unmarshal(raw, &req) != nil || req.Value == "" {
		// Not valid JSON, or no request in it: nothing to show or send.
		return nil
	}

	pane, terminalId, rel := splitSendValue(req.Value)

	// The view shows the hint as the placeholder (design D6), so the panel
	// reads the prompt file once more; a read failure leaves the placeholder
	// empty and the panel still works.
	hint := ""
	if abs := resolveUnderRoot(settingRoot(), rel); abs != "" {
		if p, err := loadPrompt(abs); err == nil {
			hint = p.Hint
		}
	}
	model := newAskModel(promptName(rel), labelForTarget(pane, terminalId), hint)
	final, err := runAskPanel(model, in, out)
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	// Deliver first, focus second (design D7). On cancel no text reaches any
	// agent (spec: cancel sends nothing).
	var deliverErr error
	if final.submitted {
		deliverErr = deliver(pane, terminalId, rel, final.input.Value())
	}
	if req.ReturnPane != "" {
		// Focus goes back before the exit, so luvus's close_pane does not
		// move the focus to the first leaf (design D7). The pane can be gone;
		// a failure is ignored.
		luvus([]string{"pane", "focus", req.ReturnPane}, "")
	}
	return deliverErr
}
