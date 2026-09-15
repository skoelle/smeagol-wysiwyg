package vault

import (
\t"os"
\t"path/filepath"
\t"testing"
)

func setupTestVault(t *testing.T) *Vault {
\tt.Helper()
\tdir := t.TempDir()
\tif err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
\t\tt.Fatal(err)
\t}
\tif err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Root\n"), 0o644); err != nil {
\t\tt.Fatal(err)
\t}
\tif err := os.WriteFile(filepath.Join(dir, "sub", "page.md"), []byte("# Sub Page\n"), 0o644); err != nil {
\t\tt.Fatal(err)
\t}
\tv, err := New(dir)
\tif err != nil {
\t\tt.Fatal(err)
\t}
\treturn v
}

func TestResolveWithinRoot(t *testing.T) {
\tv := setupTestVault(t)
\tfull, err := v.Resolve("sub/page.md")
\tif err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
\twant := filepath.Join(v.Root, "sub", "page.md")
\tif full != want {
\t\tt.Fatalf("got %q, want %q", full, want)
\t}
}

func TestResolveBlocksTraversal(t *testing.T) {
\tv := setupTestVault(t)
\tcases := []string{
\t\t"../../etc/passwd",
\t\t"../outside.md",
\t\t"sub/../../outside.md",
\t\t"/../../../etc/passwd",
\t}
\tfor _, c := range cases {
\t\tfull, err := v.Resolve(c)
\t\tif err == nil {
\t\t\tt.Fatalf("path %q resolved to %q without error, expected ErrOutsideVault", c, full)
\t\t}
\t}
}

func TestReadFile(t *testing.T) {
\tv := setupTestVault(t)
\tcontent, err := v.ReadFile("README.md")
\tif err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
\tif string(content) != "# Root\n" {
\t\tt.Fatalf("unexpected content: %q", content)
\t}
}

func TestWriteFileAtomic(t *testing.T) {
\tv := setupTestVault(t)
\tif err := v.WriteFileAtomic("sub/page.md", []byte("# Updated\n")); err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
\tcontent, err := v.ReadFile("sub/page.md")
\tif err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
\tif string(content) != "# Updated\n" {
\t\tt.Fatalf("unexpected content after write: %q", content)
\t}

\tentries, err := os.ReadDir(filepath.Join(v.Root, "sub"))
\tif err != nil {
\t\tt.Fatal(err)
\t}
\tfor _, e := range entries {
\t\tif filepath.Ext(e.Name()) == "" && e.Name() != "page.md" {
\t\t\tt.Fatalf("unexpected leftover file: %s", e.Name())
\t\t}
\t}
}

func TestWriteFileAtomicBlocksTraversal(t *testing.T) {
\tv := setupTestVault(t)
\tif err := v.WriteFileAtomic("../evil.md", []byte("x")); err == nil {
\t\tt.Fatal("expected error writing outside vault root")
\t}
}

func TestTree(t *testing.T) {
\tv := setupTestVault(t)
\ttree, err := v.Tree()
\tif err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
\tif !tree.IsDir {
\t\tt.Fatal("root node should be a directory")
\t}
\tif len(tree.Children) != 2 {
\t\tt.Fatalf("expected 2 children (README.md, sub/), got %d", len(tree.Children))
\t}
}

func TestExists(t *testing.T) {
\tv := setupTestVault(t)
\tif !v.Exists("README.md") {
\t\tt.Fatal("expected README.md to exist")
\t}
\tif v.Exists("does-not-exist.md") {
\t\tt.Fatal("expected does-not-exist.md to not exist")
\t}
}
