package safeio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeRelRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "a", "b.txt")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}
	if rel, err := SafeRel(root, inside); err != nil || rel != filepath.Join("a", "b.txt") {
		t.Fatalf("SafeRel inside = %q, %v", rel, err)
	}

	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if _, err := SafeRel(root, outside); err == nil {
		t.Fatal("expected escape error")
	}
}

func TestJoinUnderRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	if _, err := JoinUnder(root, "../escape"); err == nil {
		t.Fatal("expected parent escape error")
	}
	if _, err := JoinUnder(root, filepath.Join("ok", "file")); err != nil {
		t.Fatalf("expected safe path: %v", err)
	}
}

func TestHasSymlinkInPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	has, err := HasSymlinkInPath(root, filepath.Join(link, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected symlink in path")
	}
}
