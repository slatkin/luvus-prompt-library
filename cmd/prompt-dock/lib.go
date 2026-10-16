// lib.go — shared helpers for the Prompt Dock module. Ported from lib.ts.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// LuvusResult is the outcome of one luvus CLI call.
type LuvusResult struct {
	Status int
	Stdout string
	Stderr string
}

// luvusBinPath returns the luvus binary to call back into: LUVUS_BIN_PATH,
// else `luvus` on PATH.
func luvusBinPath() string {
	if bin := os.Getenv("LUVUS_BIN_PATH"); bin != "" {
		return bin
	}
	return "luvus"
}

// luvus runs a luvus CLI command as one argv array, no shell (design D4).
// It never returns an error to the caller; the caller inspects Status. A
// spawn failure (no binary, argv too long) has no stderr, so its Go error
// text stands in for the TS `code`, and Status is -1. A non-zero exit keeps
// the child's captured stderr, as in TS.
func luvus(args []string, input string) LuvusResult {
	cmd := exec.Command(luvusBinPath(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	err := cmd.Run()
	if err == nil {
		return LuvusResult{Status: 0, Stdout: stdout.String(), Stderr: stderr.String()}
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return LuvusResult{Status: exit.ExitCode(), Stdout: stdout.String(), Stderr: stderr.String()}
	}
	// A spawn failure has no stderr; the error text says why.
	return LuvusResult{Status: -1, Stdout: stdout.String(), Stderr: err.Error()}
}

// homeDir returns the user's home folder, or "" when it cannot be found.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// expandHome expands a leading `~` to the user's home folder (spec:
// configurable root).
func expandHome(p string) string {
	home := homeDir()
	if p == "~" {
		return home
	}
	// Windows users type `~\`, so accept both separators.
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(home, p[2:])
	}
	return p
}

// resolveRoot resolves a `root` value: trim it, treat an empty value as the
// `~/.prompts` default, and expand `~`.
func resolveRoot(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "~/.prompts"
	}
	expanded := expandHome(trimmed)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return filepath.Clean(expanded)
	}
	return abs
}

// settingRoot returns the prompt library root: the `root` module setting,
// with `~` expanded.
func settingRoot() string {
	return resolveRoot(os.Getenv("LUVUS_SETTING_ROOT"))
}

// resolveUnderRoot resolves a relpath against the root, or returns "" when it
// escapes the root (design D3). Absolute relpaths and any `..` that climbs
// past the root are rejected, so a stale row value can never point outside
// the current root. This check is lexical: a symlink under the root is
// followed, so its target may lie outside.
func resolveUnderRoot(root string, rel string) string {
	base, err := filepath.Abs(root)
	if err != nil {
		base = filepath.Clean(root)
	}
	// Like path.resolve: an absolute rel replaces the base outright.
	abs := rel
	if !filepath.IsAbs(rel) {
		abs = filepath.Join(base, rel)
	}
	abs = filepath.Clean(abs)
	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return ""
	}
	return abs
}

// moduleStateDir returns the module's persistent state folder (design D5).
// luvus always sets it; if it is missing, fail rather than write the expanded
// set into the working folder.
func moduleStateDir() (string, error) {
	dir := os.Getenv("LUVUS_MODULE_STATE_DIR")
	if dir == "" {
		return "", fmt.Errorf("module: LUVUS_MODULE_STATE_DIR is unset")
	}
	return dir, nil
}

// readExpanded reads the expanded-folder set: a JSON array of folder relpaths
// (design D5). A missing or corrupt file means "all collapsed".
func readExpanded(dir string) map[string]bool {
	raw, err := os.ReadFile(filepath.Join(dir, "expanded.json"))
	if err != nil {
		return map[string]bool{}
	}
	var parsed []any
	if json.Unmarshal(raw, &parsed) != nil {
		return map[string]bool{}
	}
	expanded := map[string]bool{}
	for _, v := range parsed {
		if s, ok := v.(string); ok {
			expanded[s] = true
		}
	}
	return expanded
}

// writeExpanded writes the expanded-folder set (design D5). An empty set is
// written as `[]`, never `null`.
func writeExpanded(dir string, expanded map[string]bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Folder order is sorted: Go set iteration is random, and no reader
	// depends on the order.
	rels := make([]string, 0, len(expanded))
	for rel := range expanded {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	// Write a temp file, then rename it over the old one, so a refresh never
	// reads half a file.
	// ponytail: two clicks in quick succession can still race; add a lock file if that bites.
	file := filepath.Join(dir, "expanded.json")
	encoded, err := json.MarshalIndent(rels, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(file, encoded)
}

// readSelected reads the selected prompt relpath (design D9), or "" when
// nothing is selected. The dock has no selection of its own, so the module
// keeps it. A missing or corrupt file means "nothing selected".
func readSelected(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "selected.json"))
	if err != nil {
		return ""
	}
	var parsed any
	if json.Unmarshal(raw, &parsed) != nil {
		return ""
	}
	if s, ok := parsed.(string); ok {
		return s
	}
	return ""
}

// writeSelected writes the selected prompt relpath (design D9).
func writeSelected(dir string, rel string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// Temp file + rename, for the same reason as the expanded set.
	file := filepath.Join(dir, "selected.json")
	encoded, err := json.Marshal(rel)
	if err != nil {
		return err
	}
	return writeAtomic(file, encoded)
}

// writeAtomic writes a state file through a `.tmp` sibling, then renames it
// over the old file. The bytes are already indented JSON plus a newline, as
// the TS code wrote them.
func writeAtomic(file string, encoded []byte) error {
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, append(encoded, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// AgentInfo is one agent as `luvus agent list` reports it: the fields the
// module needs. A nil pointer stands for the TS null.
type AgentInfo struct {
	Pane  string
	Agent string
	// TerminalID is the terminal id as luvus reports it, or nil when the pane
	// has no terminal runtime.
	TerminalID *string
	// Name is the live alias, or nil when the agent has none.
	Name *string
	// Workspace is the id of the workspace that holds the pane, as `agent
	// list` reports it.
	Workspace string
}

// agentEnvelope mirrors the RPC envelope of `luvus agent list`.
type agentEnvelope struct {
	Result struct {
		Agents []json.RawMessage `json:"agents"`
	} `json:"result"`
}

// agentRaw mirrors one agent entry, with every field optional so a wrong type
// can be told from a missing one.
type agentRaw struct {
	Pane       *string `json:"pane"`
	Agent      *string `json:"agent"`
	TerminalID *string `json:"terminal_id"`
	Name       *string `json:"name"`
	Workspace  *string `json:"workspace"`
}

// parseAgents pulls the agent array out of `luvus agent list` output. The CLI
// prints the RPC envelope (`{"id":"1","result":{"agents":[…]}}`). Entries
// without a pane or agent kind are dropped; a null `terminal_id` is kept as
// nil and the caller decides what to do with it.
func parseAgents(stdout string) []AgentInfo {
	var parsed agentEnvelope
	// Unparseable reply: no targets, never a crash.
	if json.Unmarshal([]byte(stdout), &parsed) != nil {
		return []AgentInfo{}
	}
	agents := []AgentInfo{}
	for _, raw := range parsed.Result.Agents {
		var a agentRaw
		if json.Unmarshal(raw, &a) != nil {
			continue
		}
		// A missing pane or agent kind drops the entry, as in TS.
		if a.Pane == nil || a.Agent == nil {
			continue
		}
		agent := AgentInfo{Pane: *a.Pane, Agent: *a.Agent}
		agent.TerminalID = a.TerminalID
		agent.Name = a.Name
		if a.Workspace != nil {
			agent.Workspace = *a.Workspace
		}
		agents = append(agents, agent)
	}
	return agents
}

// listAgents lists the live agents across all workspaces and tabs (`luvus
// agent list`, design D2). A failed or unparseable reply means no targets,
// never a crash.
func listAgents() []AgentInfo {
	res := luvus([]string{"agent", "list"}, "")
	if res.Status != 0 {
		return []AgentInfo{}
	}
	return parseAgents(res.Stdout)
}

// currentWorkspaceId returns the focused workspace id, as luvus injects it
// (`LUVUS_WORKSPACE_ID`, design D11).
func currentWorkspaceId() string {
	return os.Getenv("LUVUS_WORKSPACE_ID")
}

// agentsInWorkspace returns the agents that belong to one workspace (design
// D11). An empty id keeps every agent: the id comes from the environment, and
// hiding every send target is a worse failure than showing too many.
func agentsInWorkspace(agents []AgentInfo, workspaceId string) []AgentInfo {
	if workspaceId == "" {
		return agents
	}
	kept := []AgentInfo{}
	for _, a := range agents {
		if a.Workspace == workspaceId {
			kept = append(kept, a)
		}
	}
	return kept
}

// encodeSendValue builds the "Send to" row value: `<pane>|<terminal_id>|<rel>`
// (design D3). The terminal id is what makes luvus reject a reused pane id
// (design D4).
func encodeSendValue(pane string, terminalId string, rel string) string {
	return pane + "|" + terminalId + "|" + rel
}

// splitSendValue splits a send value into `[pane, terminalId, relpath]`, on
// the first two `|` only (design D3), so a `|` in a file name still works.
func splitSendValue(value string) (string, string, string) {
	first := strings.Index(value, "|")
	if first == -1 {
		return "", "", value
	}
	second := strings.Index(value[first+1:], "|")
	if second == -1 {
		return value[:first], "", value[first+1:]
	}
	second += first + 1
	return value[:first], value[first+1 : second], value[second+1:]
}

// targetLabel is the target label for menus and toasts (design D8): `<name>
// (<agent>) · pane <pane>`, or `<agent> · pane <pane>` when the agent has no
// alias. The menu lists every live agent, so status and workspace are left
// out: the label stays short enough to read at a glance.
func targetLabel(a AgentInfo) string {
	base := a.Agent
	if a.Name != nil && *a.Name != "" {
		base = fmt.Sprintf("%s (%s)", *a.Name, a.Agent)
	}
	return fmt.Sprintf("%s · pane %s", base, a.Pane)
}

// TreeRow is one entry of the scanned tree, in dock-row order (spec: tree
// rows).
type TreeRow struct {
	// Rel is the path relative to the root, `/`-separated on every platform.
	Rel string
	// Name is the display name: the folder name, or the file name without
	// `.md`.
	Name string
	// Depth is 0 for root-level entries, 1 for their children, and so on.
	Depth int
	// Kind is "folder" or "file".
	Kind string
	// IsLast is true when no later row shares this row's parent. A collapsed
	// folder's children never show, so they do not count as rows after it
	// (spec: tree rows).
	IsLast bool
}

// scanTree walks the prompt library depth-first (spec: tree rows). Folders
// come before files at each level, each group sorted by name; only `.md`
// files show; names starting with `.` are skipped; the children of a folder
// that is not in `expanded` are never read (design D5: collapsed costs
// nothing). Relpaths use `/` separators on every platform so the expanded-set
// keys in `expanded.json` are stable across platforms. Symlinks count as what
// they point at. A folder that links back up the tree is listed but not
// walked again.
func scanTree(root string, expanded map[string]bool) []TreeRow {
	rows := []TreeRow{}
	var walk func(dir string, rel string, depth int, ancestors map[string]bool)
	walk = func(dir string, rel string, depth int, ancestors map[string]bool) {
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			return // unreadable folder: render its other siblings, nothing under it
		}
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return
		}
		if ancestors[real] {
			return
		}
		chain := make(map[string]bool, len(ancestors)+1)
		for a := range ancestors {
			chain[a] = true
		}
		chain[real] = true
		var folders, files []string
		for _, entry := range dirEntries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			// A symlink is judged by its target; a broken link matches nothing.
			isDir, isFile := false, false
			if entry.Type()&os.ModeSymlink != 0 {
				info, statErr := os.Stat(filepath.Join(dir, entry.Name()))
				if statErr != nil {
					continue
				}
				isDir, isFile = info.IsDir(), info.Mode().IsRegular()
			} else {
				isDir, isFile = entry.IsDir(), entry.Type().IsRegular()
			}
			if isDir {
				folders = append(folders, entry.Name())
			} else if isFile && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				files = append(files, entry.Name())
			}
		}
		// Default sort: ascending by code point (spec: tree rows).
		sort.Strings(folders)
		sort.Strings(files)
		for i, name := range folders {
			childRel := name
			if rel != "" {
				childRel = rel + "/" + name
			}
			// A folder is last only when no file follows it at this level: the
			// children of a collapsed folder are not rows of their own.
			rows = append(rows, TreeRow{
				Rel:    childRel,
				Name:   name,
				Depth:  depth,
				Kind:   "folder",
				IsLast: i == len(folders)-1 && len(files) == 0,
			})
			if expanded[childRel] {
				walk(filepath.Join(dir, name), childRel, depth+1, chain)
			}
		}
		for i, name := range files {
			rows = append(rows, TreeRow{
				Rel:    joinRel(rel, name),
				Name:   name[:len(name)-3],
				Depth:  depth,
				Kind:   "file",
				IsLast: i == len(files)-1,
			})
		}
	}
	walk(root, "", 0, map[string]bool{})
	return rows
}

// joinRel joins a folder relpath and a name with `/`.
func joinRel(rel string, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

// toast shows a one-line message in the UI (design Context).
func toast(text string) {
	luvus([]string{"ui", "toast", text}, "")
}
