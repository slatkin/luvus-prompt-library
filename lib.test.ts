// lib.test.ts — run with `node --test`.
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  chmodSync,
  existsSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { homedir, tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";

import { resolveInlineEditor, splitExternalValue } from "./edit.ts";
import { editorMenuEntries, sendMenuEntries } from "./refresh.ts";

import {
  encodeSendValue,
  expandHome,
  findOnPath,
  parseAgents,
  readExpanded,
  resolveRoot,
  resolveUnderRoot,
  scanTree,
  splitSendValue,
  targetLabel,
  writeExpanded,
} from "./lib.ts";

// Tests that run a POSIX shell script or create symlinks are skipped on Windows.
const posixOnly = { skip: process.platform === "win32" };

test("expandHome expands a leading ~ to the home folder", () => {
  assert.equal(expandHome("~/.prompts"), path.join(homedir(), ".prompts"));
  assert.equal(expandHome("~"), homedir());
  assert.equal(expandHome("~\\prompts"), path.join(homedir(), "prompts"));
});

test("expandHome leaves plain and absolute paths alone", () => {
  assert.equal(expandHome("prompts"), "prompts");
  assert.equal(expandHome("/tmp/prompts"), "/tmp/prompts");
});

test("resolveRoot defaults an empty setting to ~/.prompts", () => {
  assert.equal(resolveRoot(""), path.join(homedir(), ".prompts"));
});

test("resolveRoot reads the root setting and expands ~", () => {
  assert.equal(resolveRoot("~/notes"), path.join(homedir(), "notes"));
  assert.equal(resolveRoot("  ~/notes  "), path.join(homedir(), "notes"));
  assert.equal(resolveRoot("/tmp/prompts"), path.resolve("/tmp/prompts"));
});

test("resolveUnderRoot resolves relpaths inside the root", () => {
  const root = path.resolve("/tmp/library");
  assert.equal(resolveUnderRoot(root, "plan.md"), path.join(root, "plan.md"));
  assert.equal(resolveUnderRoot(root, "review/rust.md"), path.join(root, "review", "rust.md"));
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

test("scanTree: symlinks show as their targets, and a folder loop stops", posixOnly, () => {
  const root = tempLibrary();
  symlinkSync(path.join(root, "plan.md"), path.join(root, "linked.md"));
  symlinkSync(path.join(root, "review"), path.join(root, "alias"), "dir");
  symlinkSync(root, path.join(root, "review", "up"), "dir");
  const rows = scanTree(root, new Set(["alias", "review", "review/up"]));
  assert.deepEqual(
    rows.map((r) => [r.rel, r.depth, r.kind]),
    [
      ["alias", 0, "folder"],
      ["alias/up", 1, "folder"],
      ["alias/rust.md", 1, "file"],
      ["alias/ts.md", 1, "file"],
      ["review", 0, "folder"],
      ["review/up", 1, "folder"],
      ["review/rust.md", 1, "file"],
      ["review/ts.md", 1, "file"],
      ["linked.md", 0, "file"],
      ["plan.md", 0, "file"],
    ],
  );
  rmSync(root, { recursive: true, force: true });
});

test("readExpanded treats a missing or corrupt file as empty", () => {
  const empty = mkdtempSync(path.join(tmpdir(), "prompt-dock-state-"));
  assert.deepEqual(readExpanded(empty), new Set());
  rmSync(empty, { recursive: true, force: true });
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

test("findOnPath finds a binary that exists on PATH (sh)", posixOnly, () => {
  assert.equal(findOnPath("sh"), true);
});

test("findOnPath returns false for a missing binary", () => {
  assert.equal(findOnPath("definitely-not-a-real-binary-xyz"), false);
});

test("editorMenuEntries use the editor|relpath encoding", () => {
  // Editors are passed in, so the test does not depend on what is on PATH.
  const entries = editorMenuEntries("review/plan.md", [
    { bin: "zed", title: "Open in zed" },
    { bin: "code", title: "Open in VS Code" },
  ]);
  assert.deepEqual(
    entries.map((e) => [e.title, e.action, e.value]),
    [
      ["Open in zed", "edit-external", "zed|review/plan.md"],
      ["Open in VS Code", "edit-external", "code|review/plan.md"],
    ],
  );
});

test("splitExternalValue splits on the first | only, so | in a filename works", () => {
  const [editor, rel] = splitExternalValue("code|weird|name.md");
  assert.equal(editor, "code");
  assert.equal(rel, "weird|name.md");
});

// ---- send targets (tasks 4.1 / 4.2) ----

test("encodeSendValue then splitSendValue round-trips a send value", () => {
  const value = encodeSendValue("118", "a".repeat(32), "plan.md");
  assert.deepEqual(splitSendValue(value), ["118", "a".repeat(32), "plan.md"]);
});

test("splitSendValue splits on the first two | only, so | in a relpath works", () => {
  assert.deepEqual(splitSendValue("12|abc|review|plan.md"), ["12", "abc", "review|plan.md"]);
  assert.deepEqual(splitSendValue("12|abc|plan.md"), ["12", "abc", "plan.md"]);
  assert.deepEqual(splitSendValue("no-separators.md"), ["", "", "no-separators.md"]);
});

test("targetLabel: a named agent shows alias, kind, status, and workspace", () => {
  const label = targetLabel({
    pane: "9",
    agent: "pi",
    terminal_id: "a".repeat(32),
    name: "pipi",
    status: "idle",
    workspace_name: "pi-luvus",
  });
  for (const part of ["pipi", "pi", "idle", "pi-luvus"]) {
    assert.ok(label.includes(part), `label ${label} should contain ${part}`);
  }
});

test("targetLabel: an unnamed agent shows kind and pane id, without the parenthetical", () => {
  const label = targetLabel({
    pane: "118",
    agent: "claude",
    terminal_id: "a".repeat(32),
    name: null,
    status: "working",
    workspace_name: "mbv",
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
        {
          pane: "9",
          agent: "pi",
          terminal_id: "a".repeat(32),
          name: "pipi",
          status: "idle",
          workspace_name: "pi-luvus",
        },
        {
          pane: "10",
          agent: "claude",
          terminal_id: null,
          name: null,
          status: "working",
          workspace_name: "mbv",
        },
        { pane: "11" }, // no agent kind: dropped
      ],
    },
  });
  const agents = parseAgents(stdout);
  assert.deepEqual(
    agents.map((a) => [a.pane, a.agent, a.terminal_id, a.name]),
    [
      ["9", "pi", "a".repeat(32), "pipi"],
      ["10", "claude", null, null],
    ],
  );
});

test("parseAgents returns no agents for other shapes, nulls, and garbage", () => {
  const agent = {
    pane: "7",
    agent: "codex",
    terminal_id: "b".repeat(32),
    name: null,
    status: "idle",
    workspace_name: "w",
  };
  assert.deepEqual(parseAgents(JSON.stringify({ agents: [agent] })), []);
  assert.deepEqual(parseAgents(JSON.stringify([agent])), []);
  assert.deepEqual(parseAgents("not json"), []);
  assert.deepEqual(parseAgents("null"), []);
  assert.deepEqual(parseAgents(JSON.stringify({ result: { agents: [null, 5] } })), []);
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

test(
  "sendMenuEntries: one Send to entry per agent; no terminal id encodes as empty",
  posixOnly,
  () => {
    const hex = "c".repeat(32);
    const script = fakeLuvus(
      JSON.stringify({
        id: "1",
        result: {
          agents: [
            {
              pane: "9",
              agent: "pi",
              terminal_id: hex,
              name: "pipi",
              status: "idle",
              workspace_name: "pi-luvus",
            },
            {
              pane: "10",
              agent: "claude",
              terminal_id: null,
              name: null,
              status: "working",
              workspace_name: "mbv",
            },
          ],
        },
      }),
    );
    const prevBin = process.env.LUVUS_BIN_PATH;
    process.env.LUVUS_BIN_PATH = script;
    try {
      const entries = sendMenuEntries("plan.md");
      assert.deepEqual(
        entries.map((e) => [e.action, e.value]),
        [
          ["send", `9|${hex}|plan.md`],
          ["send", "10||plan.md"],
        ],
      );
      assert.ok(entries[0].title.includes("pipi"), entries[0].title);
      assert.ok(entries[1].title.startsWith("Send to "), entries[1].title);
    } finally {
      if (prevBin === undefined) delete process.env.LUVUS_BIN_PATH;
      else process.env.LUVUS_BIN_PATH = prevBin;
      rmSync(path.dirname(script), { recursive: true, force: true });
    }
  },
);

/** Run send.ts with a fake luvus that logs each call; returns exit status and the log. */
function runSend(value: string, root: string): { status: number | null; calls: string } {
  const dir = mkdtempSync(path.join(tmpdir(), "prompt-dock-send-"));
  const log = path.join(dir, "calls.log");
  const script = path.join(dir, "fake-luvus");
  writeFileSync(script, `#!/bin/sh\necho "$@" >> "${log}"\n`);
  chmodSync(script, 0o755);
  const sendScript = fileURLToPath(new URL("./send.ts", import.meta.url));
  const res = spawnSync(process.execPath, [sendScript], {
    encoding: "utf8",
    env: {
      ...process.env,
      LUVUS_BIN_PATH: script,
      LUVUS_SETTING_ROOT: root,
      LUVUS_MODULE_ROW_VALUE: value,
    },
  });
  const calls = existsSync(log) ? readFileSync(log, "utf8") : "";
  rmSync(dir, { recursive: true, force: true });
  return { status: res.status, calls };
}

test("send refuses an agent with no terminal id before any agent prompt call", posixOnly, () => {
  const root = tempLibrary();
  const { status, calls } = runSend("9||plan.md", root);
  assert.notEqual(status, 0);
  assert.ok(calls.includes("Send to pane 9 failed: no terminal id"), calls);
  assert.ok(!calls.includes("agent prompt"), calls);
  rmSync(root, { recursive: true, force: true });
});

test("send refuses an empty prompt file and names the agent", posixOnly, () => {
  const root = tempLibrary();
  writeFileSync(path.join(root, "empty.md"), "  \n");
  const { status, calls } = runSend(`9|${"d".repeat(32)}|empty.md`, root);
  assert.notEqual(status, 0);
  assert.ok(calls.includes('Send to pane 9 failed: "empty" is empty.'), calls);
  assert.ok(!calls.includes("agent prompt"), calls);
  rmSync(root, { recursive: true, force: true });
});

test("send's read-failure toast names the agent", posixOnly, () => {
  const root = tempLibrary();
  const { status, calls } = runSend(`9|${"d".repeat(32)}|missing.md`, root);
  assert.notEqual(status, 0);
  assert.ok(calls.includes('Send to pane 9 failed: couldn\'t read "missing".'), calls);
  rmSync(root, { recursive: true, force: true });
});
