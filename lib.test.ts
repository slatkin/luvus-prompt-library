// lib.test.ts — run with `node --test`.
import assert from "node:assert/strict";
import { homedir } from "node:os";
import path from "node:path";
import { test } from "node:test";

import { expandHome, resolveRoot, resolveUnderRoot } from "./lib.ts";

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
