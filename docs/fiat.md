# Fiat file format

A file named `fiat` (or files ending in `.fiat`) defines variables and
targets.

## Variables

```
$GO=go build
$GOFLAGS := -trimpath -ldflags="-s -w"
```

Variables start with `$`.  `=` is lazy (re-evaluated every time), `:=`
is eager (expanded once at parse time).  Reference them with `$NAME`
or `$(NAME)` — parentheses let you append text directly, e.g.
`$(bin)-debug`.

Lines starting with `#` are comments.  On property lines, inline `#`
strips the rest of the line.

## Targets

```
target-name: language optkey=optval
    property=value
```

A target starts with its name, a colon, and optionally a language
keyword.  After the language you can set target-local variables:

```
build: go NAME=myapp VER=1.0
    cmd=echo "Building $NAME v$VER"
```

Properties are indented with **one tab** — required, not optional.
Two tabs continue the previous line's value.

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

## Properties

| Property | What it does |
|---|---|
| `cmd=` | Shell command to run (repeatable — runs in sequence) |
| `bin=` | Path to the output binary |
| `sources=` | File patterns checked for rebuild detection |
| `tmp=` | Files cleaned before and after the target |
| `require=` | Targets that must run first |
| `desc=` | Human-readable description shown by `creo -l` |
| `install=` | Copy built binaries to a destination (repeatable) |
| `arch=` | Architecture for cross-compile (space-separated) |
| `os=` | OS for cross-compile (space-separated) |
| `args=` | Extra arguments injected into the default command |

### Source patterns

- `*` — files in the current directory
- `**.go` — `.go` files recursively
- `src/**/*.go` — `.go` files under `src/`

## Built-in variables

When not explicitly defined by the user:

| Variable | Default |
|---|---|
| `$GO` | `go build` |
| `$GOFLAGS` | `-trimpath …` (release) or `-gcflags="all=-N -l"` (debug) |
| `$CC` | `cc` |
| `$CXX` / `$CPP` | `c++` |
| `$PROJECT` | Inferred from project file (`go.mod`, `Cargo.toml`, `pyproject.toml`, `package.json`, `settings.gradle.kts`, `pom.xml`) or directory name |
| `$BUILDDIR` | `build` |
| `$VERSION` | Inferred from `git describe --tags` |
| `$COMMIT` | Short commit hash from `git rev-parse --short HEAD` |
| `$DATE` | Current UTC timestamp (ISO 8601) |
| `$THIS` | The target's own name |
| `$DIR` | Absolute path to the directory containing the fiat file |

### `$VERSION` derivation

| Repo state | Example |
|---|---|
| No tags at all | `dev` |
| Exact tag, clean | `v0.1.0` |
| Exact tag, dirty | `v0.1.0-dirty` |
| Commits after tag, clean | `v0.1.0-3-a1b2c3d4` |
| Commits after tag, dirty | `v0.1.0-3-a1b2c3d4-dirty` |

Outside a Git repo `$VERSION` defaults to `dev`.  Override with
`$VERSION := custom` in the fiat file.

## Output variables

When a dependency target produces a binary, its path is available to
the requiring target as `$OUTPUT_<name>`:

```
build: go

image: oci
    appdir=/srv
    require=build
    repo=ghcr.io/myorg/myapp
    tag=latest
```

When the dependency uses `arch=` or `os=`, each combo gets its own
`$OUTPUT_<name>` — the requiring target reads the value matching its
own `arch`/`os` combination.

## Multi-arch and multi-OS

```
nix: go
    os=linux freebsd
    arch=amd64 arm64
    bin=$bin-$os-$arch
```

`os` and `arch` take space-separated values.  Every combination is
built.  `$bin` here is the language default path; the expansion
produces e.g. `build/test-linux-amd64`.

Cross-compilation environment variables are set per language.

## Install

```
install: go
    install=$bin:$HOME/bin/
    require=build
```

Format is `source:destination`.  When only a destination is given,
source defaults to the target's binary.  Environment variables like
`$HOME` are expanded.

Multiple `install=` lines are allowed:

```
install: go
    install=$bin:$HOME/bin/
    install=$(bin)-debug:$HOME/bin/
    require=build debug
```

The install phase runs unconditionally.  `creo -c` removes installed
files alongside the binary.

## Virtual targets

A target name starting with `.` is virtual — it has no output file and
always runs:

```
.test: go
    cmd=go test ./...
```

Virtual targets get no language defaults.  Dependencies resolve
normally; clean silently skips them.
