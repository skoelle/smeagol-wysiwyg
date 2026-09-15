// Package watcher watches the vault directory for external filesystem
// changes and broadcasts change events to interested subscribers (SSE
// connections in internal/server), so the browser can live-reload.
//
// See SPEC.md section 5.2. The watcher only detects changes; it never
// reads file content itself.
package watcher

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	Path string
	Op   string
}

type Watcher struct {
	root string
	fsw  *fsnotify.Watcher
	mu   sync.Mutex
	subs map[chan Event]struct{}
	done chan struct{}
}

func New(root string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		root: root,
		fsw:  fsw,
		subs: make(map[chan Event]struct{}),
		done: make(chan struct{}),
	}
	if err := w.addRecursive(root); err != nil {
		fsw.Close()
		return nil, err
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			if err := w.fsw.Add(path); err != nil {
				log.Printf("watcher: failed to watch %s: %v", path, err)
			}
		}
		return nil
	})
}

func (w *Watcher) loop() {
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleRaw(ev)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher: error: %v", err)
		case <-w.done:
			return
		}
	}
}

func (w *Watcher) handleRaw(ev fsnotify.Event) {
	if ev.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			if err := w.addRecursive(ev.Name); err != nil {
				log.Printf("watcher: failed to watch new dir %s: %v", ev.Name, err)
			}
		}
	}

	rel, err := filepath.Rel(w.root, ev.Name)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)

	var op string
	switch {
	case ev.Op&fsnotify.Write == fsnotify.Write:
		op = "write"
	case ev.Op&fsnotify.Create == fsnotify.Create:
		op = "create"
	case ev.Op&fsnotify.Remove == fsnotify.Remove:
		op = "remove"
	case ev.Op&fsnotify.Rename == fsnotify.Rename:
		op = "rename"
	default:
		return
	}

	w.broadcast(Event{Path: rel, Op: op})
}

func (w *Watcher) broadcast(e Event) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for ch := range w.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (w *Watcher) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 16)
	w.mu.Lock()
	w.subs[ch] = struct{}{}
	w.mu.Unlock()
	return ch, func() {
		w.mu.Lock()
		delete(w.subs, ch)
		w.mu.Unlock()
		close(ch)
	}
}

func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}
