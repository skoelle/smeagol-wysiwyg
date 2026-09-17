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
  milkdownModules: null,
  ignoreNextReloadFor: null,
  scrollSpyObserver: null,
  lastMarkdown: "",
};

function el(id) { return document.getElementById(id); }

function updateTopbarHeight() {
  const topbar = el("topbar");
  if (topbar) {
    document.documentElement.style.setProperty("--topbar-height", topbar.offsetHeight + "px");
  }
}

function isMobile() {
  return window.innerWidth <= 640;
}

function init() {
  const contentEl = el("content");
  state.path = contentEl.dataset.path || "README.md";
  state.exists = contentEl.dataset.exists === "true";

  el("btn-overview").addEventListener("click", toggleOverview);
  el("btn-toc").addEventListener("click", toggleTOC);
  el("btn-edit").addEventListener("click", toggleEdit);
  el("search-input").addEventListener("input", debounce(onSearchInput, 250));

  updateTopbarHeight();
  window.addEventListener("resize", debounce(updateTopbarHeight, 100));

  connectEvents();
  if (!isMobile()) buildTOC();
  highlightSearchMatch();
  if (new URLSearchParams(location.search).get("edit") === "1") {
    enterEditMode();
    history.replaceState(null, "", location.pathname);
  }
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
    const createBtn = document.createElement("button");
    createBtn.className = "btn tree-create";
    createBtn.textContent = "+ Neue Seite";
    createBtn.addEventListener("click", createNewPage);
    panel.appendChild(createBtn);
  } catch (e) {
    panel.innerHTML = "Fehler beim Laden der Übersicht.";
  }
}

async function createNewPage() {
  const input = prompt("Pfad (z.B. notes/meeting):");
  if (!input || !input.trim()) return;
  const clean = input.trim().replace(/\.md$/, "").replace(/[\/\\]/g, (m) => m === "\\" ? "/" : m);
  if (clean.includes("..")) return;
  const path = clean + ".md";
  try {
    const res = await fetch("/api/raw/" + path, {
      method: "PUT",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: "",
    });
    if (!res.ok) throw new Error("HTTP " + res.status);
    state.path = path;
    state.exists = true;
    window.location = "/page/" + path + "?edit=1";
  } catch (e) {
    alert("Fehler beim Anlegen: " + e.message);
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
    renderSearchResults(panel, results, q);
  } catch (e) {
    panel.innerHTML = "Fehler bei der Suche.";
  }
}

function renderSearchResults(panel, results, query) {
  panel.innerHTML = "";
  if (!results || results.length === 0) {
    panel.innerHTML = '<p class="empty-state">Keine Treffer.</p>';
    return;
  }
  const pathCounts = {};
  results.forEach((r) => {
    pathCounts[r.path] = (pathCounts[r.path] || 0) + 1;
  });
  const pathIdx = {};
  results.forEach((r) => {
    const idx = pathIdx[r.path] || 0;
    pathIdx[r.path] = idx + 1;
    const div = document.createElement("div");
    div.className = "search-result";
    const a = document.createElement("a");
    a.href = "/page/" + r.path + "?q=" + encodeURIComponent(query) + "&idx=" + idx;
    a.textContent = r.title || r.path;
    const snippet = document.createElement("div");
    snippet.className = "snippet";
    snippet.textContent = r.snippet;
    div.appendChild(a);
    div.appendChild(snippet);
    panel.appendChild(div);
  });
}

function highlightSearchMatch() {
  const params = new URLSearchParams(location.search);
  const q = params.get("q");
  if (!q) return;
  const targetIdx = parseInt(params.get("idx"), 10);
  const contentEl = el("content");
  const regex = new RegExp("(" + q.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + ")", "gi");
  const textNodes = [];
  const walker = document.createTreeWalker(contentEl, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) textNodes.push(walker.currentNode);
  const matches = [];
  textNodes.forEach((node) => {
    if (!regex.test(node.nodeValue)) return;
    regex.lastIndex = 0;
    const span = document.createElement("span");
    span.innerHTML = node.nodeValue.replace(regex, '<mark class="search-highlight">$1</mark>');
    node.parentNode.replaceChild(span, node);
    matches.push(...span.querySelectorAll(".search-highlight"));
  });
  if (!matches.length) return;
  const idx = (!isNaN(targetIdx) && targetIdx < matches.length) ? targetIdx : 0;
  matches[idx].scrollIntoView({ behavior: "smooth", block: "center" });
}

function toggleTOC() {
  const panel = el("toc-panel");
  if (!panel.hidden) {
    panel.hidden = true;
    return;
  }
  panel.hidden = false;
  buildTOC();
}

function buildTOC() {
  const panel = el("toc-panel");
  panel.innerHTML = "";

  let headings = [];

  if (state.editing && state.milkdownEditor && state.milkdownModules) {
    const { editorViewCtx } = state.milkdownModules;
    state.milkdownEditor.action((ctx) => {
      const view = ctx.get(editorViewCtx);
      view.state.doc.descendants((node) => {
        if (node.type.name === "heading") {
          const text = node.textContent;
          headings.push({
            level: node.attrs.level,
            text: text,
            id: text.toLowerCase().replace(/[^\w]+/g, "-"),
          });
        }
      });
    });
  } else {
    const contentEl = el("content");
    contentEl.querySelectorAll("h1, h2, h3").forEach((h) => {
      const id = h.textContent.toLowerCase().replace(/[^\w]+/g, "-");
      h.id = id;
      headings.push({
        level: parseInt(h.tagName[1]),
        text: h.textContent,
        id: id,
      });
    });
  }

  if (headings.length === 0) {
    panel.innerHTML = '<p class="empty-state">Keine Ueberschriften gefunden.</p>';
    return;
  }

  headings.forEach((h) => {
    const a = document.createElement("a");
    a.className = "toc-item";
    a.dataset.level = h.level;
    a.textContent = h.text;
    a.addEventListener("click", (e) => {
      e.preventDefault();
      if (state.editing) {
        const mount = el("editor-mount");
        const heading = mount.querySelector("h" + h.level);
        if (heading) heading.scrollIntoView({ behavior: "smooth", block: "start" });
      } else {
        const target = document.getElementById(h.id);
        if (target) target.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
    panel.appendChild(a);
  });

  initScrollSpy();
}

function initScrollSpy() {
  if (state.scrollSpyObserver) state.scrollSpyObserver.disconnect();

  const items = el("toc-panel").querySelectorAll(".toc-item");
  if (items.length === 0) return;

  const targets = [];
  items.forEach((item, i) => {
    let target = null;
    if (state.editing) {
      const mount = el("editor-mount");
      const headings = mount.querySelectorAll("h1, h2, h3");
      if (headings[i]) target = headings[i];
    } else {
      const id = item.textContent.toLowerCase().replace(/[^\w]+/g, "-");
      target = document.getElementById(id);
    }
    if (target) targets.push({ el: target, item: item });
  });

  state.scrollSpyObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        const match = targets.find((t) => t.el === entry.target);
        if (match) {
          if (entry.isIntersecting) {
            items.forEach((i) => i.classList.remove("active"));
            match.item.classList.add("active");
          }
        }
      });
    },
    { rootMargin: "-80px 0px -70% 0px" }
  );

  targets.forEach((t) => state.scrollSpyObserver.observe(t.el));
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
  setSaveState("saved");

  try {
    const {
      Editor, rootCtx, defaultValueCtx, editorViewCtx, commonmark, listener, listenerCtx,
      history, toggleMark, wrapIn, setBlockType, wrapInList,
    } = await import(MILKDOWN_URL);

    state.milkdownModules = { editorViewCtx, toggleMark, wrapIn, setBlockType, wrapInList };

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
      .use(history)
      .use(listener)
      .config((ctx) => {
        const l = ctx.get(listenerCtx);
        l.markdownUpdated((_ctx, markdown, prevMarkdown) => {
          state.lastMarkdown = markdown;
          if (markdown !== prevMarkdown) {
            scheduleSave(markdown);
            if (!el("toc-panel").hidden && !isMobile()) buildTOC();
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

  const isEmpty = !state.lastMarkdown.trim();

  if (state.milkdownEditor && typeof state.milkdownEditor.destroy === "function") {
    await state.milkdownEditor.destroy();
  }
  state.milkdownEditor = null;
  state.milkdownModules = null;
  mount.hidden = true;
  mount.innerHTML = "";
  toolbar.hidden = true;
  btn.textContent = "Bearbeiten";
  state.editing = false;

  if (isEmpty && state.exists) {
    try {
      await fetch("/api/raw/" + state.path, { method: "DELETE" });
      state.exists = false;
      setSaveState("saved");
    } catch (e) {
      console.error("Delete failed:", e);
      setSaveState("error");
    }
  }

  const res = await fetch("/page/" + state.path);
  const html = await res.text();
  const parser = new DOMParser();
  const doc = parser.parseFromString(html, "text/html");
  const newContent = doc.getElementById("content");
  if (newContent) {
    contentEl.innerHTML = newContent.innerHTML;
    contentEl.dataset.exists = state.exists ? "true" : "false";
  }
  contentEl.hidden = false;

  if (!el("toc-panel").hidden && !isMobile()) buildTOC();
}

function scheduleSave(markdown) {
  setSaveState("editing");
  if (state.saveTimer) clearTimeout(state.saveTimer);
  state.saveTimer = setTimeout(() => save(markdown), state.saveDebounceMs);
}

async function save(markdown) {
  setSaveState("saving");
  try {
    state.ignoreNextReloadFor = state.path;
    const res = await fetch("/api/raw/" + state.path, {
      method: "PUT",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: markdown,
    });
    if (!res.ok) throw new Error("HTTP " + res.status);
    const now = new Date();
    setSaveState("saved");
    state.exists = true;
  } catch (err) {
    console.error("Speichern fehlgeschlagen:", err);
    setSaveState("error");
  }
}

function setSaveState(stateName) {
  const indicator = el("save-indicator");
  indicator.dataset.state = stateName;
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
