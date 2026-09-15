// Package vault implements the filesystem layer for smeagol-wysiwyg.
//
// Design principles (see SPEC.md section 5.1):
//   - No upfront scan or in-memory cache. Every read happens on demand.
//   - All paths are resolved and validated against the vault root before
//     any filesystem operation, to prevent path traversal.
//   - Writes are atomic: write to a temp file in the same directory, then
//     rename over the target file.
package vault

import (
\t"errors"
\t"fmt"
\t"os"
\t"path/filepath"
\t"sort"
\t"strings"
)

var ErrOutsideVault = errors.New("vault: path escapes vault root")

type Vault struct {
\tRoot string
}

func New(root string) (*Vault, error) {
\tabs, err := filepath.Abs(root)
\tif err != nil {
\t\treturn nil, fmt.Errorf("vault: resolving root: %w", err)
\t}
\tinfo, err := os.Stat(abs)
\tif err != nil {
\t\treturn nil, fmt.Errorf("vault: stat root: %w", err)
\t}
\tif !info.IsDir() {
\t\treturn nil, fmt.Errorf("vault: root %q is not a directory", abs)
\t}
\treturn &Vault{Root: abs}, nil
}

func (v *Vault) Resolve(reqPath string) (string, error) {
\tcleaned := filepath.Clean("/" + strings.TrimPrefix(reqPath, "/"))
\tfull := filepath.Join(v.Root, cleaned)

\trootWithSep := v.Root
\tif !strings.HasSuffix(rootWithSep, string(os.PathSeparator)) {
\t\trootWithSep += string(os.PathSeparator)
\t}
\tif full != v.Root && !strings.HasPrefix(full, rootWithSep) {
\t\treturn "", ErrOutsideVault
\t}
\treturn full, nil
}

func (v *Vault) ReadFile(reqPath string) ([]byte, error) {
\tfull, err := v.Resolve(reqPath)
\tif err != nil {
\t\treturn nil, err
\t}
\treturn os.ReadFile(full)
}

func (v *Vault) WriteFileAtomic(reqPath string, content []byte) error {
\tfull, err := v.Resolve(reqPath)
\tif err != nil {
\t\treturn err
\t}
\tdir := filepath.Dir(full)
\tif err := os.MkdirAll(dir, 0o755); err != nil {
\t\treturn fmt.Errorf("vault: creating parent dir: %w", err)
\t}

\ttmp, err := os.CreateTemp(dir, ".smeagol-tmp-*")
\tif err != nil {
\t\treturn fmt.Errorf("vault: creating temp file: %w", err)
\t}
\ttmpName := tmp.Name()
\tdefer func() {
\t\t_ = os.Remove(tmpName)
\t}()

\tif _, err := tmp.Write(content); err != nil {
\t\ttmp.Close()
\t\treturn fmt.Errorf("vault: writing temp file: %w", err)
\t}
\tif err := tmp.Sync(); err != nil {
\t\ttmp.Close()
\t\treturn fmt.Errorf("vault: syncing temp file: %w", err)
\t}
\tif err := tmp.Close(); err != nil {
\t\treturn fmt.Errorf("vault: closing temp file: %w", err)
\t}

\tif info, statErr := os.Stat(full); statErr == nil {
\t\t_ = os.Chmod(tmpName, info.Mode())
\t}

\tif err := os.Rename(tmpName, full); err != nil {
\t\treturn fmt.Errorf("vault: renaming into place: %w", err)
\t}
\treturn nil
}

type Node struct {
\tName     string  `json:"name"`
\tPath     string  `json:"path"`
\tIsDir    bool    `json:"isDir"`
\tChildren []*Node `json:"children,omitempty"`
}

func (v *Vault) Tree() (*Node, error) {
\troot := &Node{Name: filepath.Base(v.Root), Path: "", IsDir: true}
\tif err := walkDir(v.Root, v.Root, root); err != nil {
\t\treturn nil, err
\t}
\treturn root, nil
}

func walkDir(root, dir string, node *Node) error {
\tentries, err := os.ReadDir(dir)
\tif err != nil {
\t\treturn err
\t}
\tsort.Slice(entries, func(i, j int) bool {
\t\treturn entries[i].Name() < entries[j].Name()
\t})
\tfor _, e := range entries {
\t\tname := e.Name()
\t\tif strings.HasPrefix(name, ".") {
\t\t\tcontinue
\t\t}
\t\tfull := filepath.Join(dir, name)
\t\trel, err := filepath.Rel(root, full)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\trel = filepath.ToSlash(rel)

\t\tif e.IsDir() {
\t\t\tchild := &Node{Name: name, Path: rel, IsDir: true}
\t\t\tif err := walkDir(root, full, child); err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t\tnode.Children = append(node.Children, child)
\t\t\tcontinue
\t\t}
\t\tif strings.EqualFold(filepath.Ext(name), ".md") {
\t\t\tnode.Children = append(node.Children, &Node{Name: name, Path: rel, IsDir: false})
\t\t}
\t}
\treturn nil
}

func (v *Vault) Exists(reqPath string) bool {
\tfull, err := v.Resolve(reqPath)
\tif err != nil {
\t\treturn false
\t}
\tinfo, err := os.Stat(full)
\treturn err == nil && !info.IsDir()
}
