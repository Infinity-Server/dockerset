package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath        = "/etc/netfin.yml"
	DefaultConfigDir         = "/config"
	DefaultDBPath            = "/db/jellyfin.db"
	DefaultFileBackupDir     = "/backup/files"
	DefaultSQLiteBackupDir   = "/backup/sqlite"
	DefaultDebounce          = 2 * time.Second
	DefaultReconcile         = 30 * time.Minute
	DefaultSourceRetry       = 5 * time.Second
	DefaultLogLevel          = "warning"
	DefaultSQLitePattern     = "*.db"
	DefaultSQLiteReplicaType = "file"
)

type Config struct {
	Netfin     NetfinConfig `yaml:"netfin"`
	Litestream Litestream   `yaml:",inline"`
}

type NetfinConfig struct {
	ConfigDir         string        `yaml:"config-dir"`
	FileBackupDir     string        `yaml:"file-backup-dir"`
	Debounce          time.Duration `yaml:"debounce"`
	ReconcileInterval time.Duration `yaml:"reconcile-interval"`
	SourceRetry       time.Duration `yaml:"source-retry"`
	LogLevel          string        `yaml:"loglevel"`
	ExcludePatterns   []string      `yaml:"exclude"`
	DryRun            bool          `yaml:"dry-run"`
}

type Litestream struct {
	DBs []DBConfig `yaml:"dbs"`
}

type DBConfig struct {
	Path               string          `yaml:"path"`
	Dir                string          `yaml:"dir"`
	Pattern            string          `yaml:"pattern"`
	Recursive          bool            `yaml:"recursive"`
	Watch              bool            `yaml:"watch"`
	MetaPath           string          `yaml:"meta-path"`
	MetaDir            string          `yaml:"meta-dir"`
	MonitorInterval    time.Duration   `yaml:"monitor-interval"`
	CheckpointInterval time.Duration   `yaml:"checkpoint-interval"`
	BusyTimeout        time.Duration   `yaml:"busy-timeout"`
	Replica            ReplicaConfig   `yaml:"replica"`
	Replicas           []ReplicaConfig `yaml:"replicas"`
}

type ReplicaConfig struct {
	Type         string        `yaml:"type"`
	Path         string        `yaml:"path"`
	URL          string        `yaml:"url"`
	SyncInterval time.Duration `yaml:"sync-interval"`
}

func Load(path string) (Config, error) {
	cfg := Default()
	buf, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.applyDefaultsAndValidate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	return Config{
		Netfin: NetfinConfig{
			ConfigDir:         DefaultConfigDir,
			FileBackupDir:     DefaultFileBackupDir,
			Debounce:          DefaultDebounce,
			ReconcileInterval: DefaultReconcile,
			SourceRetry:       DefaultSourceRetry,
			LogLevel:          DefaultLogLevel,
			ExcludePatterns:   BaseExcludePatterns(),
		},
		Litestream: Litestream{
			DBs: []DBConfig{
				{
					Path: DefaultDBPath,
					Replica: ReplicaConfig{
						Type: DefaultSQLiteReplicaType,
						Path: DefaultSQLiteBackupDir,
					},
				},
			},
		},
	}
}

func BaseExcludePatterns() []string {
	return []string{
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
	}
}

func DefaultExcludePatterns() []string {
	cfg := Default()
	return cfg.EffectiveExcludePatterns()
}

func ParseConfigFlag(args []string, command string) (string, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	configPath := DefaultConfigPath
	fs.StringVar(&configPath, "config", configPath, "netfin YAML config path")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	return configPath, nil
}

func (c *Config) applyDefaultsAndValidate() error {
	if c.Netfin.ConfigDir == "" {
		c.Netfin.ConfigDir = DefaultConfigDir
	}
	if c.Netfin.FileBackupDir == "" {
		c.Netfin.FileBackupDir = DefaultFileBackupDir
	}
	if c.Netfin.Debounce == 0 {
		c.Netfin.Debounce = DefaultDebounce
	}
	if c.Netfin.ReconcileInterval == 0 {
		c.Netfin.ReconcileInterval = DefaultReconcile
	}
	if c.Netfin.SourceRetry == 0 {
		c.Netfin.SourceRetry = DefaultSourceRetry
	}
	if c.Netfin.LogLevel == "" {
		c.Netfin.LogLevel = DefaultLogLevel
	}
	if c.Netfin.Debounce < 0 || c.Netfin.ReconcileInterval < 0 || c.Netfin.SourceRetry < 0 {
		return fmt.Errorf("durations must be positive")
	}
	switch strings.ToLower(c.Netfin.LogLevel) {
	case "debug", "info", "warn", "warning", "error":
		c.Netfin.LogLevel = strings.ToLower(c.Netfin.LogLevel)
	default:
		return fmt.Errorf("netfin.loglevel must be one of debug, info, warn, warning, error")
	}
	if len(c.Litestream.DBs) == 0 {
		c.Litestream.DBs = Default().Litestream.DBs
	}
	c.Netfin.ExcludePatterns = dedupe(append(BaseExcludePatterns(), c.Netfin.ExcludePatterns...))

	for i := range c.Litestream.DBs {
		db := &c.Litestream.DBs[i]
		if db.Path == "" && db.Dir == "" {
			if i == 0 {
				db.Path = DefaultDBPath
			} else {
				return fmt.Errorf("dbs[%d]: path or dir required", i)
			}
		}
		if db.Pattern == "" {
			db.Pattern = DefaultSQLitePattern
		}
		if db.Path != "" {
			abs, err := filepath.Abs(db.Path)
			if err != nil {
				return err
			}
			db.Path = abs
		}
		if db.Dir != "" {
			abs, err := filepath.Abs(db.Dir)
			if err != nil {
				return err
			}
			db.Dir = abs
		}
		replica, err := normalizeReplica(db)
		if err != nil {
			return fmt.Errorf("dbs[%d]: %w", i, err)
		}
		db.Replica = replica
		db.Replicas = nil
		if db.MetaPath != "" {
			abs, err := filepath.Abs(db.MetaPath)
			if err != nil {
				return err
			}
			db.MetaPath = abs
		}
		if db.MetaDir != "" {
			abs, err := filepath.Abs(db.MetaDir)
			if err != nil {
				return err
			}
			db.MetaDir = abs
		}
	}
	var err error
	c.Netfin.ConfigDir, err = filepath.Abs(c.Netfin.ConfigDir)
	if err != nil {
		return err
	}
	c.Netfin.FileBackupDir, err = filepath.Abs(c.Netfin.FileBackupDir)
	if err != nil {
		return err
	}
	c.Netfin.ExcludePatterns = c.EffectiveExcludePatterns()
	return nil
}

func normalizeReplica(db *DBConfig) (ReplicaConfig, error) {
	if db.Replica.Path != "" || db.Replica.URL != "" || db.Replica.Type != "" {
		if len(db.Replicas) > 0 {
			return ReplicaConfig{}, fmt.Errorf("cannot specify both replica and replicas")
		}
		r := db.Replica
		if r.Type == "" {
			r.Type = DefaultSQLiteReplicaType
		}
		if r.Type != DefaultSQLiteReplicaType {
			return ReplicaConfig{}, fmt.Errorf("only file replica is supported by embedded netfin restore/replicate path")
		}
		if r.Path == "" {
			r.Path = DefaultSQLiteBackupDir
		}
		abs, err := filepath.Abs(r.Path)
		if err != nil {
			return ReplicaConfig{}, err
		}
		r.Path = abs
		return r, nil
	}
	if len(db.Replicas) > 1 {
		return ReplicaConfig{}, fmt.Errorf("multiple replicas are not supported")
	}
	if len(db.Replicas) == 1 {
		db.Replica = db.Replicas[0]
		db.Replicas = nil
		return normalizeReplica(db)
	}
	abs, err := filepath.Abs(DefaultSQLiteBackupDir)
	if err != nil {
		return ReplicaConfig{}, err
	}
	return ReplicaConfig{Type: DefaultSQLiteReplicaType, Path: abs}, nil
}

func (c Config) EffectiveExcludePatterns() []string {
	out := append([]string(nil), c.Netfin.ExcludePatterns...)
	for _, db := range c.Litestream.DBs {
		out = append(out, SQLiteExcludePatterns(db)...)
	}
	return dedupe(out)
}

func SQLiteExcludePatterns(db DBConfig) []string {
	var bases []string
	if db.Pattern != "" {
		bases = append(bases, db.Pattern)
	}
	if db.Path != "" {
		bases = append(bases, filepath.Base(db.Path))
	}
	var out []string
	for _, p := range bases {
		p = filepath.ToSlash(filepath.Clean(p))
		if p == "." || p == "" {
			continue
		}
		out = append(out, p)
		if strings.HasSuffix(p, ".db") || strings.Contains(p, ".db") {
			out = append(out, p+"-wal", p+"-shm", p+"-journal")
		}
		if strings.HasSuffix(p, ".sqlite") || strings.Contains(p, ".sqlite") {
			out = append(out, p+"-wal", p+"-shm")
		}
	}
	return out
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
