# Jellyfin config backup sidecar

Single-binary Kubernetes sidecar for backing up Jellyfin configuration.

It runs two independent pipelines:

- ordinary files: `fsnotify` plus periodic lightweight reconcile mirrors `netfin.config-dir` into `netfin.file-backup-dir`
- SQLite: embedded Litestream Go library replicates databases configured in Litestream-style `dbs:` entries

All runtime configuration is YAML. The file keeps Litestream-style top-level
configuration and adds a `netfin:` top-level section for this sidecar's own
settings.

## Config

Default config path:

```bash
/etc/netfin.yml
```

Example:

```yaml
netfin:
  config-dir: /config
  file-backup-dir: /backup/files
  debounce: 2s
  reconcile-interval: 30m
  source-retry: 5s
  dry-run: false
  exclude:
    - "*.tmp"

dbs:
  - path: /db/jellyfin.db
    monitor-interval: 1s
    checkpoint-interval: 5s
    replica:
      type: file
      path: /backup/sqlite
      sync-interval: 1s
```

Directory-style Litestream config is supported for sidecar replication:

```yaml
netfin:
  config-dir: /config
  file-backup-dir: /backup/files

dbs:
  - dir: /db
    pattern: "*.db"
    recursive: true
    watch: true
    replica:
      type: file
      path: /backup/sqlite
```

When a Litestream DB entry has `pattern: "*.db"`, netfin automatically adds
`*.db`, `*.db-wal`, `*.db-shm`, and `*.db-journal` to ordinary file exclusions.
For `*.sqlite`, it adds `*.sqlite`, `*.sqlite-wal`, and `*.sqlite-shm`. Explicit
`dbs[].path` names also derive matching `-wal`, `-shm`, and `-journal`
exclusions.

Built-in ordinary-file exclusions also include:

- `SQLiteBackups/**`
- `cache/**`
- `transcodes/**`
- `log/**`

## Run

```bash
make build
./bin/netfin run -config /etc/netfin.yml
```

The sidecar tolerates Jellyfin starting before or after it:

- if `netfin.config-dir` does not exist yet, file sync waits and retries
- if a configured SQLite DB does not exist yet, Litestream replication waits and retries
- a startup deep reconcile repairs stale backups; periodic lightweight reconcile repairs missed source events
- directory-mode DB configs with `watch: true` keep discovering new matching SQLite databases

## Restore

Use the same YAML:

```bash
./bin/netfin restore -config /etc/netfin.yml
```

Restore behavior:

- copies `netfin.file-backup-dir` back into `netfin.config-dir` using atomic file replacement
- restores each explicit `dbs[].path` from its local Litestream file replica
- skips `dbs[].dir` restore entries; configure explicit `dbs[].path` entries for restore
- does not delete extra files already present in `/config`

Run restore while Jellyfin is stopped.

## Consistency model

This tool does not require Jellyfin to stop for backup.

SQLite is backed up through Litestream WAL replication. Ordinary files are
mirrored independently by file events and periodic reconcile. There is no global
transactional consistency between SQLite state and ordinary file state.

## Litestream library

This project embeds `github.com/benbjohnson/litestream` as a Go library and pins
it to `v0.5.14` in `go.mod`. Litestream's library API is not treated as a stable
public API, so upgrades should be tested with real Jellyfin database traffic and
restore drills.

Only local `type: file` replicas are supported by the embedded netfin path.

## Kubernetes sidecar example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jellyfin-netfin
data:
  netfin.yml: |
    netfin:
      config-dir: /config
      file-backup-dir: /backup/files
      debounce: 2s
      reconcile-interval: 30m
      source-retry: 5s
    dbs:
      - path: /db/jellyfin.db
        replica:
          type: file
          path: /backup/sqlite
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jellyfin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jellyfin
  template:
    metadata:
      labels:
        app: jellyfin
    spec:
      containers:
        - name: jellyfin
          image: jellyfin/jellyfin:latest
          volumeMounts:
            - name: config
              mountPath: /config
            - name: db
              mountPath: /db
            - name: backup
              mountPath: /backup
        - name: netfin
          image: ghcr.io/example/netfin:latest
          args:
            - run
            - -config=/etc/netfin/netfin.yml
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
            - name: db
              mountPath: /db
              readOnly: true
            - name: backup
              mountPath: /backup
            - name: netfin-config
              mountPath: /etc/netfin
              readOnly: true
      volumes:
        - name: config
          persistentVolumeClaim:
            claimName: jellyfin-config
        - name: db
          persistentVolumeClaim:
            claimName: jellyfin-db
        - name: backup
          persistentVolumeClaim:
            claimName: jellyfin-backup
        - name: netfin-config
          configMap:
            name: jellyfin-netfin
```
