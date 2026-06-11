# creo

A build tool that respects your time.  A Go project builds from a single
line.  Variables, rebuild detection, cross-compilation, language
auto-configuration, and recursive builds are all built in — no arcane
syntax to memorise.

## Quick start

```sh
$ creo -i          # create a bare project file
$ creo -i go       # initialise a Go project
$ creo -i go:1.25  # initialise with a pinned Go toolchain
$ creo             # build
$ creo -v          # see what's happening
$ creo all         # run every target
$ ./bootstrap.sh   # build from source and install to ~/bin
```

A minimal Go project needs only this in a file named `fiat`:

```
build: go
```

creo picks up the directory name as the binary, compiles all `.go` files,
strips debug symbols, and only rebuilds when sources change.

## Anatomy of a fiat file

```
# Variables
$GO=go build

build: go
    sources=*.go
    cmd=$GO $args $GOFLAGS -o $bin $sources
```

Variables start with `$`; `=` is lazy, `:=` is eager.  Targets list a
name, an optional language keyword, and indented properties.

## Documentation

Detailed references live in the `docs/` directory:

| Topic | File |
|-------|------|
| Common format (variables, targets, properties) | [docs/fiat.md](docs/fiat.md) |
| Go | [docs/go.md](docs/go.md) |
| C / C++ | [docs/c.md](docs/c.md) |
| Rust | [docs/rust.md](docs/rust.md) |
| Python | [docs/python.md](docs/python.md) |
| Node / TypeScript | [docs/node.md](docs/node.md) |
| Java / Kotlin / Gradle | [docs/java.md](docs/java.md) |
| OCI container images | [docs/oci.md](docs/oci.md) |
| CLI reference | [docs/cli.md](docs/cli.md) |

## Why not Make?

No `$(eval ...)`, no `.PHONY`, no `.SUFFIXES`, no conditionals.
Just variables with `$`, targets with properties, and shell commands.
Language support makes the common case — a Go project — a single line.
OCI image building is built in.

## License

MIT
