package safeio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func AtomicCopyFile(src, dst string, mode os.FileMode, dryRun bool) error {
	if dryRun {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode.Perm()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(dst))
}

func RemovePath(path string, dryRun bool) error {
	if dryRun {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return fsyncDir(filepath.Dir(path))
}

func fsyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open parent dir for fsync: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("fsync parent dir: %w", err)
	}
	return nil
}
