# Future

## Remote build cache via SSH+rsync

A shared build cache for teams, using nothing but SSH and rsync on a
plain Linux/BSD server. No new daemons, no SDK dependencies.

### Storage layout

```
/var/creo-cache/<project-id>/
  <sha256-of-inputs>.json              # manifest
  <sha256-of-inputs>/                  # artifact directory
    build_linux_amd64.bin
    build_linux_arm64.bin
    server_linux_amd64.oci.tar
```

### Manifest format

```json
{
  "key": "abc123def...",
  "go_version": "go1.23.4 darwin/arm64",
  "created": "2026-06-09T12:00:00Z",
  "artifacts": [
    {"name": "build_linux_amd64.bin", "sha256": "f00...", "size": 5242880, "type": "binary"},
    {"name": "server_linux_amd64.oci.tar", "sha256": "baa...", "size": 10485760, "type": "oci"}
  ]
}
```

### Two-level cache flow

- L1: local `.creo/cache/` (same as today, fast, no network)
- L2: remote SSH path (checked only on L1 miss)

Build loop (in `runner.go`):

1. Compute input hash
2. Check local manifest -> hit: skip build
3. Check remote manifest via `rsync -a` -> hit: download artifacts, write local, skip build
4. Build
5. Upload artifacts via `rsync -a`, write local + remote manifests

### Configuration

```fiat
build: go
    cache-remote=ssh://build-cache.example.com/var/creo-cache/my-project
```

CLI flag (overrides fiat): `--cache-remote user@host:/path`
Env var (overrides all): `CREO_CACHE_REMOTE`

SSH key config lives in `~/.ssh/config` -- no code change needed.

### GC (cron tool)

`creo cache gc` subcommand, run daily via cron:

- `--max-age` (default 7d): delete manifests + artifacts older than this
- `--max-size` (default 10G): delete oldest entries until under this
- Operates entirely over SSH: `ssh user@host "rm -rf ..."`

### Open questions

- Should the manifest stay at the project-ID level or be per-target?
  (Per-target means more round-trips but smaller downloads on partial hits.)

- If a remote hit returns partial results (say 2 of 3 artifacts), should
  we build the missing ones locally or treat it as a miss? (Partial-hit
  rebuild with local upload of only the missing artifacts could save time
  on flaky CI jobs.)

- Should the remote cache path embed the fiat file's git SHA so different
  branches are isolated? (Prevents cross-branch cache poisoning at the
  cost of fewer hits on shared branches.)

- For watch mode: skip remote check entirely, or have an opt-in flag?
  (Watch is local iteration; network round-trips every second would be
  painful.)

- Should uploading be synchronous (blocking the build) or spawn a
  background goroutine? (Fire-and-forget is faster for the developer but
  risks losing artifacts if the machine dies.)

- How to handle HTTP proxy environments (corporate laptops where SSH to
  the cache server goes through a jump host)? (ProxyJump in
  `~/.ssh/config` is already transparent to rsync, but we could document
  it.)
