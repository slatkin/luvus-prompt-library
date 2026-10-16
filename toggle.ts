// toggle.ts — expand or collapse the folder in LUVUS_MODULE_ROW_VALUE
// (design D5), persist the set in the module state dir, then repaint.
import { moduleStateDir, readExpanded, writeExpanded } from "./lib.ts";
import { refreshDock } from "./refresh.ts";

const value = process.env.LUVUS_MODULE_ROW_VALUE;
if (!value) {
  console.error("toggle: no folder value (LUVUS_MODULE_ROW_VALUE unset)");
  process.exit(1);
}

const dir = moduleStateDir();
const expanded = readExpanded(dir);
if (expanded.has(value)) {
  expanded.delete(value);
} else {
  expanded.add(value);
}
writeExpanded(dir, expanded);

refreshDock();
