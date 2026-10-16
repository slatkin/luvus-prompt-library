// lib.ts — shared helpers for the Prompt Dock module.
import { execFileSync } from "node:child_process";
import { homedir } from "node:os";
import path from "node:path";
import {
  accessSync,
  constants,
  mkdirSync,
  readFileSync,
  readdirSync,
  realpathSync,
  renameSync,
  statSync,
  writeFileSync,
  type Dirent,
} from "node:fs";
import { pathToFileURL } from "node:url";

interface LuvusResult {
  status: number;
  stdout: string;
  stderr: string;
}

/** The luvus binary to call back into: LUVUS_BIN_PATH, else `luvus` on PATH. */
function luvusBinPath(): string {
  return process.env.LUVUS_BIN_PATH || "luvus";
}

/** Decode a failed execFileSync error without runtime typeof checks. */
interface SyncError {
  status?: number;
  stdout?: string;
  stderr?: string;
  code?: string;
}

/**
 * Run a luvus CLI command as one argv array, no shell (design D4).
 * Never throws on a non-zero exit; the caller inspects `status`.
 */
export function luvus(args: readonly string[], opts: { input?: string } = {}): LuvusResult {
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
    // A spawn failure (no binary, argv too long) has no stderr; its code says why.
    return { status: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr || e.code || "" };
  }
}

/** Expand a leading `~` to the user's home folder (spec: configurable root). */
export function expandHome(p: string): string {
  if (p === "~") return homedir();
  // Windows users type `~\`, so accept both separators.
  if (/^~[\\/]/.test(p)) return path.join(homedir(), p.slice(2));
  return p;
}

/**
 * Resolve a `root` value: trim it, treat an empty value as the `~/.prompts`
 * default, and expand `~`.
 */
export function resolveRoot(raw: string): string {
  return path.resolve(expandHome(raw.trim() || "~/.prompts"));
}

/** The prompt library root: the `root` module setting, with `~` expanded. */
export function settingRoot(): string {
  return resolveRoot(process.env.LUVUS_SETTING_ROOT ?? "");
}

/**
 * Resolve a relpath against the root, or return null when it escapes the root
 * (design D3). Absolute relpaths and any `..` that climbs past the root are
 * rejected, so a stale row value can never point outside the current root.
 * This check is lexical: a symlink under the root is followed, so its target may lie outside.
 */
export function resolveUnderRoot(root: string, rel: string): string | null {
  const base = path.resolve(root);
  const abs = path.resolve(base, rel);
  if (abs !== base && !abs.startsWith(base + path.sep)) return null;
  return abs;
}

/**
 * The module's persistent state folder (design D5). luvus always sets it; if it
 * is missing, fail rather than write the expanded set into the working folder.
 */
export function moduleStateDir(): string {
  const dir = process.env.LUVUS_MODULE_STATE_DIR;
  if (!dir) fail("module", "LUVUS_MODULE_STATE_DIR is unset");
  return dir;
}

/**
 * The expanded-folder set: a JSON array of folder relpaths (design D5). A
 * missing or corrupt file means "all collapsed".
 */
export function readExpanded(dir: string): Set<string> {
  try {
    const parsed: unknown = JSON.parse(readFileSync(path.join(dir, "expanded.json"), "utf8"));
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((v): v is string => typeof v === "string"));
  } catch {
    return new Set();
  }
}

export function writeExpanded(dir: string, expanded: Iterable<string>): void {
  mkdirSync(dir, { recursive: true });
  // Write a temp file, then rename it over the old one, so a refresh never reads half a file.
  // ponytail: two clicks in quick succession can still race; add a lock file if that bites.
  const file = path.join(dir, "expanded.json");
  writeFileSync(`${file}.tmp`, `${JSON.stringify([...expanded], null, 2)}\n`);
  renameSync(`${file}.tmp`, file);
}

/**
 * The selected prompt relpath (design D9), or null when nothing is selected.
 * The dock has no selection of its own, so the module keeps it. A missing or
 * corrupt file means "nothing selected".
 */
export function readSelected(dir: string): string | null {
  try {
    const parsed: unknown = JSON.parse(readFileSync(path.join(dir, "selected.json"), "utf8"));
    return typeof parsed === "string" ? parsed : null;
  } catch {
    return null;
  }
}

export function writeSelected(dir: string, rel: string): void {
  mkdirSync(dir, { recursive: true });
  // Temp file + rename, for the same reason as the expanded set.
  const file = path.join(dir, "selected.json");
  writeFileSync(`${file}.tmp`, `${JSON.stringify(rel, null, 2)}\n`);
  renameSync(`${file}.tmp`, file);
}

/** One agent as `luvus agent list` reports it: the fields the module needs. */
export interface AgentInfo {
  pane: string;
  agent: string;
  /** The terminal id as luvus reports it, or null when the pane has no terminal runtime. */
  terminal_id: string | null;
  /** The live alias, or null when the agent has none. */
  name: string | null;
  /** The id of the workspace that holds the pane, as `agent list` reports it. */
  workspace: string;
}

/**
 * Pull the agent array out of `luvus agent list` output. The CLI prints the
 * RPC envelope (`{"id":"1","result":{"agents":[…]}}`). Entries without a pane
 * or agent kind are dropped; a null `terminal_id` is kept and the caller
 * decides what to do with it.
 */
export function parseAgents(stdout: string): AgentInfo[] {
  let parsed: { result?: { agents?: unknown } } | null;
  try {
    parsed = JSON.parse(stdout);
  } catch {
    return []; // unparseable reply: no targets, never a crash
  }
  const list = parsed?.result?.agents;
  if (!Array.isArray(list)) return [];
  const agents: AgentInfo[] = [];
  for (const raw of list) {
    const a = (raw ?? {}) as Partial<AgentInfo>;
    if (typeof a.pane !== "string" || typeof a.agent !== "string") continue;
    agents.push({
      pane: a.pane,
      agent: a.agent,
      terminal_id: typeof a.terminal_id === "string" ? a.terminal_id : null,
      name: typeof a.name === "string" ? a.name : null,
      workspace: typeof a.workspace === "string" ? a.workspace : "",
    });
  }
  return agents;
}

/**
 * The live agents across all workspaces and tabs (`luvus agent list`, design
 * D2). A failed or unparseable reply means no targets, never a crash.
 */
export function listAgents(): AgentInfo[] {
  const res = luvus(["agent", "list"]);
  return res.status === 0 ? parseAgents(res.stdout) : [];
}

/** The focused workspace id, as luvus injects it (`LUVUS_WORKSPACE_ID`, design D11). */
export function currentWorkspaceId(): string {
  return process.env.LUVUS_WORKSPACE_ID ?? "";
}

/**
 * The agents that belong to one workspace (design D11). An empty id keeps every
 * agent: the id comes from the environment, and hiding every send target is a
 * worse failure than showing too many.
 */
export function agentsInWorkspace(agents: readonly AgentInfo[], workspaceId: string): AgentInfo[] {
  if (!workspaceId) return [...agents];
  return agents.filter((a) => a.workspace === workspaceId);
}

/**
 * The "Send to" row value: `<pane>|<terminal_id>|<relpath>` (design D3). The
 * terminal id is what makes luvus reject a reused pane id (design D4).
 */
export function encodeSendValue(pane: string, terminalId: string, rel: string): string {
  return `${pane}|${terminalId}|${rel}`;
}

/**
 * Split a send value into `[pane, terminalId, relpath]`, on the first two `|`
 * only (design D3), so a `|` in a file name still works.
 */
export function splitSendValue(value: string): [string, string, string] {
  const first = value.indexOf("|");
  if (first === -1) return ["", "", value];
  const second = value.indexOf("|", first + 1);
  if (second === -1) {
    return [value.slice(0, first), "", value.slice(first + 1)];
  }
  return [value.slice(0, first), value.slice(first + 1, second), value.slice(second + 1)];
}

/**
 * The target label for menus and toasts (design D8): `<name> (<agent>) · pane
 * <pane>`, or `<agent> · pane <pane>` when the agent has no alias.
 * The menu lists every live agent, so status and workspace are left out: the
 * label stays short enough to read at a glance.
 */
export function targetLabel(a: AgentInfo): string {
  const base = a.name ? `${a.name} (${a.agent})` : a.agent;
  return `${base} · pane ${a.pane}`;
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

/**
 * Depth-first scan of the prompt library (spec: tree rows). Folders come
 * before files at each level, each group sorted by name; only `.md` files
 * show; names starting with `.` are skipped; the children of a folder that is
 * not in `expanded` are never read (design D5: collapsed costs nothing).
 * Relpaths use `/` separators on every platform so the expanded-set keys in
 * `expanded.json` are stable across platforms. Symlinks count as what they
 * point at. A folder that links back up the tree is listed but not walked again.
 */
export function scanTree(root: string, expanded: ReadonlySet<string>): TreeRow[] {
  const rows: TreeRow[] = [];
  // `ancestors` holds the real paths of the folders above `dir` (loop guard).
  const walk = (dir: string, rel: string, depth: number, ancestors: ReadonlySet<string>): void => {
    let entries: Dirent[];
    let real: string;
    try {
      entries = readdirSync(dir, { withFileTypes: true });
      real = realpathSync(dir);
    } catch {
      return; // unreadable folder: render its other siblings, nothing under it
    }
    if (ancestors.has(real)) return;
    const chain = new Set(ancestors).add(real);
    const folders: string[] = [];
    const files: string[] = [];
    for (const entry of entries) {
      if (entry.name.startsWith(".")) continue;
      // A symlink is judged by its target; a broken link matches nothing.
      const target = entry.isSymbolicLink()
        ? statSync(path.join(dir, entry.name), { throwIfNoEntry: false })
        : entry;
      if (target?.isDirectory()) folders.push(entry.name);
      else if (target?.isFile() && /\.md$/i.test(entry.name)) files.push(entry.name);
    }
    // Default sort: ascending by code unit (spec: tree rows).
    folders.sort();
    files.sort();
    for (const name of folders) {
      const childRel = rel ? `${rel}/${name}` : name;
      rows.push({ rel: childRel, name, depth, kind: "folder" });
      if (expanded.has(childRel)) {
        walk(path.join(dir, name), childRel, depth + 1, chain);
      }
    }
    for (const name of files) {
      rows.push({
        rel: rel ? `${rel}/${name}` : name,
        name: name.slice(0, -3),
        depth,
        kind: "file",
      });
    }
  };
  walk(root, "", 0, new Set());
  return rows;
}

/** Show a one-line message in the UI (design Context). */
export function toast(text: string): void {
  luvus(["ui", "toast", text]);
}

/** Fail loudly in `luvus module log` and stop (non-zero exit). */
export function fail(context: string, detail: string): never {
  console.error(`${context}: ${detail}`);
  throw new Error(`${context}: ${detail}`);
}

/**
 * True when the module at `metaUrl` is the entry script, so action scripts
 * run only when executed directly and tests can import their helpers.
 */
export function isMain(metaUrl: string): boolean {
  const entry = process.argv[1];
  if (!entry) return false;
  // Compare real paths: Node resolves symlinks for import.meta.url, not for argv[1].
  try {
    return pathToFileURL(realpathSync(entry)).href === metaUrl;
  } catch {
    return false;
  }
}

/**
 * Stdlib PATH probe (design D7): split PATH and check each folder for the
 * binary. On Windows, PATHEXT extensions are appended so `code` also matches
 * `code.cmd`. No `which` dependency.
 */
export function findOnPath(bin: string): boolean {
  const pathEnv = process.env.PATH || "";
  const exts =
    process.platform === "win32" ? (process.env.PATHEXT || ".EXE;.CMD;.BAT").split(";") : [""];
  for (const dir of pathEnv.split(path.delimiter)) {
    if (!dir) continue;
    for (const ext of exts) {
      const candidate = path.join(dir, bin + ext.toLowerCase());
      try {
        if (statSync(candidate).isFile()) {
          accessSync(candidate, constants.X_OK); // not executable: keep scanning
          return true;
        }
      } catch {
        // not here — keep scanning
      }
    }
  }
  return false;
}
