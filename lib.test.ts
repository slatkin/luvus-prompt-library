// lib.test.ts — run with `node --test`.
import assert from "node:assert/strict";
import { chmodSync, mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { homedir } from "node:os";
import path from "node:path";
import { test } from "node:test";

import { findOnPath, resolveInlineEditor, splitExternalValue } from "./edit.ts";
import { editorMenuEntries, sendMenuEntries } from "./refresh.ts";

import {
  encodeSendValue,
  expandHome,
  parseAgents,
  readExpanded,
  resolveRoot,
  resolveUnderRoot,
  scanTree,
  splitSendValue,
  targetLabel,
  writeExpanded,
} from "./lib.ts";

test("expandHome expands a leading ~ to the home folder", () => {
  assert.equal(expandHome("~/prompts"), path.join(homedir(), "prompts"));
  assert.equal(expandHome("~"), homedir());
});

test("expandHome leaves plain and absolute paths alone", () => {
  assert.equal(expandHome("prompts"), "prompts");
  assert.equal(expandHome("/tmp/prompts"), "/tmp/prompts");
});

test("resolveRoot defaults an empty setting to ~/prompts", () => {
  assert.equal(resolveRoot(""), path.join(homedir(), "prompts"));
});

test("resolveRoot reads the root setting and expands ~", () => {
  assert.equal(resolveRoot("~/notes"), path.join(homedir(), "notes"));
  assert.equal(resolveRoot("  ~/notes  "), path.join(homedir(), "notes"));
  assert.equal(resolveRoot("/tmp/prompts"), "/tmp/prompts");
});

test("resolveUnderRoot resolves relpaths inside the root", () => {
  const root = "/tmp/library";
  assert.equal(resolveUnderRoot(root, "plan.md"), "/tmp/library/plan.md");
  assert.equal(
    resolveUnderRoot(root, "review/rust.md"),
    "/tmp/library/review/rust.md",
  );
});

test("resolveUnderRoot rejects a relpath that escapes the root", () => {
  assert.equal(resolveUnderRoot("/tmp/library", "../x.md"), null);
  assert.equal(resolveUnderRoot("/tmp/library", "review/../../x.md"), null);
  assert.equal(resolveUnderRoot("/tmp/library", "/etc/passwd"), null);
});

/** A temp prompt library matching the spec scenarios, removed on dispose. */
function tempLibrary(): string {
  const root = mkdtempSync(path.join(tmpdir(), "prompt-dock-test-"));
  mkdirSync(path.join(root, "review"));
  mkdirSync(path.join(root, ".git"));
  writeFileSync(path.join(root, "plan.md"), "plan\n");
  writeFileSync(path.join(root, "review", "rust.md"), "rust\n");
  writeFileSync(path.join(root, "review", "ts.md"), "ts\n");
  writeFileSync(path.join(root, "notes.txt"), "not a prompt\n");
  writeFileSync(path.join(root, ".hidden.md"), "hidden\n");
  return root;
}

test("scanTree: nested library, folders before files, sorted, indented", () => {
  const root = tempLibrary();
  const rows = scanTree(root, new Set(["review"]));
  assert.deepEqual(
    rows.map((r) => [r.rel, r.name, r.depth, r.kind]),
    [
      ["review", "review", 0, "folder"],
      ["review/rust.md", "rust", 1, "file"],
      ["review/ts.md", "ts", 1, "file"],
      ["plan.md", "plan", 0, "file"],
    ],
  );
  rmSync(root, { recursive: true, force: true });
});

test("scanTree: a collapsed folder leaves its children out", () => {
  const root = tempLibrary();
  const rows = scanTree(root, new Set());
  assert.deepEqual(
    rows.map((r) => [r.rel, r.depth, r.kind]),
    [
      ["review", 0, "folder"],
      ["plan.md", 0, "file"],
    ],
  );
  rmSync(root, { recursive: true, force: true });
});

test("scanTree: ignored files — non-.md files and hidden entries never show", () => {
  const root = tempLibrary();
  const rows = scanTree(root, new Set(["review"]));
  const names = rows.map((r) => r.name);
  assert.ok(!names.includes("notes.txt"));
  assert.ok(!names.includes(".git"));
  assert.ok(!names.includes(".hidden.md"));
  assert.ok(!rows.some((r) => r.rel.includes(".git/")));
  rmSync(root, { recursive: true, force: true });
});

test("readExpanded treats a missing or corrupt file as empty", () => {
  assert.deepEqual(readExpanded(tmpdir()), new Set());
  const dir = mkdtempSync(path.join(tmpdir(), "prompt-dock-state-"));
  writeFileSync(path.join(dir, "expanded.json"), "{not json");
  assert.deepEqual(readExpanded(dir), new Set());
  rmSync(dir, { recursive: true, force: true });
});

test("writeExpanded then readExpanded round-trips the folder set", () => {
  const dir = mkdtempSync(path.join(tmpdir(), "prompt-dock-state-"));
  writeExpanded(dir, ["review", "a/b"]);
  assert.deepEqual(readExpanded(dir), new Set(["review", "a/b"]));
  // A corrupt file after a good write still reads as empty (design D5).
  writeFileSync(path.join(dir, "expanded.json"), "null");
  assert.deepEqual(readExpanded(dir), new Set());
  rmSync(dir, { recursive: true, force: true });
});


// ---- editing helpers (tasks 3.1 / 3.2) ----
test("resolveInlineEditor keeps the configured editor", () => {
  assert.equal(resolveInlineEditor("hx"), "hx");
  assert.equal(resolveInlineEditor("emacs -nw"), "emacs -nw");
});

test("resolveInlineEditor falls back to nvim when empty, readonly, or missing", () => {
  assert.equal(resolveInlineEditor(null), "nvim");
  assert.equal(resolveInlineEditor(""), "nvim");
  assert.equal(resolveInlineEditor("  "), "nvim");
  assert.equal(resolveInlineEditor("readonly"), "nvim");
});

test("findOnPath finds a binary that exists on PATH (sh)", () => {
  assert.equal(findOnPath("sh"), true);
});

test("findOnPath returns false for a missing binary", () => {
  assert.equal(findOnPath("definitely-not-a-real-binary-xyz"), false);
});

test("editorMenuEntries use the editor|relpath encoding", () => {
  const entries = editorMenuEntries("review/plan.md");
  const zed = entries.find((e) => e.title === "Open in zed");
  assert.ok(zed, "zed entry present when zed is on PATH");
  assert.equal(zed.action, "edit-external");
  assert.equal(zed.value, "zed|review/plan.md");
  const code = entries.find((e) => e.title === "Open in VS Code");
  if (code) {
    assert.equal(code.action, "edit-external");
    assert.equal(code.value, "code|review/plan.md");
  }
});

test("splitExternalValue splits on the first | only, so | in a filename works", () => {
  const [editor, rel] = splitExternalValue("code|weird|name.md");
  assert.equal(editor, "code");
  assert.equal(rel, "weird|name.md");
});

// ---- send targets (tasks 4.1 / 4.2) ----

test("encodeSendValue then splitSendValue round-trips, even with | in the file name", () => {
  const value = encodeSendValue("118", "a".repeat(32), "weird|name.md");
  assert.deepEqual(splitSendValue(value), ["118", "a".repeat(32), "weird|name.md"]);
});

test("splitSendValue splits on the first two | only, so | in a relpath works", () => {
  assert.deepEqual(splitSendValue("12|abc|review|plan.md"), ["12", "abc", "review|plan.md"]);
  assert.deepEqual(splitSendValue("12|abc|plan.md"), ["12", "abc", "plan.md"]);
  assert.deepEqual(splitSendValue("no-separators.md"), ["", "", "no-separators.md"]);
});

test("targetLabel: a named agent shows alias, kind, status, and workspace", () => {
  const label = targetLabel({
    pane: "9", agent: "pi", terminal_id: "a".repeat(32),
    name: "pipi", status: "idle", workspace_name: "pi-luvus",
  });
  for (const part of ["pipi", "pi", "idle", "pi-luvus"]) {
    assert.ok(label.includes(part), `label ${label} should contain ${part}`);
  }
});

test("targetLabel: an unnamed agent shows kind and pane id, without the parenthetical", () => {
  const label = targetLabel({
    pane: "118", agent: "claude", terminal_id: "a".repeat(32),
    name: null, status: "working", workspace_name: "mbv",
  });
  assert.ok(label.includes("claude"), label);
  assert.ok(label.includes("118"), label);
  assert.ok(!label.includes("(claude)"), `label ${label} should have no parenthetical kind`);
});

test("parseAgents reads the agent array out of the CLI envelope", () => {
  const stdout = JSON.stringify({
    id: "1",
    result: {
      agents: [
        { pane: "9", agent: "pi", terminal_id: "a".repeat(32), name: "pipi", status: "idle", workspace_name: "pi-luvus" },
        { pane: "10", agent: "claude", terminal_id: null, name: null, status: "working", workspace_name: "mbv" },
        { pane: "11" }, // no agent kind: dropped
      ],
    },
  });
  const agents = parseAgents(stdout);
  assert.deepEqual(
    agents.map((a) => [a.pane, a.agent, a.terminal_id, a.name]),
    [["9", "pi", "a".repeat(32), "pipi"], ["10", "claude", null, null]],
  );
});

test("parseAgents tolerates a bare agents object, a bare array, and garbage", () => {
  const agent = { pane: "7", agent: "codex", terminal_id: "b".repeat(32), name: null, status: "idle", workspace_name: "w" };
  assert.equal(parseAgents(JSON.stringify({ agents: [agent] })).length, 1);
  assert.equal(parseAgents(JSON.stringify([agent])).length, 1);
  assert.deepEqual(parseAgents("not json"), []);
  assert.deepEqual(parseAgents(JSON.stringify({ other: true })), []);
});

/** A fake `luvus` binary whose `agent list` prints the given JSON. */
function fakeLuvus(stdout: string): string {
  const dir = mkdtempSync(path.join(tmpdir(), "prompt-dock-bin-"));
  const script = path.join(dir, "fake-luvus");
  writeFileSync(script, `#!/bin/sh\ncat <<'JSON'\n${stdout}\nJSON\n`);
  chmodSync(script, 0o755);
  return script;
}

test("sendMenuEntries: one Send to entry per guarded agent, none for the rest", () => {
  const hex = "c".repeat(32);
  const script = fakeLuvus(JSON.stringify({
    id: "1",
    result: {
      agents: [
        { pane: "9", agent: "pi", terminal_id: hex, name: "pipi", status: "idle", workspace_name: "pi-luvus" },
        { pane: "10", agent: "claude", terminal_id: null, name: null, status: "working", workspace_name: "mbv" },
      ],
    },
  }));
  const prevBin = process.env.LUVUS_BIN_PATH;
  process.env.LUVUS_BIN_PATH = script;
  try {
    const entries = sendMenuEntries("plan.md");
    // The agent without a terminal id cannot be guarded, so it gets no entry.
    assert.equal(entries.length, 1);
    assert.ok(entries[0].title.startsWith("Send to "), entries[0].title);
    assert.ok(entries[0].title.includes("pipi"), entries[0].title);
    assert.equal(entries[0].action, "send");
    assert.equal(entries[0].value, `9|${hex}|plan.md`);
  } finally {
    if (prevBin === undefined) delete process.env.LUVUS_BIN_PATH;
    else process.env.LUVUS_BIN_PATH = prevBin;
    rmSync(path.dirname(script), { recursive: true, force: true });
  }
});
