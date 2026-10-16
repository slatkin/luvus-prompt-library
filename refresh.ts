// refresh.ts — scan the prompt library and push the dock rows (design D2).
// Runs on startup, on every event hook, for the `refresh` action, and from
// toggle.ts after it flips the expanded set. Sends one JSON row array to
// `ui dock push` on stdin.
import { statSync } from "node:fs";

import {
  encodeSendValue,
  findOnPath,
  isMain,
  listAgents,
  luvus,
  moduleStateDir,
  readExpanded,
  scanTree,
  settingRoot,
  targetLabel,
  type AgentInfo,
  type TreeRow,
} from "./lib.ts";

const DOCK_ID = "prompts";
const INDENT = "  ";

/** One row as `ui dock push` accepts it (JSON on stdin). */
interface DockRow {
  text: string;
  tone?: string;
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
  title: string;
}

/** The external editors found on PATH; a refresh checks this once. */
function availableEditors(): Editor[] {
  return [
    { bin: "zed", title: "Open in zed" },
    { bin: "code", title: "Open in VS Code" },
  ].filter((e) => findOnPath(e.bin));
}

/**
 * External editor entries for a prompt row, shown only when the binary is
 * actually on PATH (design D7, spec: external editors).
 */
export function editorMenuEntries(
  rel: string,
  editors: readonly Editor[] = availableEditors(),
): DockMenuEntry[] {
  return editors.map((e) => ({
    title: e.title,
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
function buildRows(root: string, expanded: ReadonlySet<string>): DockRow[] {
  if (!isFolder(root)) {
    // Spec: missing root — one muted row that names the folder, no other rows.
    return [{ text: `root not found: ${root}`, tone: "muted" }];
  }
  // Agents and editors don't change between rows: look them up once.
  const editors = availableEditors();
  const agents = listAgents();
  return scanTree(root, expanded).map((row) =>
    toDockRow(row, expanded.has(row.rel), editors, agents),
  );
}

function toDockRow(
  row: TreeRow,
  isExpanded: boolean,
  editors: readonly Editor[],
  agents: readonly AgentInfo[],
): DockRow {
  const indent = INDENT.repeat(row.depth);
  if (row.kind === "folder") {
    return {
      text: `${indent}${isExpanded ? "▾" : "▸"} ${row.name}`,
      action: "toggle",
      value: row.rel,
    };
  }
  return {
    text: `${indent}${row.name}`,
    action: "edit-inline",
    value: row.rel,
    menu: [...editorMenuEntries(row.rel, editors), ...sendMenuEntries(row.rel, agents)],
  };
}

/** Scan the configured root and push the rows into the `prompts` dock. */
export function refreshDock(): void {
  const root = settingRoot();
  const expanded = readExpanded(moduleStateDir());
  const rows = buildRows(root, expanded);
  const res = luvus(["ui", "dock", "push", "--id", DOCK_ID], {
    input: JSON.stringify(rows),
  });
  if (res.status !== 0) {
    console.error(
      `ui dock push failed (status ${res.status}): ${res.stderr.trim() || res.stdout.trim()}`,
    );
  }
}

if (isMain(import.meta.url)) refreshDock();
