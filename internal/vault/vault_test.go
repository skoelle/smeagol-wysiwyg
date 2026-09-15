package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestVault(t *testing.T) *Vault {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "page.md"), []byte("# Sub Page\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestResolveWithinRoot(t *testing.T) {
	v := setupTestVault(t)
	full, err := v.Resolve("sub/page.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(v.Root, "sub", "page.md")
	if full != want {
		t.Fatalf("got %q, want %q", full, want)
	}
}

func TestResolveBlocksTraversal(t *testing.T) {
	v := setupTestVault(t)
	cases := []string{
		"../../etc/passwd",
		"../outside.md",
		"sub/../../outside.md",
		"/../../../etc/passwd",
	}
	for _, c := range cases {
		full, err := v.Resolve(c)
		if err == nil {
			t.Fatalf("path %q resolved to %q without error, expected ErrOutsideVault", c, full)
		}
	}
}

func TestReadFile(t *testing.T) {
	v := setupTestVault(t)
	content, err := v.ReadFile("README.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "# Root\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	v := setupTestVault(t)
	if err := v.WriteFileAtomic("sub/page.md", []byte("# Updated\n")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, err := v.ReadFile("sub/page.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "# Updated\n" {
		t.Fatalf("unexpected content after write: %q", content)
	}

	entries, err := os.ReadDir(filepath.Join(v.Root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == "" && e.Name() != "page.md" {
			t.Fatalf("unexpected leftover file: %s", e.Name())
		}
	}
}

func TestWriteFileAtomicBlocksTraversal(t *testing.T) {
	v := setupTestVault(t)
	if err := v.WriteFileAtomic("../evil.md", []byte("x")); err == nil {
		t.Fatal("expected error writing outside vault root")
	}
}

func TestTree(t *testing.T) {
	v := setupTestVault(t)
	tree, err := v.Tree()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tree.IsDir {
		t.Fatal("root node should be a directory")
	}
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children (README.md, sub/), got %d", len(tree.Children))
	}
}

func TestExists(t *testing.T) {
	v := setupTestVault(t)
	if !v.Exists("README.md") {
		t.Fatal("expected README.md to exist")
	}
	if v.Exists("does-not-exist.md") {
		t.Fatal("expected does-not-exist.md to not exist")
	}
}
