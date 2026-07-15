package restore

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"netfin/internal/config"
	"netfin/internal/safeio"

	"github.com/benbjohnson/litestream"
	"github.com/benbjohnson/litestream/file"
)

type Options struct {
	Config config.Config
	DryRun bool
	Logger *slog.Logger
}

func Run(ctx context.Context, opts Options) error {
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	if err := restoreFiles(ctx, opts.Config.Netfin.FileBackupDir, opts.Config.Netfin.ConfigDir, opts.DryRun, log.With("component", "restore-files")); err != nil {
		return err
	}
	for _, db := range opts.Config.Litestream.DBs {
		if db.Dir != "" {
			log.Warn("skipping directory-mode sqlite restore; configure explicit dbs[].path for restore", "dir", db.Dir)
			continue
		}
		if err := restoreDB(ctx, db, opts.DryRun, log.With("component", "restore-sqlite")); err != nil {
			return err
		}
	}
	return nil
}

func restoreFiles(ctx context.Context, backupDir, targetDir string, dryRun bool, log *slog.Logger) error {
	log.Info("restoring ordinary files", "from", backupDir, "to", targetDir, "dry_run", dryRun)
	return filepath.WalkDir(backupDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := safeio.SafeRel(backupDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		dst, err := safeio.JoinUnder(targetDir, rel)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			log.Info("restore directory", "path", rel, "dry_run", dryRun)
			if dryRun {
				return nil
			}
			return os.MkdirAll(dst, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		log.Info("restore file", "path", rel, "bytes", info.Size(), "dry_run", dryRun)
		return safeio.AtomicCopyFile(path, dst, info.Mode(), info.ModTime(), dryRun)
	})
}

func restoreDB(ctx context.Context, db config.DBConfig, dryRun bool, log *slog.Logger) error {
	if db.Path == "" {
		return fmt.Errorf("sqlite restore requires db path")
	}
	replicaPath := db.Replica.Path
	if replicaPath == "" {
		return fmt.Errorf("sqlite restore requires file replica path for %s", db.Path)
	}
	log.Info("restoring sqlite database", "from", replicaPath, "to", db.Path, "dry_run", dryRun)
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(db.Path), 0o755); err != nil {
		return err
	}
	client := file.NewReplicaClient(replicaPath)
	replica := litestream.NewReplicaWithClient(nil, client)
	client.Replica = replica
	opt := litestream.NewRestoreOptions()
	opt.OutputPath = db.Path
	if err := replica.Restore(ctx, opt); err != nil {
		return fmt.Errorf("restore sqlite %s: %w", db.Path, err)
	}
	return nil
}
