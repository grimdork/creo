# Future

### GC (cron tool)

`creo cache gc` subcommand, run daily via cron:

- `--max-age` (default 7d): delete manifests + artifacts older than this
- `--max-size` (default 10G): delete oldest entries until under this
- `--keep-latest` flag to always leave one built binary of each arch even if it's old
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

## Cache layers

```
L1: .creo/cache/<target>.json          (local, always checked first)
L2: SSH+rsync     (remote, --cache-remote flag or CREO_CACHE_REMOTE env)
```

L1 is always active.  L2 is optional and checked only on L1 miss.
Both share the same input hash computation.

## Init templates

Extend `creo -i` with `--template` to scaffold a ready-to-build
project including fiat file, source code, and README:

```
creo -i go --template web       # Go HTTP server with Dockerfile
creo -i python --template cli   # Python CLI with uv + argparse
creo -i rust --template lib     # Rust library with Cargo + fiat
```

### Template resolution

```
1. --template flag (overrides)
2. ~/.config/creo/templates/<language>/<name>/
3. built-in templates bundled in the binary (embedded fs)
```

Built-in templates ship with the binary (zero network).  Users extend
by placing directories in `~/.config/creo/templates/`.

### Template format

A template directory contains:

| File | Required | Purpose |
|------|----------|---------|
| `template.ini` | yes | Metadata: name, description, language, files |
| `src/` | yes | Source files to copy (file names support `$VAR` expansion) |
| `fiat.tmpl` | no | Fiat file template (variables expanded at scaffold time) |
| `README.md.tmpl` | no | README template |
| `.gitignore.tmpl` | no | Gitignore template |

Example `template.ini`:

```ini
[template]
name=web
description=Go HTTP server with OCI
language=go
files=main.go, go.mod.tmpl

[vars]
PORT=8080
OCI_REPO=ghcr.io/$PROJECT
```

## Local dev server

A first-class dev-loop built into creo, no external watchers needed:

```
creo run [target]     # build then execute
creo dev [target]     # watch, rebuild, restart on change
```

### `creo run`

1. Build `target` (default: `build`)
2. Exec the binary, passing extra `--` args through

```
creo run build -- -port 9090 -debug
creo run build -- --help
```

For interpreted languages (Python, Node), `creo run` invokes the
interpreter directly with the project entrypoint.

### `creo dev`

Like `creo -w` but also starts the process and restarts it on rebuild:

1. Build target
2. Start binary (stdout/stderr forwarded, stdin connected)
3. Watch sources
4. On change: send SIGTERM, wait 3s, SIGKILL, rebuild, restart

Graceful shutdown lets the old process drain connections before the
new one starts.  Flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--sig` | `SIGTERM` | Signal to send on restart |
| `--timeout` | `3s` | Graceful shutdown wait |
| `--no-restart` | false | Exit on first crash instead of restarting |

### Process management

The dev server tracks the child PID, forwards signals (`SIGINT` from
Ctrl-C goes to creo, which forwards to the child before exiting).
Windows support is a non-goal (use WSL).

## Platform config in fiat

Some projects need different settings per OS (e.g. different library
paths, compiler flags, or commands).  Add a `[platform:<os>]` block
in any target to override properties for a specific OS:

```fiat
build: c
    cmd=cc -o $bin $sources

    [platform:darwin]
    cmd=cc -o $bin sources/*.c -framework CoreFoundation

    [platform:linux]
    cmd=cc -o $bin $sources -lrt
```

`<os>` matches `runtime.GOOS` values (`darwin`, `linux`, `freebsd`,
etc.).  Platform blocks can appear inside any target and override any
property.  Outside targets, platform blocks can set file-level
variables:

```fiat
[platform:darwin]
$CC=clang
$EXTRA_FLAGS=-framework CoreFoundation

[platform:linux]
$CC=gcc
$EXTRA_FLAGS=-lrt

build: c
    cmd=$CC -o $bin $sources $EXTRA_FLAGS
```

### Resolution

1. Target-level `[platform:<os>]` block (highest priority)
2. File-level `[platform:<os>]` block
3. Default (non-platform) property value

Platform blocks are transparent during cross-compilation (`arch=`/`os=`
on the target) — the build loop picks the block matching the current
combo's OS.

## CI generation

Auto-generate CI pipeline files from fiat targets:

```
creo init --ci github-actions     # emits .github/workflows/creo.yml
creo init --ci gitlab             # emits .gitlab-ci.yml
creo init --ci woodpecker         # emits .woodpecker.yml
```

The generator reads all targets, their `arch=`/`os=` combos, and
dependency order, then emits a matrix build.  Example for a project
with Go + OCI targets:

```yaml
# .github/workflows/creo.yml (auto-generated)
name: creo CI
on: [push, pull_request]
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest]     # derived from fiat targets
        arch: [amd64, arm64]    # derived from arch= on fiat targets
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: creo build
      - run: creo image
```

### Extending

Users write their own generators as shell scripts or Go plugins
(resolved from `~/.config/creo/ci/`).  The generator receives a JSON
representation of all targets on stdin.

## Remote builds

Bootstrap build tools on a remote machine, installing only what the
local project's fiat files require:

```
creo bootstrap ssh://[user@]host[:port] [flags]
```

### What it does

1. Parses all fiat files in the local project
2. Collects unique language/target types (go, c, rust, python, etc.)
3. Maps each to required packages or toolchains
4. SSHs into the remote, detects OS and distro
5. Installs missing tools (or prints instructions if it can't)
6. Optionally syncs source and runs a test build

### Root vs non-root

| Connection | Go | C/C++ | Rust | Python | nfpm |
|---|---|---|---|---|---|
| `root@host` | system Go via package manager, or download from go.dev | `build-essential` / `Development Tools` | `rustup` system-wide | system python3 + pip | `go install` |
| `user@host` | download tarball to `~/go/` | print: *"ask an admin for build-essential, or use root@host"* | `curl rustup.rs \| sh` | `uv` in user dir | `go install` in user GOBIN |

### Tool-to-package mapping

| Language | Debian/Ubuntu | RHEL/Fedora | Alpine |
|---|---|---|---|
| go | `golang-go` or tarball from go.dev | `golang` or tarball | `go` |
| c/c++ | `build-essential` | `"Development Tools" group` | `build-base` |
| rust | `curl rustup.rs \| sh` | `curl rustup.rs \| sh` | `curl rustup.rs \| sh` |
| python | `python3 python3-pip` | `python3 python3-pip` | `python3 py3-pip` |
| node | `nodejs npm` | `nodejs npm` | `nodejs npm` |
| java | `default-jdk` | `java-latest-openjdk` | `openjdk17` |
| tinygo | tarball from tinygo.org | tarball | tarball |
| nfpm | `go install ...` | `go install ...` | `go install ...` |

### URL format

- `ssh://user@host:22/path` — explicit port and optional remote
  working directory
- `user@host` — shorthand (defaults port 22, no remote path)

### Security

- `root@` accepted for `bootstrap` only (system-wide installs
  require it)
- `root@` **always rejected** for SSH cache (`--cache-remote`,
  `cache-remote=`) — separate enforced check before any rsync
  operation
- Connection uses `~/.ssh/config` transparently (same mechanism
  as rsync cache)

### Output

```
$ creo bootstrap ssh://buildbox.local
🔍 Parsed languages: go, c
📡 Connecting to buildbox.local...
🔧 Detected OS: Debian 12
✅ Go (go1.24.3) — already installed
⏳ build-essential — installing via apt...
✅ build-essential installed
✨ Done. Run 'creo build' on the remote to verify.
```

### Optional flags

| Flag | Description |
|------|-------------|
| `--sync` | rsync the project to the remote before bootstrapping |
| `--test-build` | run `creo build` on the remote after install |
| `--user-install` | force user-mode installs even when connected as root |

### Non-root C/C++

Print: *"C/C++ compilers require system packages (build-essential).
Install via `creo bootstrap ssh://root@host` or ask an admin."*

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
