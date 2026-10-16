// select.ts — mark the prompt in LUVUS_MODULE_ROW_VALUE as the selected one
// (design D9), persist it, then repaint. The dock has no selection of its own,
// so a left-click on a prompt row runs this instead of opening a file.
import {
  isMain,
  moduleStateDir,
  readSelected,
  resolveUnderRoot,
  settingRoot,
  writeSelected,
} from "./lib.ts";
import { refreshDock } from "./refresh.ts";

function runSelect(): void {
  const value = process.env.LUVUS_MODULE_ROW_VALUE;
  if (!value) {
    console.error("select: no row value (LUVUS_MODULE_ROW_VALUE unset)");
    process.exit(1);
  }
  if (!resolveUnderRoot(settingRoot(), value)) {
    console.error(`select: prompt is outside the prompt library root: ${value}`);
    process.exit(1);
  }

  const dir = moduleStateDir();
  if (readSelected(dir) === value) return; // already selected: no repaint needed

  writeSelected(dir, value);
  refreshDock();
}

if (isMain(import.meta.url)) runSelect();
