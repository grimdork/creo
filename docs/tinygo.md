# TinyGo

Build small, statically-linked Go binaries with [TinyGo](https://tinygo.org)
instead of the standard `go` toolchain.

## Target

```
build: tinygo
```

The `tinygo` language handler sets these defaults:

| Variable | Default |
|----------|---------|
| `$TINYGO` | `tinygo build` |
| `$TINYGOFLAGS` | `-no-debug -panic=trap -scheduler=none` |
| `$PROJECT` | module name from `go.mod`, or directory name |
| `sources` | `*.go go.mod go.sum` |
| `bin` | `$BUILDDIR/$PROJECT` |
| `cmd` | `$TINYGO $TINYGOFLAGS -o $bin` |

Cross-compilation works via `GOOS` / `GOARCH` environment
variables (same as Go).

## Init

```
creo -i tinygo
```

Scaffolds `main.go`, `version.go`, `go.mod`, and a fiat file with
a `build: tinygo` target.

## Prerequisites

Install TinyGo: <https://tinygo.org/getting-started/install/>
