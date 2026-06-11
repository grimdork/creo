# Rust

Initialise a Rust project:

```sh
$ creo -i rust
```

This runs `cargo init` and creates a `fiat` file.

## Defaults

| Property | Value |
|---|---|
| `bin=` | `$BUILDDIR/release/<crate>` — from `Cargo.toml` `[package] name`, or directory name |
| `cmd=` | `$CARGO build --release $args` |
| `sources=` | `*.rs Cargo.toml Cargo.lock` |

`debug` targets (and names ending in `-debug`) use:
- `bin=` → `$BUILDDIR/debug/<crate>`
- `cmd=` → `$CARGO build $args` (no `--release`)

## Variables

| Variable | Default |
|---|---|
| `$CARGO` | `cargo` |

## Cross-compilation

Targets with `arch=` or `os=` set `CARGO_BUILD_TARGET` to the
appropriate Rust triple:

| arch | linux | darwin/macos | windows |
|---|---|---|---|
| `amd64` / `x86_64` | `x86_64-unknown-linux-gnu` | `x86_64-apple-darwin` | `x86_64-pc-windows-msvc` |
| `arm64` / `aarch64` | `aarch64-unknown-linux-gnu` | `aarch64-apple-darwin` | `aarch64-pc-windows-msvc` |
| `arm` | `armv7-unknown-linux-gnueabihf` | — | — |
