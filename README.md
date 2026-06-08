# creo

A build tool that respects your time. A Go project builds from a single
line. Variables, rebuild detection, cross-compilation, language
auto-configuration, and recursive builds are all built in — no arcane
syntax to memorise.

## Quick start

```sh
$ creo -i          # create a bare project file
$ creo -i go       # initialise a Go project
$ creo -i go:1.25  # initialise with a pinned Go toolchain
$ creo -i c        # initialise a C project
$ creo -i cxx      # initialise a C++ project
$ creo -i oci      # initialise a container image project
$ creo -i go oci   # initialise multiple languages
$ creo             # build
$ creo -v          # see what's happening
$ creo all         # run every target
```

Running `creo -i go` in a directory with some files already present
creates only the ones that are missing — safe to run repeatedly.

A minimal Go project:

```
build: go
```

That's it. `creo` picks up the directory name as the binary, compiles all
`.go` files in the current directory, strips debug symbols, and checks
whether the binary is newer than the sources before rebuilding.

## Format

A file named `fiat` (or `*.fiat` if you run without one) defines
variables and targets.

### Variables

```
$GO=go build
$GOFLAGS := -trimpath -ldflags="-s -w"
```

Variables start with `$`.  `=` is lazy (re-evaluated every time), `:=`
is eager (expanded once at parse time).  Reference them with `$NAME`
or `$(NAME)` — parentheses let you append text directly (e.g. `$(bin)-debug`).

Two built-in variables are available in every target:

| Variable | Value |
|---|---|
| `$THIS` | The target's own name (`"build"`, `"debug"`, etc.) |
| `$DIR` | Absolute path to the directory containing the fiat file |

After a dependency completes, `$OUTPUT_<name>` is set to its binary
path for the requiring target.

Lines starting with `#` are comments.  Inline `#` (on property lines)
strips the rest of the line.

### Targets

```
target-name:
	property=value
```

A target starts with its name, a colon, and optionally a language
keyword.  After the language, you can set target-local variables with
`KEY=VALUE` pairs:

```
build: go NAME=myapp VER=1.0
	cmd=echo "Building $NAME v$VER"
```

Properties are indented with **one tab** — required, not
optional.  Two tabs continue the previous line's value.

```
build: go
	sources=*.go
		**.c
		**/*.h
	require=lint
	tmp=*.o
```

If no target is given on the command line, `build` runs.  The special
target `all` runs every target in order, starting with `build` and its
dependencies.

### Language mode

A target with a language keyword gets automatic defaults:

```
target: go
```

| Language | `bin=` | `cmd=` | `sources=` |
|---|---|---|---|
| `go` | `./<name>` (from `go.mod`) | `$GO $GOFLAGS -o $bin` | `*.go go.mod go.sum` |
| `c` | `./<name>` (from directory) | `$CC $CFLAGS $LDFLAGS -o $bin $sources $LIBS` | `*.c` |
| `cxx` / `cpp` | `./<name>` (from directory) | `$CXX $CXXFLAGS $LDFLAGS -o $bin $sources $LIBS` | `*.cpp` |
| `oci` | — | — (packaging-only; uses `$OUTPUT_<target>` from required build) | — |

For `go`: `build` targets get release flags; `debug` and any target
ending in `-debug` get debug flags.  Define `$GOFLAGS` to override.

For `c`: `build` targets get `$CFLAGS` (`-O2 -Wall`); `debug` targets
get `$CDEBUGFLAGS` (`-O0 -g -Wall`).  Same pattern for `cxx`/`cpp`
with `$CXXFLAGS` / `$CXXDEBUGFLAGS`.

All variables are overridable in the fiat file.

For `oci`: packages a compiled binary into an OCI container image and
writes a tarball or pushes to a registry.  OCI targets are packaging-only
(no compilation themselves) — use `require=` to reference a Go build
target and `$OUTPUT_<target>` to locate its binary.  Example:

```
build: go

image: oci
    repo=ghcr.io/myorg/myapp
    tag=latest
    require=build
```

After `build` completes, its binary path is available as `$OUTPUT_build`.
The image places the binary at `/app/<name>` (override with `appdir=`).

### OCI properties

| Property | What it does |
|---|---|
| `repo=` | Container registry (e.g. `ghcr.io/user/repo`) |
| `tag=` | Image tag (default: `latest` for tarball; push uses this if set) |
| `tarball=` | Write image as a docker-compatible `.tar` file |
| `appdir=` | Directory in the image for the binary (default: `/app`) |
| `ociuser=` | Registry username (for basic auth) |
| `ocipass=` | Registry password or token |

If no `tarball=` is set, the image is only pushed.  If no `repo=` is
set, the image is only written to a tarball.  Auth: when both
`ociuser=` and `ocipass=` are set, basic auth is used; otherwise the
default Docker keychain (`~/.docker/config.json`) is consulted.

### Output variables

When a dependency target produces a binary (`bin=` property), its path
is available to the requiring target as `$OUTPUT_<name>`.  This replaces
the old `ko` `$SRCDIR` approach and works for any language:

```
build: go

image: oci
    appdir=/srv
    require=build
    repo=ghcr.io/myorg/myapp
    tag=latest
```

Here `build` sets `$bin`, which becomes `$OUTPUT_build` for the `image`
target.  The `oci` language target reads it as the binary source for the
container layer.

### Legacy `ko` migration

The `language: ko` keyword is no longer recognised.  If you have an
existing `ko` target, replace it with:

```
image: oci
    repo=<your-registry>
    tarball=<path>.tar
    require=<build-target>
```

Where `<build-target>` is the name of your Go build target.  The binary
path is automatically injected via `$OUTPUT_<target>`.

### Multi-arch and multi-OS

```
nix: go
	os=linux freebsd
	arch=amd64 arm64
	bin=$bin-$os-$arch
```

`os` and `arch` can take space-separated values.  The runner builds
every combination — the example above produces four binaries.  Clean
and skip checks respect all combinations.

Multi-arch builds run in parallel by default (one goroutine per CPU).
Use `-j N` to control concurrency, or `-j 1` for serial execution.

Cross-compilation environment variables are set per language:

| Language | Env vars set |
|---|---|
| `go` | `GOARCH`, `GOOS` (used by Go toolchain) |
| `c`, `cxx`, `cpp` | none (C/C++ cross-compilers must be configured via `$CC` / `$CXX`) |
| `oci` | none (packaging-only target) |

For C/C++ cross-compilation, set `$CC` or `$CXX` to the target
toolchain prefix:

```
nix: c
    os=linux
    arch=arm64
    $CC=aarch64-linux-gnu-gcc
```

### Install

```
install: go
	install=$bin:$HOME/bin/
	require=build
```

The `install=` property copies files after a target runs.  Format is
`source:destination`; when only a destination is given, source defaults
to the target's binary (`$bin`).  Environment variables like `$HOME` are
expanded so paths like `$HOME/bin/` work naturally.

Multiple `install=` lines are allowed.  Combined with `$(bin)-debug`
(parenthesised references) and `require=` this handles complex setups:

```
install: go
	install=$bin:$HOME/bin/
	install=$(bin)-debug:$HOME/bin/
	require=build debug
```

The install phase runs unconditionally — binaries are always copied,
even when the source already exists.  `creo -c` removes installed files
alongside the build artefacts from `bin=`.

### Properties

| Property | What it does |
|---|---|
| `cmd=` | Shell command to run (repeatable — runs in sequence) |
| `bin=` | Path to the output binary |
| `sources=` | File patterns checked for rebuild detection |
| `tmp=` | Files cleaned before and after the target |
| `require=` | Targets that must run first |
| `desc=` | Human-readable description shown by `creo -l` |
| `install=` | Copy built binaries to a destination (repeatable — see below) |
| `arch=` | Architecture for cross-compile (space-separated; sets per-language env) |
| `os=` | OS for cross-compile (space-separated; sets per-language env) |
| `args=` | Extra arguments injected into the default command (empty by default) |
| `repo=` | Container registry for OCI images (e.g. `ghcr.io/user/repo`) |
| `tag=` | Image tag for OCI (default: `latest` for tarball) |
| `tarball=` | Path to write an OCI image tarball |
| `appdir=` | Directory in the OCI image for the binary (default: `/app`) |
| `ociuser=` | Registry username (basic auth) |
| `ocipass=` | Registry password or token |

Source patterns: `*` matches files in the current directory, `**.go` and
`**/*.go` match `.go` files recursively, and `src/**/*.go` matches `.go`
files under `src/` only. When a binary already exists and is newer than
all sources, creo skips it with a message.

### Built-in variables

When not explicitly defined by the user:

| Variable | Default |
|---|---|
| `$GO` | `go build` |
| `$GOFLAGS` | `-trimpath -ldflags="-s -w"` (release) or `-gcflags="all=-N -l"` (debug) |
| `$GODEBUGFLAGS` | `-gcflags="all=-N -l"` |
| `$APPDIR` | `/app` (OCI app directory; overridable per target via `appdir=`) |
| `$CC` | `cc` (C compiler) |
| `$CFLAGS` | `-O2 -Wall` (release), `$CDEBUGFLAGS`: `-O0 -g -Wall` (debug) |
| `$CXX`, `$CPP` | `c++` (C++ compiler) |
| `$CXXFLAGS`, `$CPPFLAGS` | `-O2 -Wall` (release), `$CXXDEBUGFLAGS`: `-O0 -g -Wall` (debug) |
| `$LDFLAGS` | *(empty — override for `-L` flags)* |
| `$LIBS` | *(empty — override for `-lm -lpthread`)* |
| `$SRCDIR` | *(empty — override to build from a sub-package)* |
| `$VERSION` | Inferred from `git describe --tags` (see below) |
| `$COMMIT` | Short commit hash from `git rev-parse --short HEAD` |
| `$DATE` | Current UTC timestamp (ISO 8601) |

`$VERSION` is derived from Git history at parse time:

| Repo state | Example (`$VERSION`) |
|---|---|
| No tags at all | `dev` |
| Exact tag, clean | `v0.1.0` |
| Exact tag, uncommitted changes | `v0.1.0-dirty` |
| Commits after tag, clean | `v0.1.0-3-a1b2c3d4` |
| Commits after tag, dirty | `v0.1.0-3-a1b2c3d4-dirty` |

Outside a Git repo, `$VERSION` defaults to `dev`.

`creo --version` prints the embedded version string.  Release and debug
builds inject it automatically via `-X main.version=$VERSION` in the
linker flags.  Define `$VERSION := custom` in the fiat file to override
it, or use `$VERSION` in any `cmd=` or `bin=` expression.

### Virtual targets

A target name starting with `.` *(dot targets)* is virtual — it has no
output file and always runs.  Useful for tests, linting, or release
tasks:

```
.test: go
    cmd=go test ./...
```

Virtual targets get no language defaults (no auto `bin`/`cmd`/`sources`).
Give them what you need, or just a `cmd=` with no language at all.
Dependencies resolve normally; clean silently skips them.

### Target listing

```
creo -l
```

Prints every target, its language, and its `desc=` description:

```
  build       (go)   Build the project binary
  debug       (go)   Debug build with symbols
  install     (go)   Build and install to ~/bin
```

Add a description to any target with the `desc=` property.

### Watch mode

```
creo -w [target]
```

Watches a target's source files and rebuilds on every change.  Useful
during development — edit, save, and the build happens automatically.
The default target is `build`.  Polls every second (no external
dependencies).

### Parallel builds

Multi-arch targets (targets with multiple `arch=` or `os=` values)
build each combination in parallel.  Use `-j N` to limit concurrency:

```
creo -j 2 nix
```

Without `-j`, the number of CPUs is used.  `-j 1` runs serially.

### Shell completion

```
creo --completion
```

Outputs a bash completion script providing tab completion for flags,
target names, and `init` languages.  Install it with:

```
source <(creo --completion)
```

Add it to your `~/.bashrc` for persistence.

## CLI

```
creo [flags] [target...]
```

| Flag | Description |
|---|---|
| `-i`, `--init` | Initialise project (optionally with languages: `go`/`go:1.25`/`c`/`cxx`/`cpp`/`oci`; multiple accepted) |
| `-f`, `--file` | Alternative fiat file path |
| `-F`, `--force` | Force rebuild |
| `-l`, `--list` | List available targets with descriptions |
| `-w`, `--watch` | Watch sources and rebuild on change |
| `-k`, `--keep-going` | Continue past errors, report all at the end |
| `-n`, `--dry-run` | Print commands and install actions without executing |
| `-j`, `--jobs` | Parallel jobs for multi-arch builds (default: CPU count) |
| `-r`, `--recursive` | Walk subdirectories for fiat files |
| `-c`, `--clean` | Remove target binaries and installed files |
| `-v`, `--verbose` | Show what's happening |
| `--completion` | Print bash shell completion script |
| `--version` | Print version and exit |
| `-h`, `--help` | Show help |

Targets are positional: `creo debug test` runs both.  Without targets,
`build` is the default.  `all` runs every target.

Error messages include the fiat file path and line number by default:

```
Error: fiat:12: install of ./creo: no such file or directory
```

## Why not Make?

No `$(eval ...)`, no `.PHONY`, no `.SUFFIXES`, no `ifeq`/`else`/`endif`.
Just variables with `$`, targets with properties, and shell commands.
Language support makes the common case — a Go project — a single line.
OCI image building is built in — no external `ko` binary needed.

## License

MIT
