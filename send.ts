// send.ts — give a stored prompt to a live agent and submit it (design D4).
// Row menu value: `<pane>|<terminal_id>|<relpath>` (design D3). The file is
// read at action time, so edits saved after the last dock repaint go out.
import { readFileSync } from "node:fs";
import path from "node:path";

import {
  fail,
  isMain,
  listAgents,
  luvus,
  resolveUnderRoot,
  settingRoot,
  splitSendValue,
  targetLabel,
  toast,
  type AgentInfo,
} from "./lib.ts";

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
 * terminal id (design D4 — the id is the guard; an empty id matches an agent
 * with none), or `pane <id>` when luvus no longer reports that pair.
 */
function labelForTarget(pane: string, terminalId: string): string {
  const match: AgentInfo | undefined = listAgents().find(
    (a) => a.pane === pane && (a.terminal_id ?? "") === terminalId,
  );
  return match ? targetLabel(match) : `pane ${pane}`;
}

function send(value: string): void {
  const [pane, terminalId, rel] = splitSendValue(value);
  if (!pane) {
    fail("send", `malformed value (expected <pane>|<terminal_id>|<relpath>): ${value}`);
  }

  const label = labelForTarget(pane, terminalId);
  const abs = resolveUnderRoot(settingRoot(), rel);
  if (!abs) {
    toast(`Send to ${label} failed: that file is outside your prompt library root.`);
    fail("send", `relpath escapes root: ${rel}`);
  }

  if (!terminalId) {
    // Without a terminal id the send can't be guarded against a reused pane id (design D4).
    toast(`Send to ${label} failed: no terminal id`);
    fail("send", `no terminal id for pane ${pane}`);
  }

  // Read at action time, so edits saved after the last repaint are included
  // (spec: send submits the prompt).
  let text: string;
  try {
    // A UTF-8 byte-order mark is not part of the prompt; drop it.
    text = readFileSync(abs, "utf8").replace(/^\uFEFF/, "");
  } catch (err) {
    toast(`Send to ${label} failed: couldn't read "${promptName(rel)}".`);
    fail("send", `read failed for ${abs}: ${err}`);
  }
  if (!text.trim()) {
    toast(`Send to ${label} failed: "${promptName(rel)}" is empty.`);
    fail("send", `prompt file is empty: ${abs}`);
  }

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

if (isMain(import.meta.url)) runSend();
