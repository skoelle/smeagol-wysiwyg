// smeagol-wysiwyg frontend logic.
//
// Shipped directly (no build step required) and loaded via
// <script src="/assets/main.js" defer> from the server-rendered page
// shell (see internal/server/server.go).
//
// The WYSIWYG editor itself is Milkdown, loaded on demand (only once the
// user actually switches into edit mode) from the local vendor bundle
// at /assets/vendor/milkdown.js (no internet connection required).

const MILKDOWN_URL = "/assets/vendor/milkdown.js";

const state = {
  path: "",
  exists: false,
  editing: false,
  saveTimer: null,
  saveDebounceMs: 1000,
  milkdownEditor: null,
  ignoreNextReloadFor: null,
};

function el(id) { return document.getElementById(id); }

function init() {
  const contentEl = el("content");
  state.path = contentEl.dataset.path || "README.md";
  state.exists = contentEl.dataset.exists === "true";

  el("btn-overview").addEventListener("click", toggleOverview);
  el("btn-edit").addEventListener("click", toggleEdit);
  el("search-input").addEventListener("input", debounce(onSearchInput, 250));

  connectEvents();
}

async function toggleOverview() {
  const panel = el("overview-panel");
  if (!panel.hidden) {
    panel.hidden = true;
    return;
  }
  panel.hidden = false;
  panel.innerHTML = "Lade...";
  try {
    const res = await fetch("/api/tree");
    const tree = await res.json();
    panel.innerHTML = "";
    panel.appendChild(renderTreeNode(tree, true));
  } catch (e) {
    panel.innerHTML = "Fehler beim Laden der Übersicht.";
  }
}

function renderTreeNode(node, isRoot) {
  const wrapper = document.createElement("div");
  if (!isRoot) wrapper.className = "tree-node";

  if (node.isDir) {
    if (!isRoot) {
      const label = document.createElement("div");
      label.className = "tree-dir-label";
      label.textContent = "\ud83d\udcc1 " + node.name;
      wrapper.appendChild(label);
    }
    (node.children || []).forEach((child) => {
      wrapper.appendChild(renderTreeNode(child, false));
    });
  } else {
    const link = document.createElement("a");
    link.href = "/page/" + node.path;
    link.textContent = "\ud83d\udcc4 " + node.name;
    wrapper.appendChild(link);
  }
  return wrapper;
}

async function onSearchInput(evt) {
  const q = evt.target.value.trim();
  const panel = el("search-panel");
  if (!q) {
    panel.hidden = true;
    panel.innerHTML = "";
    return;
  }
  panel.hidden = false;
  panel.innerHTML = "Suche...";
  try {
    const res = await fetch("/api/search?q=" + encodeURIComponent(q));
    const results = await res.json();
    renderSearchResults(panel, results);
  } catch (e) {
    panel.innerHTML = "Fehler bei der Suche.";
  }
}

function renderSearchResults(panel, results) {
  panel.innerHTML = "";
  if (!results || results.length === 0) {
    panel.innerHTML = '<p class="empty-state">Keine Treffer.</p>';
    return;
  }
  results.forEach((r) => {
    const div = document.createElement("div");
    div.className = "search-result";
    const a = document.createElement("a");
    a.href = "/page/" + r.path;
    a.textContent = r.title || r.path;
    const snippet = document.createElement("div");
    snippet.className = "snippet";
    snippet.textContent = r.snippet;
    div.appendChild(a);
    div.appendChild(snippet);
    panel.appendChild(div);
  });
}

async function toggleEdit() {
  if (state.editing) {
    await exitEditMode();
    return;
  }
  await enterEditMode();
}

async function enterEditMode() {
  const contentEl = el("content");
  const mount = el("editor-mount");
  const toolbar = el("editor-toolbar");
  const btn = el("btn-edit");

  const res = await fetch("/api/raw/" + state.path);
  const raw = await res.text();

  contentEl.hidden = true;
  mount.hidden = false;
  toolbar.hidden = false;
  mount.innerHTML = "";
  btn.textContent = "Fertig";
  state.editing = true;
  setSaveState("saved", "Bereit");

  try {
    const {
      Editor, rootCtx, defaultValueCtx, editorViewCtx, commonmark, listener, listenerCtx,
      toggleMark, wrapIn, setBlockType, wrapInList,
    } = await import(MILKDOWN_URL);

    function run(ed, build) {
      ed.action((ctx) => {
        const view = ctx.get(editorViewCtx);
        const cmd = build(view.state.schema);
        if (cmd) cmd(view.state, view.dispatch, view);
      });
    }

    const cmdMap = {
      bold:         (ed) => run(ed, (s) => toggleMark(s.marks.strong)),
      italic:       (ed) => run(ed, (s) => toggleMark(s.marks.emphasis)),
      inlinecode:   (ed) => run(ed, (s) => toggleMark(s.marks.inlineCode)),
      blockquote:   (ed) => run(ed, (s) => wrapIn(s.nodes.blockquote)),
      bulletlist:   (ed) => run(ed, (s) => wrapInList(s.nodes.bullet_list)),
      orderedlist:  (ed) => run(ed, (s) => wrapInList(s.nodes.ordered_list)),
      h1:           (ed) => run(ed, (s) => setBlockType(s.nodes.heading, { level: 1 })),
      h2:           (ed) => run(ed, (s) => setBlockType(s.nodes.heading, { level: 2 })),
      h3:           (ed) => run(ed, (s) => setBlockType(s.nodes.heading, { level: 3 })),
      paragraph:    (ed) => run(ed, (s) => setBlockType(s.nodes.paragraph)),
      codeblock:    (ed) => run(ed, (s) => setBlockType(s.nodes.code_block)),
      hr:           (ed) => run(ed, (s) => (state, dispatch) => dispatch(state.tr.replaceSelectionWith(s.nodes.hr.create()))),
      hardbreak:    (ed) => run(ed, (s) => (state, dispatch) => dispatch(state.tr.replaceSelectionWith(s.nodes.hardbreak.create()))),
    };

    toolbar.querySelectorAll("button[data-cmd]").forEach((btn) => {
      btn.addEventListener("mousedown", (e) => {
        e.preventDefault();
        const fn = cmdMap[btn.dataset.cmd];
        if (fn) fn(editor);
      });
    });

    const editor = await Editor.make()
      .config((ctx) => {
        ctx.set(rootCtx, mount);
        ctx.set(defaultValueCtx, raw);
      })
      .use(commonmark)
      .use(listener)
      .config((ctx) => {
        const l = ctx.get(listenerCtx);
        l.markdownUpdated((_ctx, markdown, prevMarkdown) => {
          if (markdown !== prevMarkdown) {
            scheduleSave(markdown);
          }
        });
      })
      .create();

    state.milkdownEditor = editor;
  } catch (err) {
    console.error("Milkdown konnte nicht geladen werden:", err);
    mount.innerHTML =
      '<p class="empty-state">Der WYSIWYG-Editor konnte nicht geladen werden (vermutlich keine Internetverbindung fuer den einmaligen CDN-Abruf von Milkdown). ' +
      "Bitte Internetzugang pruefen und erneut versuchen.</p>";
  }
}

async function exitEditMode() {
  const contentEl = el("content");
  const mount = el("editor-mount");
  const toolbar = el("editor-toolbar");
  const btn = el("btn-edit");

  if (state.milkdownEditor && typeof state.milkdownEditor.destroy === "function") {
    await state.milkdownEditor.destroy();
  }
  state.milkdownEditor = null;
  mount.hidden = true;
  mount.innerHTML = "";
  toolbar.hidden = true;
  btn.textContent = "Bearbeiten";
  state.editing = false;

  const res = await fetch("/page/" + state.path);
  const html = await res.text();
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  const newContent = doc.getElementById("content");
  if (newContent) {
    contentEl.innerHTML = newContent.innerHTML;
    contentEl.dataset.exists = "true";
  }
  contentEl.hidden = false;
}

function scheduleSave(markdown) {
  setSaveState("editing", "Ungespeicherte Aenderungen");
  if (state.saveTimer) clearTimeout(state.saveTimer);
  state.saveTimer = setTimeout(() => save(markdown), state.saveDebounceMs);
}

async function save(markdown) {
  setSaveState("saving", "Speichert...");
  try {
    state.ignoreNextReloadFor = state.path;
    const res = await fetch("/api/raw/" + state.path, {
      method: "PUT",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: markdown,
    });
    if (!res.ok) throw new Error("HTTP " + res.status);
    const now = new Date();
    setSaveState("saved", "Gespeichert " + now.toLocaleTimeString());
    state.exists = true;
  } catch (err) {
    console.error("Speichern fehlgeschlagen:", err);
    setSaveState("error", "Fehler beim Speichern");
  }
}

function setSaveState(stateName, text) {
  const indicator = el("save-indicator");
  indicator.dataset.state = stateName;
  indicator.textContent = text;
}

function connectEvents() {
  const src = new EventSource("/api/events");
  src.onmessage = (evt) => {
    let data;
    try {
      data = JSON.parse(evt.data);
    } catch {
      return;
    }
    if (data.path !== state.path) return;

    if (state.ignoreNextReloadFor === data.path) {
      state.ignoreNextReloadFor = null;
      return;
    }

    if (state.editing) {
      showReloadBanner();
    } else {
      window.location.reload();
    }
  };
  src.onerror = () => {};
}

function showReloadBanner() {
  if (document.querySelector(".reload-banner")) return;
  const banner = document.createElement("div");
  banner.className = "reload-banner";
  banner.innerHTML =
    "Diese Seite wurde extern geaendert. " +
    '<button id="reload-now">Neu laden</button> <button id="reload-dismiss">Ignorieren</button>';
  document.body.appendChild(banner);
  banner.querySelector("#reload-now").addEventListener("click", () => window.location.reload());
  banner.querySelector("#reload-dismiss").addEventListener("click", () => banner.remove());
}

function debounce(fn, ms) {
  let t;
  return (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
}

document.addEventListener("DOMContentLoaded", init);
