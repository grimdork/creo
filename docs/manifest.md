# Package manifest

A `manifest.ini` file in the project root provides shared metadata,
file lists, scripts, dependencies, and architecture overrides for
all packaging targets (archive, deb, rpm, brew) and OCI images.

## Lookup order

1. `manifest=<path>` property on the target (overrides default)
2. `manifest.ini` next to the fiat file
3. Best-effort: binary + `README.md` + `LICENSE` only

## Sections

### `[package]`

```
[package]
maintainer=Developer <dev@example.com>
vendor=Example Corp
homepage=https://github.com/user/project
license=MIT
section=contrib
priority=extra
description=My awesome project
```

Fields are consumed by deb/rpm targets and used as fallback
metadata for brew and archive.

### `[depends]`, `[recommends]`, `[suggests]`

```
[depends]
libc6 = >=2.31

[recommends]
ca-certificates
```

Package dependencies. The format is `name = version_constraint`.
If the version constraint is empty, the package name is used as-is.

### `[files]`

Local files to include. Format: `dst = src`.

```
[files]
/usr/share/doc/$PROJECT/FAQ.md = FAQ.md
```

Paths are relative to the fiat file directory. Variable expansion
is not applied to manifest file and download paths.

### `[download]`

URLs fetched at build time and included in OCI images.
Format: `dst = url`.

```
[download]
/usr/lib/libfoo.so = https://example.com/libfoo.so
```

Files are downloaded once during the build and placed at the
specified path inside the image. Only public (unauthenticated)
URLs are supported.

### `[scripts]`

Pre/post installation scripts for deb/rpm packages.

```
[scripts]
preinstall = scripts/preinst.sh
postinstall = scripts/postinst.sh
preremove = scripts/prerm.sh
postremove = scripts/postrm.sh
```

### `[arch:<name>]`

Override values for a specific architecture. Any section key
can appear inside an arch block. The arch name matches what
creo passes as `$arch` (e.g. `amd64`, `arm64`).

```
[arch:arm64]
depends = libc6 >=2.35
```
