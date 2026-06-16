# Archive

Create `.tar.gz` or `.zip` release archives containing the binary plus
documentation and other files.

## Target

```
archive: archive
    require=build
```

| Property | Default | Description |
|----------|---------|-------------|
| `format` | `tar.gz` | Archive format (`tar.gz` or `zip`) |
| `require` | — | Build target whose output to package |

The archive includes the binary at the top level, plus any files from
a `manifest.ini` `[files]` section. Without a manifest, `README.md`
and `LICENSE` are included automatically if they exist.

## Init

```
creo -i archive
```

Adds an `archive: archive` target requiring `build`.

## Manifest

See [manifest.md](manifest.md) for the full manifest reference.
