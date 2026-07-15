package filesync

import "testing"

func TestExcluder(t *testing.T) {
	ex := NewExcluder([]string{
		"*.db",
		"*.db-wal",
		"*.db-shm",
		"*.db-journal",
		"*.sqlite",
		"*.sqlite-wal",
		"*.sqlite-shm",
		"SQLiteBackups/**",
		"cache/**",
		"transcodes/**",
		"log/**",
	})

	tests := map[string]bool{
		"jellyfin.db":                true,
		"data/library.db-wal":        true,
		"data/library.db-shm":        true,
		"data/library.db-journal":    true,
		"data/library.sqlite":        true,
		"data/library.sqlite-wal":    true,
		"data/library.sqlite-shm":    true,
		"SQLiteBackups/backup.db":    true,
		"SQLiteBackups/nested/a.txt": true,
		"cache/images/a.jpg":         true,
		"transcodes/session/file.ts": true,
		"log/jellyfin.log":           true,
		"plugins/config.json":        false,
		"metadata/library.db.backup": false,
		"SQLiteBackupsSibling/a.txt": false,
		"cache-file/a.txt":           false,
	}

	for rel, want := range tests {
		if got := ex.Excluded(rel); got != want {
			t.Fatalf("Excluded(%q) = %v, want %v", rel, got, want)
		}
	}
}
