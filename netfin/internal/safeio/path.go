package safeio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanRoot(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root path required")
	}
	return filepath.Abs(root)
}

func SafeRel(root, path string) (string, error) {
	root, err := CleanRoot(root)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == "" {
		return ".", nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes root: %s", path)
	}
	return filepath.Clean(rel), nil
}

func JoinUnder(root, rel string) (string, error) {
	root, err := CleanRoot(root)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == "" {
		return root, nil
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("relative path escapes root: %s", rel)
	}
	return filepath.Join(root, clean), nil
}

func HasSymlinkInPath(root, path string) (bool, error) {
	rel, err := SafeRel(root, path)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return false, nil
	}
	cur, err := CleanRoot(root)
	if err != nil {
		return false, err
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true, nil
		}
	}
	return false, nil
}
