package safeio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "nested", "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := AtomicCopyFile(src, dst, 0o640, false); err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != "hello" {
		t.Fatalf("dst content = %q", buf)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(dst), ".tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}
