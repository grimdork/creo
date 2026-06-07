# creo

A build tool with simpler rules than Make. Variables, rebuild detection,
cross-compilation, and recursive builds are built in.

## Quick start

```sh
$ creo -i          # create a fiat file
$ creo             # build it
$ creo -v          # see what's happening
$ creo all         # run every target
```

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
is eager (expanded once at parse time).  Reference them with `$NAME`.

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
| `cmd=` | `$GO <flags> -o $bin` |
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

### Properties

| Property | What it does |
|---|---|
| `cmd=` | Shell command to run (repeatable — runs in sequence) |
| `bin=` | Path to the output binary |
| `sources=` | File patterns checked for rebuild detection |
| `tmp=` | Files cleaned before and after the target |
| `require=` | Targets that must run first |
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

## CLI

```
creo [flags] [target...]
```

| Flag | Description |
|---|---|
| `-i`, `--init` | Create `fiat` and `.gitignore` |
| `-f`, `--force` | Force rebuild |
| `-r`, `--recursive` | Walk subdirectories for fiat files |
| `-c`, `--clean` | Remove target binaries |
| `-v`, `--verbose` | Show what's happening |
| `-h`, `--help` | Show help |

Targets are positional: `creo debug test` runs both.  Without targets,
`build` is the default.  `all` runs every target.

## Why not Make?

No `$(eval ...)`, no `.PHONY`, no `.SUFFIXES`, no `ifeq`/`else`/`endif`.
Just variables with `$`, targets with properties, and shell commands.
Language support makes the common case — a Go project — a single line.

## License

MIT
