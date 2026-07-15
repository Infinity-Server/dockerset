package filesync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"netfin/internal/safeio"

	"github.com/fsnotify/fsnotify"
)

type Syncer struct {
	src         string
	dst         string
	excluder    Excluder
	debounce    time.Duration
	reconcile   time.Duration
	sourceRetry time.Duration
	dryRun      bool
	log         *slog.Logger

	watcher *fsnotify.Watcher
	mu      sync.Mutex
	pending map[string]struct{}
}

type Options struct {
	Source            string
	Destination       string
	ExcludePatterns   []string
	Debounce          time.Duration
	ReconcileInterval time.Duration
	SourceRetry       time.Duration
	DryRun            bool
	Logger            *slog.Logger
}

func New(opts Options) (*Syncer, error) {
	src, err := safeio.CleanRoot(opts.Source)
	if err != nil {
		return nil, err
	}
	dst, err := safeio.CleanRoot(opts.Destination)
	if err != nil {
		return nil, err
	}
	if opts.Debounce <= 0 {
		return nil, fmt.Errorf("debounce must be positive")
	}
	if opts.ReconcileInterval <= 0 {
		return nil, fmt.Errorf("reconcile interval must be positive")
	}
	if opts.SourceRetry <= 0 {
		opts.SourceRetry = 5 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Syncer{
		src:         src,
		dst:         dst,
		excluder:    NewExcluder(opts.ExcludePatterns),
		debounce:    opts.Debounce,
		reconcile:   opts.ReconcileInterval,
		sourceRetry: opts.SourceRetry,
		dryRun:      opts.DryRun,
		log:         opts.Logger,
		pending:     make(map[string]struct{}),
	}, nil
}

func (s *Syncer) Run(ctx context.Context) error {
	if !s.dryRun {
		if err := os.MkdirAll(s.dst, 0o755); err != nil {
			return err
		}
	}

	for {
		if err := waitForDir(ctx, s.src, s.sourceRetry, s.log); err != nil {
			return nil
		}
		if err := s.runWatching(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			s.log.Warn("file watcher stopped; retrying", "error", err)
			if !sleepContext(ctx, s.sourceRetry) {
				return nil
			}
			continue
		}
		return nil
	}
}

func (s *Syncer) runWatching(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher
	defer watcher.Close()

	if err := s.addWatchTree(); err != nil {
		return err
	}
	if err := s.Reconcile(ctx); err != nil {
		s.log.Warn("initial file reconcile failed", "error", err)
	}

	debounce := time.NewTimer(s.debounce)
	if !debounce.Stop() {
		<-debounce.C
	}
	reconcileTicker := time.NewTicker(s.reconcile)
	defer reconcileTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watcher.Errors:
			if err != nil {
				s.log.Warn("fsnotify error", "error", err)
			}
		case event := <-watcher.Events:
			s.handleEvent(event)
			resetTimer(debounce, s.debounce)
		case <-debounce.C:
			s.flushPending(ctx)
		case <-reconcileTicker.C:
			if err := s.Reconcile(ctx); err != nil {
				s.log.Warn("file reconcile failed", "error", err)
			}
		}
	}
}

func waitForDir(ctx context.Context, path string, retry time.Duration, log *slog.Logger) error {
	for {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return nil
		}
		if err == nil {
			return fmt.Errorf("source path is not a directory: %s", path)
		}
		if !os.IsNotExist(err) {
			return err
		}
		log.Info("source directory not found; waiting", "path", path)
		if !sleepContext(ctx, retry) {
			return ctx.Err()
		}
	}
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

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}

func (s *Syncer) handleEvent(event fsnotify.Event) {
	if event.Name == "" {
		return
	}
	rel, err := safeio.SafeRel(s.src, event.Name)
	if err != nil {
		s.log.Warn("ignored event outside source", "path", event.Name, "error", err)
		return
	}
	if rel == "." {
		return
	}
	s.queue(rel)

	if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		if info, err := os.Lstat(event.Name); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if err := s.addWatchTreeAt(event.Name); err != nil {
				s.log.Warn("add watcher for new directory", "path", event.Name, "error", err)
			}
		}
	}
}

func (s *Syncer) queue(rel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[rel] = struct{}{}
}

func (s *Syncer) flushPending(ctx context.Context) {
	s.mu.Lock()
	pending := make([]string, 0, len(s.pending))
	for rel := range s.pending {
		pending = append(pending, rel)
	}
	s.pending = make(map[string]struct{})
	s.mu.Unlock()

	for _, rel := range pending {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := s.SyncRel(rel); err != nil {
			s.log.Warn("sync file event failed", "path", rel, "error", err)
		}
	}
}

func (s *Syncer) Reconcile(ctx context.Context) error {
	s.log.Info("file reconcile started", "source", s.src, "destination", s.dst, "dry_run", s.dryRun)
	seen := map[string]struct{}{}
	if err := filepath.WalkDir(s.src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := safeio.SafeRel(s.src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if s.excluder.Excluded(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		seen[filepath.Clean(rel)] = struct{}{}
		if entry.IsDir() {
			return s.ensureDir(rel, info.Mode())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return s.copyRel(rel, info)
	}); err != nil {
		return err
	}

	if err := s.removeDeleted(seen); err != nil {
		return err
	}
	s.log.Info("file reconcile finished")
	return nil
}

func (s *Syncer) SyncRel(rel string) error {
	if rel == "." || s.excluder.Excluded(rel) {
		return nil
	}
	srcPath, err := safeio.JoinUnder(s.src, rel)
	if err != nil {
		return err
	}
	dstPath, err := safeio.JoinUnder(s.dst, rel)
	if err != nil {
		return err
	}
	hasSymlink, err := safeio.HasSymlinkInPath(s.src, srcPath)
	if err != nil {
		return err
	}
	if hasSymlink {
		s.log.Info("skip symlink path", "path", rel)
		return nil
	}
	info, err := os.Lstat(srcPath)
	if os.IsNotExist(err) {
		s.log.Info("remove backup path", "path", rel, "dry_run", s.dryRun)
		return safeio.RemovePath(dstPath, s.dryRun)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		s.log.Info("remove backup for symlink", "path", rel, "dry_run", s.dryRun)
		return safeio.RemovePath(dstPath, s.dryRun)
	}
	if info.IsDir() {
		return s.ensureDir(rel, info.Mode())
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	return s.copyRel(rel, info)
}

func (s *Syncer) ensureDir(rel string, mode os.FileMode) error {
	dstPath, err := safeio.JoinUnder(s.dst, rel)
	if err != nil {
		return err
	}
	s.log.Info("ensure backup directory", "path", rel, "dry_run", s.dryRun)
	if s.dryRun {
		return nil
	}
	return os.MkdirAll(dstPath, mode.Perm())
}

func (s *Syncer) copyRel(rel string, info os.FileInfo) error {
	srcPath, err := safeio.JoinUnder(s.src, rel)
	if err != nil {
		return err
	}
	dstPath, err := safeio.JoinUnder(s.dst, rel)
	if err != nil {
		return err
	}
	s.log.Info("copy backup file", "path", rel, "bytes", info.Size(), "dry_run", s.dryRun)
	return safeio.AtomicCopyFile(srcPath, dstPath, info.Mode(), s.dryRun)
}

func (s *Syncer) removeDeleted(seen map[string]struct{}) error {
	if s.dryRun {
		return nil
	}
	return filepath.WalkDir(s.dst, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := safeio.SafeRel(s.dst, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.Clean(rel)
		if _, ok := seen[rel]; ok {
			return nil
		}
		s.log.Info("remove deleted backup path", "path", rel)
		if entry.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		return os.Remove(path)
	})
}

func (s *Syncer) addWatchTree() error {
	return s.addWatchTreeAt(s.src)
}

func (s *Syncer) addWatchTreeAt(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := safeio.SafeRel(s.src, path)
		if err != nil {
			return err
		}
		if rel != "." && s.excluder.Excluded(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if err := s.watcher.Add(path); err != nil {
			return err
		}
		s.log.Info("watching directory", "path", path)
		return nil
	})
}
