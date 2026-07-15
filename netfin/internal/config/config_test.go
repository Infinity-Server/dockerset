package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadYAMLWithNetfinAndLitestream(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "netfin.yml")
	if err := os.WriteFile(path, []byte(`
netfin:
  config-dir: ./config
  file-backup-dir: ./backup/files
  debounce: 3s
  reconcile-interval: 7m
  source-retry: 4s
  exclude:
    - "*.tmp"
dbs:
  - path: ./db/library.db
    monitor-interval: 1s
    checkpoint-interval: 5s
    replica:
      type: file
      path: ./backup/sqlite
      sync-interval: 2s
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Netfin.Debounce != 3*time.Second {
		t.Fatalf("Debounce = %s", cfg.Netfin.Debounce)
	}
	if cfg.Netfin.ReconcileInterval != 7*time.Minute {
		t.Fatalf("ReconcileInterval = %s", cfg.Netfin.ReconcileInterval)
	}
	if cfg.Netfin.SourceRetry != 4*time.Second {
		t.Fatalf("SourceRetry = %s", cfg.Netfin.SourceRetry)
	}
	if len(cfg.Litestream.DBs) != 1 {
		t.Fatalf("db count = %d", len(cfg.Litestream.DBs))
	}
	if cfg.Litestream.DBs[0].MonitorInterval != time.Second {
		t.Fatalf("MonitorInterval = %s", cfg.Litestream.DBs[0].MonitorInterval)
	}
	if !contains(cfg.Netfin.ExcludePatterns, "*.db") || !contains(cfg.Netfin.ExcludePatterns, "*.db-wal") || !contains(cfg.Netfin.ExcludePatterns, "library.db-wal") {
		t.Fatalf("derived excludes missing: %#v", cfg.Netfin.ExcludePatterns)
	}
	if !contains(cfg.Netfin.ExcludePatterns, "*.tmp") {
		t.Fatalf("custom exclude missing: %#v", cfg.Netfin.ExcludePatterns)
	}
}

func TestSQLiteExcludePatternsFromPattern(t *testing.T) {
	patterns := SQLiteExcludePatterns(DBConfig{Pattern: "*.sqlite"})
	if !contains(patterns, "*.sqlite") || !contains(patterns, "*.sqlite-wal") || !contains(patterns, "*.sqlite-shm") {
		t.Fatalf("unexpected patterns: %#v", patterns)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
