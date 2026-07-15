package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"netfin/internal/config"
	"netfin/internal/filesync"
	"netfin/internal/restore"
	"netfin/internal/sqliterepl"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("sidecar exited with error", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command; expected run or restore")
	}
	command := args[0]
	switch command {
	case "run", "restore":
		args = args[1:]
	default:
		return fmt.Errorf("unknown command %q; expected run or restore", command)
	}

	configPath, err := config.ParseConfigFlag(args, command)
	if err != nil {
		return err
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	switch command {
	case "run":
		return runSidecar(cfg, logger)
	case "restore":
		return runRestore(cfg, logger)
	default:
		return fmt.Errorf("unhandled command: %s", command)
	}
}

func runSidecar(cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fileSyncer, err := filesync.New(filesync.Options{
		Source:            cfg.Netfin.ConfigDir,
		Destination:       cfg.Netfin.FileBackupDir,
		ExcludePatterns:   cfg.Netfin.ExcludePatterns,
		Debounce:          cfg.Netfin.Debounce,
		ReconcileInterval: cfg.Netfin.ReconcileInterval,
		SourceRetry:       cfg.Netfin.SourceRetry,
		DryRun:            cfg.Netfin.DryRun,
		Logger:            logger.With("component", "filesync"),
	})
	if err != nil {
		return err
	}

	sqliteReplicator, err := sqliterepl.New(sqliterepl.Options{
		DBs:         cfg.Litestream.DBs,
		DryRun:      cfg.Netfin.DryRun,
		SourceRetry: cfg.Netfin.SourceRetry,
		Logger:      logger.With("component", "sqliterepl"),
	})
	if err != nil {
		return err
	}

	logger.Info("sidecar starting",
		"config_dir", cfg.Netfin.ConfigDir,
		"file_backup_dir", cfg.Netfin.FileBackupDir,
		"db_count", len(cfg.Litestream.DBs),
		"debounce", cfg.Netfin.Debounce.String(),
		"reconcile_interval", cfg.Netfin.ReconcileInterval.String(),
		"source_retry", cfg.Netfin.SourceRetry.String(),
		"dry_run", cfg.Netfin.DryRun,
	)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		errCh <- fileSyncer.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		errCh <- sqliteReplicator.Run(ctx)
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = err
		}
		cancel()
	}
	wg.Wait()
	logger.Info("sidecar stopped")
	return runErr
}

func runRestore(cfg config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger.Info("restore starting", "config_dir", cfg.Netfin.ConfigDir, "file_backup_dir", cfg.Netfin.FileBackupDir, "dry_run", cfg.Netfin.DryRun)
	if err := restore.Run(ctx, restore.Options{Config: cfg, DryRun: cfg.Netfin.DryRun, Logger: logger}); err != nil {
		return err
	}
	logger.Info("restore finished")
	return nil
}
