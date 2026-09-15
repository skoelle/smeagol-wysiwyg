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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrOutsideVault = errors.New("vault: path escapes vault root")

type Vault struct {
	Root string
}

func New(root string) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("vault: resolving root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("vault: stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("vault: root %q is not a directory", abs)
	}
	return &Vault{Root: abs}, nil
}

func (v *Vault) Resolve(reqPath string) (string, error) {
	full := filepath.Clean(filepath.Join(v.Root, reqPath))

	rootWithSep := v.Root
	if !strings.HasSuffix(rootWithSep, string(os.PathSeparator)) {
		rootWithSep += string(os.PathSeparator)
	}
	if full != v.Root && !strings.HasPrefix(full, rootWithSep) {
		return "", ErrOutsideVault
	}
	return full, nil
}

func (v *Vault) ReadFile(reqPath string) ([]byte, error) {
	full, err := v.Resolve(reqPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

func (v *Vault) WriteFileAtomic(reqPath string, content []byte) error {
	full, err := v.Resolve(reqPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("vault: creating parent dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".smeagol-tmp-*")
	if err != nil {
		return fmt.Errorf("vault: creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("vault: writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("vault: syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("vault: closing temp file: %w", err)
	}

	if info, statErr := os.Stat(full); statErr == nil {
		_ = os.Chmod(tmpName, info.Mode())
	}

	if err := os.Rename(tmpName, full); err != nil {
		return fmt.Errorf("vault: renaming into place: %w", err)
	}
	return nil
}

type Node struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	IsDir    bool    `json:"isDir"`
	Children []*Node `json:"children,omitempty"`
}

func (v *Vault) Tree() (*Node, error) {
	root := &Node{Name: filepath.Base(v.Root), Path: "", IsDir: true}
	if err := walkDir(v.Root, v.Root, root); err != nil {
		return nil, err
	}
	return root, nil
}

func walkDir(root, dir string, node *Node) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(dir, name)
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if e.IsDir() {
			child := &Node{Name: name, Path: rel, IsDir: true}
			if err := walkDir(root, full, child); err != nil {
				return err
			}
			node.Children = append(node.Children, child)
			continue
		}
		if strings.EqualFold(filepath.Ext(name), ".md") {
			node.Children = append(node.Children, &Node{Name: name, Path: rel, IsDir: false})
		}
	}
	return nil
}

func (v *Vault) Exists(reqPath string) bool {
	full, err := v.Resolve(reqPath)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && !info.IsDir()
}
