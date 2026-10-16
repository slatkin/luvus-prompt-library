// lib.ts — shared helpers for the Prompt Dock module.
import { execFileSync } from "node:child_process";
import { homedir } from "node:os";
import path from "node:path";
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";

export interface LuvusResult {
  status: number;
  stdout: string;
  stderr: string;
}

/** The luvus binary to call back into: LUVUS_BIN_PATH, else `luvus` on PATH. */
export function luvusBinPath(): string {
  return process.env.LUVUS_BIN_PATH || "luvus";
}

/** Decode a failed execFileSync error without runtime typeof checks. */
interface SyncError {
  status?: number;
  stdout?: string;
  stderr?: string;
}

/**
 * Run a luvus CLI command as one argv array, no shell (design D4).
 * Never throws on a non-zero exit; the caller inspects `status`.
 */
export function luvus(
  args: readonly string[],
  opts: { input?: string } = {},
): LuvusResult {
  try {
    const stdout = execFileSync(luvusBinPath(), [...args], {
      shell: false,
      encoding: "utf8",
      input: opts.input,
      maxBuffer: 64 * 1024 * 1024,
    });
    return { status: 0, stdout, stderr: "" };
  } catch (err) {
    const e = err as SyncError;
    return { status: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}

/** Expand a leading `~` to the user's home folder (spec: configurable root). */
export function expandHome(p: string): string {
  if (p === "~") return homedir();
  if (p.startsWith("~/")) return path.join(homedir(), p.slice(2));
  return p;
}

/**
 * Resolve a `root` value: trim it, treat an empty value as the `~/prompts`
 * default, and expand `~`.
 */
export function resolveRoot(raw: string): string {
  return path.resolve(expandHome(raw.trim() || "~/prompts"));
}

/** The prompt library root: the `root` module setting, with `~` expanded. */
export function settingRoot(): string {
  return resolveRoot(process.env.LUVUS_SETTING_ROOT ?? "");
}

/**
 * Resolve a relpath against the root, or return null when it escapes the root
 * (design D3). Absolute relpaths and any `..` that climbs past the root are
 * rejected, so a stale row value can never point outside the current root.
 */
export function resolveUnderRoot(root: string, rel: string): string | null {
  const base = path.resolve(root);
  const abs = path.resolve(base, rel);
  if (abs !== base && !abs.startsWith(base + path.sep)) return null;
  return abs;
}

/** The module's persistent state folder (design D5). */
export function moduleStateDir(): string {
  return process.env.LUVUS_MODULE_STATE_DIR || ".";
}

/**
 * The expanded-folder set: a JSON array of folder relpaths (design D5). A
 * missing or corrupt file means "all collapsed".
 */
export function readExpanded(dir: string): Set<string> {
  try {
    const parsed: unknown = JSON.parse(
      readFileSync(path.join(dir, "expanded.json"), "utf8"),
    );
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((v): v is string => typeof v === "string"));
  } catch {
    return new Set();
  }
}

export function writeExpanded(dir: string, expanded: Iterable<string>): void {
  mkdirSync(dir, { recursive: true });
  writeFileSync(
    path.join(dir, "expanded.json"),
    `${JSON.stringify([...expanded], null, 2)}\n`,
  );
}

/** One entry of the scanned tree, in dock-row order (spec: tree rows). */
export interface TreeRow {
  /** Path relative to the root, `/`-separated on every platform. */
  rel: string;
  /** Display name: the folder name, or the file name without `.md`. */
  name: string;
  /** 0 for root-level entries, 1 for their children, and so on. */
  depth: number;
  kind: "folder" | "file";
}

/** Ascending by-name order for folder and file name groups (spec: tree rows). */
const byName = (a: string, b: string): number => (a < b ? -1 : a > b ? 1 : 0);

/**
 * Depth-first scan of the prompt library (spec: tree rows). Folders come
 * before files at each level, each group sorted by name; only `.md` files
 * show; names starting with `.` are skipped; the children of a folder that is
 * not in `expanded` are never read (design D5: collapsed costs nothing).
 * Relpaths use `/` separators on every platform so the expanded-set keys in
 * `expanded.json` are stable across platforms.
 */
export function scanTree(
  root: string,
  expanded: ReadonlySet<string>,
): TreeRow[] {
  const rows: TreeRow[] = [];
  const walk = (dir: string, rel: string, depth: number): void => {
    let entries: ReturnType<typeof readdirSync>;
    try {
      entries = readdirSync(dir, { withFileTypes: true });
    } catch {
      return; // unreadable folder: render its other siblings, nothing under it
    }
    const folders: string[] = [];
    const files: string[] = [];
    for (const entry of entries) {
      if (entry.name.startsWith(".")) continue;
      if (entry.isDirectory()) folders.push(entry.name);
      else if (entry.isFile() && entry.name.endsWith(".md")) files.push(entry.name);
    }
    folders.sort(byName);
    files.sort(byName);
    for (const name of folders) {
      const childRel = rel ? `${rel}/${name}` : name;
      rows.push({ rel: childRel, name, depth, kind: "folder" });
      if (expanded.has(childRel)) {
        walk(path.join(dir, name), childRel, depth + 1);
      }
    }
    for (const name of files) {
      rows.push({ rel: rel ? `${rel}/${name}` : name, name: name.slice(0, -3), depth, kind: "file" });
    }
  };
  walk(root, "", 0);
  return rows;
}
