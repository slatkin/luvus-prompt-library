// edit.ts — open a prompt file from a dock row (design D6, D7).
// Two modes, chosen by LUVUS_MODULE_ACTION_ID:
//   edit-inline    — split a pane without taking focus and type the editor
//                    command (from luvus "Open files with", layout.file_open)
//                    into its shell; falls back to nvim.
//   edit-external  — spawn zed or VS Code detached on the file and exit at
//                    once (row value is `<editor>|<relpath>`, design D3).
import { spawn } from "node:child_process";
import { homedir } from "node:os";
import path from "node:path";
import { readFileSync } from "node:fs";

import { fail, isMain, luvus, resolveUnderRoot, settingRoot, toast } from "./lib.ts";

const FALLBACK_EDITOR = "nvim";
// The external editors refresh.ts offers; anything else is refused.
const EXTERNAL_EDITORS = ["zed", "code"];

/**
 * Read luvus "Open files with" from ~/.luvus/config.json (design D6).
 * Returns the raw setting value, or null when the file is missing,
 * unreadable or not valid JSON.
 */
function readRawFileOpen(): string | null {
  try {
    const config = JSON.parse(readFileSync(path.join(homedir(), ".luvus", "config.json"), "utf8"));
    const value = config?.layout?.file_open;
    return typeof value === "string" ? value : null;
  } catch {
    return null;
  }
}

/**
 * The editor to open the file with: luvus "Open files with" when it is set
 * and readable, otherwise nvim (spec: inline edit fallback).
 */
export function resolveInlineEditor(raw: string | null): string {
  const value = (raw ?? "").trim();
  // `readonly` is luvus's built-in viewer, which can't edit — use nvim instead.
  if (!value || value === "readonly") return FALLBACK_EDITOR;
  return value;
}

/** Inline mode: split a pane without focusing it, then type the editor command. */
function editInline(value: string): void {
  const root = settingRoot();
  const abs = resolveUnderRoot(root, value);
  if (!abs) {
    toast("That file is outside your prompt library root, so it can't be opened.");
    fail("edit-inline", `relpath escapes root: ${value}`);
  }
  if (/['\\\r\n]/.test(abs)) {
    // `pane run` types the command into a shell, so the path must be single-quoted;
    // an apostrophe, a backslash (fish reads \\ as one \), or a line break would break it.
    toast(
      "Files with an apostrophe, a backslash, or a line break in the name can't be opened inline. Please rename the file.",
    );
    fail("edit-inline", `path contains a quote, backslash, or line break: ${abs}`);
  }

  const editor = resolveInlineEditor(readRawFileOpen());
  const split = luvus(["pane", "split", "--no-focus"]);
  if (split.status !== 0) {
    toast("Couldn't open a pane for editing. Please try again.");
    fail(
      "edit-inline",
      `pane split failed (status ${split.status}): ${split.stderr.trim() || split.stdout.trim()}`,
    );
  }
  let paneId: string;
  try {
    // luvus prints the RPC envelope; a flat object is accepted too.
    const parsed = JSON.parse(split.stdout);
    paneId = String((parsed.result ?? parsed).pane);
  } catch {
    toast("Couldn't open a pane for editing. Please try again.");
    fail("edit-inline", `unexpected pane split output: ${split.stdout.trim()}`);
  }
  if (!paneId || paneId === "undefined") {
    toast("Couldn't open a pane for editing. Please try again.");
    fail("edit-inline", `no pane id in pane split output: ${split.stdout.trim()}`);
  }

  const run = luvus(["pane", "run", paneId, `${editor} '${abs}'`]);
  if (run.status !== 0) {
    // Close the pane we just split, so a failed run leaves no empty pane behind.
    luvus(["pane", "close", paneId]);
    toast("Couldn't start the editor in the new pane. Please try again.");
    fail(
      "edit-inline",
      `pane run failed (status ${run.status}): ${run.stderr.trim() || run.stdout.trim()}`,
    );
  }
}

/**
 * Split an edit-external row value on the FIRST `|` only (design D3), so a
 * `|` in a file name still works. Returns [editor, relpath].
 */
export function splitExternalValue(value: string): [string, string] {
  const sep = value.indexOf("|");
  if (sep === -1) return ["", value];
  return [value.slice(0, sep), value.slice(sep + 1)];
}

/** External mode: start the editor detached and finish at once (design D7). */
function editExternal(value: string): void {
  const [editor, rel] = splitExternalValue(value);
  if (!editor) fail("edit-external", `malformed value (expected <editor>|<relpath>): ${value}`);
  if (!EXTERNAL_EDITORS.includes(editor)) fail("edit-external", `unknown editor: ${editor}`);

  const abs = resolveUnderRoot(settingRoot(), rel);
  if (!abs) {
    toast("That file is outside your prompt library root, so it can't be opened.");
    fail("edit-external", `relpath escapes root: ${rel}`);
  }

  try {
    // Windows starts `code` through its .cmd shim, which spawn can't run without a
    // shell, so the path is quoted for cmd. ponytail: cmd still expands %VAR% inside quotes.
    const win = process.platform === "win32";
    const child = spawn(editor, [win ? `"${abs}"` : abs], {
      detached: true,
      stdio: "ignore",
      shell: win,
    });
    // A failed launch (e.g. binary vanished) must not crash the action.
    child.on("error", (err) => {
      console.error(`edit-external: spawn error: ${err}`);
      toast(`Couldn't start ${editor}. Please try again.`);
    });
    child.unref();
  } catch (err) {
    toast(`Couldn't start ${editor}. Please try again.`);
    fail("edit-external", `spawn failed: ${err}`);
  }
}

function runEdit(): void {
  const action = process.env.LUVUS_MODULE_ACTION_ID;
  const value = process.env.LUVUS_MODULE_ROW_VALUE;
  if (!value) fail("edit", "no row value (LUVUS_MODULE_ROW_VALUE unset)");

  if (action === "edit-inline") {
    editInline(value);
  } else if (action === "edit-external") {
    editExternal(value);
  } else {
    fail("edit", `unknown action: ${action ?? "(none)"}`);
  }
}

if (isMain(import.meta.url)) runEdit();
