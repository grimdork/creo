# creo

A build tool with a simpler, more readable format than Make. Variable
expansion, rebuild detection, multi-command targets, and recursive builds
are baked in — no arcane syntax needed.

## Quick start

```sh
# Bootstrap a new project
$ creo -i

# Edit the generated "fiat" file...
```

A minimal build rule:

```
$CC=gcc
$CFLAGS := -O2 -Wall

build:
        bin=./myapp
        cmd=$CC $CFLAGS -o $bin main.c
        sources=*.c
```

```
$ creo              # build (or "up to date")
$ creo -r           # force rebuild
$ creo debug        # run the debug target
$ creo -R           # drill into subdirectories
```

## Format

### Variables

```
$KEY=value
$KEY:=value
```

The `=` form re-evaluates variable references every time (lazy). The `:=`
form expands once at parse time (eager).

### Targets

```
target-name:
	property=value
```

A target starts with its name followed by a colon, at the start of a line
(no indentation).  **One tab of indentation is required** for every
property line under a target.  This is not optional — the parser enforces
it.

If no target is specified on the command line, `build` runs.  The special
target `all` runs every target in the file.

### Properties

| Property | Purpose |
|---|---|
| `bin=` | Path to the output binary |
| `cmd=` | Shell command to run (repeatable — runs in sequence) |
| `sources=` | File patterns for rebuild detection |
| `tmp=` | Temporary files to clean before and after the target |
| `require=` | Targets that must run first, in order |

Source patterns: `*` matches files in the current directory, `**.ext`
matches recursively.  If `bin` is newer than all sources, the target is
skipped.

### Multi-line properties

`require`, `sources`, `tmp` (and `cmd`, `bin`, or custom variables) can
span multiple lines.  A line indented with **two tabs** (`\t\t`) continues
the value of the preceding property.

```
build:
	require=dep1
		dep2
		dep3
	sources=*.go
		**.c
		**/*.h
	tmp=*.o *.tmp
```

This is equivalent to one-liners:

```
build:
	require=dep1 dep2 dep3
	sources=*.go **.c **/*.h
	tmp=*.o *.tmp
```

For `cmd` each continuation adds a separate command that runs sequentially.

### Language targets

A target can specify a language instead of inline properties:

```
target-name: go
```

When the language is `go`, sensible defaults are filled in automatically:

| Property | Default |
|---|---|
| `bin=` | `./<dirname>` (appends `-debug` for targets named `debug`) |
| `cmd=` | `go build <flags> -o $bin` |
| `sources=` | `*.go` (current directory only) |

The default `$GOFLAGS` varies by target name:

| Target | `$GOFLAGS` |
|---|---|
| `build` | `-trimpath -ldflags="-s -w"` |
| `debug` | `-gcflags="all=-N -l"` |
| any other | `-trimpath -ldflags="-s -w"` (release) |

If `$GOFLAGS` is defined explicitly in the fiat file, it takes precedence.

A minimal Go project's fiat file:

```
build: go

debug: go
```

That's it — no variables, no properties, no boilerplate.  `creo` builds
the release binary, `creo debug` builds a debug binary with the name
suffixed with `-debug`.

### Example

```
$APP=creo
$GO=go build
$FLAGS := -trimpath -ldflags="-s -w"

build:
	bin=./$APP
	cmd=$GO $FLAGS -o $bin
	sources=**.go
	require=lint

lint:
	cmd=staticcheck ./...

clean:
	cmd=rm -f ./$APP
	tmp=*.o
```

## CLI

```
creo [flags] [target...]
```

| Flag | Description |
|---|---|
| `-i`, `--init` | Create `fiat`, `.gitignore` (skip existing) |
| `-f`, `--force` | Overwrite with init; force rebuild otherwise |
| `-r`, `--rebuild` | Remove binary before building |
| `-R`, `--recursive` | Walk subdirectories for fiat files |
| `-c`, `--clean` | Remove target binaries |
| `-v`, `--verbose` | Show diagnostic and cleanup output |
| `-h`, `--help` | Show help |

Target names are positional: `creo debug test` runs `debug` then `test`.
Without targets, `build` is the default.

## Why not Make?

No `$(eval ...)` directives, no `.PHONY` targets, no `.SUFFIXES` rules,
no `ifeq`/`else`/`endif` conditionals.  Just variables with `$`
references, inline properties, and shell commands.  `sources`, `tmp`, and
`require` are built in — common patterns that take pages of Make macros
are a single line here.

## License

MIT
