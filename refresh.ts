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
// The tree is drawn with box-drawing branches. Each row with a parent starts
// with one guide cell per ancestor, then a connector; root rows carry no prefix,
// so their glyph sits at the dock's left edge.
const GUIDE_PIPE = "│ ";
const GUIDE_BLANK = "  ";
const BRANCH_MID = "├─ ";
const BRANCH_LAST = "└─ ";
// Row prefixes, written as escapes so the codepoints stay exact and reviewable.
// These are Nerd Font glyphs (Material Design set), so the terminal needs a Nerd
// Font — the same assumption luvus's own web client makes in its font stack.
const PROMPT_GLYPH = "\u{F0B79}"; // md-chat
const FOLDER_GLYPH = "\u{F024B}"; // md-folder
const FOLDER_OPEN_GLYPH = "\u{F0770}"; // md-folder_open

/** One row as `ui dock push` accepts it (JSON on stdin). */
interface DockRow {
  text: string;
  spans?: DockSpan[];
  action?: string;
  value?: string;
  menu?: DockMenuEntry[];
}

/**
 * One drawn segment of a row. A span with no tone takes the row's tone, and a
 * row with no tone takes the default row colour; `text` is drawn only when
 * `spans` is empty.
 */
interface DockSpan {
  text: string;
  tone?: string;
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

/**
 * The branch prefix for every row, in row order (spec: tree rows). A row draws
 * one two-column guide cell per ancestor, then a connector: `├─` when a sibling
 * follows it, `└─` when it is the last. Root rows get no prefix.
 *
 * A guide cell is a pipe only when its ancestor has a later sibling AND is not a
 * root row: a pipe continues the ancestor's own connector down to the next
 * sibling, and a root row has no connector. So the children of a top-level
 * folder keep the two columns but draw no line back up to the folder.
 */
export function branchPrefixes(rows: readonly TreeRow[]): string[] {
  // `hasLater[d]` is the last row seen at depth `d`. It is the ancestor of every
  // deeper row that follows it, since `scanTree` emits rows depth-first.
  const hasLater: boolean[] = [];
  return rows.map((row) => {
    const guides = hasLater
      .slice(0, row.depth)
      .map((later, depth) => (depth > 0 && later ? GUIDE_PIPE : GUIDE_BLANK))
      .join("");
    hasLater[row.depth] = !row.isLast;
    if (row.depth === 0) return "";
    return guides + (row.isLast ? BRANCH_LAST : BRANCH_MID);
  });
}

/**
 * The drawn segments of a row: the branch prefix, then the glyph and name. The
 * prefix never carries a tone, so only the label takes the accent tone — a row
 * `tone` would paint its branch glyphs and its text the same colour (spec: row
 * colours). The row's `text` stays the plain label a click hands to the action.
 */
function rowSpans(prefix: string, label: string, labelTone?: string): DockSpan[] {
  const labelSpan: DockSpan = labelTone ? { text: label, tone: labelTone } : { text: label };
  // luvus drops a span with empty text, so a root row sends the label alone.
  return prefix ? [{ text: prefix }, labelSpan] : [labelSpan];
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
    const label = `root not found: ${root}`;
    return [{ text: label, spans: rowSpans("", label, "muted") }];
  }
  // Agents and editors don't change between rows: look them up once. Only the
  // focused workspace's agents are offered (design D11).
  const editors = availableEditors();
  const agents = agentsInWorkspace(listAgents(), currentWorkspaceId());
  const tree = scanTree(root, expanded);
  const prefixes = branchPrefixes(tree);
  return tree.map((row, i) =>
    toDockRow(row, prefixes[i], expanded.has(row.rel), row.rel === selected, editors, agents),
  );
}

/** One dock row: its branch prefix, then the glyph and name for its kind. */
export function toDockRow(
  row: TreeRow,
  prefix: string,
  isExpanded: boolean,
  isSelected: boolean,
  editors: readonly Editor[],
  agents: readonly AgentInfo[],
): DockRow {
  if (row.kind === "folder") {
    const glyph = isExpanded ? FOLDER_OPEN_GLYPH : FOLDER_GLYPH;
    const label = `${glyph} ${row.name}`;
    return {
      text: `${prefix}${label}`,
      // Folders take the accent tone, so they read differently from prompts.
      spans: rowSpans(prefix, label, "accent"),
      action: "toggle",
      value: row.rel,
    };
  }
  // A left-click only selects (design D9): opening a file is a menu choice, so a
  // stray click cannot pull focus into a new editor pane.
  const label = `${PROMPT_GLYPH} ${row.name}`;
  const dockRow: DockRow = {
    text: `${prefix}${label}`,
    // Mirrors the built-in docks' focused row as far as the dock API allows
    // (design D10): accent text, the exact colour they use, and no marker glyph
    // of its own — every prompt row already carries the prompt glyph.
    spans: rowSpans(prefix, label, isSelected ? "accent" : undefined),
    action: "select",
    value: row.rel,
    // Send targets first: they are the common action, and the menu reads top-down.
    menu: [
      ...sendMenuEntries(row.rel, agents),
      { title: "Open", action: "edit-inline", value: row.rel },
      ...editMenuEntries(row.rel, editors),
    ],
  };
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
