// refresh.ts — scan the prompt library and push the dock rows (design D2).
// Runs on startup, on every event hook, for the `refresh` action, and from
// toggle.ts after it flips the expanded set. Sends one JSON row array to
// `ui dock push` on stdin.
import { existsSync, statSync } from "node:fs";
import { pathToFileURL } from "node:url";

import {
  luvus,
  moduleStateDir,
  readExpanded,
  scanTree,
  settingRoot,
  type TreeRow,
} from "./lib.ts";

const DOCK_ID = "prompts";
const INDENT = "  ";

/** One row as `ui dock push` accepts it (JSON on stdin). */
export interface DockRow {
  text: string;
  tone?: string;
  action?: string;
  value?: string;
}

/** Build the dock rows for `root`: the tree, or the missing-root row. */
export function buildRows(root: string, expanded: ReadonlySet<string>): DockRow[] {
  if (!existsSync(root) || !statSync(root).isDirectory()) {
    // Spec: missing root — one muted row that names the folder, no other rows.
    return [{ text: `root not found: ${root}`, tone: "muted" }];
  }
  return scanTree(root, expanded).map((row) =>
    toDockRow(row, expanded.has(row.rel))
  );
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
