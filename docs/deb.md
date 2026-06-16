# Debian packages

Build `.deb` packages via [nfpm](https://nfpm.goreleaser.com).

## Target

```
deb: deb
    require=build
    maintainer=Developer <dev@example.com>
```

| Property | Default | Description |
|----------|---------|-------------|
| `maintainer` | `git config user` or `packager <root@localhost>` | Package maintainer |
| `vendor` | project name | Organisation name |
| `homepage` | — | Project homepage URL |
| `license` | `MIT` | Package license |
| `section` | `contrib` | Debian section |
| `priority` | `extra` | Package priority |

The binary is installed at `/usr/bin/$PROJECT`. Additional files,
dependencies, scripts, and architecture overrides come from a
shared `manifest.ini`.

## Init

```
creo -i deb
```

## Prerequisites

Install nfpm:

```
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```
