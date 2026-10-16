// refresh.ts — scan the prompt library and push the dock rows (design D2).
// Runs on startup, on every event hook, for the `refresh` action, and from
// toggle.ts after it flips the expanded set. Sends one JSON row array to
// `ui dock push` on stdin.
import { existsSync, statSync } from "node:fs";
import { pathToFileURL } from "node:url";

import {
  encodeSendValue,
  listAgents,
  luvus,
  moduleStateDir,
  readExpanded,
  scanTree,
  settingRoot,
  targetLabel,
  type TreeRow,
} from "./lib.ts";
import { findOnPath } from "./edit.ts";

const DOCK_ID = "prompts";
const INDENT = "  ";

/** One row as `ui dock push` accepts it (JSON on stdin). */
export interface DockRow {
  text: string;
  tone?: string;
  action?: string;
  value?: string;
  menu?: DockMenuEntry[];
}

/** One per-row right-click menu entry as `ui dock push` accepts it. */
export interface DockMenuEntry {
  title: string;
  action: string;
  value: string;
}

/**
 * External editor entries for a prompt row, shown only when the binary is
 * actually on PATH (design D7, spec: external editors).
 */
export function editorMenuEntries(rel: string): DockMenuEntry[] {
  const entries: DockMenuEntry[] = [];
  if (findOnPath("zed")) {
    entries.push({ title: "Open in zed", action: "edit-external", value: `zed|${rel}` });
  }
  if (findOnPath("code")) {
    entries.push({ title: "Open in VS Code", action: "edit-external", value: `code|${rel}` });
  }
  return entries;
}

/**
 * One "Send to" entry per live agent for a prompt row (spec: agent send
 * targets). Only agents with a terminal id can be guarded against a reused
 * pane id (design D4), so the rest are left out.
 */
export function sendMenuEntries(rel: string): DockMenuEntry[] {
  return listAgents()
    .filter((a) => a.terminal_id !== null)
    .map((a) => ({
      title: `Send to ${targetLabel(a)}`,
      action: "send",
      value: encodeSendValue(a.pane, a.terminal_id as string, rel),
    }));
}

/** Build the dock rows for `root`: the tree, or the missing-root row. */
export function buildRows(root: string, expanded: ReadonlySet<string>): DockRow[] {
  if (!existsSync(root) || !statSync(root).isDirectory()) {
    // Spec: missing root — one muted row that names the folder, no other rows.
    return [{ text: `root not found: ${root}`, tone: "muted" }];
  }
  return scanTree(root, expanded).map((row) => toDockRow(row, expanded.has(row.rel)));
}

function toDockRow(row: TreeRow, isExpanded: boolean): DockRow {
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
    menu: [...editorMenuEntries(row.rel), ...sendMenuEntries(row.rel)],
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

// Run only when executed directly, so toggle.ts can import refreshDock
// without a second dock push (the dock push replaces all rows, design D2).
function isMain(): boolean {
  const entry = process.argv[1];
  if (!entry) return false;
  try {
    return pathToFileURL(entry).href === import.meta.url;
  } catch {
    return false;
  }
}

if (isMain()) refreshDock();
