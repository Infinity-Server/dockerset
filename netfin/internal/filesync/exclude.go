package filesync

import (
	"path/filepath"
	"strings"
)

type Excluder struct {
	patterns []string
}

func NewExcluder(patterns []string) Excluder {
	cp := append([]string(nil), patterns...)
	for i := range cp {
		cp[i] = filepath.ToSlash(strings.TrimSpace(cp[i]))
	}
	return Excluder{patterns: cp}
}

func (e Excluder) Excluded(rel string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." {
		return false
	}
	base := pathBase(rel)
	for _, pattern := range e.patterns {
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if !strings.Contains(pattern, "/") {
			if ok, _ := filepath.Match(pattern, base); ok {
				return true
			}
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
	}
	return false
}

func pathBase(path string) string {
	i := strings.LastIndexByte(path, '/')
	if i < 0 {
		return path
	}
	return path[i+1:]
}
