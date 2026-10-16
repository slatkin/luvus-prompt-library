// select.go — mark the prompt in LUVUS_MODULE_ROW_VALUE as the selected one
// (design D9), persist it, then repaint. The dock has no selection of its own,
// so a left-click on a prompt row runs this instead of opening a file.
package main

import (
	"fmt"
	"os"
)

// runSelect selects the prompt named by LUVUS_MODULE_ROW_VALUE, then repaints
// the dock.
func runSelect() error {
	value := os.Getenv("LUVUS_MODULE_ROW_VALUE")
	if value == "" {
		return fmt.Errorf("select: no row value (LUVUS_MODULE_ROW_VALUE unset)")
	}
	if resolveUnderRoot(settingRoot(), value) == "" {
		return fmt.Errorf("select: prompt is outside the prompt library root: %s", value)
	}

	dir, err := moduleStateDir()
	if err != nil {
		return err
	}
	if readSelected(dir) == value {
		return nil // already selected: no repaint needed
	}

	if err := writeSelected(dir, value); err != nil {
		return err
	}
	return refreshDock()
}
