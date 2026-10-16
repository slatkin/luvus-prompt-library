// refresh.ts — scan the prompt library and push the dock rows (design D2).
// Runs on startup, on every event hook, for the `refresh` action, and from
// toggle.ts after it flips the expanded set. Sends one JSON row array to
// `ui dock push` on stdin.
import { statSync } from "node:fs";

import {
  agentsInWorkspace,
  currentWorkspaceId,
  encodeSendValue,
  findOnPath,
  isMain,
  listAgents,
  luvus,
  moduleStateDir,
  readExpanded,
  readSelected,
  scanTree,
  settingRoot,
  targetLabel,
  type AgentInfo,
  type TreeRow,
} from "./lib.ts";

const DOCK_ID = "prompts";
const DOCK_TITLE = "PROMPTS";
// One level of nesting. A folder row spends two of these columns on its `▸ `
// glyph, so a three-space step puts every child one space right of its parent's
// name; a two-space step puts the child flush under that name instead.
const INDENT = "   ";
// `ui dock push` draws a row's dot as "<dot> " before its text, so a dotted row
// starts two columns further right. A selected row drops those columns from its
// indent instead, and its name keeps the column its siblings use.
const DOT_PREFIX_WIDTH = 2;
// The dot on the selected row. A module dock row can set a status dot and a tone,
// but not bold or a row background, so the accent tone plus a filled dot is the
// closest a module gets to the focused row of the built-in docks.
const SELECTED_DOT = "done";

/** One row as `ui dock push` accepts it (JSON on stdin). */
interface DockRow {
  text: string;
  tone?: string;
  /** A status dot: `idle` / `working` / `blocked` / `done`. */
  dot?: string;
  action?: string;
  value?: string;
  menu?: DockMenuEntry[];
}

/** One per-row right-click menu entry as `ui dock push` accepts it. */
interface DockMenuEntry {
  title: string;
  action: string;
  value: string;
}

interface Editor {
  bin: string;
  /** The editor's name, for the menu entry label. */
  label: string;
}

/** The external editors found on PATH; a refresh checks this once. */
function availableEditors(): Editor[] {
  return [
    { bin: "zed", label: "zed" },
    { bin: "code", label: "VS Code" },
  ].filter((e) => findOnPath(e.bin));
}

/**
 * The "Edit" entries for a prompt row: one per external editor on PATH
 * (design D7, spec: external editors). "Open" is the inline editor and is
 * built in `toDockRow`; these entries are the detached applications.
 */
export function editMenuEntries(
  rel: string,
  editors: readonly Editor[] = availableEditors(),
): DockMenuEntry[] {
  return editors.map((e) => ({
    title: `Edit in ${e.label}`,
    action: "edit-external",
    value: `${e.bin}|${rel}`,
  }));
}

/**
 * One "Send to" entry per live agent for a prompt row (spec: agent send
 * targets). An agent without a terminal id still gets an entry: its value
 * carries an empty id, and send.ts refuses it with a toast (design D4).
 */
export function sendMenuEntries(
  rel: string,
  agents: readonly AgentInfo[] = listAgents(),
): DockMenuEntry[] {
  return agents.map((a) => ({
    title: `Send to ${targetLabel(a)}`,
    action: "send",
    value: encodeSendValue(a.pane, a.terminal_id ?? "", rel),
  }));
}

function isFolder(p: string): boolean {
  try {
    return statSync(p).isDirectory();
  } catch {
    return false; // missing, or not readable: show the missing-root row
  }
}

/** Build the dock rows for `root`: the tree, or the missing-root row. */
function buildRows(
  root: string,
  expanded: ReadonlySet<string>,
  selected: string | null,
): DockRow[] {
  if (!isFolder(root)) {
    // Spec: missing root — one muted row that names the folder, no other rows.
    return [{ text: `root not found: ${root}`, tone: "muted" }];
  }
  // Agents and editors don't change between rows: look them up once. Only the
  // focused workspace's agents are offered (design D11).
  const editors = availableEditors();
  const agents = agentsInWorkspace(listAgents(), currentWorkspaceId());
  return scanTree(root, expanded).map((row) =>
    toDockRow(row, expanded.has(row.rel), row.rel === selected, editors, agents),
  );
}

function toDockRow(
  row: TreeRow,
  isExpanded: boolean,
  isSelected: boolean,
  editors: readonly Editor[],
  agents: readonly AgentInfo[],
): DockRow {
  const baseIndent = INDENT.repeat(row.depth);
  if (row.kind === "folder") {
    return {
      text: `${baseIndent}${isExpanded ? "▾" : "▸"} ${row.name}`,
      // Folders take the accent tone, so they read differently from prompts.
      tone: "accent",
      action: "toggle",
      value: row.rel,
    };
  }
  const indent = isSelected ? baseIndent.slice(DOT_PREFIX_WIDTH) : baseIndent;
  // A left-click only selects (design D9): opening a file is a menu choice, so a
  // stray click cannot pull focus into a new editor pane.
  const dockRow: DockRow = {
    text: `${indent}${row.name}`,
    action: "select",
    value: row.rel,
    // Send targets first: they are the common action, and the menu reads top-down.
    menu: [
      ...sendMenuEntries(row.rel, agents),
      { title: "Open", action: "edit-inline", value: row.rel },
      ...editMenuEntries(row.rel, editors),
    ],
  };
  if (isSelected) {
    // Mirrors the built-in docks' focused row as far as the dock API allows
    // (design D10): accent text, which is the exact colour they use, plus a dot.
    dockRow.tone = "accent";
    dockRow.dot = SELECTED_DOT;
  }
  return dockRow;
}

/** Scan the configured root and push the rows into the `prompts` dock. */
export function refreshDock(): void {
  const root = settingRoot();
  const dir = moduleStateDir();
  const rows = buildRows(root, readExpanded(dir), readSelected(dir));
  // Push the title too: without one, luvus titles the dock from its id.
  const res = luvus(["ui", "dock", "push", "--id", DOCK_ID, "--title", DOCK_TITLE], {
    input: JSON.stringify(rows),
  });
  if (res.status !== 0) {
    console.error(
      `ui dock push failed (status ${res.status}): ${res.stderr.trim() || res.stdout.trim()}`,
    );
  }
}

if (isMain(import.meta.url)) refreshDock();
