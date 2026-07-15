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
