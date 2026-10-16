// lib.test.ts — run with `node --test`.
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { homedir } from "node:os";
import path from "node:path";
import { test } from "node:test";

import {
  expandHome,
  readExpanded,
  resolveRoot,
  resolveUnderRoot,
  scanTree,
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
