// toggle.go — expand or collapse the folder in LUVUS_MODULE_ROW_VALUE
// (design D5), persist the set in the module state dir, then repaint.
package main

import (
	"fmt"
	"os"
)

// runToggle expands or collapses the folder named by LUVUS_MODULE_ROW_VALUE,
// then repaints the dock.
func runToggle() error {
	value := os.Getenv("LUVUS_MODULE_ROW_VALUE")
	if value == "" {
		return fmt.Errorf("toggle: no folder value (LUVUS_MODULE_ROW_VALUE unset)")
	}
	if resolveUnderRoot(settingRoot(), value) == "" {
		return fmt.Errorf("toggle: folder is outside the prompt library root: %s", value)
	}

	dir, err := moduleStateDir()
	if err != nil {
		return err
	}
	expanded := readExpanded(dir)
	if expanded[value] {
		delete(expanded, value)
	} else {
		expanded[value] = true
	}
	if err := writeExpanded(dir, expanded); err != nil {
		return err
	}

	return refreshDock()
}
