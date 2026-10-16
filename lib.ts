// lib.ts — shared helpers for the Prompt Dock module.
import { execFileSync } from "node:child_process";
import { homedir } from "node:os";
import path from "node:path";

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
