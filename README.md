# creo

A build tool that respects your time. A Go project builds from a single
line. Variables, rebuild detection, cross-compilation, language
auto-configuration, and recursive builds are all built in — no arcane
syntax to memorise.

## Quick start

```sh
$ creo -i          # create a bare project file
$ creo -i go       # initialise a full Go project
$ creo -i go:1.25  # initialise with a pinned Go toolchain
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

Lines starting with `#` are comments.  Inline `#` (on property lines)
strips the rest of the line.

### Targets

```
target-name:
	property=value
```

A target starts with its name, a colon, and optionally a language
keyword.  Properties are indented with **one tab** — required, not
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

For Go, this fills in:

| Property | Default |
|---|---|
| `bin=` | `./<name>` (from `go.mod` module path; falls back to directory name; `-debug` suffix for targets ending in `-debug`) |
| `cmd=` | `$GO <flags> -o $bin` (only when no `install=` lines present) |
| `sources=` | `*.go` |

Flags vary by target name: `build` and most targets get release flags;
`debug` and any target ending in `-debug` get debug flags.  Define
`$GOFLAGS` in the file to override.

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
|---|---|---|
| `cmd=` | Shell command to run (repeatable — runs in sequence) |
| `bin=` | Path to the output binary |
| `sources=` | File patterns checked for rebuild detection |
| `tmp=` | Files cleaned before and after the target |
| `require=` | Targets that must run first |
| `desc=` | Human-readable description shown by `creo -l` |
| `install=` | Copy built binaries to a destination (repeatable — see below) |
| `arch=` | `GOARCH` value (space-separated for cross-compile) |
| `os=` | `GOOS` value (space-separated for cross-compile) |

Source patterns: `*` matches files in the current directory, `**.ext`
matches recursively.  When a binary already exists and is newer than all
sources, creo skips it with a message.

### Built-in variables

When not explicitly defined by the user:

| Variable | Default |
|---|---|
| `$GO` | `go build` |
| `$GOFLAGS` | `-trimpath -ldflags="-s -w"` (release) or `-gcflags="all=-N -l"` (debug) |
| `$GODEBUGFLAGS` | `-gcflags="all=-N -l"` |
| `$VERSION` | Inferred from `git describe --tags` (see below) |

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

## CLI

```
creo [flags] [target...]
```

| Flag | Description |
|---|---|---|
| `-i`, `--init` | Initialise project (optionally with language, e.g. `go` or `go:1.25`) |
| `-f`, `--force` | Force rebuild |
| `-l`, `--list` | List available targets with descriptions |
| `-w`, `--watch` | Watch sources and rebuild on change |
| `-j`, `--jobs` | Parallel jobs for multi-arch builds (default: CPU count) |
| `-r`, `--recursive` | Walk subdirectories for fiat files |
| `-c`, `--clean` | Remove target binaries and installed files |
| `-v`, `--verbose` | Show what's happening |
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

## License

MIT
