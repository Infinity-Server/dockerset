package filesync

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"netfin/internal/config"
)

func TestReconcileCopiesOrdinaryFilesAndExcludesDatabases(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	write := func(rel, value string) {
		t.Helper()
		path := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("system.xml", "config")
	write("metadata/poster.txt", "poster")
	write("jellyfin.db", "db")
	write("jellyfin.db-wal", "wal")
	write("cache/ignored.txt", "cache")

	syncer, err := New(Options{
		Source:            src,
		Destination:       dst,
		ExcludePatterns:   config.DefaultExcludePatterns(),
		Debounce:          time.Millisecond,
		ReconcileInterval: time.Minute,
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := syncer.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}

	assertExists(t, filepath.Join(dst, "system.xml"), "config")
	assertExists(t, filepath.Join(dst, "metadata", "poster.txt"), "poster")
	assertMissing(t, filepath.Join(dst, "jellyfin.db"))
	assertMissing(t, filepath.Join(dst, "jellyfin.db-wal"))
	assertMissing(t, filepath.Join(dst, "cache", "ignored.txt"))
}

func TestRunSyncsEventsAndExcludesDatabases(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	syncer, err := New(Options{
		Source:            src,
		Destination:       dst,
		ExcludePatterns:   config.DefaultExcludePatterns(),
		Debounce:          20 * time.Millisecond,
		ReconcileInterval: time.Hour,
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- syncer.Run(ctx)
	}()
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(filepath.Join(src, "system.xml"), []byte("config"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "jellyfin.db"), []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	waitFor(t, time.Second, func() bool {
		buf, err := os.ReadFile(filepath.Join(dst, "system.xml"))
		return err == nil && string(buf) == "config"
	})
	assertMissing(t, filepath.Join(dst, "jellyfin.db"))

	cancel()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestSyncRelSkipsUnchangedFileFromState(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	path := filepath.Join(src, "system.xml")
	if err := os.WriteFile(path, []byte("config"), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.SyncRel("system.xml"); err != nil {
		t.Fatal(err)
	}
	dstPath := filepath.Join(dst, "system.xml")
	if err := os.Chtimes(dstPath, modTime.Add(time.Hour), modTime.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := syncer.SyncRel("system.xml"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modTime.Add(time.Hour)) {
		t.Fatalf("unchanged state was rewritten, dst modtime = %s", info.ModTime())
	}
}

func TestLightReconcileRemovesDeletedPathFromState(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "system.xml", "config")

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(src, "system.xml")); err != nil {
		t.Fatal(err)
	}
	if err := syncer.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, filepath.Join(dst, "system.xml"))
}

func TestSyncTreeCopiesMovedDirectoryContents(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "metadata/images/poster.txt", "poster")
	writeFile(t, src, "metadata/images/jellyfin.db", "db")

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.SyncTree(context.Background(), "metadata"); err != nil {
		t.Fatal(err)
	}

	assertExists(t, filepath.Join(dst, "metadata", "images", "poster.txt"), "poster")
	assertMissing(t, filepath.Join(dst, "metadata", "images", "jellyfin.db"))
}

func TestSyncRelDirectoryDeleteForgetsChildState(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "metadata/poster.txt", "poster")

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.SyncTree(context.Background(), "metadata"); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(src, "metadata")); err != nil {
		t.Fatal(err)
	}
	if err := syncer.SyncRel("metadata"); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, filepath.Join(dst, "metadata", "poster.txt"))

	writeFile(t, src, "metadata/poster.txt", "new")
	if err := syncer.SyncTree(context.Background(), "metadata"); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(dst, "metadata", "poster.txt"), "new")
}

func TestSyncRelRemovesBackupWhenFileBecomesSymlink(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "system.xml", "config")

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.SyncRel("system.xml"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(src, "system.xml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", filepath.Join(src, "system.xml")); err != nil {
		t.Fatal(err)
	}
	if err := syncer.SyncRel("system.xml"); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, filepath.Join(dst, "system.xml"))
}

func TestSyncRelExcludedPathDoesNotTouchUnknownBackup(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, src, "jellyfin.db", "db")
	writeFile(t, dst, "jellyfin.db", "stale")

	syncer := newTestSyncer(t, src, dst)
	if err := syncer.SyncRel("jellyfin.db"); err != nil {
		t.Fatal(err)
	}
	assertExists(t, filepath.Join(dst, "jellyfin.db"), "stale")
}

func newTestSyncer(t *testing.T, src, dst string) *Syncer {
	t.Helper()
	syncer, err := New(Options{
		Source:            src,
		Destination:       dst,
		ExcludePatterns:   config.DefaultExcludePatterns(),
		Debounce:          time.Millisecond,
		ReconcileInterval: time.Minute,
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	})
	if err != nil {
		t.Fatal(err)
	}
	return syncer
}

func writeFile(t *testing.T, root, rel, value string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path, want string) {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != want {
		t.Fatalf("%s = %q, want %q", path, buf, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, err=%v", path, err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
