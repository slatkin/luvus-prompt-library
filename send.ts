// send.ts — give a stored prompt to a live agent and submit it (design D4).
// Row menu value: `<pane>|<terminal_id>|<relpath>` (design D3). The file is
// read at action time, so edits saved after the last dock repaint go out.
import { readFileSync } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

import {
  listAgents,
  luvus,
  resolveUnderRoot,
  settingRoot,
  splitSendValue,
  targetLabel,
  type AgentInfo,
} from "./lib.ts";

/** Show a one-line message in the UI (design Context). */
function toast(text: string): void {
  luvus(["ui", "toast", text]);
}

/** Fail loudly in `luvus module log` and stop (non-zero exit). */
function fail(context: string, detail: string): never {
  console.error(`${context}: ${detail}`);
  throw new Error(`${context}: ${detail}`);
}

/** One line, bounded, for a toast: stderr if luvus wrote any, else stdout. */
function summarize(res: { stdout: string; stderr: string }): string {
  return (res.stderr.trim() || res.stdout.trim()).replace(/\s+/g, " ").slice(0, 160);
}

/** The prompt's display name: the file name without `.md`. */
function promptName(rel: string): string {
  return path.basename(rel).replace(/\.md$/i, "");
}

/**
 * The agent label for toasts: the agent now at `pane` with that exact
 * terminal id (design D4 — the id is the guard), or `pane <id>` when luvus
 * no longer reports that pair.
 */
function targetLabelFor(pane: string, terminalId: string): string {
  const match: AgentInfo | undefined = listAgents().find(
    (a) => a.pane === pane && a.terminal_id === terminalId,
  );
  return match ? targetLabel(match) : `pane ${pane}`;
}

function send(value: string): void {
  const [pane, terminalId, rel] = splitSendValue(value);
  if (!pane || !terminalId) {
    fail("send", `malformed value (expected <pane>|<terminal_id>|<relpath>): ${value}`);
  }

  const abs = resolveUnderRoot(settingRoot(), rel);
  if (!abs) {
    toast("That file is outside your prompt library root, so it can't be sent.");
    fail("send", `relpath escapes root: ${rel}`);
  }

  // Read at action time, so edits saved after the last repaint are included
  // (spec: send submits the prompt).
  let text: string;
  try {
    text = readFileSync(abs, "utf8");
  } catch (err) {
    toast(`Couldn't read "${promptName(rel)}". Please try again.`);
    fail("send", `read failed for ${abs}: ${err}`);
  }

  const label = targetLabelFor(pane, terminalId);
  // The text is ONE argv element (`execFileSync`, no shell, design D4), so
  // its newlines survive; `agent prompt` submits it in one atomic request.
  const res = luvus(["agent", "prompt", pane, text, "--terminal-id", terminalId]);
  if (res.status !== 0) {
    // `agent_not_ready` (permission prompt up), a stale target, or any other
    // refusal: nothing was queued — say so, with the agent label.
    toast(`Send to ${label} failed: ${summarize(res)}`);
    fail("send", `agent prompt failed (status ${res.status}): ${summarize(res)}`);
  }
  toast(`Sent "${promptName(rel)}" to ${label}.`);
}

function runSend(): void {
  const value = process.env.LUVUS_MODULE_ROW_VALUE;
  if (!value) fail("send", "no row value (LUVUS_MODULE_ROW_VALUE unset)");
  send(value);
}

// Run only when executed directly (as the module action), so the tests can
// import helpers without triggering a send or an early exit.
function isMain(): boolean {
  const entry = process.argv[1];
  if (!entry) return false;
  try {
    return pathToFileURL(entry).href === import.meta.url;
  } catch {
    return false;
  }
}

if (isMain()) runSend();
