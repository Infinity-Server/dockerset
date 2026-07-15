package sqliterepl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"netfin/internal/config"

	"github.com/benbjohnson/litestream"
	"github.com/benbjohnson/litestream/file"
)

type Replicator struct {
	dbs         []config.DBConfig
	dryRun      bool
	sourceRetry time.Duration
	log         *slog.Logger
}

type Options struct {
	DBs         []config.DBConfig
	DryRun      bool
	SourceRetry time.Duration
	Logger      *slog.Logger
}

func New(opts Options) (*Replicator, error) {
	if len(opts.DBs) == 0 {
		return nil, fmt.Errorf("at least one litestream db is required")
	}
	if opts.SourceRetry <= 0 {
		opts.SourceRetry = config.DefaultSourceRetry
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Replicator{
		dbs:         opts.DBs,
		dryRun:      opts.DryRun,
		sourceRetry: opts.SourceRetry,
		log:         opts.Logger,
	}, nil
}

func (r *Replicator) Run(ctx context.Context) error {
	if r.dryRun {
		r.log.Info("sqlite replication disabled by dry-run")
		<-ctx.Done()
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(r.dbs))
	for _, db := range r.dbs {
		db := db
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- r.runDBConfig(ctx, db)
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		<-done
		return nil
	case <-done:
		close(errCh)
		for err := range errCh {
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
		}
		return nil
	}
}

func (r *Replicator) runDBConfig(ctx context.Context, db config.DBConfig) error {
	known := map[string]struct{}{}
	ticker := time.NewTicker(r.sourceRetry)
	defer ticker.Stop()
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		paths, err := discoverDBPaths(db)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				r.log.Info("sqlite source not found; waiting", "path", db.Path, "dir", db.Dir)
			} else {
				r.log.Warn("sqlite discovery failed; retrying", "error", err)
			}
		}
		for _, path := range paths {
			if _, ok := known[path]; ok {
				continue
			}
			known[path] = struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.replicatePath(ctx, db, path)
			}()
		}
		if db.Dir == "" || !db.Watch {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if len(known) > 0 {
					continue
				}
			}
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Replicator) replicatePath(ctx context.Context, db config.DBConfig, dbPath string) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.runOnce(ctx, db, dbPath); err != nil {
			if ctx.Err() == nil {
				r.log.Warn("sqlite replication stopped; retrying", "db", dbPath, "error", err)
			}
			if !sleepContext(ctx, r.sourceRetry) {
				return
			}
			continue
		}
		return
	}
}

func (r *Replicator) runOnce(ctx context.Context, cfg config.DBConfig, dbPath string) error {
	if info, err := os.Stat(dbPath); err != nil {
		return err
	} else if !info.Mode().IsRegular() {
		return fmt.Errorf("sqlite db path is not a regular file: %s", dbPath)
	}

	db := litestream.NewDB(dbPath)
	if cfg.MetaPath != "" {
		db.SetMetaPath(cfg.MetaPath)
	} else if cfg.MetaDir != "" {
		rel := filepath.Base(dbPath)
		if cfg.Dir != "" {
			if v, err := filepath.Rel(cfg.Dir, dbPath); err == nil {
				rel = v
			}
		}
		db.SetMetaPath(filepath.Join(cfg.MetaDir, rel+litestream.MetaDirSuffix))
	} else {
		db.SetMetaPath(filepath.Join(cfg.Replica.Path, ".metadata", filepath.Base(dbPath)+litestream.MetaDirSuffix))
	}
	if cfg.MonitorInterval > 0 {
		db.MonitorInterval = cfg.MonitorInterval
	}
	if cfg.CheckpointInterval > 0 {
		db.CheckpointInterval = cfg.CheckpointInterval
	}
	if cfg.BusyTimeout > 0 {
		db.BusyTimeout = cfg.BusyTimeout
	}

	replicaPath := cfg.Replica.Path
	if cfg.Dir != "" {
		rel, err := filepath.Rel(cfg.Dir, dbPath)
		if err != nil {
			return err
		}
		replicaPath = filepath.Join(replicaPath, rel)
	}
	if err := os.MkdirAll(replicaPath, 0o755); err != nil {
		return err
	}
	client := file.NewReplicaClient(replicaPath)
	replica := litestream.NewReplicaWithClient(db, client)
	if cfg.Replica.SyncInterval > 0 {
		replica.SyncInterval = cfg.Replica.SyncInterval
	}
	db.Replica = replica
	client.Replica = replica

	store := litestream.NewStore([]*litestream.DB{db}, litestream.DefaultCompactionLevels)
	r.log.Info("starting sqlite replication", "db", dbPath, "replica", replicaPath)
	if err := store.Open(ctx); err != nil {
		return fmt.Errorf("open litestream store: %w", err)
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.Close(shutdownCtx); err != nil {
		if isIgnorableCloseError(err) {
			r.log.Debug("ignored litestream shutdown sync error", "error", err)
			return nil
		}
		return fmt.Errorf("close litestream store: %w", err)
	}
	r.log.Info("sqlite replication stopped", "db", dbPath)
	return nil
}

func discoverDBPaths(db config.DBConfig) ([]string, error) {
	if db.Dir == "" {
		if _, err := os.Stat(db.Path); err != nil {
			return nil, err
		}
		return []string{db.Path}, nil
	}
	var paths []string
	err := filepath.WalkDir(db.Dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if !db.Recursive && path != db.Dir {
				return filepath.SkipDir
			}
			return nil
		}
		matched, err := filepath.Match(db.Pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && isSQLiteDatabase(path) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func isSQLiteDatabase(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 16)
	if _, err := file.Read(header); err != nil {
		return false
	}
	return string(header) == "SQLite format 3\x00"
}

func isIgnorableCloseError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "ensure wal exists") && strings.Contains(msg, "no such table: _litestream_seq")
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
