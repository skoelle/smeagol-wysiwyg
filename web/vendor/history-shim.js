// Shim: wire up prosemirror-history for Milkdown.
// esbuild can't resolve @milkdown/prose/history subpath exports,
// so we build Milkdown-compatible plugins from raw prosemirror packages.

import { $prose } from "@milkdown/utils";
import { history as pmHistory, undo, redo } from "prosemirror-history";

const historyPlugin = $prose(() => pmHistory());

// Keymap plugin: Ctrl+Z / Ctrl+Y / Ctrl+Shift+Z
import { keymap } from "prosemirror-keymap";
const historyKeymap = $prose(() => keymap({
  "Mod-z": undo,
  "Mod-y": redo,
  "Shift-Mod-z": redo,
}));

export const history = [historyPlugin, historyKeymap];
