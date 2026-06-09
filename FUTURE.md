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

## OCI build cache

A cache backend that uses OCI artifact manifests in any container
registry — zero new infrastructure. Auth reuses the same mechanism
as `oci.Push` (keychain, `ociuser`/`ocipass`, `ocicred=`).

### How it maps

```
<registry>/<target>/<sha256-of-inputs>
  └── artifact manifest (artifactType: application/vnd.creo.cache)
      ├── config blob: {"key":"abc...", "go_version":"go1.23", "created":"..."}
      └── blobs: one per `cache-paths=` file
```

### Configuration

File-level default, per-target override:

```fiat
$CACHE_OCI=ghcr.io/myorg/creo-cache

build: go
    cache-paths=$bin

server: oci
    require=build
    cache-oci=ghcr.io/team/other-cache
    cache-paths=$tarball
```

Resolution: target `cache-oci=` → `$CACHE_OCI` fiat var → `--cache-oci` flag
→ `CREO_CACHE_OCI` env.  `cache-paths` is repeatable, space-separated,
defaults to `$bin` if set, else nothing.

### Flow

1. Compute input hash (same as today)
2. Check local `.creo/cache/<target>.json` → hit: skip
3. Check registry: `HEAD` the manifest at `<cache-oci>/<target>/<sha256>` → hit:
   pull blob(s), unpack to expected paths, write local cache → skip
4. Build
5. Push: create artifact manifest, upload blobs, tag with hash

### Retrieval policy

`HEAD` request (cheap, no body) with 30-day cache freshness.  If the
manifest is found and 30 days haven't elapsed since the local cache
was written, skip the remote check entirely.  On miss, rebuild and
push the new entry.

### GC

Registry-native retention policies (most registries support tag-based
retention).  No custom cron needed.

### Differences from SSH+rsync

| Aspect | SSH+rsync | OCI cache |
|--------|-----------|-----------|
| Infrastructure | Dedicated SSH server | Nothing — uses your existing registry |
| Auth | SSH keys | Same as push/pull (keychain, helpers) |
| GC | Custom cron over SSH | Registry-native retention policies |
| Speed | rsync incremental (diff only) | Full blob downloads |
| Large blobs | Incremental transfer | Full download each time |

OCI cache is simpler to set up (no SSH server).  SSH+rsync is better
for large outputs.  Both are L2 backends behind the same L1 local cache.
If both `cache-oci` and `cache-remote` are set, OCI is tried first
(faster round-trip for `HEAD`), then SSH.

## Artifact registries (push / pull / remote targets)

OCI registries can store arbitrary files — not just container images.
The OCI Artifact Manifest spec (1.1+) lets you push any binary, tarball,
SBOM, or report with a custom `artifactType`.  No base image needed.

### New CLI subcommands

```
creo push <target>       # push binary/tarball as OCI artifact
creo pull <ref> [path]   # download artifact blob to disk
```

`creo push` reads `cache-oci`/`cache-paths` to determine destination
and what to upload — it's the same artifact upload the OCI cache does
automatically, just invoked manually.

`creo pull` fetches the artifact manifest, finds the blob descriptor,
and downloads it to the given path (or stdout).

### Remote target type

A target that resolves its binary from a remote registry pull instead
of a local build:

```fiat
server: remote
    use=ghcr.io/myorg/server-bin:latest
    os=linux
    arch=amd64
```

The `remote` language:
- Pulls the artifact matching the combo's platform
- Sets `$bin` to the downloaded file path
- Makes `$OUTPUT_server` available to requiring targets

Resolution: target `use=` → implicit from `cache-oci` if target name
matches  → error.

### Wire it together

```fiat
$CACHE_OCI=ghcr.io/myorg/creo-cache

build: go
    cache-paths=$bin

image: oci
    require=build
    tag=latest
```

With `creo push build`, the project pushes the build binary as an OCI
artifact.  Another project pulls it with a `remote` target:

```fiat
lib: remote
    use=ghcr.io/myorg/creo-cache/build:latest

myapp: go
    require=lib
    # $OUTPUT_lib is the downloaded binary
```

### Multi-platform artifacts

When `arch=`/`os=` are set, each combo gets its own tag:

```
<registry>/<target>:<sha256>-<arch>-<os>
```

Or use a manifest list (fat manifest) referencing each platform's
artifact blob.

### `artifact-type` property

Override the default `application/vnd.creo.binary` media type:

```fiat
build: go
    artifact-type=application/vnd.creo.sbom
```

### Implementation

| File | Δ lines | Change |
|------|---------|--------|
| `internal/oci/artifact.go` | ~120 | `PushArtifact`, `PullArtifact`, artifact manifest structs |
| `internal/runner/cache.go` | ~100 | `pushCacheOCI`, `pullCacheOCI` — calls PushArtifact/PullArtifact |
| `internal/runner/runner.go` | ~30 | OCI cache hooks after build, remote target execution |
| `internal/lang/remote.go` | ~30 | `applyRemote` — parse `use=`, `artifact-type=` |
| `main.go` | ~20 | `push`/`pull` subcommands |
| `internal/fiat/types.go` | +3 | `Use`, `CacheOCI`, `CachePaths` on `Target` |

~300 lines total, zero new Go dependencies.

## Summary of planned cache layers

```
L1: .creo/cache/<target>.json          (local, always checked first)
L2: OCI registry (remote, cheap HEAD check)
L3: SSH+rsync     (remote, for large/incremental artifacts)
```

All three share the same input hash computation.  L2 and L3 are
optional and independent of each other.

## Open questions

### Cache

- OCI cache: should `HEAD` check the manifest digest against a local
  copy, or just check existence?  (Digest comparison catches registry
  re-tags, but adds a header round-trip.)

- Should `cache-paths` default to `$bin` for compile targets and
  `$tarball` for OCI targets?  (Sensible default, but surprising if
  the user didn't expect caching.)

- For multi-combo builds with OCI cache: push one blob per combo or
  one manifest list with all platforms?  (Flat per-combo tags are
  simpler but don't let CI skip platforms it doesn't need.)

### Artifact registries

- Should `remote` targets be a separate language or just a property
  on any target?  (E.g. `build: go use=ghcr.io/...` to override the
  build with a remote pull.  Magical but concise.)

- `creo push` without any `cache-oci` configured: error, or push to
  the registry implied by the target's `repo=`?  (Pushing to `repo=`
  is sensible for release workflows.)

- When pushing multi-platform as a manifest list, should the list
  use `application/vnd.oci.image.index.v1+json` (the standard fat
  manifest) or a custom list type?  (Standard is better for
  interoperability, but registries may reject non-image index types.)

- How should `creo pull` handle version resolution?  `latest` tag,
  semver ranges (`~1.2`), or explicit SHA256 pinning?  (Semver ranges
  are useful but require a separate tag-to-version mapping.)

### SSH cache

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
