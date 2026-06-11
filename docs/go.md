# Go

Initialise a Go project:

```sh
$ creo -i go           # latest toolchain
$ creo -i go:1.25      # pin to Go 1.25
```

This creates `go.mod`, `main.go`, and a `fiat` file.

## Defaults

| Property | Value |
|---|---|
| `bin=` | `$BUILDDIR/<name>` — project name from `go.mod` `module` line, or directory name |
| `cmd=` | `$GO $args $GOFLAGS -o $bin` |
| `sources=` | `*.go go.mod go.sum` |

`build` targets get release flags: `-trimpath -ldflags="-s -w -buildid=reproducible -X main.version=$VERSION"`.

`debug` targets (and any target ending in `-debug`) get debug flags:
`-gcflags="all=-N -l" -ldflags="-buildid=reproducible -X main.version=$VERSION"`.

Define `$GOFLAGS` to override entirely.

## Cross-compilation

Targets with `arch=` or `os=` set `GOARCH` / `GOOS` per combo:

```
nix: go
    os=linux
    arch=amd64 arm64
    bin=$bin-$os-$arch
```

## Variables

| Variable | Default (release) | Default (debug) |
|---|---|---|
| `$GO` | `go build` | same |
| `$GOFLAGS` | `-trimpath -ldflags="-s -w -buildid=reproducible -X main.version=$VERSION"` | `-gcflags="all=-N -l" -ldflags="-buildid=reproducible -X main.version=$VERSION"` |
| `$GODEBUGFLAGS` | — | `-gcflags="all=-N -l"` |
| `$SRCDIR` | (empty) | same |
