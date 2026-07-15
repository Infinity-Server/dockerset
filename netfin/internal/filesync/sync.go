package filesync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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
	pending map[string]pendingSync
	state   map[string]pathState

	reconciled bool
}

type pendingSync struct {
	tree bool
}

type pendingEntry struct {
	rel  string
	tree bool
}

type pathKind uint8

const (
	pathKindDir pathKind = iota + 1
	pathKindFile
)

type pathState struct {
	kind    pathKind
	mode    os.FileMode
	size    int64
	modTime time.Time
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
		pending:     make(map[string]pendingSync),
		state:       make(map[string]pathState),
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

	if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		if info, err := os.Lstat(event.Name); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if err := s.addWatchTreeAt(event.Name); err != nil {
				s.log.Warn("add watcher for new directory", "path", event.Name, "error", err)
			}
			s.queue(rel, true)
			return
		}
	}
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		if s.stateKind(rel) == pathKindDir {
			s.queue(rel, true)
			return
		}
	}
	s.queue(rel, false)
}

func (s *Syncer) queue(rel string, tree bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rel = filepath.Clean(rel)
	pending := s.pending[rel]
	pending.tree = pending.tree || tree
	s.pending[rel] = pending
}

func (s *Syncer) flushPending(ctx context.Context) {
	s.mu.Lock()
	pending := make([]pendingEntry, 0, len(s.pending))
	for rel, item := range s.pending {
		pending = append(pending, pendingEntry{rel: rel, tree: item.tree})
	}
	s.pending = make(map[string]pendingSync)
	s.mu.Unlock()

	for _, item := range collapsePending(pending) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if item.tree {
			if err := s.SyncTree(ctx, item.rel); err != nil {
				s.log.Warn("sync directory tree event failed", "path", item.rel, "error", err)
			}
			continue
		}
		if err := s.SyncRel(item.rel); err != nil {
			s.log.Warn("sync file event failed", "path", item.rel, "error", err)
		}
	}
}

func (s *Syncer) Reconcile(ctx context.Context) error {
	deep := s.needsDeepReconcile()
	s.log.Info("file reconcile started", "source", s.src, "destination", s.dst, "deep", deep, "dry_run", s.dryRun)
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
		rel = filepath.Clean(rel)
		seen[rel] = struct{}{}
		if entry.IsDir() {
			return s.syncDir(rel, info)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return s.copyRel(rel, info)
	}); err != nil {
		return err
	}

	if deep {
		if err := s.removeDeletedByWalkingDestination(seen); err != nil {
			return err
		}
	} else if err := s.removeDeletedFromState(seen); err != nil {
		return err
	}
	s.markReconciled()
	s.log.Info("file reconcile finished")
	return nil
}

func (s *Syncer) SyncRel(rel string) error {
	if rel == "." {
		return nil
	}
	rel = filepath.Clean(rel)
	srcPath, err := safeio.JoinUnder(s.src, rel)
	if err != nil {
		return err
	}
	dstPath, err := safeio.JoinUnder(s.dst, rel)
	if err != nil {
		return err
	}
	if s.excluder.Excluded(rel) {
		return s.removeKnownBackupPath(rel, dstPath)
	}
	hasSymlink, err := safeio.HasSymlinkInPath(s.src, srcPath)
	if err != nil {
		return err
	}
	if hasSymlink {
		s.log.Info("skip symlink path", "path", rel)
		return s.removeKnownBackupPath(rel, dstPath)
	}
	info, err := os.Lstat(srcPath)
	if os.IsNotExist(err) {
		return s.removeKnownBackupPath(rel, dstPath)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		s.log.Info("remove backup for symlink", "path", rel, "dry_run", s.dryRun)
		return s.removeKnownBackupPath(rel, dstPath)
	}
	if info.IsDir() {
		return s.syncDir(rel, info)
	}
	if !info.Mode().IsRegular() {
		return s.removeKnownBackupPath(rel, dstPath)
	}
	return s.copyRel(rel, info)
}

func (s *Syncer) SyncTree(ctx context.Context, rel string) error {
	if rel == "." {
		return nil
	}
	rel = filepath.Clean(rel)
	srcPath, err := safeio.JoinUnder(s.src, rel)
	if err != nil {
		return err
	}
	dstPath, err := safeio.JoinUnder(s.dst, rel)
	if err != nil {
		return err
	}
	if s.excluder.Excluded(rel) {
		return s.removeKnownBackupPath(rel, dstPath)
	}
	hasSymlink, err := safeio.HasSymlinkInPath(s.src, srcPath)
	if err != nil {
		return err
	}
	if hasSymlink {
		s.log.Info("skip symlink path", "path", rel)
		return s.removeKnownBackupPath(rel, dstPath)
	}
	info, err := os.Lstat(srcPath)
	if os.IsNotExist(err) {
		return s.removeKnownBackupPath(rel, dstPath)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return s.SyncRel(rel)
	}

	seen := map[string]struct{}{}
	if err := filepath.WalkDir(srcPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		childRel, err := safeio.SafeRel(s.src, path)
		if err != nil {
			return err
		}
		if childRel == "." {
			return nil
		}
		if s.excluder.Excluded(childRel) {
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
		childRel = filepath.Clean(childRel)
		seen[childRel] = struct{}{}
		if entry.IsDir() {
			return s.syncDir(childRel, info)
		}
		if !info.Mode().IsRegular() {
			childDst, err := safeio.JoinUnder(s.dst, childRel)
			if err != nil {
				return err
			}
			return s.removeBackupPath(childRel, childDst)
		}
		return s.copyRel(childRel, info)
	}); err != nil {
		return err
	}
	return s.removeDeletedFromStateUnder(rel, seen)
}

func (s *Syncer) syncDir(rel string, info os.FileInfo) error {
	state := pathStateFromInfo(pathKindDir, info)
	if s.stateMatches(rel, state) {
		return nil
	}
	if err := s.ensureDir(rel, info.Mode()); err != nil {
		return err
	}
	s.rememberState(rel, state)
	return nil
}

func (s *Syncer) removeBackupPath(rel, dstPath string) error {
	s.log.Info("remove backup path", "path", rel, "dry_run", s.dryRun)
	if err := safeio.RemovePath(dstPath, s.dryRun); err != nil {
		return err
	}
	s.forgetStateTree(rel)
	return nil
}

func (s *Syncer) removeKnownBackupPath(rel, dstPath string) error {
	if !s.hasStateTree(rel) {
		return nil
	}
	return s.removeBackupPath(rel, dstPath)
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
	state := pathStateFromInfo(pathKindFile, info)
	if s.stateMatches(rel, state) {
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
	if fileMetadataMatches(dstPath, info) {
		s.log.Debug("skip unchanged backup file", "path", rel)
		s.rememberState(rel, state)
		return nil
	}
	s.log.Info("copy backup file", "path", rel, "bytes", info.Size(), "dry_run", s.dryRun)
	if err := safeio.AtomicCopyFile(srcPath, dstPath, info.Mode(), info.ModTime(), s.dryRun); err != nil {
		return err
	}
	s.rememberState(rel, state)
	return nil
}

func (s *Syncer) removeDeletedByWalkingDestination(seen map[string]struct{}) error {
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
			s.forgetStateTree(rel)
			return filepath.SkipDir
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		s.forgetState(rel)
		return nil
	})
}

func (s *Syncer) removeDeletedFromState(seen map[string]struct{}) error {
	return s.removeDeletedFromStateUnder(".", seen)
}

func (s *Syncer) removeDeletedFromStateUnder(root string, seen map[string]struct{}) error {
	root = filepath.Clean(root)
	for _, rel := range s.statePaths() {
		if root != "." && rel != root && !hasPathPrefix(rel, root+string(os.PathSeparator)) {
			continue
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		if !s.hasState(rel) {
			continue
		}
		dstPath, err := safeio.JoinUnder(s.dst, rel)
		if err != nil {
			return err
		}
		s.log.Info("remove deleted backup path", "path", rel, "dry_run", s.dryRun)
		if err := safeio.RemovePath(dstPath, s.dryRun); err != nil {
			return err
		}
		s.forgetStateTree(rel)
	}
	return nil
}

func fileMetadataMatches(path string, src os.FileInfo) bool {
	dst, err := os.Stat(path)
	if err != nil {
		return false
	}
	return dst.Mode().Perm() == src.Mode().Perm() &&
		dst.Size() == src.Size() &&
		dst.ModTime().Equal(src.ModTime())
}

func pathStateFromInfo(kind pathKind, info os.FileInfo) pathState {
	state := pathState{
		kind: kind,
		mode: info.Mode().Perm(),
	}
	if kind == pathKindFile {
		state.size = info.Size()
		state.modTime = info.ModTime()
	}
	return state
}

func (s *Syncer) needsDeepReconcile() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.reconciled
}

func (s *Syncer) markReconciled() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconciled = true
}

func (s *Syncer) stateMatches(rel string, state pathState) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[filepath.Clean(rel)] == state
}

func (s *Syncer) stateKind(rel string) pathKind {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state[filepath.Clean(rel)].kind
}

func (s *Syncer) hasState(rel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.state[filepath.Clean(rel)]
	return ok
}

func (s *Syncer) hasStateTree(rel string) bool {
	rel = filepath.Clean(rel)
	prefix := rel + string(os.PathSeparator)
	s.mu.Lock()
	defer s.mu.Unlock()
	for path := range s.state {
		if path == rel || hasPathPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (s *Syncer) rememberState(rel string, state pathState) {
	if s.dryRun {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[filepath.Clean(rel)] = state
}

func (s *Syncer) forgetState(rel string) {
	if s.dryRun {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state, filepath.Clean(rel))
}

func (s *Syncer) forgetStateTree(rel string) {
	if s.dryRun {
		return
	}
	rel = filepath.Clean(rel)
	prefix := rel + string(os.PathSeparator)
	s.mu.Lock()
	defer s.mu.Unlock()
	for path := range s.state {
		if path == rel || hasPathPrefix(path, prefix) {
			delete(s.state, path)
		}
	}
}

func (s *Syncer) statePaths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	paths := make([]string, 0, len(s.state))
	for rel := range s.state {
		paths = append(paths, rel)
	}
	sort.Slice(paths, func(i, j int) bool {
		depthI := pathDepth(paths[i])
		depthJ := pathDepth(paths[j])
		if depthI == depthJ {
			return paths[i] < paths[j]
		}
		return depthI < depthJ
	})
	return paths
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

func pathDepth(path string) int {
	if path == "." || path == "" {
		return 0
	}
	depth := 1
	for _, ch := range path {
		if ch == os.PathSeparator {
			depth++
		}
	}
	return depth
}

func collapsePending(entries []pendingEntry) []pendingEntry {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].rel == entries[j].rel {
			return entries[i].tree && !entries[j].tree
		}
		return entries[i].rel < entries[j].rel
	})

	collapsed := make([]pendingEntry, 0, len(entries))
	for _, item := range entries {
		skip := false
		for _, existing := range collapsed {
			if existing.tree && (item.rel == existing.rel || hasPathPrefix(item.rel, existing.rel+string(os.PathSeparator))) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		collapsed = append(collapsed, item)
	}
	return collapsed
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
