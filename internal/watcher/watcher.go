// Package watcher watches the vault directory for external filesystem
// changes and broadcasts change events to interested subscribers (SSE
// connections in internal/server), so the browser can live-reload.
//
// See SPEC.md section 5.2. The watcher only detects changes; it never
// reads file content itself.
package watcher

import (
\t"io/fs"
\t"log"
\t"os"
\t"path/filepath"
\t"strings"
\t"sync"

\t"github.com/fsnotify/fsnotify"
)

type Event struct {
\tPath string
\tOp   string
}

type Watcher struct {
\troot string
\tfsw  *fsnotify.Watcher
\tmu   sync.Mutex
\tsubs map[chan Event]struct{}
\tdone chan struct{}
}

func New(root string) (*Watcher, error) {
\tfsw, err := fsnotify.NewWatcher()
\tif err != nil {
\t\treturn nil, err
\t}
\tw := &Watcher{
\t\troot: root,
\t\tfsw:  fsw,
\t\tsubs: make(map[chan Event]struct{}),
\t\tdone: make(chan struct{}),
\t}
\tif err := w.addRecursive(root); err != nil {
\t\tfsw.Close()
\t\treturn nil, err
\t}
\tgo w.loop()
\treturn w, nil
}

func (w *Watcher) addRecursive(dir string) error {
\treturn filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
\t\tif err != nil {
\t\t\treturn nil
\t\t}
\t\tif d.IsDir() {
\t\t\tif strings.HasPrefix(d.Name(), ".") && path != dir {
\t\t\t\treturn filepath.SkipDir
\t\t\t}
\t\t\tif err := w.fsw.Add(path); err != nil {
\t\t\t\tlog.Printf("watcher: failed to watch %s: %v", path, err)
\t\t\t}
\t\t}
\t\treturn nil
\t})
}

func (w *Watcher) loop() {
\tfor {
\t\tselect {
\t\tcase ev, ok := <-w.fsw.Events:
\t\t\tif !ok {
\t\t\t\treturn
\t\t\t}
\t\t\tw.handleRaw(ev)
\t\tcase err, ok := <-w.fsw.Errors:
\t\t\tif !ok {
\t\t\t\treturn
\t\t\t}
\t\t\tlog.Printf("watcher: error: %v", err)
\t\tcase <-w.done:
\t\t\treturn
\t\t}
\t}
}

func (w *Watcher) handleRaw(ev fsnotify.Event) {
\tif ev.Op&fsnotify.Create == fsnotify.Create {
\t\tif info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
\t\t\tif err := w.addRecursive(ev.Name); err != nil {
\t\t\t\tlog.Printf("watcher: failed to watch new dir %s: %v", ev.Name, err)
\t\t\t}
\t\t}
\t}

\trel, err := filepath.Rel(w.root, ev.Name)
\tif err != nil {
\t\treturn
\t}
\trel = filepath.ToSlash(rel)

\tvar op string
\tswitch {
\tcase ev.Op&fsnotify.Write == fsnotify.Write:
\t\top = "write"
\tcase ev.Op&fsnotify.Create == fsnotify.Create:
\t\top = "create"
\tcase ev.Op&fsnotify.Remove == fsnotify.Remove:
\t\top = "remove"
\tcase ev.Op&fsnotify.Rename == fsnotify.Rename:
\t\top = "rename"
\tdefault:
\t\treturn
\t}

\tw.broadcast(Event{Path: rel, Op: op})
}

func (w *Watcher) broadcast(e Event) {
\tw.mu.Lock()
\tdefer w.mu.Unlock()
\tfor ch := range w.subs {
\t\tselect {
\t\tcase ch <- e:
\t\tdefault:
\t\t}
\t}
}

func (w *Watcher) Subscribe() (<-chan Event, func()) {
\tch := make(chan Event, 16)
\tw.mu.Lock()
\tw.subs[ch] = struct{}{}
\tw.mu.Unlock()
\treturn ch, func() {
\t\tw.mu.Lock()
\t\tdelete(w.subs, ch)
\t\tw.mu.Unlock()
\t\tclose(ch)
\t}
}

func (w *Watcher) Close() error {
\tclose(w.done)
\treturn w.fsw.Close()
}
