package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanRelativePath(t *testing.T) {
	if got, err := cleanRelativePath(`config\app.json`); err != nil || got != "config/app.json" {
		t.Fatalf("got %q %v", got, err)
	}
	for _, p := range []string{"../x", `..\x`, `C:\x`, "/x", ""} {
		if _, err := cleanRelativePath(p); err == nil {
			t.Fatalf("%q should fail", p)
		}
	}
}

func TestCopyDirMergeAndRollbackSupport(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	backup := filepath.Join(root, "backup")
	mustWrite(t, filepath.Join(dst, "keep.txt"), "keep")
	mustWrite(t, filepath.Join(dst, "replace.txt"), "old")
	mustWrite(t, filepath.Join(src, "replace.txt"), "new")
	mustWrite(t, filepath.Join(src, "config", "app.json"), "{}")

	if err := copyDir(dst, backup, nil); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(src, dst, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "keep.txt"), "keep")
	assertFile(t, filepath.Join(dst, "replace.txt"), "new")
	assertFile(t, filepath.Join(dst, "config", "app.json"), "{}")

	if err := os.RemoveAll(dst); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(backup, dst, nil); err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(dst, "replace.txt"), "old")
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Fatalf("%s got %q want %q", path, string(b), want)
	}
}
